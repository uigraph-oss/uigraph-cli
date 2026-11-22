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

const loginVideo = "walkthrough: login\ntitle: How login works\ndescription: Sign-in, start to end.\nscenes: [{say: hi}]\n"

func TestVideosAreFoundWithoutAnySection(t *testing.T) {
	cfg, err := Load(videoRepo(t, "", map[string]string{
		".uigraph/videos/login.video.yaml": loginVideo,
		".uigraph/videos/plain.video.yaml": "title: Not a video\nscenes: []\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Docs) != 1 || cfg.Docs[0].Name != "How login works" || cfg.Docs[0].FileType != WalkthroughFileType ||
		cfg.Docs[0].Description != "Sign-in, start to end." {
		t.Fatalf("docs: %+v", cfg.Docs)
	}
}

func TestVideosSectionListsPathsAndGlobs(t *testing.T) {
	cfg, err := Load(videoRepo(t, "videos:\n  - tours/*.video.yaml\n  - extra/one.yaml\n", map[string]string{
		"tours/a.video.yaml":                 loginVideo,
		"tours/b.video.yaml":                 "walkthrough: b\nscenes: []\n",
		"extra/one.yaml":                     "walkthrough: one\ntitle: One\nscenes: []\n",
		".uigraph/videos/ignored.video.yaml": loginVideo, // not listed, so not synced
	}))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, d := range cfg.Docs {
		names = append(names, d.Name)
	}
	if got := strings.Join(names, ","); got != "How login works,b,One" {
		t.Fatalf("docs = %s", got)
	}
}

func TestVideosSectionErrors(t *testing.T) {
	_, err := Load(videoRepo(t, "videos:\n  - nope/*.video.yaml\n", nil))
	if err == nil || !strings.Contains(err.Error(), "videos[0] matches no file") {
		t.Fatalf("missing file: %v", err)
	}
	_, err = Load(videoRepo(t, "videos:\n  - a.yaml\n", map[string]string{"a.yaml": "title: nope\n"}))
	if err == nil || !strings.Contains(err.Error(), "is not a video file") {
		t.Fatalf("not a video: %v", err)
	}
}

func TestVideoAlsoListedAsDocIsNotDuplicated(t *testing.T) {
	root := videoRepo(t, "", map[string]string{".uigraph/videos/login.video.yaml": loginVideo})
	dir := filepath.Dir(root)
	p := filepath.Join(dir, ".uigraph/videos/login.video.yaml")
	if err := os.WriteFile(root, []byte("version: 1\nservice:\n  name: Svc\ndocs:\n  - name: Login\n    path: "+p+"\n    fileType: walkthrough\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Docs) != 1 || cfg.Docs[0].Name != "Login" {
		t.Fatalf("docs: %+v", cfg.Docs)
	}
}
