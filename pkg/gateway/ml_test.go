package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSyncMLProject_Success(t *testing.T) {
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
		_ = json.NewEncoder(w).Encode(mlSyncResponse{Synced: len(gotReq), Created: 1})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token")

	res, err := client.SyncMLProject(context.Background(), MLProjectItem{
		Name: "Facebook", Type: "model", SourceType: "mlflow", SourceURL: "http://localhost:5000", Team: "uigraph",
	})
	if err != nil {
		t.Fatalf("SyncMLProject() error = %v", err)
	}
	if res.Created != 1 || res.Updated != 0 {
		t.Errorf("result = %+v, want created 1 updated 0", res)
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
}

func TestSyncMLRun_OneRequestPerItem(t *testing.T) {
	var paths []string
	var bodySizes []int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		raw, _ := io.ReadAll(r.Body)
		var items []MLRunItem
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		bodySizes = append(bodySizes, len(items))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mlSyncResponse{Synced: len(items), Updated: len(items)})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token")

	runs := []MLRunItem{
		{MLflowID: "run-abc", ExperimentMLflowID: "exp-1", Name: "run-abc", Metrics: map[string]any{"loss": 1.2}},
		{MLflowID: "run-def", ExperimentMLflowID: "exp-1", Name: "run-def", Metrics: map[string]any{"loss": 1.5}},
		{MLflowID: "run-ghi", ExperimentMLflowID: "exp-1", Name: "run-ghi", Metrics: map[string]any{"loss": 1.9}},
	}
	for _, run := range runs {
		res, err := client.SyncMLRun(context.Background(), run)
		if err != nil {
			t.Fatalf("SyncMLRun() error = %v", err)
		}
		if res.Updated != 1 {
			t.Errorf("result = %+v, want updated 1", res)
		}
	}

	if len(paths) != len(runs) {
		t.Fatalf("requests = %d, want %d (one per run)", len(paths), len(runs))
	}
	for i, path := range paths {
		if path != "/v1/sync/ml/runs" {
			t.Errorf("request %d path = %s, want /v1/sync/ml/runs", i, path)
		}
		if bodySizes[i] != 1 {
			t.Errorf("request %d carried %d items, want exactly 1", i, bodySizes[i])
		}
	}
}

func TestSyncMLProject_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token")

	_, err := client.SyncMLProject(context.Background(), MLProjectItem{Name: "P", Type: "model"})
	if err == nil {
		t.Fatal("SyncMLProject() error = nil, want error on 500 status")
	}
}
