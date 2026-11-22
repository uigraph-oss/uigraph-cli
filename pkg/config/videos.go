package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// WalkthroughFileType is the doc file type a video's YAML is stored under. A repo does not
// write it any more: videos have their own top-level section. ("video" is an uploaded video file.)
const WalkthroughFileType = "walkthrough"

// DefaultVideoGlob is what the videos section covers when the config does not list any.
const DefaultVideoGlob = ".uigraph/videos/*.video.yaml"

// videoDocs turns the videos section into docs, named by each video's title, so videos sync
// through the same path as other files. Entries are paths or globs relative to the config file.
// Without a section, the default glob applies and files that are not videos are skipped; a file
// the config names itself must be a video, or the sync stops with a clear error.
func videoDocs(root string, entries []string) ([]DocRef, error) {
	explicit := len(entries) > 0
	if !explicit {
		entries = []string{DefaultVideoGlob}
	}
	seen := map[string]bool{}
	var docs []DocRef
	for i, entry := range entries {
		pattern := entry
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(root, pattern)
		}
		paths, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("videos[%d] is not a valid path or pattern: %s", i, entry)
		}
		sort.Strings(paths)
		if explicit && len(paths) == 0 {
			return nil, fmt.Errorf("videos[%d] matches no file: %s", i, entry)
		}
		for _, p := range paths {
			if seen[p] {
				continue
			}
			seen[p] = true
			doc, ok, err := videoDoc(p)
			if err != nil {
				return nil, err
			}
			if !ok {
				if explicit {
					return nil, fmt.Errorf("videos[%d] is not a video file (it has no top-level \"walkthrough:\" key): %s", i, p)
				}
				continue
			}
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func videoDoc(path string) (DocRef, bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return DocRef{}, false, fmt.Errorf("could not read video file %s: %v", path, err)
	}
	var head struct {
		Walkthrough string `yaml:"walkthrough"`
		Title       string `yaml:"title"`
		Description string `yaml:"description"`
	}
	if yaml.Unmarshal(b, &head) != nil || head.Walkthrough == "" {
		return DocRef{}, false, nil
	}
	name := head.Title
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), ".video.yaml")
	}
	return DocRef{Name: name, Path: path, FileType: WalkthroughFileType, Description: head.Description}, true, nil
}
