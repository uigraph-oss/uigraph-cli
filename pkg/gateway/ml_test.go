package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSyncMLProjects_Success(t *testing.T) {
	var gotPath, gotMethod, gotToken string
	var gotReq []MLProjectItem

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotToken = r.Header.Get("X-API-Token")

		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mlSyncResponse{Synced: len(gotReq)})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token")

	synced, err := client.SyncMLProjects(context.Background(), []MLProjectItem{
		{Name: "Facebook", Type: "model", SourceType: "mlflow", SourceURL: "http://localhost:5000", Team: "uigraph"},
	})
	if err != nil {
		t.Fatalf("SyncMLProjects() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/v1/sync/ml/projects" {
		t.Errorf("path = %s, want /v1/sync/ml/projects", gotPath)
	}
	if gotToken != "secret-token" {
		t.Errorf("X-API-Token = %s, want secret-token", gotToken)
	}
	if len(gotReq) != 1 || gotReq[0].Name != "Facebook" || gotReq[0].Type != "model" {
		t.Errorf("request body mismatch: %+v", gotReq)
	}
	if synced != 1 {
		t.Errorf("synced = %d, want 1", synced)
	}
}

func TestSyncMLRunSeries_Path(t *testing.T) {
	var gotPath, gotToken string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-API-Token")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mlSyncResponse{Synced: 2})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token")

	synced, err := client.SyncMLRunSeries(context.Background(), "run-abc", []MLSeriesPoint{
		{Key: "loss", Step: 0, Value: 1.5},
		{Key: "loss", Step: 1, Value: 1.2},
	})
	if err != nil {
		t.Fatalf("SyncMLRunSeries() error = %v", err)
	}
	if gotPath != "/v1/sync/ml/runs/run-abc/series" {
		t.Errorf("path = %s, want /v1/sync/ml/runs/run-abc/series", gotPath)
	}
	if gotToken != "secret-token" {
		t.Errorf("X-API-Token = %s, want secret-token", gotToken)
	}
	if synced != 2 {
		t.Errorf("synced = %d, want 2", synced)
	}
}

func TestSyncMLProjects_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"boom"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token")

	_, err := client.SyncMLProjects(context.Background(), []MLProjectItem{{Name: "P", Type: "model"}})
	if err == nil {
		t.Fatal("SyncMLProjects() error = nil, want error on 500 status")
	}
}
