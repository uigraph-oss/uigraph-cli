package mlflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/uigraph-oss/uigraph-cli/pkg/config"
	"github.com/uigraph-oss/uigraph-cli/pkg/gateway"
)

func TestSearchRunsFilter(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"runs":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	since := time.UnixMilli(1700000000000).UTC()
	discard := func([]Run) error { return nil }

	if err := client.SearchRuns(context.Background(), "1", time.Time{}, discard); err != nil {
		t.Fatalf("SearchRuns(zero): %v", err)
	}
	if _, ok := bodies[0]["filter"]; ok {
		t.Errorf("zero since sent a filter: %v", bodies[0]["filter"])
	}
	if _, ok := bodies[0]["order_by"]; ok {
		t.Errorf("zero since sent an order_by: %v", bodies[0]["order_by"])
	}

	if err := client.SearchRuns(context.Background(), "1", since, discard); err != nil {
		t.Fatalf("SearchRuns(since): %v", err)
	}
	want := "attributes.start_time > 1700000000000"
	if bodies[1]["filter"] != want {
		t.Errorf("filter = %v, want %q", bodies[1]["filter"], want)
	}
	if bodies[1]["order_by"] == nil {
		t.Error("expected order_by alongside the filter")
	}
}

func TestVersionChangedSince(t *testing.T) {
	since := time.UnixMilli(1000).UTC()

	tests := []struct {
		name    string
		version ModelVersion
		since   time.Time
		want    bool
	}{
		{"zero watermark always changed", ModelVersion{LastUpdatedTimestamp: ptr(int64(500))}, time.Time{}, true},
		{"missing timestamp treated as changed", ModelVersion{}, since, true},
		{"newer than watermark", ModelVersion{LastUpdatedTimestamp: ptr(int64(1001))}, since, true},
		{"equal to watermark", ModelVersion{LastUpdatedTimestamp: ptr(int64(1000))}, since, false},
		{"older than watermark", ModelVersion{LastUpdatedTimestamp: ptr(int64(999))}, since, false},
	}
	for _, tt := range tests {
		if got := versionChangedSince(tt.version, tt.since); got != tt.want {
			t.Errorf("%s: versionChangedSince = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func modelServer(t *testing.T, versions []ModelVersion) (*httptest.Server, *[]string) {
	t.Helper()
	var fetched []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/2.0/mlflow/")
		switch {
		case path == "registered-models/get":
			_, _ = w.Write([]byte(`{"registered_model":{"name":"Saba"}}`))
		case path == "model-versions/search":
			out, _ := json.Marshal(map[string]any{"model_versions": versions})
			_, _ = w.Write(out)
		case strings.HasPrefix(path, "logged-models/"):
			modelID := strings.TrimPrefix(path, "logged-models/")
			fetched = append(fetched, modelID)
			_, _ = w.Write([]byte(fmt.Sprintf(
				`{"model":{"info":{"model_id":%q,"source_run_id":"train-run"},"data":{"metrics":[{"key":"f1","value":0.9,"run_id":"eval-run","model_id":%q}]}}}`,
				modelID, modelID)))
		case path == "runs/get":
			_, _ = w.Write([]byte(`{"run":{"info":{"run_id":"eval-run","experiment_id":"1"},"data":{}}}`))
		default:
			t.Errorf("unexpected MLflow request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server, &fetched
}

func collectModels(versions *[]gateway.MLVersionItem, evaluations *[]gateway.MLEvaluationItem, production *[]gateway.MLModelItem) ModelSink {
	return ModelSink{
		Model: func(gateway.MLModelItem) error { return nil },
		Version: func(item gateway.MLVersionItem) error {
			*versions = append(*versions, item)
			return nil
		},
		Evaluation: func(item gateway.MLEvaluationItem) error {
			*evaluations = append(*evaluations, item)
			return nil
		},
		ProductionModel: func(item gateway.MLModelItem) error {
			*production = append(*production, item)
			return nil
		},
	}
}

func TestSyncModelsSkipsUnchangedVersions(t *testing.T) {
	versions := []ModelVersion{
		{Name: "Saba", Version: "1", LastUpdatedTimestamp: ptr(int64(500)), Source: "models:/old-model/artifacts"},
		{Name: "Saba", Version: "2", LastUpdatedTimestamp: ptr(int64(1500)), Source: "models:/new-model/artifacts"},
	}
	server, fetched := modelServer(t, versions)
	defer server.Close()

	project := config.MLProjectRef{Name: "p", Type: "model", Models: []config.MLModelRef{{Name: "Saba"}}}
	since := time.UnixMilli(1000).UTC()

	var gotVersions []gateway.MLVersionItem
	var gotEvaluations []gateway.MLEvaluationItem
	var gotProduction []gateway.MLModelItem
	summary, err := SyncModels(context.Background(), NewClient(server.URL, ""), project, since, map[string]bool{}, collectModels(&gotVersions, &gotEvaluations, &gotProduction))
	if err != nil {
		t.Fatalf("SyncModels: %v", err)
	}

	if len(gotVersions) != 1 || gotVersions[0].MLflowID != "Saba/2" {
		t.Errorf("Versions = %+v, want only Saba/2", gotVersions)
	}
	if summary.UnchangedVersions != 1 {
		t.Errorf("UnchangedVersions = %d, want 1", summary.UnchangedVersions)
	}
	if len(*fetched) != 1 || (*fetched)[0] != "new-model" {
		t.Errorf("fetched logged models = %v, want only new-model", *fetched)
	}
}

func TestSyncModelsScansEvaluatedUnchangedVersion(t *testing.T) {
	versions := []ModelVersion{
		{Name: "Saba", Version: "1", LastUpdatedTimestamp: ptr(int64(500)), Source: "models:/old-model/artifacts"},
	}
	server, fetched := modelServer(t, versions)
	defer server.Close()

	project := config.MLProjectRef{Name: "p", Type: "model", Models: []config.MLModelRef{{Name: "Saba"}}}
	since := time.UnixMilli(1000).UTC()

	var gotVersions []gateway.MLVersionItem
	var gotEvaluations []gateway.MLEvaluationItem
	var gotProduction []gateway.MLModelItem
	_, err := SyncModels(context.Background(), NewClient(server.URL, ""), project, since, map[string]bool{"old-model": true}, collectModels(&gotVersions, &gotEvaluations, &gotProduction))
	if err != nil {
		t.Fatalf("SyncModels: %v", err)
	}

	if len(gotVersions) != 0 {
		t.Errorf("Versions = %+v, want none (the version row itself did not change)", gotVersions)
	}
	if len(*fetched) != 1 || (*fetched)[0] != "old-model" {
		t.Errorf("fetched logged models = %v, want old-model", *fetched)
	}
	if len(gotEvaluations) != 1 || gotEvaluations[0].VersionMLflowID != "Saba/1" {
		t.Errorf("Evaluations = %+v, want one for Saba/1", gotEvaluations)
	}
}

func TestSyncModelsKeepsProductionVersionWhenUnchanged(t *testing.T) {
	versions := []ModelVersion{
		{Name: "Saba", Version: "1", LastUpdatedTimestamp: ptr(int64(500)), CurrentStage: "Production"},
		{Name: "Saba", Version: "2", LastUpdatedTimestamp: ptr(int64(1500))},
	}
	server, _ := modelServer(t, versions)
	defer server.Close()

	project := config.MLProjectRef{Name: "p", Type: "model", Models: []config.MLModelRef{{Name: "Saba"}}}

	var gotVersions []gateway.MLVersionItem
	var gotEvaluations []gateway.MLEvaluationItem
	var gotProduction []gateway.MLModelItem
	_, err := SyncModels(context.Background(), NewClient(server.URL, ""), project, time.UnixMilli(1000).UTC(), map[string]bool{}, collectModels(&gotVersions, &gotEvaluations, &gotProduction))
	if err != nil {
		t.Fatalf("SyncModels: %v", err)
	}

	if len(gotProduction) != 1 {
		t.Fatalf("ProductionModel emitted %d times, want 1", len(gotProduction))
	}
	production := gotProduction[0].ProductionVersionMLflowID
	if production == nil || *production != "Saba/1" {
		t.Errorf("ProductionVersionMLflowID = %v, want Saba/1", production)
	}
}

func TestSyncTrainingCollectsEvaluatedModelIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/2.0/mlflow/")
		switch path {
		case "experiments/get-by-name":
			_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"1","name":"exp","lifecycle_stage":"active"}}`))
		case "runs/search":
			_, _ = w.Write([]byte(`{"runs":[{"info":{"run_id":"eval-run","experiment_id":"1"},"data":{"tags":[{"key":"mlflow.datasets","value":"[{\"model\":\"old-model\"}]"}]}}]}`))
		default:
			t.Errorf("unexpected MLflow request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	project := config.MLProjectRef{Name: "p", Type: "training", Experiments: []config.MLExperimentRef{{Name: "exp"}}}

	var gotRuns []gateway.MLRunItem
	sink := TrainingSink{
		Experiment: func(gateway.MLExperimentItem) error { return nil },
		Dataset:    func(gateway.MLDatasetItem) error { return nil },
		Run: func(item gateway.MLRunItem) error {
			gotRuns = append(gotRuns, item)
			return nil
		},
		Artifact: func(gateway.MLArtifactItem) error { return nil },
	}

	summary, err := SyncTraining(context.Background(), NewClient(server.URL, ""), project, time.UnixMilli(1000).UTC(), sink)
	if err != nil {
		t.Fatalf("SyncTraining: %v", err)
	}

	if len(gotRuns) != 0 {
		t.Errorf("Runs = %+v, want none (evaluation runs are not training runs)", gotRuns)
	}
	if len(summary.EvaluatedModelIDs) != 1 || summary.EvaluatedModelIDs[0] != "old-model" {
		t.Errorf("EvaluatedModelIDs = %v, want [old-model]", summary.EvaluatedModelIDs)
	}
}
