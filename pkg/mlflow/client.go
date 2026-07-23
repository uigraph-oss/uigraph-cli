package mlflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Experiment struct {
	ExperimentID   string     `json:"experiment_id"`
	Name           string     `json:"name"`
	LifecycleStage string     `json:"lifecycle_stage"`
	CreationTime   *int64     `json:"creation_time"`
	Tags           []KeyValue `json:"tags"`
}

type Metric struct {
	Key       string  `json:"key"`
	Value     float64 `json:"value"`
	Timestamp *int64  `json:"timestamp"`
	Step      int64   `json:"step"`
}

type Dataset struct {
	Name       string `json:"name"`
	Digest     string `json:"digest"`
	SourceType string `json:"source_type"`
	Source     string `json:"source"`
	Schema     string `json:"schema"`
	Profile    string `json:"profile"`
}

type DatasetInput struct {
	Tags    []KeyValue `json:"tags"`
	Dataset Dataset    `json:"dataset"`
}

type RunInputs struct {
	DatasetInputs []DatasetInput `json:"dataset_inputs"`
}

type RunInfo struct {
	RunID        string `json:"run_id"`
	ExperimentID string `json:"experiment_id"`
	RunName      string `json:"run_name"`
	Status       string `json:"status"`
	StartTime    *int64 `json:"start_time"`
	EndTime      *int64 `json:"end_time"`
}

type RunData struct {
	Metrics []Metric   `json:"metrics"`
	Params  []KeyValue `json:"params"`
	Tags    []KeyValue `json:"tags"`
}

type ModelOutput struct {
	ModelID string `json:"model_id"`
}

type RunOutputs struct {
	ModelOutputs []ModelOutput `json:"model_outputs"`
}

type Run struct {
	Info    RunInfo     `json:"info"`
	Data    RunData     `json:"data"`
	Inputs  *RunInputs  `json:"inputs"`
	Outputs *RunOutputs `json:"outputs"`
}

type FileInfo struct {
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	FileSize *int64 `json:"file_size"`
}

type RegisteredModel struct {
	Name                 string     `json:"name"`
	CreationTimestamp    *int64     `json:"creation_timestamp"`
	LastUpdatedTimestamp *int64     `json:"last_updated_timestamp"`
	Description          string     `json:"description"`
	Tags                 []KeyValue `json:"tags"`
}

type ModelVersion struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	CreationTimestamp *int64 `json:"creation_timestamp"`
	CurrentStage      string `json:"current_stage"`
	Description       string `json:"description"`
	RunID             string `json:"run_id"`
	Status            string `json:"status"`
}

func (c *Client) get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	u := fmt.Sprintf("%s/api/2.0/mlflow/%s", c.baseURL, path)
	if len(query) > 0 {
		u = u + "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return c.do(req)
}

func (c *Client) post(ctx context.Context, path string, body any) ([]byte, error) {
	u := fmt.Sprintf("%s/api/2.0/mlflow/%s", c.baseURL, path)
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mlflow request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read mlflow response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mlflow error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (c *Client) GetExperimentByName(ctx context.Context, name string) (*Experiment, error) {
	q := url.Values{}
	q.Set("experiment_name", name)
	body, err := c.get(ctx, "experiments/get-by-name", q)
	if err != nil {
		return nil, err
	}
	var out struct {
		Experiment Experiment `json:"experiment"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to parse experiment: %w", err)
	}
	return &out.Experiment, nil
}

func (c *Client) SearchRuns(ctx context.Context, experimentID string) ([]Run, error) {
	var runs []Run
	token := ""
	for {
		reqBody := map[string]any{
			"experiment_ids": []string{experimentID},
			"run_view_type":  "ALL",
			"max_results":    1000,
		}
		if token != "" {
			reqBody["page_token"] = token
		}
		body, err := c.post(ctx, "runs/search", reqBody)
		if err != nil {
			return nil, err
		}
		var out struct {
			Runs          []Run  `json:"runs"`
			NextPageToken string `json:"next_page_token"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, fmt.Errorf("failed to parse runs: %w", err)
		}
		runs = append(runs, out.Runs...)
		token = out.NextPageToken
		if token == "" {
			break
		}
	}
	return runs, nil
}

func (c *Client) MetricHistory(ctx context.Context, runID, key string) ([]Metric, error) {
	q := url.Values{}
	q.Set("run_id", runID)
	q.Set("metric_key", key)
	body, err := c.get(ctx, "metrics/get-history", q)
	if err != nil {
		return nil, err
	}
	var out struct {
		Metrics []Metric `json:"metrics"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to parse metric history: %w", err)
	}
	return out.Metrics, nil
}

func (c *Client) Artifacts(ctx context.Context, runID string) ([]FileInfo, error) {
	return c.artifactsRecursive(ctx, runID, "")
}

func (c *Client) artifactsRecursive(ctx context.Context, runID, path string) ([]FileInfo, error) {
	q := url.Values{}
	q.Set("run_id", runID)
	if path != "" {
		q.Set("path", path)
	}
	body, err := c.get(ctx, "artifacts/list", q)
	if err != nil {
		return nil, err
	}
	var out struct {
		Files []FileInfo `json:"files"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to parse artifacts: %w", err)
	}
	var files []FileInfo
	for _, item := range out.Files {
		if item.IsDir {
			nested, err := c.artifactsRecursive(ctx, runID, item.Path)
			if err != nil {
				return nil, err
			}
			files = append(files, nested...)
		} else {
			files = append(files, item)
		}
	}
	return files, nil
}

func (c *Client) LoggedModelArtifacts(ctx context.Context, modelID string) ([]FileInfo, error) {
	return c.loggedModelArtifactsRecursive(ctx, modelID, "")
}

func (c *Client) loggedModelArtifactsRecursive(ctx context.Context, modelID, path string) ([]FileInfo, error) {
	q := url.Values{}
	if path != "" {
		q.Set("artifact_directory_path", path)
	}
	body, err := c.get(ctx, "logged-models/"+modelID+"/artifacts/directories", q)
	if err != nil {
		return nil, err
	}
	var out struct {
		Files []FileInfo `json:"files"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to parse logged model artifacts: %w", err)
	}
	var files []FileInfo
	for _, item := range out.Files {
		if item.IsDir {
			nested, err := c.loggedModelArtifactsRecursive(ctx, modelID, item.Path)
			if err != nil {
				return nil, err
			}
			files = append(files, nested...)
		} else {
			files = append(files, item)
		}
	}
	return files, nil
}

func (c *Client) GetRegisteredModel(ctx context.Context, name string) (*RegisteredModel, error) {
	q := url.Values{}
	q.Set("name", name)
	body, err := c.get(ctx, "registered-models/get", q)
	if err != nil {
		return nil, err
	}
	var out struct {
		RegisteredModel RegisteredModel `json:"registered_model"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to parse registered model: %w", err)
	}
	return &out.RegisteredModel, nil
}

func (c *Client) ModelVersions(ctx context.Context, modelName string) ([]ModelVersion, error) {
	var versions []ModelVersion
	token := ""
	for {
		q := url.Values{}
		q.Set("filter", fmt.Sprintf("name='%s'", modelName))
		q.Set("max_results", "1000")
		if token != "" {
			q.Set("page_token", token)
		}
		body, err := c.get(ctx, "model-versions/search", q)
		if err != nil {
			return nil, err
		}
		var out struct {
			ModelVersions []ModelVersion `json:"model_versions"`
			NextPageToken string         `json:"next_page_token"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, fmt.Errorf("failed to parse model versions: %w", err)
		}
		versions = append(versions, out.ModelVersions...)
		token = out.NextPageToken
		if token == "" {
			break
		}
	}
	return versions, nil
}
