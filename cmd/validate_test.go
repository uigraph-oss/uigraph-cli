package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func runValidateFixture(t *testing.T, fixture string, args ...string) (string, error) {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fixtureDirectory := filepath.Join(workingDirectory, "testdata", fixture)
	if err := os.Chdir(fixtureDirectory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	var output bytes.Buffer
	command := newValidateCmd()
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(args)
	err = command.Execute()
	return output.String(), err
}

func TestValidateCommandValidWithoutCredentialsOrNetwork(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	t.Setenv("UIGRAPH_TOKEN", "")
	t.Setenv("UIGRAPH_GATEWAY_URL", server.URL)
	t.Setenv("VALIDATE_TEST_MLFLOW_URL", "")
	t.Setenv("VALIDATE_TEST_MLFLOW_TOKEN", "")

	output, err := runValidateFixture(t, "valid", "--config", ".uigraph.yaml")
	if err != nil {
		t.Fatalf("validate error = %v, output = %q", err, output)
	}
	if !strings.Contains(output, "Valid UiGraph configuration: .uigraph.yaml") {
		t.Fatalf("output = %q, want success message", output)
	}
	if requests.Load() != 0 {
		t.Fatalf("network requests = %d, want 0", requests.Load())
	}
}

func TestValidateCommandErrors(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		want    string
	}{
		{name: "missing referenced file", fixture: "missing-reference", want: "apis[0].path file does not exist"},
		{name: "malformed yaml", fixture: "malformed-yaml", want: "failed to parse YAML"},
		{name: "malformed json", fixture: "malformed-json", want: "contextPath contains malformed JSON"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := runValidateFixture(t, test.fixture)
			if err == nil {
				t.Fatalf("validate error = nil, output = %q", output)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validate error = %v, want %q", err, test.want)
			}
		})
	}
}
