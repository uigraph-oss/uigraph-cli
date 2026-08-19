package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validScreenshotsTestConfig(t *testing.T) (*Config, string) {
	t.Helper()
	dir := t.TempDir()
	shot := filepath.Join(dir, "login.png")
	if err := os.WriteFile(shot, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Version: 1,
		Service: Service{
			Name:        "payments",
			Category:    "backend",
			Description: "handles payments",
			Repository:  Repository{Provider: "github", URL: "https://github.com/example/payments"},
			Ownership:   Ownership{Team: "payments-team"},
		},
		TestPacks: []TestPackRef{{
			Name: "smoke",
			Type: "smoke",
			TestCases: []TestCaseRef{{
				Type:        "manual",
				Title:       "Login",
				Screenshots: []string{shot},
			}},
		}},
	}
	return cfg, dir
}

func TestConfigValidate_Screenshots(t *testing.T) {
	cfg, _ := validScreenshotsTestConfig(t)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestConfigValidate_ScreenshotMissingFile(t *testing.T) {
	cfg, dir := validScreenshotsTestConfig(t)
	cfg.TestPacks[0].TestCases[0].Screenshots[0] = filepath.Join(dir, "missing.png")
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "screenshots[0] file does not exist") {
		t.Fatalf("Validate() error = %v, want screenshots[0] file does not exist", err)
	}
}

func TestConfigValidate_ScreenshotIsDirectory(t *testing.T) {
	cfg, dir := validScreenshotsTestConfig(t)
	sub := filepath.Join(dir, "shots.png")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg.TestPacks[0].TestCases[0].Screenshots[0] = sub
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "must be a file, not a directory") {
		t.Fatalf("Validate() error = %v, want must be a file, not a directory", err)
	}
}

func TestConfigValidate_ScreenshotNonImageExtension(t *testing.T) {
	cfg, dir := validScreenshotsTestConfig(t)
	notes := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(notes, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.TestPacks[0].TestCases[0].Screenshots[0] = notes
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "must be an image") {
		t.Fatalf("Validate() error = %v, want must be an image", err)
	}
}

func TestConfigValidate_ScreenshotUppercaseExtension(t *testing.T) {
	cfg, dir := validScreenshotsTestConfig(t)
	shot := filepath.Join(dir, "Login.PNG")
	if err := os.WriteFile(shot, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.TestPacks[0].TestCases[0].Screenshots[0] = shot
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestConfigLoad_ScreenshotsFromTestCasesFile(t *testing.T) {
	dir := t.TempDir()
	shot := filepath.Join(dir, "login.png")
	if err := os.WriteFile(shot, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	casesPath := filepath.Join(dir, "cases.yaml")
	cases := "testCases:\n  - type: manual\n    title: Login\n    screenshots:\n      - " + shot + "\n"
	if err := os.WriteFile(casesPath, []byte(cases), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, ".uigraph.yaml")
	cfgYAML := "version: 1\ntestPacks:\n  - name: smoke\n    type: smoke\n    testCasesPath: " + casesPath + "\n"
	if err := os.WriteFile(cfgPath, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	got := cfg.TestPacks[0].TestCases[0].Screenshots
	if len(got) != 1 || got[0] != shot {
		t.Fatalf("Screenshots = %v, want [%s]", got, shot)
	}
}
