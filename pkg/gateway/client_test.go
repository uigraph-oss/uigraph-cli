package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uigraph-oss/uigraph-cli/pkg/config"
)

func TestSyncSavedQuery_Success(t *testing.T) {
	var gotPath, gotMethod, gotToken string
	var gotReq SavedQuerySyncRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotToken = r.Header.Get("X-API-Token")

		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SavedQuerySyncResponse{
			SourceRef: gotReq.SourceRef,
			ID:        "query-123",
			Created:   true,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token")

	req := SavedQuerySyncRequest{
		ServiceName: "payments",
		DBName:      "primary",
		SourceRef:   "top-customers",
		Title:       "top-customers",
		Description: "Top customers by revenue",
		QueryText:   "select * from customers",
		Tags:        []string{"reporting"},
		Git:         GitMetadataMinimal{CommitHash: "abc123"},
	}

	resp, err := client.SyncSavedQuery(context.Background(), req)
	if err != nil {
		t.Fatalf("SyncSavedQuery() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/v1/sync/service/database/query" {
		t.Errorf("path = %s, want /v1/sync/service/database/query", gotPath)
	}
	if gotToken != "secret-token" {
		t.Errorf("X-API-Token = %s, want secret-token", gotToken)
	}
	if gotReq.ServiceName != "payments" || gotReq.DBName != "primary" || gotReq.SourceRef != "top-customers" {
		t.Errorf("request body mismatch: %+v", gotReq)
	}
	if gotReq.QueryText != "select * from customers" {
		t.Errorf("queryText = %q, want %q", gotReq.QueryText, "select * from customers")
	}
	if len(gotReq.Tags) != 1 || gotReq.Tags[0] != "reporting" {
		t.Errorf("tags = %v, want [reporting]", gotReq.Tags)
	}

	if !resp.Created {
		t.Errorf("resp.Created = false, want true")
	}
	if resp.ID != "query-123" {
		t.Errorf("resp.ID = %q, want query-123", resp.ID)
	}
}

func TestSyncSavedQuery_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"boom"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token")

	_, err := client.SyncSavedQuery(context.Background(), SavedQuerySyncRequest{
		ServiceName: "payments",
		DBName:      "primary",
		SourceRef:   "q",
		Title:       "q",
		QueryText:   "select 1",
	})
	if err == nil {
		t.Fatal("SyncSavedQuery() error = nil, want error on 500 status")
	}
}

func TestSyncServiceDependencies_Success(t *testing.T) {
	var gotPath, gotMethod, gotToken string
	var gotReq ServiceDependenciesSyncRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotToken = r.Header.Get("X-API-Token")
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ServiceDependenciesSyncResponse{Message: "dependencies synced"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token")
	resp, err := client.SyncServiceDependencies(context.Background(), ServiceDependenciesSyncRequest{
		ServiceName: "payments",
		Dependencies: []config.DependencyRef{{
			Name: "payment-provider", Service: "Stripe Payments", Type: "http", Criticality: "hard", APIGroupName: "payments-v1", APIEndpointNames: []string{"CreatePayment"},
		}},
	})
	if err != nil {
		t.Fatalf("SyncServiceDependencies() error = %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/sync/service/dependencies" || gotToken != "secret-token" {
		t.Errorf("request = %s %s token=%s", gotMethod, gotPath, gotToken)
	}
	if gotReq.ServiceName != "payments" || gotReq.Dependencies[0].Service != "Stripe Payments" || gotReq.Dependencies[0].APIGroupName != "payments-v1" {
		t.Errorf("request body mismatch: %+v", gotReq)
	}
	if resp.Message != "dependencies synced" {
		t.Errorf("response = %+v", resp)
	}
}
