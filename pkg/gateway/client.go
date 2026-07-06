package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/uigraph-oss/uigraph-cli/pkg/config"
	"github.com/uigraph-oss/uigraph-cli/pkg/git"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type Source struct {
	Type string `json:"type"`
	Tool string `json:"tool"`
}

type ServiceSyncRequest struct {
	Project config.Project `json:"project"`
	Service config.Service `json:"service"`
	Git     git.Metadata   `json:"git"`
	Source  Source         `json:"source"`
}

type ServiceSyncResponse struct {
	Name    string `json:"name"`
	Message string `json:"message,omitempty"`
}

type APIGroup struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Spec struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

type GitMetadataMinimal struct {
	CommitHash string `json:"commitHash"`
}

type APIGroupSyncRequest struct {
	APIGroup    APIGroup           `json:"apiGroup"`
	Spec        Spec               `json:"spec"`
	Git         GitMetadataMinimal `json:"git"`
	ServiceName string             `json:"serviceName"`
}

type APIGroupSyncResponse struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
}

type ArchitectureDiagramSyncRequest struct {
	ServiceName    string `json:"serviceName"`
	Name           string `json:"name"`
	MermaidContent string `json:"mermaidContent"`
	ContextContent string `json:"contextContent,omitempty"`
	GitCommitHash  string `json:"gitCommitHash,omitempty"`
}

type ArchitectureDiagramSyncResponse struct {
	Name           string `json:"name"`
	Message        string `json:"message,omitempty"`
	VersionCreated bool   `json:"versionCreated,omitempty"`
}

type TestPackSyncRequest struct {
	TestPack    TestPackInfoPayload `json:"testPack"`
	Git         GitMetadataMinimal  `json:"git"`
	ServiceName string              `json:"serviceName"`
	ServiceID   string              `json:"serviceId,omitempty"`
}

type TestPackInfoPayload struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Environment  *string `json:"environment,omitempty"`
	ReleaseLabel *string `json:"releaseLabel,omitempty"`
}

type TestPackSyncResponse struct {
	TestPackID  string `json:"testPackId"`
	ServiceName string `json:"serviceName,omitempty"`
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Message     string `json:"message,omitempty"`
}

type TestCaseSyncRequest struct {
	TestCase     TestCaseInfoPayload `json:"testCase"`
	Git          GitMetadataMinimal  `json:"git"`
	TestPackName string              `json:"testPackName"`
	TestPackID   string              `json:"testPackId,omitempty"`
	ServiceName  string              `json:"serviceName"`
}

type TestCaseStepPayload struct {
	Action         string `json:"action"`
	ExpectedResult string `json:"expectedResult,omitempty"`
}

type AssertionPayload struct {
	Field string `json:"field"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type TestCaseInfoPayload struct {
	Type  string  `json:"type"`
	Title string  `json:"title"`
	Order float64 `json:"order"`

	Description           *string  `json:"description,omitempty"`
	Priority              *string  `json:"priority,omitempty"`
	Labels                []string `json:"labels,omitempty"`
	LinkedTicket          *string  `json:"linkedTicket,omitempty"`
	EstimatedDurationMins *int     `json:"estimatedDurationMins,omitempty"`
	TestOwner             *string  `json:"testOwner,omitempty"`

	MapName        *string `json:"mapName,omitempty"`
	FrameName      *string `json:"frameName,omitempty"`
	FocalPointName *string `json:"focalPointName,omitempty"`

	APIGroupName       *string            `json:"apiGroupName,omitempty"`
	OperationID        *string            `json:"operationId,omitempty"`
	ExpectedStatusCode *int               `json:"expectedStatusCode,omitempty"`
	RequestTemplate    *string            `json:"requestTemplate,omitempty"`
	MaxResponseTimeMs  *int               `json:"maxResponseTimeMs,omitempty"`
	ResponseBody       *string            `json:"responseBody,omitempty"`
	Assertions         []AssertionPayload `json:"assertions,omitempty"`

	StepsList        []TestCaseStepPayload `json:"stepsList,omitempty"`
	ExpectedOutcome  *string               `json:"expectedOutcome,omitempty"`
	Preconditions    *string               `json:"preconditions,omitempty"`
	TestData         *string               `json:"testData,omitempty"`
	Postconditions   *string               `json:"postconditions,omitempty"`
	RequiresEvidence bool                  `json:"requiresEvidence"`

	BaselineRunResultID *string `json:"baselineRunResultId,omitempty"`
	IsCritical          bool    `json:"isCritical"`
}

type TestCaseSyncResponse struct {
	Title   string `json:"title"`
	Message string `json:"message,omitempty"`
}

type ServiceDocPrepareRequest struct {
	ServiceName string `json:"serviceName"`
	DocName     string `json:"docName"`
	ContentHash string `json:"contentHash"`
	FileSize    int64  `json:"fileSize"`
	FilePath    string `json:"filePath,omitempty"`
	FileType    string `json:"fileType,omitempty"`
	Description string `json:"description,omitempty"`
}

type ServiceDocPrepareResponse struct {
	Action       string  `json:"action"`
	Reason       string  `json:"reason,omitempty"`
	UploadURL    *string `json:"uploadUrl,omitempty"`
	FileID       *string `json:"fileId,omitempty"`
	ExistingHash *string `json:"existingHash,omitempty"`
}

type ServiceDocCompleteRequest struct {
	ServiceName string `json:"serviceName"`
	DocName     string `json:"docName"`
	FileID      string `json:"fileId"`
	ContentHash string `json:"contentHash"`
	FileType    string `json:"fileType,omitempty"`
	Description string `json:"description,omitempty"`
	CommitHash  string `json:"commitHash,omitempty"`
}

type ServiceDocCompleteResponse struct {
	Name       string `json:"name"`
	Message    string `json:"message"`
	ServiceDoc string `json:"serviceDocId,omitempty"`
}

type ServiceDatabaseSyncRequest struct {
	ServiceName       string             `json:"serviceName"`
	DBName            string             `json:"dbName"`
	Dialect           string             `json:"dialect"`
	DBType            string             `json:"dbType,omitempty"`
	SchemaFileContent string             `json:"schemaFileContent,omitempty"`
	Git               GitMetadataMinimal `json:"git"`
}

type ServiceDatabaseSyncResponse struct {
	DBName         string `json:"dbName"`
	Message        string `json:"message,omitempty"`
	VersionCreated bool   `json:"versionCreated,omitempty"`
}

type SavedQuerySyncRequest struct {
	ServiceName string             `json:"serviceName"`
	DBName      string             `json:"dbName"`
	SourceRef   string             `json:"sourceRef"`
	Title       string             `json:"title"`
	Description string             `json:"description,omitempty"`
	QueryText   string             `json:"queryText"`
	Tags        []string           `json:"tags,omitempty"`
	Git         GitMetadataMinimal `json:"git"`
}

type SavedQuerySyncResponse struct {
	SourceRef string `json:"sourceRef"`
	ID        string `json:"id"`
	Created   bool   `json:"created"`
}

type MapSyncRequest struct {
	MapName     string `json:"mapName"`
	Description string `json:"description,omitempty"`
	CommitHash  string `json:"commitHash,omitempty"`
}

type MapSyncResponse struct {
	MapID   string `json:"mapId"`
	Message string `json:"message"`
}

type FramePrepareRequest struct {
	MapName     string `json:"mapName"`
	FrameName   string `json:"frameName"`
	Description string `json:"description,omitempty"`
	ContentHash string `json:"contentHash,omitempty"`
	FileSize    int64  `json:"fileSize,omitempty"`
	ImagePath   string `json:"imagePath,omitempty"`
	CommitHash  string `json:"commitHash,omitempty"`
}

type FramePrepareResponse struct {
	Action    string  `json:"action"`
	PageID    string  `json:"pageId"`
	UploadURL *string `json:"uploadUrl,omitempty"`
	FileID    *string `json:"fileId,omitempty"`
	Message   string  `json:"message,omitempty"`
}

type FrameCompleteRequest struct {
	MapName     string `json:"mapName"`
	FrameName   string `json:"frameName"`
	FileID      string `json:"fileId"`
	ContentHash string `json:"contentHash"`
	Description string `json:"description,omitempty"`
	CommitHash  string `json:"commitHash,omitempty"`
}

type FrameCompleteResponse struct {
	PageID  string `json:"pageId"`
	Message string `json:"message"`
}

type FocalPointSyncRequest struct {
	MapName        string  `json:"mapName"`
	FrameName      string  `json:"frameName"`
	FocalPointName string  `json:"focalPointName"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Visibility     string  `json:"visibility,omitempty"`
	CommitHash     string  `json:"commitHash,omitempty"`
}

type FocalPointSyncResponse struct {
	FocalPointID string `json:"focalPointId"`
	PageID       string `json:"pageId"`
	Message      string `json:"message"`
}

type ComponentFieldItem struct {
	ComponentFieldID string        `json:"componentFieldId"`
	Label            string        `json:"label"`
	Type             string        `json:"type,omitempty"`
	Data             []interface{} `json:"data,omitempty"`
}

type FocalPointMetaSyncRequest struct {
	MapName                 string               `json:"mapName"`
	FrameName               string               `json:"frameName"`
	FocalPointName          string               `json:"focalPointName"`
	ComponentID             string               `json:"componentId"`
	ComponentLinkID         string               `json:"componentLinkId,omitempty"`
	ComponentModalFields    []ComponentFieldItem `json:"componentModalFields,omitempty"`
	ServiceName             string               `json:"serviceName,omitempty"`
	APIGroupName            string               `json:"apiGroupName,omitempty"`
	OperationID             string               `json:"operationId,omitempty"`
	TestPackName            string               `json:"testPackName,omitempty"`
	DocName                 string               `json:"docName,omitempty"`
	ArchitectureDiagramName string               `json:"architectureDiagramName,omitempty"`
	CommitHash              string               `json:"commitHash,omitempty"`
}

type FocalPointMetaSyncResponse struct {
	FocalPointMetaID string `json:"focalPointMetaId"`
	FocalPointID     string `json:"focalPointId"`
	ComponentID      string `json:"componentId"`
	Message          string `json:"message"`
}

func (c *Client) SyncService(ctx context.Context, req ServiceSyncRequest) (*ServiceSyncResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/service", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp ServiceSyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &syncResp, nil
}

func (c *Client) SyncAPIGroup(ctx context.Context, req APIGroupSyncRequest) (*APIGroupSyncResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/service/api-group", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp APIGroupSyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &syncResp, nil
}

func (c *Client) SyncArchitectureDiagram(ctx context.Context, req ArchitectureDiagramSyncRequest) (*ArchitectureDiagramSyncResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/service/architecture-diagram", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp ArchitectureDiagramSyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &syncResp, nil
}

func (c *Client) SyncTestPack(ctx context.Context, req TestPackSyncRequest) (*TestPackSyncResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/service/test-pack", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp TestPackSyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &syncResp, nil
}

func (c *Client) SyncTestCase(ctx context.Context, req TestCaseSyncRequest) (*TestCaseSyncResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/service/test-case", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp TestCaseSyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &syncResp, nil
}

func (c *Client) PrepareServiceDocUpload(ctx context.Context, req ServiceDocPrepareRequest) (*ServiceDocPrepareResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/service/doc/prepare", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var prepResp ServiceDocPrepareResponse
	if err := json.Unmarshal(respBody, &prepResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &prepResp, nil
}

func (c *Client) CompleteServiceDocUpload(ctx context.Context, req ServiceDocCompleteRequest) (*ServiceDocCompleteResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/service/doc/complete", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var completeResp ServiceDocCompleteResponse
	if err := json.Unmarshal(respBody, &completeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &completeResp, nil
}

func (c *Client) SyncMap(ctx context.Context, req MapSyncRequest) (*MapSyncResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/map", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp MapSyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &syncResp, nil
}

func (c *Client) PrepareFrameSync(ctx context.Context, req FramePrepareRequest) (*FramePrepareResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/frame/prepare", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var prepResp FramePrepareResponse
	if err := json.Unmarshal(respBody, &prepResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &prepResp, nil
}

func (c *Client) CompleteFrameSync(ctx context.Context, req FrameCompleteRequest) (*FrameCompleteResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/frame/complete", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var completeResp FrameCompleteResponse
	if err := json.Unmarshal(respBody, &completeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &completeResp, nil
}

func (c *Client) SyncFocalPoint(ctx context.Context, req FocalPointSyncRequest) (*FocalPointSyncResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/focal-point", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp FocalPointSyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &syncResp, nil
}

func (c *Client) SyncFocalPointMeta(ctx context.Context, req FocalPointMetaSyncRequest) (*FocalPointMetaSyncResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/focal-point-meta", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp FocalPointMetaSyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &syncResp, nil
}

func (c *Client) SyncServiceDatabase(ctx context.Context, req ServiceDatabaseSyncRequest) (*ServiceDatabaseSyncResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/service/database", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp ServiceDatabaseSyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &syncResp, nil
}

func (c *Client) SyncSavedQuery(ctx context.Context, req SavedQuerySyncRequest) (*SavedQuerySyncResponse, error) {
	url := fmt.Sprintf("%s/v1/sync/service/database/query", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", c.token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, formatGatewayError(resp.StatusCode, respBody)
	}

	var syncResp SavedQuerySyncResponse
	if err := json.Unmarshal(respBody, &syncResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &syncResp, nil
}

func (r *ServiceSyncRequest) Print() {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func (r *APIGroupSyncRequest) Print() {
	type dryRunRequest struct {
		APIGroup APIGroup `json:"apiGroup"`
		Spec     struct {
			Path        string `json:"path"`
			ContentSize int    `json:"contentSize"`
		} `json:"spec"`
		Git GitMetadataMinimal `json:"git"`
	}

	dr := dryRunRequest{
		APIGroup: r.APIGroup,
		Git:      r.Git,
	}
	dr.Spec.Path = r.Spec.Path
	dr.Spec.ContentSize = len(r.Spec.Content)

	data, err := json.MarshalIndent(dr, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func formatGatewayError(statusCode int, respBody []byte) error {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("authentication failed - please check your UIGRAPH_TOKEN")
	case http.StatusBadRequest:
		var errResp struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Message != "" {
			return fmt.Errorf("validation error: %s", errResp.Message)
		}
		return fmt.Errorf("invalid request - please check your .uigraph.yaml configuration")
	case http.StatusNotFound:
		return fmt.Errorf("resource not found - please verify service and resource names")
	case http.StatusTooManyRequests:
		var errResp struct {
			Message    string `json:"message"`
			RetryAfter int    `json:"retryAfter,omitempty"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.RetryAfter > 0 {
			return fmt.Errorf("rate limit exceeded - please retry after %d seconds", errResp.RetryAfter)
		}
		return fmt.Errorf("rate limit exceeded - too many requests, please try again later")
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		var errResp struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Message != "" {
			return fmt.Errorf("%s", errResp.Message)
		}
		return fmt.Errorf("service temporarily unavailable - please try again")
	default:
		return fmt.Errorf("sync failed with status %d - please try again", statusCode)
	}
}
