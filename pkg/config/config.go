package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var validScreenshotExt = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".webp": true,
	".svg":  true,
}

type Config struct {
	Version              int              `yaml:"version"`
	Service              Service          `yaml:"service"`
	APIs                 []APIRef         `yaml:"apis"`
	Dependencies         []DependencyRef  `yaml:"dependencies,omitempty"`
	ArchitectureDiagrams []ArchDiagramRef `yaml:"architectureDiagrams,omitempty"`
	TestPacks            []TestPackRef    `yaml:"testPacks,omitempty"`
	Databases            []DatabaseRef    `yaml:"databases,omitempty"`
	Queries              []QueryRef       `yaml:"queries,omitempty"`
	QueryFiles           []string         `yaml:"queryFiles,omitempty"`
	Docs                 []DocRef         `yaml:"docs,omitempty"`
	Maps                 []MapRef         `yaml:"maps,omitempty"`
	ML                   []MLProjectRef   `yaml:"ml,omitempty"`
	CostTags             []CostTag        `yaml:"costTags,omitempty"`
	Timeline             *TimelineRef     `yaml:"timeline,omitempty"`
}

type CostTag struct {
	Key   string `yaml:"key" json:"key"`
	Value string `yaml:"value"  json:"value"`
}

type TimelineRef struct {
	Decisions TimelineScanRef    `yaml:"decisions,omitempty"`
	Incidents TimelineScanRef    `yaml:"incidents,omitempty"`
	Releases  TimelineReleaseRef `yaml:"releases,omitempty"`
}

type TimelineScanRef struct {
	Paths []string `yaml:"paths,omitempty"`
}

type TimelineReleaseRef struct {
	ChangelogPath string `yaml:"changelogPath,omitempty"`
}

type MLProjectRef struct {
	Name        string            `yaml:"name"`
	Type        string            `yaml:"type"`
	Description string            `yaml:"description,omitempty"`
	Ownership   Ownership         `yaml:"ownership,omitempty"`
	Source      MLSourceRef       `yaml:"source"`
	Models      []MLModelRef      `yaml:"models,omitempty"`
	Experiments []MLExperimentRef `yaml:"experiments,omitempty"`
}

type MLSourceRef struct {
	Type     string `yaml:"type"`
	URL      string `yaml:"url,omitempty"`
	URLEnv   string `yaml:"urlEnv,omitempty"`
	TokenEnv string `yaml:"tokenEnv,omitempty"`
}

func (s MLSourceRef) ResolveURL() (string, error) {
	if s.URL != "" {
		return s.URL, nil
	}
	if s.URLEnv != "" {
		v := os.Getenv(s.URLEnv)
		if v == "" {
			return "", fmt.Errorf("environment variable %s is not set or empty", s.URLEnv)
		}
		return v, nil
	}
	return "", fmt.Errorf("neither url nor urlEnv is set")
}

func (s MLSourceRef) ResolveToken() (string, error) {
	if s.TokenEnv == "" {
		return "", nil
	}
	v := os.Getenv(s.TokenEnv)
	if v == "" {
		return "", fmt.Errorf("environment variable %s is not set or empty", s.TokenEnv)
	}
	return v, nil
}

type MLModelRef struct {
	Name            string `yaml:"name"`
	Description     string `yaml:"description,omitempty"`
	ProblemType     string `yaml:"problemType,omitempty"`
	Domain          string `yaml:"domain,omitempty"`
	License         string `yaml:"license,omitempty"`
	IntendedUse     string `yaml:"intendedUse,omitempty"`
	Limitations     string `yaml:"limitations,omitempty"`
	Recommendations string `yaml:"recommendations,omitempty"`
	Considerations  string `yaml:"considerations,omitempty"`
}

type MLExperimentRef struct {
	Name string `yaml:"name"`
}

var validProblemType = map[string]bool{
	"classification": true,
	"regression":     true,
	"ranking":        true,
	"generation":     true,
	"embedding":      true,
	"other":          true,
}

type MapRef struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description,omitempty"`
	Frames      []FrameRef `yaml:"frames,omitempty"`
}

type FrameRef struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description,omitempty"`
	ImagePath   string          `yaml:"imagePath,omitempty"`
	FocalPoints []FocalPointRef `yaml:"focalPoints,omitempty"`
}

type FocalPointRef struct {
	Name       string              `yaml:"name"`
	X          float64             `yaml:"x"`
	Y          float64             `yaml:"y"`
	Visibility string              `yaml:"visibility,omitempty"`
	Components []FocalPointMetaRef `yaml:"components,omitempty"`
}

type FocalPointMetaRef struct {
	ComponentID string `yaml:"componentId"`

	ComponentLinkID string `yaml:"componentLinkId,omitempty"`

	ServiceName             string `yaml:"serviceName,omitempty"`
	APIGroupName            string `yaml:"apiGroupName,omitempty"`
	OperationID             string `yaml:"operationId,omitempty"`
	TestPackName            string `yaml:"testPackName,omitempty"`
	DocName                 string `yaml:"docName,omitempty"`
	ArchitectureDiagramName string `yaml:"architectureDiagramName,omitempty"`

	ModalFields []ComponentModalFieldRef `yaml:"modalFields,omitempty"`
}

type ComponentModalFieldRef struct {
	ComponentFieldID string        `yaml:"componentFieldId"`
	Label            string        `yaml:"label,omitempty"`
	Type             string        `yaml:"type,omitempty"`
	Data             []interface{} `yaml:"data,omitempty"`
}

type Service struct {
	Name         string       `yaml:"name" json:"name"`
	Category     string       `yaml:"category" json:"category"`
	Description  string       `yaml:"description" json:"description"`
	Repository   Repository   `yaml:"repository" json:"repository"`
	Ownership    Ownership    `yaml:"ownership" json:"ownership"`
	Labels       []string     `yaml:"labels,omitempty" json:"labels,omitempty"`
	Integrations Integrations `yaml:"integrations,omitempty" json:"integrations,omitempty"`
}

type Repository struct {
	Provider string `yaml:"provider" json:"provider"`
	URL      string `yaml:"url" json:"url"`
}

type Ownership struct {
	Team  string `yaml:"team,omitempty" json:"team,omitempty"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
}

type Integrations struct {
	Jira  *Integration `yaml:"jira,omitempty" json:"jira,omitempty"`
	Slack *Integration `yaml:"slack,omitempty" json:"slack,omitempty"`
}

type Integration struct {
	URL string `yaml:"url" json:"url"`
}

type APIRef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

type DependencyRef struct {
	Name             string   `yaml:"name" json:"name"`
	Service          string   `yaml:"service" json:"service"`
	Direction        string   `yaml:"direction" json:"direction"`
	Type             string   `yaml:"type,omitempty" json:"type,omitempty"`
	Criticality      string   `yaml:"criticality" json:"criticality"`
	Description      string   `yaml:"description,omitempty" json:"description,omitempty"`
	APIGroupName     string   `yaml:"apiGroupName,omitempty" json:"apiGroupName,omitempty"`
	APIEndpointNames []string `yaml:"apiEndpointNames,omitempty" json:"apiEndpointNames,omitempty"`
	DatabaseName     string   `yaml:"databaseName,omitempty" json:"databaseName,omitempty"`
}

type ArchDiagramRef struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	ContextPath string `yaml:"contextPath,omitempty"`
}

type TestPackRef struct {
	Name          string        `yaml:"name"`
	Type          string        `yaml:"type"`
	Environment   string        `yaml:"environment,omitempty"`
	ReleaseLabel  string        `yaml:"releaseLabel,omitempty"`
	TestCases     []TestCaseRef `yaml:"testCases,omitempty"`
	TestCasesPath string        `yaml:"testCasesPath,omitempty"`
}

type testCasesFile struct {
	TestCases []TestCaseRef `yaml:"testCases"`
}

type queriesFile struct {
	Queries []QueryRef `yaml:"queries"`
}

type StepRef struct {
	Action         string `yaml:"action"`
	ExpectedResult string `yaml:"expectedResult,omitempty"`
}

type AssertionRef struct {
	Field string `yaml:"field"`
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

type TestCaseRef struct {
	Type  string  `yaml:"type"`
	Title string  `yaml:"title"`
	Order float64 `yaml:"order"`

	Description           string   `yaml:"description,omitempty"`
	Priority              string   `yaml:"priority,omitempty"`
	Tags                  []string `yaml:"tags,omitempty"`
	LinkedTicket          string   `yaml:"linkedTicket,omitempty"`
	EstimatedDurationMins int      `yaml:"estimatedDurationMins,omitempty"`
	TestOwner             string   `yaml:"testOwner,omitempty"`

	MapName        string `yaml:"mapName,omitempty"`
	FrameName      string `yaml:"frameName,omitempty"`
	FocalPointName string `yaml:"focalPointName,omitempty"`

	APIGroupName       string         `yaml:"apiGroupName,omitempty"`
	OperationID        string         `yaml:"operationId,omitempty"`
	ExpectedStatusCode int            `yaml:"expectedStatusCode,omitempty"`
	RequestTemplate    string         `yaml:"requestTemplate,omitempty"`
	ResponseTimeMs     int            `yaml:"responseTimeMs,omitempty"`
	ResponseBody       string         `yaml:"responseBody,omitempty"`
	Assertions         []AssertionRef `yaml:"assertions,omitempty"`

	StepsList        []StepRef `yaml:"stepsList,omitempty"`
	ExpectedOutcome  string    `yaml:"expectedOutcome,omitempty"`
	Preconditions    string    `yaml:"preconditions,omitempty"`
	TestData         string    `yaml:"testData,omitempty"`
	Postconditions   string    `yaml:"postconditions,omitempty"`
	RequiresEvidence bool      `yaml:"requiresEvidence"`
	IsCritical       bool      `yaml:"isCritical"`

	Screenshots []string `yaml:"screenshots,omitempty"`
}

type DocRef struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	FileType    string `yaml:"fileType,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type DatabaseRef struct {
	Name       string `yaml:"name"`
	Dialect    string `yaml:"dialect"`
	DBType     string `yaml:"dbType,omitempty"`
	SchemaPath string `yaml:"schemaPath"`
}

type QueryRef struct {
	Name        string   `yaml:"name"`
	Database    string   `yaml:"database"`
	Path        string   `yaml:"path,omitempty"`
	QueryText   string   `yaml:"queryText,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
}

// unmarshalErr matches the go-yaml phrasing for a value whose YAML type does
// not fit the field it is assigned to, so it can be restated in plain words.
var unmarshalErr = regexp.MustCompile("cannot unmarshal (!![a-z]+)(?: `([^`]*)`)? into ([^\\s]+)")

var yamlKindNames = map[string]string{
	"!!str":   "text",
	"!!int":   "a whole number",
	"!!float": "a number",
	"!!bool":  "true or false",
	"!!seq":   "a list",
	"!!map":   "an object",
	"!!null":  "an empty value",
}

func fieldTypeName(goType string) string {
	if strings.HasPrefix(goType, "[]") {
		return "a list"
	}
	switch goType {
	case "int", "int64":
		return "a whole number"
	case "float64":
		return "a number"
	case "string":
		return "text"
	case "bool":
		return "true or false"
	}
	return "an object"
}

// describeYAMLError restates a go-yaml failure as a reader-facing message. Type
// errors carry one entry per bad value, so all of them are reported at once.
func describeYAMLError(path string, err error) error {
	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) {
		messages := make([]string, len(typeErr.Errors))
		for i, message := range typeErr.Errors {
			match := unmarshalErr.FindStringSubmatch(message)
			if match == nil {
				messages[i] = message
				continue
			}
			restated := fmt.Sprintf("expected %s, got %s", fieldTypeName(match[3]), yamlKindNames[match[1]])
			if match[2] != "" {
				restated += fmt.Sprintf(" %q", match[2])
			}
			messages[i] = strings.Replace(message, match[0], restated, 1)
		}
		return fmt.Errorf("%s has invalid values:\n  • %s", path, strings.Join(messages, "\n  • "))
	}
	return fmt.Errorf("%s is not valid YAML: %s", path, strings.TrimPrefix(err.Error(), "yaml: "))
}

func readYAML(path string, kind string, out interface{}) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s not found: %s", kind, path)
	}
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("%s is not readable: %s", kind, path)
	}
	if err != nil {
		return fmt.Errorf("could not read %s %s: %v", kind, path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return describeYAMLError(path, err)
	}
	return nil
}

func Load(path string) (*Config, error) {
	var cfg Config
	if err := readYAML(path, "config file", &cfg); err != nil {
		return nil, err
	}

	for i := range cfg.TestPacks {
		if cfg.TestPacks[i].TestCasesPath == "" {
			continue
		}
		var f testCasesFile
		if err := readYAML(cfg.TestPacks[i].TestCasesPath, fmt.Sprintf("testPacks[%d].testCasesPath file", i), &f); err != nil {
			return nil, err
		}
		cfg.TestPacks[i].TestCases = append(cfg.TestPacks[i].TestCases, f.TestCases...)
	}

	for i, p := range cfg.QueryFiles {
		var f queriesFile
		if err := readYAML(p, fmt.Sprintf("queryFiles[%d] file", i), &f); err != nil {
			return nil, err
		}
		cfg.Queries = append(cfg.Queries, f.Queries...)
	}

	return &cfg, nil
}

// ValidationErrors reports every problem found in one pass, so a single run
// tells the user everything they have to fix rather than one item at a time.
type ValidationErrors []string

func (e ValidationErrors) Error() string {
	if len(e) == 1 {
		return e[0]
	}
	return fmt.Sprintf("%d problems found:\n  • %s", len(e), strings.Join(e, "\n  • "))
}

func (c *Config) Validate() error {
	var problems ValidationErrors

	if c.Version != 1 {
		problems = append(problems, fmt.Sprintf("unsupported config version: %d (expected 1)", c.Version))
	}

	if c.Service.Name != "" {
		if c.Service.Category == "" {
			problems = append(problems, "service.category is required")
		}
		if c.Service.Description == "" {
			problems = append(problems, "service.description is required")
		}

		validProviders := map[string]bool{"github": true, "gitlab": true, "bitbucket": true}
		if c.Service.Repository.Provider == "" {
			problems = append(problems, "service.repository.provider is required")
		} else if !validProviders[c.Service.Repository.Provider] {
			problems = append(problems, "service.repository.provider must be one of: github, gitlab, bitbucket")
		}
		if c.Service.Repository.URL == "" {
			problems = append(problems, "service.repository.url is required")
		}

		if c.Service.Ownership.Team == "" {
			problems = append(problems, "service.ownership.team is required")
		}
	} else {
		if len(c.APIs) > 0 {
			problems = append(problems, "service is required to sync apis; configs without a service may only sync maps and frames")
		}
		if len(c.Databases) > 0 {
			problems = append(problems, "service is required to sync databases; configs without a service may only sync maps and frames")
		}
		if len(c.Queries) > 0 {
			problems = append(problems, "service is required to sync queries; configs without a service may only sync maps and frames")
		}
		if len(c.ArchitectureDiagrams) > 0 {
			problems = append(problems, "service is required to sync architectureDiagrams; configs without a service may only sync maps and frames")
		}
		if len(c.TestPacks) > 0 {
			problems = append(problems, "service is required to sync testPacks; configs without a service may only sync maps and frames")
		}
		if len(c.Docs) > 0 {
			problems = append(problems, "service is required to sync docs; configs without a service may only sync maps and frames")
		}
		if len(c.Dependencies) > 0 {
			problems = append(problems, "service is required to sync dependencies; configs without a service may only sync maps and frames")
		}
		if c.CostTags != nil {
			problems = append(problems, "service is required to sync costTags; configs without a service may only sync maps and frames")
		}
		if c.Timeline != nil {
			problems = append(problems, "service is required to sync timeline; configs without a service may only sync maps and frames")
		}
	}

	validAPITypes := map[string]bool{"openapi": true, "graphql": true, "grpc": true}
	for i, api := range c.APIs {
		if api.Name == "" {
			problems = append(problems, fmt.Sprintf("apis[%d].name is required", i))
		}
		if api.Type == "" {
			problems = append(problems, fmt.Sprintf("apis[%d].type is required", i))
		} else if !validAPITypes[api.Type] {
			problems = append(problems, fmt.Sprintf("apis[%d].type must be one of: openapi, graphql, grpc", i))
		}
		if api.Path == "" {
			problems = append(problems, fmt.Sprintf("apis[%d].path is required", i))
		} else if _, err := os.Stat(api.Path); os.IsNotExist(err) {
			problems = append(problems, fmt.Sprintf("apis[%d].path file does not exist: %s", i, api.Path))
		}
	}

	validDependencyTypes := map[string]bool{"http": true, "graphql": true, "grpc": true, "database": true}
	validCriticalities := map[string]bool{"hard": true, "soft": true}
	validDirections := map[string]bool{"upstream": true, "downstream": true}
	dependencyNames := map[string]bool{}
	for i, dependency := range c.Dependencies {
		if dependency.Name == "" {
			problems = append(problems, fmt.Sprintf("dependencies[%d].name is required", i))
		} else if dependencyNames[dependency.Name] {
			problems = append(problems, fmt.Sprintf("dependencies[%d].name must be unique", i))
		} else {
			dependencyNames[dependency.Name] = true
		}
		switch dependency.Service {
		case "":
			problems = append(problems, fmt.Sprintf("dependencies[%d].service is required", i))
		case c.Service.Name:
			problems = append(problems, fmt.Sprintf("dependencies[%d].service must not reference the current service", i))
		}
		if dependency.Direction == "" {
			problems = append(problems, fmt.Sprintf("dependencies[%d].direction is required", i))
		} else if !validDirections[dependency.Direction] {
			problems = append(problems, fmt.Sprintf("dependencies[%d].direction must be one of: upstream, downstream", i))
		}
		if dependency.Type != "" && !validDependencyTypes[dependency.Type] {
			problems = append(problems, fmt.Sprintf("dependencies[%d].type must be one of: http, graphql, grpc, database", i))
		}
		if dependency.Criticality == "" {
			problems = append(problems, fmt.Sprintf("dependencies[%d].criticality is required", i))
		} else if !validCriticalities[dependency.Criticality] {
			problems = append(problems, fmt.Sprintf("dependencies[%d].criticality must be one of: hard, soft", i))
		}
		endpointNames := map[string]bool{}
		for j, endpointName := range dependency.APIEndpointNames {
			if endpointName == "" {
				problems = append(problems, fmt.Sprintf("dependencies[%d].apiEndpointNames[%d] is required", i, j))
			} else if endpointNames[endpointName] {
				problems = append(problems, fmt.Sprintf("dependencies[%d].apiEndpointNames[%d] must be unique", i, j))
			} else {
				endpointNames[endpointName] = true
			}
		}
	}

	archNames := make(map[string]bool)
	for i, ad := range c.ArchitectureDiagrams {
		if ad.Name == "" {
			problems = append(problems, fmt.Sprintf("architectureDiagrams[%d].name is required", i))
		} else {
			lowerName := strings.ToLower(ad.Name)
			if archNames[lowerName] {
				problems = append(problems, fmt.Sprintf("architectureDiagrams[%d].name %q is a duplicate (diagram names must be unique case-insensitively)", i, ad.Name))
			} else {
				archNames[lowerName] = true
			}
		}
		if ad.Path == "" {
			problems = append(problems, fmt.Sprintf("architectureDiagrams[%d].path is required", i))
		} else if _, err := os.Stat(ad.Path); os.IsNotExist(err) {
			problems = append(problems, fmt.Sprintf("architectureDiagrams[%d].path file does not exist: %s", i, ad.Path))
		}
		if ad.ContextPath != "" {
			if _, err := os.Stat(ad.ContextPath); os.IsNotExist(err) {
				problems = append(problems, fmt.Sprintf("architectureDiagrams[%d].contextPath file does not exist: %s", i, ad.ContextPath))
			}
		}
	}

	validTestPackTypes := map[string]bool{"smoke": true, "regression": true, "manual": true}
	validTestCaseTypes := map[string]bool{"api": true, "manual": true}
	for i, pack := range c.TestPacks {
		if pack.Name == "" {
			problems = append(problems, fmt.Sprintf("testPacks[%d].name is required", i))
		}
		if pack.Type == "" {
			problems = append(problems, fmt.Sprintf("testPacks[%d].type is required", i))
		} else if !validTestPackTypes[pack.Type] {
			problems = append(problems, fmt.Sprintf("testPacks[%d].type must be one of: smoke, regression, manual", i))
		}
		for j, tc := range pack.TestCases {
			if tc.Type == "" {
				problems = append(problems, fmt.Sprintf("testPacks[%d].testCases[%d].type is required", i, j))
			} else if !validTestCaseTypes[tc.Type] {
				problems = append(problems, fmt.Sprintf("testPacks[%d].testCases[%d].type must be one of: api, manual", i, j))
			}
			if tc.Title == "" {
				problems = append(problems, fmt.Sprintf("testPacks[%d].testCases[%d].title is required", i, j))
			}
			for k, shot := range tc.Screenshots {
				if shot == "" {
					problems = append(problems, fmt.Sprintf("testPacks[%d].testCases[%d].screenshots[%d] is required", i, j, k))
					continue
				}
				info, err := os.Stat(shot)
				if os.IsNotExist(err) {
					problems = append(problems, fmt.Sprintf("testPacks[%d].testCases[%d].screenshots[%d] file does not exist: %s", i, j, k, shot))
					continue
				}
				if err != nil {
					problems = append(problems, fmt.Sprintf("testPacks[%d].testCases[%d].screenshots[%d] is not readable: %s", i, j, k, shot))
					continue
				}
				if info.IsDir() {
					problems = append(problems, fmt.Sprintf("testPacks[%d].testCases[%d].screenshots[%d] must be a file, not a directory: %s", i, j, k, shot))
					continue
				}
				if !validScreenshotExt[strings.ToLower(filepath.Ext(shot))] {
					problems = append(problems, fmt.Sprintf("testPacks[%d].testCases[%d].screenshots[%d] must be an image (.png, .jpg, .jpeg, .gif, .webp, .svg): %s", i, j, k, shot))
				}
			}
		}
	}

	tagPairs := map[string]bool{}
	for i, tag := range c.CostTags {
		if tag.Key == "" {
			problems = append(problems, fmt.Sprintf("costTags[%d].key is required", i))
		}
		if tag.Value == "" {
			problems = append(problems, fmt.Sprintf("costTags[%d].value is required", i))
		}
		if tag.Key == "" || tag.Value == "" {
			continue
		}
		pair := tag.Key + "=" + tag.Value
		if tagPairs[pair] {
			problems = append(problems, fmt.Sprintf("costTags[%d]: duplicate key/value pair %q", i, pair))
		}
		tagPairs[pair] = true
	}

	if c.Timeline != nil {
		for i, p := range c.Timeline.Decisions.Paths {
			if p == "" {
				problems = append(problems, fmt.Sprintf("timeline.decisions.paths[%d] is required", i))
			}
		}
		for i, p := range c.Timeline.Incidents.Paths {
			if p == "" {
				problems = append(problems, fmt.Sprintf("timeline.incidents.paths[%d] is required", i))
			}
		}
		if c.Timeline.Releases.ChangelogPath != "" {
			if _, err := os.Stat(c.Timeline.Releases.ChangelogPath); os.IsNotExist(err) {
				problems = append(problems, fmt.Sprintf("timeline.releases.changelogPath file does not exist: %s", c.Timeline.Releases.ChangelogPath))
			}
		}
	}

	validFileTypes := map[string]bool{"pdf": true, "html": true, "markdown": true, "doc": true, "txt": true, "image": true, "video": true, "audio": true, "other": true}
	for i, doc := range c.Docs {
		if doc.Name == "" {
			problems = append(problems, fmt.Sprintf("docs[%d].name is required", i))
		}
		if doc.Path == "" {
			problems = append(problems, fmt.Sprintf("docs[%d].path is required", i))
		} else if _, err := os.Stat(doc.Path); os.IsNotExist(err) {
			problems = append(problems, fmt.Sprintf("docs[%d].path file does not exist: %s", i, doc.Path))
		}
		if doc.FileType != "" && !validFileTypes[doc.FileType] {
			problems = append(problems, fmt.Sprintf("docs[%d].fileType must be one of: pdf, html, markdown, doc, txt, image, video, audio, other", i))
		}
	}

	validComponentIDs := map[string]bool{
		"component_api-contract":               true,
		"component_test-case-suite":            true,
		"component_support-kb-troubleshooting": true,
		"component_backend-flow-diagram":       true,
	}
	for i, m := range c.Maps {
		if m.Name == "" {
			problems = append(problems, fmt.Sprintf("maps[%d].name is required", i))
		}
		for j, frame := range m.Frames {
			if frame.Name == "" {
				problems = append(problems, fmt.Sprintf("maps[%d].frames[%d].name is required", i, j))
			}
			if frame.ImagePath != "" {
				if _, err := os.Stat(frame.ImagePath); os.IsNotExist(err) {
					problems = append(problems, fmt.Sprintf("maps[%d].frames[%d].imagePath file does not exist: %s", i, j, frame.ImagePath))
				}
			}
			for k, fp := range frame.FocalPoints {
				if fp.Name == "" {
					problems = append(problems, fmt.Sprintf("maps[%d].frames[%d].focalPoints[%d].name is required", i, j, k))
				}
				for l, comp := range fp.Components {
					if comp.ComponentID == "" {
						problems = append(problems, fmt.Sprintf("maps[%d].frames[%d].focalPoints[%d].components[%d].componentId is required", i, j, k, l))
						continue
					}
					if !validComponentIDs[comp.ComponentID] {
						problems = append(problems, fmt.Sprintf("maps[%d].frames[%d].focalPoints[%d].components[%d].componentId '%s' is not valid", i, j, k, l, comp.ComponentID))
						continue
					}
					if comp.ComponentLinkID == "" && comp.ServiceName == "" && len(comp.ModalFields) == 0 {
						problems = append(problems, fmt.Sprintf("maps[%d].frames[%d].focalPoints[%d].components[%d]: either componentLinkId, serviceName, or modalFields is required", i, j, k, l))
					}
					if comp.ComponentID == "component_backend-flow-diagram" && comp.ComponentLinkID == "" {
						if comp.ServiceName == "" || comp.ArchitectureDiagramName == "" {
							problems = append(problems, fmt.Sprintf("maps[%d].frames[%d].focalPoints[%d].components[%d]: component_backend-flow-diagram requires componentLinkId, or both serviceName and architectureDiagramName", i, j, k, l))
						}
					}
					if comp.ComponentID == "component_api-contract" && comp.ComponentLinkID == "" {
						if comp.ServiceName == "" || comp.APIGroupName == "" || comp.OperationID == "" {
							problems = append(problems, fmt.Sprintf("maps[%d].frames[%d].focalPoints[%d].components[%d]: component_api-contract requires componentLinkId, or serviceName, apiGroupName, and operationId", i, j, k, l))
						}
					}
				}
			}
		}
	}

	validDialects := map[string]bool{"postgres": true, "mysql": true, "sqlite": true, "dynamodb": true, "mongodb": true, "other": true}
	for i, db := range c.Databases {
		if db.Name == "" {
			problems = append(problems, fmt.Sprintf("databases[%d].name is required", i))
		}
		if db.Dialect == "" {
			problems = append(problems, fmt.Sprintf("databases[%d].dialect is required", i))
		} else if !validDialects[db.Dialect] {
			problems = append(problems, fmt.Sprintf("databases[%d].dialect must be one of: postgres, mysql, sqlite, dynamodb, mongodb, other", i))
		}
		if db.SchemaPath == "" {
			problems = append(problems, fmt.Sprintf("databases[%d].schemaPath is required", i))
		} else if _, err := os.Stat(db.SchemaPath); os.IsNotExist(err) {
			problems = append(problems, fmt.Sprintf("databases[%d].schemaPath file does not exist: %s", i, db.SchemaPath))
		}
	}

	dbNames := map[string]bool{}
	for _, db := range c.Databases {
		dbNames[db.Name] = true
	}
	for i, q := range c.Queries {
		if q.Name == "" {
			problems = append(problems, fmt.Sprintf("queries[%d].name is required", i))
		}
		if q.Database == "" {
			problems = append(problems, fmt.Sprintf("queries[%d].database is required", i))
		} else if !dbNames[q.Database] {
			problems = append(problems, fmt.Sprintf("queries[%d].database %q does not match any databases[].name", i, q.Database))
		}
		hasPath := q.Path != ""
		hasInline := q.QueryText != ""
		if hasPath == hasInline {
			problems = append(problems, fmt.Sprintf("queries[%d]: exactly one of path or queryText is required", i))
		} else if hasPath {
			if _, err := os.Stat(q.Path); os.IsNotExist(err) {
				problems = append(problems, fmt.Sprintf("queries[%d].path file does not exist: %s", i, q.Path))
			}
		}
	}

	for i, p := range c.ML {
		if p.Name == "" {
			problems = append(problems, fmt.Sprintf("ml[%d].name is required", i))
		}
		if p.Type != "model" && p.Type != "training" {
			problems = append(problems, fmt.Sprintf("ml[%d].type must be one of: model, training", i))
		}
		if p.Source.Type != "mlflow" {
			problems = append(problems, fmt.Sprintf("ml[%d].source.type must be: mlflow", i))
		}
		if p.Source.URL != "" && p.Source.URLEnv != "" {
			problems = append(problems, fmt.Sprintf("ml[%d].source: specify either url or urlEnv, not both", i))
		}
		if p.Source.URL == "" && p.Source.URLEnv == "" {
			problems = append(problems, fmt.Sprintf("ml[%d].source: either url or urlEnv is required", i))
		}
		if p.Source.URLEnv != "" && os.Getenv(p.Source.URLEnv) == "" {
			problems = append(problems, fmt.Sprintf("ml[%d].source.urlEnv: environment variable %s is not set or empty", i, p.Source.URLEnv))
		}
		if p.Source.TokenEnv != "" && os.Getenv(p.Source.TokenEnv) == "" {
			problems = append(problems, fmt.Sprintf("ml[%d].source.tokenEnv: environment variable %s is not set or empty", i, p.Source.TokenEnv))
		}
		if p.Type == "model" {
			if len(p.Models) == 0 {
				problems = append(problems, fmt.Sprintf("ml[%d]: a model project must declare models", i))
			}
			if len(p.Experiments) > 0 {
				problems = append(problems, fmt.Sprintf("ml[%d]: a model project must not declare experiments", i))
			}
			for j, m := range p.Models {
				if m.Name == "" {
					problems = append(problems, fmt.Sprintf("ml[%d].models[%d].name is required", i, j))
				}
				if m.ProblemType != "" && !validProblemType[m.ProblemType] {
					problems = append(problems, fmt.Sprintf("ml[%d].models[%d].problemType must be one of: classification, regression, ranking, generation, embedding, other", i, j))
				}
			}
		}
		if p.Type == "training" {
			if len(p.Experiments) == 0 {
				problems = append(problems, fmt.Sprintf("ml[%d]: a training project must declare experiments", i))
			}
			if len(p.Models) > 0 {
				problems = append(problems, fmt.Sprintf("ml[%d]: a training project must not declare models", i))
			}
			for j, e := range p.Experiments {
				if e.Name == "" {
					problems = append(problems, fmt.Sprintf("ml[%d].experiments[%d].name is required", i, j))
				}
			}
		}
	}

	if len(problems) > 0 {
		return problems
	}
	return nil
}
