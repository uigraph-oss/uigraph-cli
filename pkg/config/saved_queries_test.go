package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validQueriesTestConfig(t *testing.T) *Config {
	t.Helper()
	dir := t.TempDir()
	queryPath := filepath.Join(dir, "query.sql")
	if err := os.WriteFile(queryPath, []byte("select 1"), 0o644); err != nil {
		t.Fatal(err)
	}

	return &Config{
		Version: 1,
		Project: Project{Name: "test-project"},
		Service: Service{
			Name:        "payments",
			Category:    "backend",
			Description: "handles payments",
			Repository:  Repository{Provider: "github", URL: "https://github.com/example/payments"},
			Ownership:   Ownership{Team: "payments-team"},
		},
		Databases: []DatabaseRef{
			{Name: "primary", Dialect: "postgres", SchemaPath: mustWriteSchema(t, dir)},
		},
		Queries: []QueryRef{
			{Name: "top-customers", Database: "primary", Path: queryPath},
		},
	}
}

func mustWriteSchema(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConfigValidate_Queries(t *testing.T) {
	cfg := validQueriesTestConfig(t)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestConfigValidate_QueryMissingName(t *testing.T) {
	cfg := validQueriesTestConfig(t)
	cfg.Queries[0].Name = ""
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "queries[0].name is required") {
		t.Fatalf("Validate() error = %v, want queries[0].name is required", err)
	}
}

func TestConfigValidate_QueryUnknownDatabase(t *testing.T) {
	cfg := validQueriesTestConfig(t)
	cfg.Queries[0].Database = "does-not-exist"
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "does not match any databases[].name") {
		t.Fatalf("Validate() error = %v, want database mismatch error", err)
	}
}

func TestConfigValidate_QueryRequiresExactlyOnePathOrText(t *testing.T) {
	cfg := validQueriesTestConfig(t)

	// Neither set.
	cfg.Queries[0].Path = ""
	cfg.Queries[0].QueryText = ""
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "exactly one of path or queryText") {
		t.Fatalf("Validate() error = %v, want exactly-one error", err)
	}

	// Both set.
	cfg.Queries[0].Path = "somefile.sql"
	cfg.Queries[0].QueryText = "select 1"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "exactly one of path or queryText") {
		t.Fatalf("Validate() error = %v, want exactly-one error", err)
	}
}

func TestConfigValidate_QueryPathMustExist(t *testing.T) {
	cfg := validQueriesTestConfig(t)
	cfg.Queries[0].Path = "/nonexistent/query.sql"
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "file does not exist") {
		t.Fatalf("Validate() error = %v, want file-does-not-exist error", err)
	}
}

func TestConfigValidate_QueryInlineTextIsValid(t *testing.T) {
	cfg := validQueriesTestConfig(t)
	cfg.Queries[0].Path = ""
	cfg.Queries[0].QueryText = "select * from billing"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}
