package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid config",
			content: `version: 1
project:
  name: test-project
service:
  name: Test Service
  category: Backend
  description: Test description
  repository:
    provider: github
    url: https://github.com/test/repo
`,
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			content: `invalid: yaml: content:`,
			wantErr: true,
		},
		{
			name:    "empty file",
			content: "",
			wantErr: false, // yaml.Unmarshal doesn't error on empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, ".uigraph.yaml")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := Load(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Error("Load() should return error for non-existent file")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				Service: Service{
					Name:        "Test Service",
					Category:    "Backend",
					Description: "Test description",
					Repository: Repository{
						Provider: "github",
						URL:      "https://github.com/test/repo",
					},
					Ownership: Ownership{Team: "platform"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid version",
			config: Config{
				Version: 2,
				Project: Project{Name: "test-project"},
			},
			wantErr: true,
			errMsg:  "unsupported config version",
		},
		{
			name: "missing project name",
			config: Config{
				Version: 1,
				Project: Project{},
			},
			wantErr: true,
			errMsg:  "project.name is required",
		},
		{
			name: "no service, maps only",
			config: Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				Maps: []MapRef{
					{
						Name: "Checkout",
						Frames: []FrameRef{
							{
								Name: "Cart",
								FocalPoints: []FocalPointRef{
									{Name: "Total", X: 10, Y: 20},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no service, apis present",
			config: Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				APIs:    []APIRef{{Name: "test-api", Type: "openapi", Path: "spec.yaml"}},
			},
			wantErr: true,
			errMsg:  "service is required to sync apis",
		},
		{
			name: "no service, docs present",
			config: Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				Docs:    []DocRef{{Name: "readme", Path: "README.md"}},
			},
			wantErr: true,
			errMsg:  "service is required to sync docs",
		},
		{
			name: "missing service category",
			config: Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				Service: Service{Name: "Test Service"},
			},
			wantErr: true,
			errMsg:  "service.category is required",
		},
		{
			name: "invalid repository provider",
			config: Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				Service: Service{
					Name:        "Test Service",
					Category:    "Backend",
					Description: "Test",
					Repository: Repository{
						Provider: "invalid",
						URL:      "https://example.com",
					},
				},
			},
			wantErr: true,
			errMsg:  "must be one of: github, gitlab, bitbucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestConfigValidateAPIs(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		apis    []APIRef
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid api",
			apis: []APIRef{
				{Name: "test-api", Type: "openapi", Path: tmpFile},
			},
			wantErr: false,
		},
		{
			name: "missing api name",
			apis: []APIRef{
				{Type: "openapi", Path: tmpFile},
			},
			wantErr: true,
			errMsg:  "apis[0].name is required",
		},
		{
			name: "invalid api type",
			apis: []APIRef{
				{Name: "test-api", Type: "invalid", Path: tmpFile},
			},
			wantErr: true,
			errMsg:  "must be one of: openapi, graphql, grpc",
		},
		{
			name: "missing api path",
			apis: []APIRef{
				{Name: "test-api", Type: "openapi"},
			},
			wantErr: true,
			errMsg:  "apis[0].path is required",
		},
		{
			name: "non-existent file",
			apis: []APIRef{
				{Name: "test-api", Type: "openapi", Path: "nonexistent.yaml"},
			},
			wantErr: true,
			errMsg:  "file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				Service: Service{
					Name:        "Test Service",
					Category:    "Backend",
					Description: "Test",
					Repository: Repository{
						Provider: "github",
						URL:      "https://github.com/test/repo",
					},
					Ownership: Ownership{Team: "platform"},
				},
				APIs: tt.apis,
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
