package git

import (
	"os/exec"
	"testing"
)

func TestCaptureMetadata(t *testing.T) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	meta := CaptureMetadata()

	// We can't assert exact values since they depend on the repo state,
	// but we can check that the function runs without panic
	t.Logf("Commit Hash: %s", meta.CommitHash)
	t.Logf("Branch: %s", meta.Branch)
	t.Logf("Is Dirty: %v", meta.IsDirty)
	t.Logf("Remote URL: %s", meta.RemoteURL)
}

func TestCaptureMetadataStructure(t *testing.T) {
	// Test that CaptureMetadata returns a proper struct even if git fails
	meta := CaptureMetadata()

	// Check that the struct is not nil (it's a value, not pointer)
	if meta.CommitHash == "" && meta.Branch == "" && !meta.IsDirty && meta.RemoteURL == "" {
		t.Log("Git metadata appears empty (git might not be available or not in a repo)")
	}
}

func TestRunGitCommand(t *testing.T) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid command - version",
			args:    []string{"--version"},
			wantErr: false,
		},
		{
			name:    "invalid command",
			args:    []string{"invalid-command-xyz"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runGitCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("runGitCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
