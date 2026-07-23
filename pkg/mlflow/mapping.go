package mlflow

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/uigraph-oss/uigraph-cli/pkg/gateway"
)

var experimentStatus = map[string]string{"active": "active", "deleted": "archived"}

var runStatus = map[string]string{
	"RUNNING":   "running",
	"SCHEDULED": "running",
	"FINISHED":  "completed",
	"FAILED":    "failed",
	"KILLED":    "cancelled",
}

var imageExtensions = map[string]bool{"png": true, "jpg": true, "jpeg": true, "gif": true, "svg": true}

func iso(ms *int64) *time.Time {
	if ms == nil {
		return nil
	}
	t := time.UnixMilli(*ms).UTC()
	return &t
}

func duration(startMs, endMs *int64) string {
	if startMs == nil || endMs == nil {
		return ""
	}
	seconds := (*endMs - *startMs) / 1000
	if seconds < 0 {
		seconds = 0
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

func extension(name string) string {
	if !strings.Contains(name, ".") {
		return ""
	}
	idx := strings.LastIndex(name, ".")
	return strings.ToLower(name[idx+1:])
}

func artifactType(name string) string {
	ext := extension(name)
	lower := strings.ToLower(name)
	if ext == "onnx" {
		return "ONNX"
	}
	if ext == "gguf" {
		return "GGUF"
	}
	if ext == "ipynb" {
		return "Notebook"
	}
	if strings.HasPrefix(lower, "confusion") {
		return "Confusion matrix"
	}
	if imageExtensions[ext] {
		return "Plot"
	}
	return "Model checkpoint"
}

func humanSize(size *int64) string {
	if size == nil {
		return ""
	}
	value := float64(*size)
	for _, unit := range []string{"B", "KB", "MB", "GB", "TB"} {
		if value < 1024 || unit == "TB" {
			if unit == "B" {
				return fmt.Sprintf("%d %s", int64(value), unit)
			}
			return fmt.Sprintf("%.1f %s", value, unit)
		}
		value /= 1024
	}
	return ""
}

func tagValue(tags []KeyValue, key string) string {
	for _, tag := range tags {
		if tag.Key == key {
			return tag.Value
		}
	}
	return ""
}

func datasetRowCount(profile string) int64 {
	if profile == "" {
		return 0
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(profile), &parsed); err != nil {
		return 0
	}
	value, ok := parsed["num_rows"].(float64)
	if !ok {
		return 0
	}
	return int64(value)
}

func datasetSchema(schema string) []gateway.MLSchemaField {
	if schema == "" {
		return []gateway.MLSchemaField{}
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		return []gateway.MLSchemaField{}
	}
	columns, ok := parsed["mlflow_colspec"].([]any)
	if !ok {
		return []gateway.MLSchemaField{}
	}
	fields := make([]gateway.MLSchemaField, 0, len(columns))
	for _, c := range columns {
		column, ok := c.(map[string]any)
		if !ok {
			continue
		}
		name, _ := column["name"].(string)
		typ, _ := column["type"].(string)
		fields = append(fields, gateway.MLSchemaField{Name: name, Type: typ, Description: ""})
	}
	return fields
}

func normalizeContext(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "training" || v == "train" {
		return "training"
	}
	if v == "evaluation" || v == "eval" || v == "test" || v == "validation" {
		return "evaluation"
	}
	return "training"
}

func inputContext(di DatasetInput) string {
	for _, tag := range di.Tags {
		if tag.Key == "mlflow.data.context" {
			return normalizeContext(tag.Value)
		}
	}
	return "training"
}

func experimentToItem(exp *Experiment, projectName string) gateway.MLExperimentItem {
	status, ok := experimentStatus[exp.LifecycleStage]
	if !ok {
		status = "active"
	}
	return gateway.MLExperimentItem{
		MLflowID:    exp.ExperimentID,
		ProjectName: projectName,
		Name:        exp.Name,
		Description: tagValue(exp.Tags, "mlflow.note.content"),
		Status:      status,
		StartedAt:   iso(exp.CreationTime),
	}
}

func firstDatasetMLflowID(run Run) *string {
	if run.Inputs == nil || len(run.Inputs.DatasetInputs) == 0 {
		return nil
	}
	d := run.Inputs.DatasetInputs[0].Dataset
	id := d.Digest
	if id == "" {
		id = d.Name
	}
	return &id
}

func runToItem(run Run) gateway.MLRunItem {
	status, ok := runStatus[run.Info.Status]
	if !ok {
		status = "running"
	}
	name := tagValue(run.Data.Tags, "mlflow.runName")
	if name == "" {
		name = run.Info.RunID
	}
	parameters := map[string]any{}
	for _, p := range run.Data.Params {
		parameters[p.Key] = p.Value
	}
	metrics := map[string]any{}
	for _, m := range run.Data.Metrics {
		metrics[m.Key] = m.Value
	}
	return gateway.MLRunItem{
		MLflowID:           run.Info.RunID,
		ExperimentMLflowID: run.Info.ExperimentID,
		DatasetMLflowID:    firstDatasetMLflowID(run),
		Name:               name,
		Status:             status,
		StartedAt:          iso(run.Info.StartTime),
		EndedAt:            iso(run.Info.EndTime),
		Duration:           duration(run.Info.StartTime, run.Info.EndTime),
		Notes:              tagValue(run.Data.Tags, "mlflow.note.content"),
		Parameters:         parameters,
		Metrics:            metrics,
	}
}

func metricHistoryToPoints(history []Metric) []gateway.MLSeriesPoint {
	points := make([]gateway.MLSeriesPoint, 0, len(history))
	for _, m := range history {
		points = append(points, gateway.MLSeriesPoint{
			Key:   m.Key,
			Step:  m.Step,
			Value: m.Value,
			TS:    iso(m.Timestamp),
		})
	}
	return points
}

func modelToItem(model *RegisteredModel, projectName string, productionVersionMLflowID *string) gateway.MLModelItem {
	tags := make([]string, 0, len(model.Tags))
	for _, tag := range model.Tags {
		tags = append(tags, tag.Key)
	}
	return gateway.MLModelItem{
		MLflowID:                  model.Name,
		ProjectName:               projectName,
		Name:                      model.Name,
		Description:               model.Description,
		Tags:                      tags,
		ProductionVersionMLflowID: productionVersionMLflowID,
		CreatedAt:                 iso(model.CreationTimestamp),
		UpdatedAt:                 iso(model.LastUpdatedTimestamp),
	}
}

func versionToItem(v ModelVersion) gateway.MLVersionItem {
	var runID *string
	if v.RunID != "" {
		runID = &v.RunID
	}
	return gateway.MLVersionItem{
		MLflowID:      fmt.Sprintf("%s/%s", v.Name, v.Version),
		ModelMLflowID: v.Name,
		RunMLflowID:   runID,
		Version:       v.Version,
		Description:   v.Description,
		CreatedAt:     iso(v.CreationTimestamp),
	}
}

func artifactToItem(runID string, f FileInfo) gateway.MLArtifactItem {
	name := f.Path
	if idx := strings.LastIndex(f.Path, "/"); idx >= 0 {
		name = f.Path[idx+1:]
	}
	return gateway.MLArtifactItem{
		MLflowID:    fmt.Sprintf("%s/%s", runID, f.Path),
		RunMLflowID: runID,
		Name:        name,
		Type:        artifactType(name),
		URI:         f.Path,
		Size:        humanSize(f.FileSize),
		Format:      extension(name),
	}
}

func datasetToItem(d Dataset, experimentMLflowID, context string) gateway.MLDatasetItem {
	id := d.Digest
	if id == "" {
		id = d.Name
	}
	return gateway.MLDatasetItem{
		MLflowID:           id,
		ExperimentMLflowID: experimentMLflowID,
		Name:               d.Name,
		Digest:             d.Digest,
		Source:             d.Source,
		SourceType:         d.SourceType,
		Context:            context,
		RowCount:           datasetRowCount(d.Profile),
		Schema:             datasetSchema(d.Schema),
		Tags:               map[string]string{},
	}
}
