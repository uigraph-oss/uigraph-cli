package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_QueryFilesMerge(t *testing.T) {
	dir := t.TempDir()
	schemaPath := mustWriteSchema(t, dir)
	queriesFilePath := writeFile(t, dir, "queries.yaml", `queries:
  - name: external-a
    database: primary
    queryText: "select 1"
  - name: external-b
    database: primary
    queryText: "select 2"
`)
	mainPath := writeFile(t, dir, ".uigraph.yaml", `version: 1
project:
  name: test-project
service:
  name: payments
  category: backend
  description: handles payments
  repository:
    provider: github
    url: https://github.com/example/payments
  ownership:
    team: payments-team
databases:
  - name: primary
    dialect: postgres
    schemaPath: `+schemaPath+`
queries:
  - name: inline-a
    database: primary
    queryText: "select 0"
queryFiles:
  - `+queriesFilePath+`
`)

	cfg, err := Load(mainPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Queries) != 3 {
		t.Fatalf("len(cfg.Queries) = %d, want 3", len(cfg.Queries))
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestLoad_TestCasesPathMerge(t *testing.T) {
	dir := t.TempDir()
	casesPath := writeFile(t, dir, "cases.yaml", `testCases:
  - title: External case one
    type: manual
    order: 2
  - title: External case two
    type: manual
    order: 3
`)
	mainPath := writeFile(t, dir, ".uigraph.yaml", `version: 1
project:
  name: test-project
service:
  name: payments
  category: backend
  description: handles payments
  repository:
    provider: github
    url: https://github.com/example/payments
  ownership:
    team: payments-team
testPacks:
  - name: Smoke
    type: smoke
    testCases:
      - title: Inline case
        type: manual
        order: 1
    testCasesPath: `+casesPath+`
`)

	cfg, err := Load(mainPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := len(cfg.TestPacks[0].TestCases); got != 3 {
		t.Fatalf("len(testCases) = %d, want 3", got)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestLoad_QueryFilesMissing(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeFile(t, dir, ".uigraph.yaml", `version: 1
project:
  name: test-project
queryFiles:
  - /nonexistent/queries.yaml
`)
	_, err := Load(mainPath)
	if err == nil || !strings.Contains(err.Error(), "failed to read queries file") {
		t.Fatalf("Load() error = %v, want read-queries-file error", err)
	}
}

func TestLoad_TestCasesPathMissing(t *testing.T) {
	dir := t.TempDir()
	mainPath := writeFile(t, dir, ".uigraph.yaml", `version: 1
project:
  name: test-project
testPacks:
  - name: Smoke
    type: smoke
    testCasesPath: /nonexistent/cases.yaml
`)
	_, err := Load(mainPath)
	if err == nil || !strings.Contains(err.Error(), "failed to read testCases file") {
		t.Fatalf("Load() error = %v, want read-testCases-file error", err)
	}
}

func TestLoad_QueryFilesValidationOnMerged(t *testing.T) {
	dir := t.TempDir()
	schemaPath := mustWriteSchema(t, dir)
	queriesFilePath := writeFile(t, dir, "queries.yaml", `queries:
  - name: external-bad
    database: does-not-exist
    queryText: "select 1"
`)
	mainPath := writeFile(t, dir, ".uigraph.yaml", `version: 1
project:
  name: test-project
service:
  name: payments
  category: backend
  description: handles payments
  repository:
    provider: github
    url: https://github.com/example/payments
  ownership:
    team: payments-team
databases:
  - name: primary
    dialect: postgres
    schemaPath: `+schemaPath+`
queryFiles:
  - `+queriesFilePath+`
`)

	cfg, err := Load(mainPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	err = cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "does not match any databases[].name") {
		t.Fatalf("Validate() error = %v, want database mismatch error", err)
	}
}
