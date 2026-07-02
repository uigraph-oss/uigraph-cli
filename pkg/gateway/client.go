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

// Client is a gateway API client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new gateway client
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Source represents the source of the sync
type Source struct {
	Type string `json:"type"`
	Tool string `json:"tool"`
}

// ServiceSyncRequest is the request payload for syncing a service
type ServiceSyncRequest struct {
	Project config.Project `json:"project"`
	Service config.Service `json:"service"`
	Git     git.Metadata   `json:"git"`
	Source  Source         `json:"source"`
}

// ServiceSyncResponse is the response from syncing a service
type ServiceSyncResponse struct {
	Name    string `json:"name"`
	Message string `json:"message,omitempty"`
}

// APIGroup represents an API group
type APIGroup struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Spec represents the API specification
type Spec struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

// GitMetadataMinimal represents minimal git metadata for API groups
type GitMetadataMinimal struct {
	CommitHash string `json:"commitHash"`
}

// APIGroupSyncRequest is the request payload for syncing an API group
type APIGroupSyncRequest struct {
	APIGroup    APIGroup           `json:"apiGroup"`
	Spec        Spec               `json:"spec"`
	Git         GitMetadataMinimal `json:"git"`
	ServiceName string             `json:"serviceName"`
}

// APIGroupSyncResponse is the response from syncing an API group
type APIGroupSyncResponse struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
}

// ArchitectureDiagramSyncRequest is the request payload for syncing an architecture diagram
type ArchitectureDiagramSyncRequest struct {
	ServiceName    string `json:"serviceName"`
	Name           string `json:"name"`
	MermaidContent string `json:"mermaidContent"`
	ContextContent string `json:"contextContent,omitempty"`
	GitCommitHash  string `json:"gitCommitHash,omitempty"`
}

// ArchitectureDiagramSyncResponse is the response from syncing an architecture diagram
type ArchitectureDiagramSyncResponse struct {
	Name           string `json:"name"`
	Message        string `json:"message,omitempty"`
	VersionCreated bool   `json:"versionCreated,omitempty"`
}

// TestPackSyncRequest is the request payload for syncing a test pack
type TestPackSyncRequest struct {
	TestPack    TestPackInfoPayload `json:"testPack"`
	Git         GitMetadataMinimal  `json:"git"`
	ServiceName string              `json:"serviceName"`
	ServiceID   string              `json:"serviceId,omitempty"`
}

// TestPackInfoPayload contains test pack metadata for sync
type TestPackInfoPayload struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Environment  *string `json:"environment,omitempty"`
	ReleaseLabel *string `json:"releaseLabel,omitempty"`
}

// TestPackSyncResponse is the response from syncing a test pack
type TestPackSyncResponse struct {
	TestPackID  string `json:"testPackId"`
	ServiceName string `json:"serviceName,omitempty"`
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Message     string `json:"message,omitempty"`
}

// TestCaseSyncRequest is the request payload for syncing a test case
type TestCaseSyncRequest struct {
	TestCase     TestCaseInfoPayload `json:"testCase"`
	Git          GitMetadataMinimal  `json:"git"`
	TestPackName string              `json:"testPackName"`
	TestPackID   string              `json:"testPackId,omitempty"`
	ServiceName  string              `json:"serviceName"`
}

// TestCaseStepPayload is a single step with optional expected result for sync
type TestCaseStepPayload struct {
	Action         string `json:"action"`
	ExpectedResult string `json:"expectedResult,omitempty"`
}

// AssertionPayload represents a single API assertion for sync.
type AssertionPayload struct {
	Field string `json:"field"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// TestCaseInfoPayload contains test case metadata for sync
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

	// Map/Frame/Focal Point Reference (user-friendly, will be resolved to linkedMapNodeId)
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

// TestCaseSyncResponse is the response from syncing a test case
type TestCaseSyncResponse struct {
	Title   string `json:"title"`
	Message string `json:"message,omitempty"`
}

// ServiceDocPrepareRequest is the request to prepare a doc upload
type ServiceDocPrepareRequest struct {
	ServiceName string `json:"serviceName"`
	DocName     string `json:"docName"`
	ContentHash string `json:"contentHash"`
	FileSize    int64  `json:"fileSize"`
	FilePath    string `json:"filePath,omitempty"`
	FileType    string `json:"fileType,omitempty"`
	Description string `json:"description,omitempty"`
}

// ServiceDocPrepareResponse tells the CLI whether to upload or skip
type ServiceDocPrepareResponse struct {
	Action       string  `json:"action"`
	Reason       string  `json:"reason,omitempty"`
	UploadURL    *string `json:"uploadUrl,omitempty"`
	FileID       *string `json:"fileId,omitempty"`
	ExistingHash *string `json:"existingHash,omitempty"`
}

// ServiceDocCompleteRequest is the request to complete a doc upload
type ServiceDocCompleteRequest struct {
	ServiceName string `json:"serviceName"`
	DocName     string `json:"docName"`
	FileID      string `json:"fileId"`
	ContentHash string `json:"contentHash"`
	FileType    string `json:"fileType,omitempty"`
	Description string `json:"description,omitempty"`
	CommitHash  string `json:"commitHash,omitempty"`
}

// ServiceDocCompleteResponse is the response after completing doc upload
type ServiceDocCompleteResponse struct {
	Name       string `json:"name"`
	Message    string `json:"message"`
	ServiceDoc string `json:"serviceDocId,omitempty"`
}

// ServiceDatabaseSyncRequest is the request payload for syncing a service database schema.
// The CLI sends raw file content; the gateway parses it via the adapter (SQL or NoSQL).
type ServiceDatabaseSyncRequest struct {
	ServiceName       string             `json:"serviceName"`
	DBName            string             `json:"dbName"`
	Dialect           string             `json:"dialect"`
	DBType            string             `json:"dbType,omitempty"`
	SchemaFileContent string             `json:"schemaFileContent,omitempty"` // raw file content; gateway parses via adapter
	Git               GitMetadataMinimal `json:"git"`
}

// ServiceDatabaseSyncResponse is the response from syncing a service database
type ServiceDatabaseSyncResponse struct {
	DBName         string `json:"dbName"`
	Message        string `json:"message,omitempty"`
	VersionCreated bool   `json:"versionCreated,omitempty"`
}

// SavedQuerySyncRequest is the request to upsert a saved query onto a service database.
// SourceRef is the stable key the gateway/API upsert by — a repeat sync with the same
// SourceRef updates the existing row instead of creating a duplicate.
type SavedQuerySyncRequest struct {
	ServiceName string   `json:"serviceName"`
	DBName      string   `json:"dbName"`
	SourceRef   string   `json:"sourceRef"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	QueryText   string   `json:"queryText"`
	Tags        []string `json:"tags,omitempty"`
}

// SavedQuerySyncResponse is the response from syncing a saved query
type SavedQuerySyncResponse struct {
	SourceRef string `json:"sourceRef"`
	ID        string `json:"id"`
	Created   bool   `json:"created"`
}

// MapSyncRequest is the request to upsert a map by name
type MapSyncRequest struct {
	MapName     string `json:"mapName"`
	Description string `json:"description,omitempty"`
	CommitHash  string `json:"commitHash,omitempty"`
}

// MapSyncResponse is the response after map upsert
type MapSyncResponse struct {
	MapID   string `json:"mapId"`
	Message string `json:"message"`
}

// FramePrepareRequest is the request to prepare a frame upsert (with optional image SHA check)
type FramePrepareRequest struct {
	MapName     string `json:"mapName"`
	FrameName   string `json:"frameName"`
	Description string `json:"description,omitempty"`
	ContentHash string `json:"contentHash,omitempty"`
	FileSize    int64  `json:"fileSize,omitempty"`
	ImagePath   string `json:"imagePath,omitempty"`
	CommitHash  string `json:"commitHash,omitempty"`
}

// FramePrepareResponse tells the CLI what to do next
type FramePrepareResponse struct {
	Action    string  `json:"action"` // "done", "skip", "upload"
	PageID    string  `json:"pageId"`
	UploadURL *string `json:"uploadUrl,omitempty"`
	FileID    *string `json:"fileId,omitempty"`
	Message   string  `json:"message,omitempty"`
}

// FrameCompleteRequest finalizes a frame after S3 image upload
type FrameCompleteRequest struct {
	MapName     string `json:"mapName"`
	FrameName   string `json:"frameName"`
	FileID      string `json:"fileId"`
	ContentHash string `json:"contentHash"`
	Description string `json:"description,omitempty"`
	CommitHash  string `json:"commitHash,omitempty"`
}

// FrameCompleteResponse is the response after frame finalization
type FrameCompleteResponse struct {
	PageID  string `json:"pageId"`
	Message string `json:"message"`
}

// FocalPointSyncRequest is the request to upsert a focal point by name
type FocalPointSyncRequest struct {
	MapName        string  `json:"mapName"`
	FrameName      string  `json:"frameName"`
	FocalPointName string  `json:"focalPointName"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Visibility     string  `json:"visibility,omitempty"`
	CommitHash     string  `json:"commitHash,omitempty"`
}

// FocalPointSyncResponse is the response after focal point upsert
type FocalPointSyncResponse struct {
	FocalPointID string `json:"focalPointId"`
	PageID       string `json:"pageId"`
	Message      string `json:"message"`
}

// ComponentFieldItem is a single component modal field for sync
type ComponentFieldItem struct {
	ComponentFieldID string        `json:"componentFieldId"`
	Label            string        `json:"label"`
	Type             string        `json:"type,omitempty"`
	Data             []interface{} `json:"data,omitempty"`
}

// FocalPointMetaSyncRequest is the request to upsert focal point meta
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

// FocalPointMetaSyncResponse is the response after focal point meta upsert
type FocalPointMetaSyncResponse struct {
	FocalPointMetaID string `json:"focalPointMetaId"`
	FocalPointID     string `json:"focalPointId"`
	ComponentID      string `json:"componentId"`
	Message          string `json:"message"`
}

// SyncService syncs a service to the gateway
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

// SyncAPIGroup syncs an API group to the gateway
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

// SyncArchitectureDiagram syncs an architecture diagram (mermaid) to the gateway
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

// SyncTestPack syncs a test pack to the gateway
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

// SyncTestCase syncs a test case to the gateway
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

// PrepareServiceDocUpload prepares a service doc upload (checks if upload needed, returns presigned URL)
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

// CompleteServiceDocUpload completes a service doc upload after S3 upload succeeds
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

// SyncMap upserts a map (project) by name
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

// PrepareFrameSync prepares a frame upsert and checks if the image needs uploading
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

// CompleteFrameSync finalizes a frame after S3 image upload
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

// SyncFocalPoint upserts a focal point by name within a map/frame
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

// SyncFocalPointMeta upserts focal point meta (component link) within a named focal point
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

// SyncServiceDatabase syncs a service database schema to the gateway
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

// SyncSavedQuery syncs a saved query snippet to the gateway
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

// Print outputs the service sync request as JSON (for dry-run)
func (r *ServiceSyncRequest) Print() {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// Print outputs the API group sync request as JSON (for dry-run)
func (r *APIGroupSyncRequest) Print() {
	// Don't print full spec content in dry-run (can be huge)
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

// formatGatewayError converts gateway error responses into user-friendly messages
func formatGatewayError(statusCode int, respBody []byte) error {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("authentication failed - please check your UIGRAPH_TOKEN")
	case http.StatusBadRequest:
		// Try to parse error message from gateway
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
		// Try to parse retry-after from response
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
