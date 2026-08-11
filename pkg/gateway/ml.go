package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MLProjectItem struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	SourceType  string `json:"sourceType"`
	SourceURL   string `json:"sourceUrl"`
	Team        string `json:"team"`
}

type MLModelItem struct {
	MLflowID                  string     `json:"mlflowId"`
	ProjectName               string     `json:"projectName"`
	Name                      string     `json:"name"`
	Description               string     `json:"description"`
	Tags                      []string   `json:"tags"`
	ProblemType               string     `json:"problemType,omitempty"`
	Domain                    string     `json:"domain,omitempty"`
	License                   string     `json:"license,omitempty"`
	IntendedUse               string     `json:"intendedUse,omitempty"`
	Limitations               string     `json:"limitations,omitempty"`
	Recommendations           string     `json:"recommendations,omitempty"`
	Considerations            string     `json:"considerations,omitempty"`
	ProductionVersionMLflowID *string    `json:"productionVersionMlflowId"`
	CreatedAt                 *time.Time `json:"createdAt"`
	UpdatedAt                 *time.Time `json:"updatedAt"`
	UserEmail                 string     `json:"userEmail,omitempty"`
}

type MLVersionItem struct {
	MLflowID      string     `json:"mlflowId"`
	ModelMLflowID string     `json:"modelMlflowId"`
	RunMLflowID   *string    `json:"runMlflowId"`
	Version       string     `json:"version"`
	Description   string     `json:"description"`
	CreatedAt     *time.Time `json:"createdAt"`
	UserEmail     string     `json:"userEmail,omitempty"`
}

type MLExperimentItem struct {
	MLflowID    string   `json:"mlflowId"`
	ProjectName string   `json:"projectName"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
	UserEmail   string   `json:"userEmail,omitempty"`
}

type MLRunItem struct {
	MLflowID           string         `json:"mlflowId"`
	ExperimentMLflowID string         `json:"experimentMlflowId"`
	DatasetMLflowID    *string        `json:"datasetMlflowId"`
	Name               string         `json:"name"`
	Status             string         `json:"status"`
	StartedAt          time.Time      `json:"startedAt"`
	EndedAt            *time.Time     `json:"endedAt"`
	Notes              string         `json:"notes"`
	Tags               []string       `json:"tags"`
	Parameters         map[string]any `json:"parameters"`
	Metrics            map[string]any `json:"metrics"`
	UserEmail          string         `json:"userEmail,omitempty"`
}

type MLArtifactItem struct {
	MLflowID    string `json:"mlflowId"`
	RunMLflowID string `json:"runMlflowId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	URI         string `json:"uri"`
	DownloadURI string `json:"downloadUri"`
	Size        string `json:"size"`
	Format      string `json:"format"`
	UserEmail   string `json:"userEmail,omitempty"`
}

type MLSchemaField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type MLDatasetItem struct {
	MLflowID           string          `json:"mlflowId"`
	ExperimentMLflowID string          `json:"experimentMlflowId"`
	Name               string          `json:"name"`
	Digest             string          `json:"digest"`
	Source             string          `json:"source"`
	SourceType         string          `json:"sourceType"`
	Context            string          `json:"context"`
	RowCount           int64           `json:"rowCount"`
	Schema             []MLSchemaField `json:"schema"`
	Tags               []string        `json:"tags"`
	UserEmail          string          `json:"userEmail,omitempty"`
}

type MLEvaluationItem struct {
	MLflowID           string         `json:"mlflowId"`
	VersionMLflowID    string         `json:"versionMlflowId"`
	ExperimentMLflowID string         `json:"experimentMlflowId"`
	DatasetMLflowID    *string        `json:"datasetMlflowId"`
	Name               string         `json:"name"`
	Type               string         `json:"type"`
	Description        string         `json:"description"`
	Summary            string         `json:"summary"`
	StartedAt          time.Time      `json:"startedAt"`
	EndedAt            *time.Time     `json:"endedAt"`
	Evaluator          string         `json:"evaluator"`
	Tags               []string       `json:"tags"`
	Parameters         map[string]any `json:"parameters"`
	Metrics            map[string]any `json:"metrics"`
	UserEmail          string         `json:"userEmail,omitempty"`
}

type MLProjectState struct {
	Name     string     `json:"name"`
	SyncedAt *time.Time `json:"syncedAt"`
}

type mlSyncResponse struct {
	Synced int `json:"synced"`
}

func (c *Client) ListMLProjects(ctx context.Context) ([]MLProjectState, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/sync/ml/projects", c.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("not authorized to read ML projects - the sync token needs the \"mlstudio:read\" scope")
	}
	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var listResp struct {
		Projects []MLProjectState `json:"projects"`
	}
	if err := json.Unmarshal(respBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return listResp.Projects, nil
}

func (c *Client) postMLSync(ctx context.Context, path string, payload any) (int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s%s", c.baseURL, path), bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return 0, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp mlSyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return syncResp.Synced, nil
}

func (c *Client) SyncMLProject(ctx context.Context, item MLProjectItem) error {
	_, err := c.postMLSync(ctx, "/v1/sync/ml/projects", []MLProjectItem{item})
	return err
}

func (c *Client) SyncMLModel(ctx context.Context, item MLModelItem) error {
	_, err := c.postMLSync(ctx, "/v1/sync/ml/models", []MLModelItem{item})
	return err
}

func (c *Client) SyncMLVersion(ctx context.Context, item MLVersionItem) error {
	_, err := c.postMLSync(ctx, "/v1/sync/ml/versions", []MLVersionItem{item})
	return err
}

func (c *Client) SyncMLExperiment(ctx context.Context, item MLExperimentItem) error {
	_, err := c.postMLSync(ctx, "/v1/sync/ml/experiments", []MLExperimentItem{item})
	return err
}

func (c *Client) SyncMLRun(ctx context.Context, item MLRunItem) error {
	_, err := c.postMLSync(ctx, "/v1/sync/ml/runs", []MLRunItem{item})
	return err
}

func (c *Client) SyncMLArtifact(ctx context.Context, item MLArtifactItem) error {
	_, err := c.postMLSync(ctx, "/v1/sync/ml/artifacts", []MLArtifactItem{item})
	return err
}

func (c *Client) SyncMLDataset(ctx context.Context, item MLDatasetItem) error {
	_, err := c.postMLSync(ctx, "/v1/sync/ml/datasets", []MLDatasetItem{item})
	return err
}

func (c *Client) SyncMLEvaluation(ctx context.Context, item MLEvaluationItem) error {
	_, err := c.postMLSync(ctx, "/v1/sync/ml/evaluations", []MLEvaluationItem{item})
	return err
}
