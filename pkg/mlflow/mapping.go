package mlflow

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/uigraph-oss/uigraph-cli/pkg/gateway"
)

const userTagKey = "uigraph.user"

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

func nowIfNil(t *time.Time) time.Time {
	if t == nil {
		return time.Now().UTC()
	}
	return *t
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

func normalizeTags(tags []KeyValue) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if strings.HasPrefix(tag.Key, "mlflow.") || tag.Key == userTagKey {
			continue
		}
		if tag.Value == "" {
			out = append(out, tag.Key)
			continue
		}
		out = append(out, fmt.Sprintf("%s: %s", tag.Key, tag.Value))
	}
	return out
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
		Tags:        normalizeTags(exp.Tags),
		UserEmail:   tagValue(exp.Tags, userTagKey),
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

func runToItem(run Run, experimentEmail string) gateway.MLRunItem {
	status, ok := runStatus[run.Info.Status]
	if !ok {
		status = "running"
	}
	name := tagValue(run.Data.Tags, "mlflow.runName")
	if name == "" {
		name = run.Info.RunID
	}
	userEmail := tagValue(run.Data.Tags, userTagKey)
	if userEmail == "" {
		userEmail = experimentEmail
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
		StartedAt:          nowIfNil(iso(run.Info.StartTime)),
		EndedAt:            iso(run.Info.EndTime),
		Notes:              tagValue(run.Data.Tags, "mlflow.note.content"),
		Tags:               normalizeTags(run.Data.Tags),
		Parameters:         parameters,
		Metrics:            metrics,
		UserEmail:          userEmail,
	}
}

func evaluatedModelIDs(run Run) []string {
	raw := tagValue(run.Data.Tags, "mlflow.datasets")
	if raw == "" {
		return nil
	}
	var entries []struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil
	}
	seen := map[string]bool{}
	var ids []string
	for _, e := range entries {
		if e.Model == "" || seen[e.Model] {
			continue
		}
		seen[e.Model] = true
		ids = append(ids, e.Model)
	}
	return ids
}

func producedModelIDs(run Run) map[string]bool {
	produced := map[string]bool{}
	if run.Outputs == nil {
		return produced
	}
	for _, out := range run.Outputs.ModelOutputs {
		if out.ModelID != "" {
			produced[out.ModelID] = true
		}
	}
	return produced
}

func IsEvaluationRun(run Run) bool {
	produced := producedModelIDs(run)
	for _, id := range evaluatedModelIDs(run) {
		if !produced[id] {
			return true
		}
	}
	return false
}

func loggedModelIDFromSource(source string) string {
	const prefix = "models:/"
	if !strings.HasPrefix(source, prefix) {
		return ""
	}
	id := strings.TrimPrefix(source, prefix)
	if idx := strings.Index(id, "/"); idx >= 0 {
		id = id[:idx]
	}
	return id
}

func evaluationToItem(run Run, versionMLflowID string, metrics []Metric, versionEmail string) gateway.MLEvaluationItem {
	name := tagValue(run.Data.Tags, "mlflow.runName")
	if name == "" {
		name = run.Info.RunID
	}
	userEmail := tagValue(run.Data.Tags, userTagKey)
	if userEmail == "" {
		userEmail = versionEmail
	}
	parameters := map[string]any{}
	for _, p := range run.Data.Params {
		parameters[p.Key] = p.Value
	}
	values := map[string]any{}
	var datasetMLflowID *string
	for _, m := range metrics {
		values[m.Key] = m.Value
		if m.DatasetDigest != "" && datasetMLflowID == nil {
			digest := m.DatasetDigest
			datasetMLflowID = &digest
		}
	}
	return gateway.MLEvaluationItem{
		MLflowID:           fmt.Sprintf("%s/%s", versionMLflowID, run.Info.RunID),
		VersionMLflowID:    versionMLflowID,
		ExperimentMLflowID: run.Info.ExperimentID,
		DatasetMLflowID:    datasetMLflowID,
		Name:               name,
		Type:               "Offline Benchmark",
		Description:        tagValue(run.Data.Tags, "mlflow.note.content"),
		Summary:            "",
		StartedAt:          nowIfNil(iso(run.Info.StartTime)),
		EndedAt:            iso(run.Info.EndTime),
		Evaluator:          tagValue(run.Data.Tags, "mlflow.user"),
		Tags:               normalizeTags(run.Data.Tags),
		Parameters:         parameters,
		Metrics:            values,
		UserEmail:          userEmail,
	}
}

func modelToItem(model *RegisteredModel, projectName string, productionVersionMLflowID *string) gateway.MLModelItem {
	return gateway.MLModelItem{
		MLflowID:                  model.Name,
		ProjectName:               projectName,
		Name:                      model.Name,
		Description:               model.Description,
		Tags:                      normalizeTags(model.Tags),
		ProductionVersionMLflowID: productionVersionMLflowID,
		CreatedAt:                 iso(model.CreationTimestamp),
		UpdatedAt:                 iso(model.LastUpdatedTimestamp),
		UserEmail:                 tagValue(model.Tags, userTagKey),
	}
}

func versionToItem(v ModelVersion, modelEmail string) gateway.MLVersionItem {
	var runID *string
	if v.RunID != "" {
		runID = &v.RunID
	}
	userEmail := tagValue(v.Tags, userTagKey)
	if userEmail == "" {
		userEmail = modelEmail
	}
	return gateway.MLVersionItem{
		MLflowID:      fmt.Sprintf("%s/%s", v.Name, v.Version),
		ModelMLflowID: v.Name,
		RunMLflowID:   runID,
		Version:       v.Version,
		Description:   v.Description,
		CreatedAt:     iso(v.CreationTimestamp),
		UserEmail:     userEmail,
	}
}

func artifactToItem(baseURL, runID string, f FileInfo, userEmail string) gateway.MLArtifactItem {
	name := f.Path
	if idx := strings.LastIndex(f.Path, "/"); idx >= 0 {
		name = f.Path[idx+1:]
	}
	downloadURI := fmt.Sprintf(
		"%s/get-artifact?path=%s&run_id=%s",
		strings.TrimRight(baseURL, "/"),
		url.QueryEscape(f.Path),
		url.QueryEscape(runID),
	)
	return gateway.MLArtifactItem{
		MLflowID:    fmt.Sprintf("%s/%s", runID, f.Path),
		RunMLflowID: runID,
		Name:        name,
		Type:        artifactType(name),
		URI:         f.Path,
		DownloadURI: downloadURI,
		Size:        humanSize(f.FileSize),
		Format:      extension(name),
		UserEmail:   userEmail,
	}
}

func loggedModelArtifactToItem(baseURL, runID, modelID string, f FileInfo, userEmail string) gateway.MLArtifactItem {
	name := f.Path
	if idx := strings.LastIndex(f.Path, "/"); idx >= 0 {
		name = f.Path[idx+1:]
	}
	downloadURI := fmt.Sprintf(
		"%s/ajax-api/2.0/mlflow/logged-models/%s/artifacts/files?artifact_file_path=%s",
		strings.TrimRight(baseURL, "/"),
		modelID,
		url.QueryEscape(f.Path),
	)
	return gateway.MLArtifactItem{
		MLflowID:    fmt.Sprintf("%s/%s/%s", runID, modelID, f.Path),
		RunMLflowID: runID,
		Name:        name,
		Type:        artifactType(name),
		URI:         fmt.Sprintf("%s/%s", modelID, f.Path),
		DownloadURI: downloadURI,
		Size:        humanSize(f.FileSize),
		Format:      extension(name),
		UserEmail:   userEmail,
	}
}

func datasetToItem(di DatasetInput, experimentMLflowID, userEmail string) gateway.MLDatasetItem {
	d := di.Dataset
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
		Context:            inputContext(di),
		RowCount:           datasetRowCount(d.Profile),
		Schema:             datasetSchema(d.Schema),
		Tags:               normalizeTags(di.Tags),
		UserEmail:          userEmail,
	}
}
