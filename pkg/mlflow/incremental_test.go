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
	if bodies[0]["run_view_type"] != "ACTIVE_ONLY" {
		t.Errorf("run_view_type = %v, want ACTIVE_ONLY", bodies[0]["run_view_type"])
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
	if bodies[1]["run_view_type"] != "ACTIVE_ONLY" {
		t.Errorf("run_view_type = %v, want ACTIVE_ONLY", bodies[1]["run_view_type"])
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
		Model: func(gateway.MLModelItem) (gateway.SyncResult, error) {
			return gateway.SyncResult{Updated: 1}, nil
		},
		Version: func(item gateway.MLVersionItem) (gateway.SyncResult, error) {
			*versions = append(*versions, item)
			return gateway.SyncResult{Created: 1}, nil
		},
		Evaluation: func(item gateway.MLEvaluationItem) (gateway.SyncResult, error) {
			*evaluations = append(*evaluations, item)
			return gateway.SyncResult{Created: 1}, nil
		},
		ProductionModel: func(item gateway.MLModelItem) (gateway.SyncResult, error) {
			*production = append(*production, item)
			return gateway.SyncResult{Updated: 1}, nil
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
	if summary.Versions.UpToDate != 1 {
		t.Errorf("Versions.UpToDate = %d, want 1", summary.Versions.UpToDate)
	}
	if summary.Versions.Found != 2 {
		t.Errorf("Versions.Found = %d, want 2", summary.Versions.Found)
	}
	if summary.Versions.Found != summary.Versions.UpToDate+summary.Versions.Created+summary.Versions.Updated {
		t.Errorf("Versions = %+v, want found to equal up-to-date + created + updated", summary.Versions)
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

func trainingServer(t *testing.T, active string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/2.0/mlflow/")
		switch path {
		case "experiments/get-by-name":
			_, _ = w.Write([]byte(`{"experiment":{"experiment_id":"1","name":"exp","lifecycle_stage":"active"}}`))
		case "runs/search":
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			if body["run_view_type"] != "ACTIVE_ONLY" {
				t.Errorf("unexpected run_view_type: %v", body["run_view_type"])
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(active))
		default:
			t.Errorf("unexpected MLflow request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestSyncTrainingCollectsEvaluatedModelIDs(t *testing.T) {
	server := trainingServer(t,
		`{"runs":[{"info":{"run_id":"eval-run","experiment_id":"1"},"data":{"tags":[{"key":"mlflow.datasets","value":"[{\"model\":\"old-model\"}]"}]}}]}`)
	defer server.Close()

	project := config.MLProjectRef{Name: "p", Type: "training", Experiments: []config.MLExperimentRef{{Name: "exp"}}}

	var gotRuns []gateway.MLRunItem
	sink := TrainingSink{
		Experiment: func(gateway.MLExperimentItem) (gateway.SyncResult, error) {
			return gateway.SyncResult{Updated: 1}, nil
		},
		Dataset: func(gateway.MLDatasetItem) (gateway.SyncResult, error) {
			return gateway.SyncResult{Created: 1}, nil
		},
		Run: func(item gateway.MLRunItem) (gateway.SyncResult, error) {
			gotRuns = append(gotRuns, item)
			return gateway.SyncResult{Created: 1}, nil
		},
		Artifact: func(gateway.MLArtifactItem) (gateway.SyncResult, error) {
			return gateway.SyncResult{Created: 1}, nil
		},
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
	if summary.Experiments.Found != 1 || summary.Experiments.Updated != 1 {
		t.Errorf("Experiments = %+v, want found 1 updated 1", summary.Experiments)
	}
}

func TestSyncTrainingSyncsEvaluationRunDatasets(t *testing.T) {
	server := trainingServer(t,
		`{"runs":[{"info":{"run_id":"eval-run","experiment_id":"1"},"data":{"tags":[{"key":"mlflow.datasets","value":"[{\"model\":\"old-model\"}]"}]},"inputs":{"dataset_inputs":[{"dataset":{"name":"holdout","digest":"abc123","source_type":"local","source":"/data/holdout"}}]}}]}`)
	defer server.Close()

	project := config.MLProjectRef{Name: "p", Type: "training", Experiments: []config.MLExperimentRef{{Name: "exp"}}}

	var gotRuns []gateway.MLRunItem
	var gotDatasets []gateway.MLDatasetItem
	sink := TrainingSink{
		Experiment: func(gateway.MLExperimentItem) (gateway.SyncResult, error) {
			return gateway.SyncResult{Updated: 1}, nil
		},
		Dataset: func(item gateway.MLDatasetItem) (gateway.SyncResult, error) {
			gotDatasets = append(gotDatasets, item)
			return gateway.SyncResult{Created: 1}, nil
		},
		Run: func(item gateway.MLRunItem) (gateway.SyncResult, error) {
			gotRuns = append(gotRuns, item)
			return gateway.SyncResult{Created: 1}, nil
		},
		Artifact: func(gateway.MLArtifactItem) (gateway.SyncResult, error) {
			return gateway.SyncResult{Created: 1}, nil
		},
	}

	summary, err := SyncTraining(context.Background(), NewClient(server.URL, ""), project, time.UnixMilli(1000).UTC(), sink)
	if err != nil {
		t.Fatalf("SyncTraining: %v", err)
	}

	if len(gotRuns) != 0 {
		t.Errorf("Runs = %+v, want none (evaluation runs are not training runs)", gotRuns)
	}
	if len(gotDatasets) != 1 {
		t.Fatalf("Datasets = %+v, want the evaluation run's dataset", gotDatasets)
	}
	if gotDatasets[0].MLflowID != "abc123" {
		t.Errorf("dataset MLflowID = %q, want abc123", gotDatasets[0].MLflowID)
	}
	if gotDatasets[0].ExperimentMLflowID != "1" {
		t.Errorf("dataset ExperimentMLflowID = %q, want 1", gotDatasets[0].ExperimentMLflowID)
	}
	if summary.Datasets.Found != 1 || summary.Datasets.Created != 1 {
		t.Errorf("Datasets = %+v, want found 1 created 1", summary.Datasets)
	}
}
