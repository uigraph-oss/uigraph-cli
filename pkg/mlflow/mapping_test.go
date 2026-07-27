package mlflow

import (
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }

func TestHumanSize(t *testing.T) {
	tests := []struct {
		size *int64
		want string
	}{
		{nil, ""},
		{ptr(int64(512)), "512 B"},
		{ptr(int64(2048)), "2.0 KB"},
		{ptr(int64(5 * 1024 * 1024)), "5.0 MB"},
	}
	for _, tt := range tests {
		if got := humanSize(tt.size); got != tt.want {
			t.Errorf("humanSize(%v) = %q, want %q", tt.size, got, tt.want)
		}
	}
}

func TestArtifactType(t *testing.T) {
	tests := map[string]string{
		"model.onnx":         "ONNX",
		"weights.gguf":       "GGUF",
		"explore.ipynb":      "Notebook",
		"confusion_plot.png": "Confusion matrix",
		"loss.png":           "Plot",
		"model.pt":           "Model checkpoint",
	}
	for name, want := range tests {
		if got := artifactType(name); got != want {
			t.Errorf("artifactType(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestDatasetRowCountAndSchema(t *testing.T) {
	if got := datasetRowCount(`{"num_rows": 42}`); got != 42 {
		t.Errorf("datasetRowCount = %d, want 42", got)
	}
	if got := datasetRowCount("not json"); got != 0 {
		t.Errorf("datasetRowCount(invalid) = %d, want 0", got)
	}
	fields := datasetSchema(`{"mlflow_colspec":[{"name":"age","type":"long"}]}`)
	if len(fields) != 1 || fields[0].Name != "age" || fields[0].Type != "long" {
		t.Errorf("datasetSchema = %+v", fields)
	}
}

func TestVersionToItem(t *testing.T) {
	item := versionToItem(ModelVersion{Name: "Saba", Version: "3", CurrentStage: "Production", RunID: "run-1"})
	if item.MLflowID != "Saba/3" {
		t.Errorf("MLflowID = %q, want Saba/3", item.MLflowID)
	}
	if item.RunMLflowID == nil || *item.RunMLflowID != "run-1" {
		t.Errorf("RunMLflowID = %v, want run-1", item.RunMLflowID)
	}
	empty := versionToItem(ModelVersion{Name: "Saba", Version: "1"})
	if empty.RunMLflowID != nil {
		t.Errorf("RunMLflowID = %v, want nil", empty.RunMLflowID)
	}
}

func TestRunToItem(t *testing.T) {
	run := Run{
		Info: RunInfo{RunID: "run-1", ExperimentID: "exp-1", Status: "FINISHED", StartTime: ptr(int64(0)), EndTime: ptr(int64(3000))},
		Data: RunData{
			Metrics: []Metric{{Key: "acc", Value: 0.9}},
			Params:  []KeyValue{{Key: "lr", Value: "0.01"}},
			Tags:    []KeyValue{{Key: "mlflow.runName", Value: "sunny-otter"}},
		},
	}
	item := runToItem(run)
	if item.Name != "sunny-otter" {
		t.Errorf("Name = %q, want sunny-otter", item.Name)
	}
	if item.Status != "completed" {
		t.Errorf("Status = %q, want completed", item.Status)
	}
	if item.Metrics["acc"] != 0.9 {
		t.Errorf("Metrics[acc] = %v, want 0.9", item.Metrics["acc"])
	}
	if item.Parameters["lr"] != "0.01" {
		t.Errorf("Parameters[lr] = %v, want 0.01", item.Parameters["lr"])
	}
}

func TestExperimentToItem(t *testing.T) {
	exp := Experiment{
		ExperimentID:   "exp-1",
		Name:           "Anomaly detection - fraud",
		LifecycleStage: "active",
		Tags: []KeyValue{
			{Key: "mlflow.note.content", Value: "Fraud detection sweep"},
			{Key: "team", Value: "search"},
			{Key: "baseline"},
		},
	}
	item := experimentToItem(&exp, "Training Project")
	if item.Description != "Fraud detection sweep" {
		t.Errorf("Description = %q, want Fraud detection sweep", item.Description)
	}
	if len(item.Tags) != 2 {
		t.Fatalf("Tags = %v, want 2 entries", item.Tags)
	}
	if item.Tags[0] != "team: search" {
		t.Errorf("Tags[0] = %q, want team: search", item.Tags[0])
	}
	if item.Tags[1] != "baseline" {
		t.Errorf("Tags[1] = %q, want baseline", item.Tags[1])
	}
}

func TestIsEvaluationRun(t *testing.T) {
	evaluation := Run{
		Info: RunInfo{RunID: "eval-1"},
		Data: RunData{Tags: []KeyValue{
			{Key: "mlflow.datasets", Value: `[{"name":"dataset","hash":"abc","model":"m-123"}]`},
		}},
	}
	if !IsEvaluationRun(evaluation) {
		t.Error("IsEvaluationRun(evaluation run) = false, want true")
	}

	training := Run{
		Info:    RunInfo{RunID: "train-1"},
		Data:    RunData{Tags: []KeyValue{{Key: "mlflow.runName", Value: "nosy-bee-242"}}},
		Outputs: &RunOutputs{ModelOutputs: []ModelOutput{{ModelID: "m-123"}}},
	}
	if IsEvaluationRun(training) {
		t.Error("IsEvaluationRun(training run) = true, want false")
	}

	selfEvaluating := Run{
		Info: RunInfo{RunID: "train-2"},
		Data: RunData{Tags: []KeyValue{
			{Key: "mlflow.datasets", Value: `[{"name":"dataset","hash":"abc","model":"m-456"}]`},
		}},
		Outputs: &RunOutputs{ModelOutputs: []ModelOutput{{ModelID: "m-456"}}},
	}
	if IsEvaluationRun(selfEvaluating) {
		t.Error("IsEvaluationRun(run evaluating the model it produced) = true, want false")
	}

	plain := Run{Info: RunInfo{RunID: "train-3"}}
	if IsEvaluationRun(plain) {
		t.Error("IsEvaluationRun(run without datasets tag) = true, want false")
	}
}

func TestLoggedModelIDFromSource(t *testing.T) {
	if got := loggedModelIDFromSource("models:/m-d1cbd09ebea3"); got != "m-d1cbd09ebea3" {
		t.Errorf("loggedModelIDFromSource() = %q, want m-d1cbd09ebea3", got)
	}
	if got := loggedModelIDFromSource("models:/m-d1cbd09ebea3/artifacts"); got != "m-d1cbd09ebea3" {
		t.Errorf("loggedModelIDFromSource() = %q, want m-d1cbd09ebea3", got)
	}
	if got := loggedModelIDFromSource("s3://bucket/path"); got != "" {
		t.Errorf("loggedModelIDFromSource(non-model source) = %q, want empty", got)
	}
}

func TestEvaluationToItem(t *testing.T) {
	end := int64(1784812074532)
	run := Run{
		Info: RunInfo{RunID: "47e63bc7", ExperimentID: "813", EndTime: &end},
		Data: RunData{
			Tags: []KeyValue{
				{Key: "mlflow.runName", Value: "eval-Saba-v2-s1000"},
				{Key: "mlflow.user", Value: "sayad"},
				{Key: "eval_seed", Value: "1000"},
			},
		},
	}
	metrics := []Metric{
		{Key: "r2_score", Value: -0.09, RunID: "47e63bc7", DatasetName: "dataset", DatasetDigest: "2c55db12"},
		{Key: "r2_score", Value: -0.09, RunID: "47e63bc7", DatasetName: "dataset", DatasetDigest: "2c55db12"},
		{Key: "max_error", Value: 0.525, RunID: "47e63bc7", DatasetName: "dataset", DatasetDigest: "2c55db12"},
	}

	item := evaluationToItem(run, "Saba/2", metrics)
	if item.MLflowID != "Saba/2/47e63bc7" {
		t.Errorf("MLflowID = %q, want Saba/2/47e63bc7", item.MLflowID)
	}
	if item.Type != "Offline Benchmark" {
		t.Errorf("Type = %q, want Offline Benchmark", item.Type)
	}
	if item.VersionMLflowID != "Saba/2" {
		t.Errorf("VersionMLflowID = %q, want Saba/2", item.VersionMLflowID)
	}
	if item.ExperimentMLflowID != "813" {
		t.Errorf("ExperimentMLflowID = %q, want 813", item.ExperimentMLflowID)
	}
	if item.Name != "eval-Saba-v2-s1000" {
		t.Errorf("Name = %q, want eval-Saba-v2-s1000", item.Name)
	}
	if item.Evaluator != "sayad" {
		t.Errorf("Evaluator = %q, want sayad", item.Evaluator)
	}
	if len(item.Metrics) != 2 {
		t.Errorf("Metrics = %v, want 2 deduplicated entries", item.Metrics)
	}
	if item.Parameters["eval_seed"] != "1000" {
		t.Errorf("Parameters[eval_seed] = %v, want 1000", item.Parameters["eval_seed"])
	}
	if _, ok := item.Parameters["mlflow.user"]; ok {
		t.Error("Parameters contains mlflow.user, want mlflow.* tags excluded")
	}
	if item.DatasetMLflowID == nil || *item.DatasetMLflowID != "2c55db12" {
		t.Errorf("DatasetMLflowID = %v, want 2c55db12", item.DatasetMLflowID)
	}
	if item.EndedAt == nil || !item.EndedAt.Equal(time.UnixMilli(end).UTC()) {
		t.Errorf("EndedAt = %v, want run end time", item.EndedAt)
	}
	if item.StartedAt.IsZero() {
		t.Error("StartedAt is zero, want fallback to current time")
	}
}

func TestEvaluationToItemMissingTimestamps(t *testing.T) {
	before := time.Now().UTC()
	run := Run{
		Info: RunInfo{RunID: "47e63bc7", ExperimentID: "813"},
		Data: RunData{Tags: []KeyValue{{Key: "mlflow.runName", Value: "eval-Saba-v2"}}},
	}

	item := evaluationToItem(run, "Saba/2", nil)
	if item.StartedAt.Before(before) {
		t.Errorf("StartedAt = %v, want at or after %v", item.StartedAt, before)
	}
	if item.EndedAt != nil {
		t.Errorf("EndedAt = %v, want nil for an unfinished run", item.EndedAt)
	}
}

func TestRunToItemMissingTimestamps(t *testing.T) {
	before := time.Now().UTC()
	run := Run{
		Info: RunInfo{RunID: "run-2", ExperimentID: "exp-1", Status: "RUNNING"},
		Data: RunData{Tags: []KeyValue{{Key: "mlflow.runName", Value: "windy-otter"}}},
	}

	item := runToItem(run)
	if item.StartedAt.Before(before) {
		t.Errorf("StartedAt = %v, want at or after %v", item.StartedAt, before)
	}
	if item.EndedAt != nil {
		t.Errorf("EndedAt = %v, want nil for an unfinished run", item.EndedAt)
	}
}
