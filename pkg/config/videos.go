package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// WalkthroughFileType is the doc file type a video's YAML is stored under. A repo does not
// write it: videos have their own top-level section. ("video" is an uploaded video file.)
const WalkthroughFileType = "walkthrough"

// videoDocs turns the videos section into docs so videos sync through the same path as other
// files. Each entry is the path of a video's YAML, relative to the config file, like the paths of
// diagrams and docs. A video is named by its title, or by its file name when it has none.
func videoDocs(root string, entries []string) ([]DocRef, error) {
	var docs []DocRef
	for i, entry := range entries {
		if strings.TrimSpace(entry) == "" {
			return nil, fmt.Errorf("videos[%d] is empty", i)
		}
		path := entry
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		var head struct {
			Title       string `yaml:"title"`
			Description string `yaml:"description"`
		}
		if err := readYAML(path, fmt.Sprintf("videos[%d] file", i), &head); err != nil {
			return nil, err
		}
		name := head.Title
		if name == "" {
			base := filepath.Base(path)
			name = strings.TrimSuffix(strings.TrimSuffix(base, filepath.Ext(base)), ".video")
		}
		docs = append(docs, DocRef{Name: name, Path: path, FileType: WalkthroughFileType, Description: head.Description})
	}
	return docs, nil
}
