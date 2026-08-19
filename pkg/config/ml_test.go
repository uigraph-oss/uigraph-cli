package config

import "testing"

func TestConfigValidateML(t *testing.T) {
	modelProject := func(models []MLModelRef, experiments []MLExperimentRef) Config {
		return Config{
			Version: 1,
			ML: []MLProjectRef{
				{
					Name:        "Facebook",
					Type:        "model",
					Source:      MLSourceRef{Type: "mlflow", URL: "http://localhost:5000"},
					Models:      models,
					Experiments: experiments,
				},
			},
		}
	}

	envProject := func(source MLSourceRef) Config {
		return Config{
			Version: 1,
			ML:      []MLProjectRef{{Name: "P", Type: "model", Source: source, Models: []MLModelRef{{Name: "m"}}}},
		}
	}

	tests := []struct {
		name    string
		config  Config
		env     map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid model project",
			config:  modelProject([]MLModelRef{{Name: "Saba"}}, nil),
			wantErr: false,
		},
		{
			name: "valid training project",
			config: Config{
				Version: 1,
				ML: []MLProjectRef{
					{
						Name:        "Facebook Training",
						Type:        "training",
						Source:      MLSourceRef{Type: "mlflow", URL: "http://localhost:5000"},
						Experiments: []MLExperimentRef{{Name: "Sample experiment"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: Config{
				Version: 1,
				ML:      []MLProjectRef{{Type: "model", Source: MLSourceRef{Type: "mlflow", URL: "http://x"}, Models: []MLModelRef{{Name: "m"}}}},
			},
			wantErr: true,
			errMsg:  "ml[0].name is required",
		},
		{
			name: "invalid type",
			config: Config{
				Version: 1,
				ML:      []MLProjectRef{{Name: "P", Type: "dataset", Source: MLSourceRef{Type: "mlflow", URL: "http://x"}}},
			},
			wantErr: true,
			errMsg:  "must be one of: model, training",
		},
		{
			name: "invalid source type",
			config: Config{
				Version: 1,
				ML:      []MLProjectRef{{Name: "P", Type: "model", Source: MLSourceRef{Type: "wandb", URL: "http://x"}, Models: []MLModelRef{{Name: "m"}}}},
			},
			wantErr: true,
			errMsg:  "source.type must be: mlflow",
		},
		{
			name:    "missing source url and urlEnv",
			config:  envProject(MLSourceRef{Type: "mlflow"}),
			wantErr: true,
			errMsg:  "either url or urlEnv is required",
		},
		{
			name:    "both url and urlEnv",
			config:  envProject(MLSourceRef{Type: "mlflow", URL: "http://x", URLEnv: "TEST_MLFLOW_URL"}),
			env:     map[string]string{"TEST_MLFLOW_URL": "http://y"},
			wantErr: true,
			errMsg:  "specify either url or urlEnv, not both",
		},
		{
			name:    "urlEnv points at unset variable",
			config:  envProject(MLSourceRef{Type: "mlflow", URLEnv: "TEST_MLFLOW_URL"}),
			env:     map[string]string{"TEST_MLFLOW_URL": ""},
			wantErr: true,
			errMsg:  "urlEnv: environment variable TEST_MLFLOW_URL is not set or empty",
		},
		{
			name:    "urlEnv resolves",
			config:  envProject(MLSourceRef{Type: "mlflow", URLEnv: "TEST_MLFLOW_URL"}),
			env:     map[string]string{"TEST_MLFLOW_URL": "http://y"},
			wantErr: false,
		},
		{
			name:    "tokenEnv points at unset variable",
			config:  envProject(MLSourceRef{Type: "mlflow", URL: "http://x", TokenEnv: "TEST_MLFLOW_TOKEN"}),
			env:     map[string]string{"TEST_MLFLOW_TOKEN": ""},
			wantErr: true,
			errMsg:  "tokenEnv: environment variable TEST_MLFLOW_TOKEN is not set or empty",
		},
		{
			name:    "tokenEnv resolves",
			config:  envProject(MLSourceRef{Type: "mlflow", URL: "http://x", TokenEnv: "TEST_MLFLOW_TOKEN"}),
			env:     map[string]string{"TEST_MLFLOW_TOKEN": "secret"},
			wantErr: false,
		},
		{
			name:    "model project without models",
			config:  modelProject(nil, nil),
			wantErr: true,
			errMsg:  "a model project must declare models",
		},
		{
			name:    "model project with experiments",
			config:  modelProject([]MLModelRef{{Name: "Saba"}}, []MLExperimentRef{{Name: "e"}}),
			wantErr: true,
			errMsg:  "a model project must not declare experiments",
		},
		{
			name: "training project without experiments",
			config: Config{
				Version: 1,
				ML:      []MLProjectRef{{Name: "T", Type: "training", Source: MLSourceRef{Type: "mlflow", URL: "http://x"}}},
			},
			wantErr: true,
			errMsg:  "a training project must declare experiments",
		},
		{
			name: "training project with models",
			config: Config{
				Version: 1,
				ML:      []MLProjectRef{{Name: "T", Type: "training", Source: MLSourceRef{Type: "mlflow", URL: "http://x"}, Experiments: []MLExperimentRef{{Name: "e"}}, Models: []MLModelRef{{Name: "m"}}}},
			},
			wantErr: true,
			errMsg:  "a training project must not declare models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
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
