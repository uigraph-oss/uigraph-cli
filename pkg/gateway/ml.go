package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type MLProjectItem struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	SourceType  string `json:"sourceType"`
	SourceURL   string `json:"sourceUrl"`
	Team        string `json:"team"`
	Email       string `json:"email"`
}

type MLModelItem struct {
	MLflowID                  string     `json:"mlflowId"`
	ProjectName               string     `json:"projectName"`
	Name                      string     `json:"name"`
	Description               string     `json:"description"`
	Tags                      []string   `json:"tags"`
	ProductionVersionMLflowID *string    `json:"productionVersionMlflowId"`
	CreatedAt                 *time.Time `json:"createdAt"`
	UpdatedAt                 *time.Time `json:"updatedAt"`
}

type MLVersionItem struct {
	MLflowID      string     `json:"mlflowId"`
	ModelMLflowID string     `json:"modelMlflowId"`
	RunMLflowID   *string    `json:"runMlflowId"`
	Version       string     `json:"version"`
	Description   string     `json:"description"`
	CreatedAt     *time.Time `json:"createdAt"`
}

type MLExperimentItem struct {
	MLflowID    string     `json:"mlflowId"`
	ProjectName string     `json:"projectName"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"startedAt"`
}

type MLRunItem struct {
	MLflowID           string         `json:"mlflowId"`
	ExperimentMLflowID string         `json:"experimentMlflowId"`
	DatasetMLflowID    *string        `json:"datasetMlflowId"`
	Name               string         `json:"name"`
	Status             string         `json:"status"`
	StartedAt          *time.Time     `json:"startedAt"`
	EndedAt            *time.Time     `json:"endedAt"`
	Duration           string         `json:"duration"`
	Notes              string         `json:"notes"`
	Parameters         map[string]any `json:"parameters"`
	Metrics            map[string]any `json:"metrics"`
}

type MLSeriesPoint struct {
	Key   string     `json:"key"`
	Step  int64      `json:"step"`
	Value float64    `json:"value"`
	TS    *time.Time `json:"ts"`
}

type MLArtifactItem struct {
	MLflowID    string `json:"mlflowId"`
	RunMLflowID string `json:"runMlflowId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	URI         string `json:"uri"`
	Size        string `json:"size"`
	Format      string `json:"format"`
}

type MLSchemaField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type MLDatasetItem struct {
	MLflowID           string            `json:"mlflowId"`
	ExperimentMLflowID string            `json:"experimentMlflowId"`
	Name               string            `json:"name"`
	Digest             string            `json:"digest"`
	Source             string            `json:"source"`
	SourceType         string            `json:"sourceType"`
	Context            string            `json:"context"`
	RowCount           int64             `json:"rowCount"`
	Schema             []MLSchemaField   `json:"schema"`
	Tags               map[string]string `json:"tags"`
}

type mlSyncResponse struct {
	Synced int `json:"synced"`
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
	defer resp.Body.Close()

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

func (c *Client) SyncMLProjects(ctx context.Context, items []MLProjectItem) (int, error) {
	return c.postMLSync(ctx, "/v1/sync/ml/projects", items)
}

func (c *Client) SyncMLModels(ctx context.Context, items []MLModelItem) (int, error) {
	return c.postMLSync(ctx, "/v1/sync/ml/models", items)
}

func (c *Client) SyncMLVersions(ctx context.Context, items []MLVersionItem) (int, error) {
	return c.postMLSync(ctx, "/v1/sync/ml/versions", items)
}

func (c *Client) SyncMLExperiments(ctx context.Context, items []MLExperimentItem) (int, error) {
	return c.postMLSync(ctx, "/v1/sync/ml/experiments", items)
}

func (c *Client) SyncMLRuns(ctx context.Context, items []MLRunItem) (int, error) {
	return c.postMLSync(ctx, "/v1/sync/ml/runs", items)
}

func (c *Client) SyncMLRunSeries(ctx context.Context, runMLflowID string, items []MLSeriesPoint) (int, error) {
	return c.postMLSync(ctx, fmt.Sprintf("/v1/sync/ml/runs/%s/series", url.PathEscape(runMLflowID)), items)
}

func (c *Client) SyncMLArtifacts(ctx context.Context, items []MLArtifactItem) (int, error) {
	return c.postMLSync(ctx, "/v1/sync/ml/artifacts", items)
}

func (c *Client) SyncMLDatasets(ctx context.Context, items []MLDatasetItem) (int, error) {
	return c.postMLSync(ctx, "/v1/sync/ml/datasets", items)
}
