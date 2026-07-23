package config

import "testing"

func TestConfigValidateML(t *testing.T) {
	modelProject := func(models []MLModelRef, experiments []MLExperimentRef) Config {
		return Config{
			Version: 1,
			Project: Project{Name: "test-project"},
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

	tests := []struct {
		name    string
		config  Config
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
				Project: Project{Name: "test-project"},
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
				Project: Project{Name: "test-project"},
				ML:      []MLProjectRef{{Type: "model", Source: MLSourceRef{Type: "mlflow", URL: "http://x"}, Models: []MLModelRef{{Name: "m"}}}},
			},
			wantErr: true,
			errMsg:  "ml[0].name is required",
		},
		{
			name: "invalid type",
			config: Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				ML:      []MLProjectRef{{Name: "P", Type: "dataset", Source: MLSourceRef{Type: "mlflow", URL: "http://x"}}},
			},
			wantErr: true,
			errMsg:  "must be one of: model, training",
		},
		{
			name: "invalid source type",
			config: Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				ML:      []MLProjectRef{{Name: "P", Type: "model", Source: MLSourceRef{Type: "wandb", URL: "http://x"}, Models: []MLModelRef{{Name: "m"}}}},
			},
			wantErr: true,
			errMsg:  "source.type must be: mlflow",
		},
		{
			name: "missing source url",
			config: Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				ML:      []MLProjectRef{{Name: "P", Type: "model", Source: MLSourceRef{Type: "mlflow"}, Models: []MLModelRef{{Name: "m"}}}},
			},
			wantErr: true,
			errMsg:  "source.url is required",
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
				Project: Project{Name: "test-project"},
				ML:      []MLProjectRef{{Name: "T", Type: "training", Source: MLSourceRef{Type: "mlflow", URL: "http://x"}}},
			},
			wantErr: true,
			errMsg:  "a training project must declare experiments",
		},
		{
			name: "training project with models",
			config: Config{
				Version: 1,
				Project: Project{Name: "test-project"},
				ML:      []MLProjectRef{{Name: "T", Type: "training", Source: MLSourceRef{Type: "mlflow", URL: "http://x"}, Experiments: []MLExperimentRef{{Name: "e"}}, Models: []MLModelRef{{Name: "m"}}}},
			},
			wantErr: true,
			errMsg:  "a training project must not declare models",
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
