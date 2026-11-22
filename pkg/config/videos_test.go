package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func videoRepo(t *testing.T, config string, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := filepath.Join(root, ".uigraph.yaml")
	if err := os.WriteFile(cfg, []byte("version: 1\nservice:\n  name: Svc\n"+config), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestVideosSectionListsPaths(t *testing.T) {
	cfg, err := Load(videoRepo(t, "videos:\n  - tours/checkout.yaml\n  - login.video.yaml\n  - plain.yml\n", map[string]string{
		"tours/checkout.yaml": "title: How checkout works\ndescription: Cart to receipt.\nscenes: []\n",
		"login.video.yaml":    "scenes: []\n",
		"plain.yml":           "title: Plain\n",
		"other.video.yaml":    "title: Not listed\n", // not listed, so not synced
	}))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, d := range cfg.Docs {
		if d.FileType != WalkthroughFileType {
			t.Errorf("%s has file type %q", d.Name, d.FileType)
		}
		names = append(names, d.Name)
	}
	if got := strings.Join(names, ","); got != "How checkout works,login,Plain" {
		t.Fatalf("docs = %s", got)
	}
	if cfg.Docs[0].Description != "Cart to receipt." {
		t.Errorf("description = %q", cfg.Docs[0].Description)
	}
}

func TestNoVideosSectionSyncsNoVideos(t *testing.T) {
	cfg, err := Load(videoRepo(t, "", map[string]string{".uigraph/videos/login.video.yaml": "title: Login\n"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Docs) != 0 {
		t.Fatalf("docs: %+v", cfg.Docs)
	}
}

func TestVideoThatDoesNotExistIsAnError(t *testing.T) {
	_, err := Load(videoRepo(t, "videos:\n  - nope.yaml\n", nil))
	if err == nil || !strings.Contains(err.Error(), "videos[0] file not found") {
		t.Fatalf("err = %v", err)
	}
}
