package mlflow

import "testing"

func ptr[T any](v T) *T { return &v }

func TestDuration(t *testing.T) {
	tests := []struct {
		start, end *int64
		want       string
	}{
		{ptr(int64(0)), ptr(int64(3661000)), "1h 1m"},
		{ptr(int64(0)), ptr(int64(303000)), "5m 3s"},
		{ptr(int64(0)), ptr(int64(3000)), "3s"},
		{ptr(int64(1000)), ptr(int64(0)), "0s"},
		{nil, ptr(int64(1000)), ""},
		{ptr(int64(1000)), nil, ""},
	}
	for _, tt := range tests {
		if got := duration(tt.start, tt.end); got != tt.want {
			t.Errorf("duration(%v,%v) = %q, want %q", tt.start, tt.end, got, tt.want)
		}
	}
}

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
	if item.Duration != "3s" {
		t.Errorf("Duration = %q, want 3s", item.Duration)
	}
	if item.Metrics["acc"] != 0.9 {
		t.Errorf("Metrics[acc] = %v, want 0.9", item.Metrics["acc"])
	}
	if item.Parameters["lr"] != "0.01" {
		t.Errorf("Parameters[lr] = %v, want 0.01", item.Parameters["lr"])
	}
}
