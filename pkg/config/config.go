package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version              int              `yaml:"version"`
	Project              Project          `yaml:"project"`
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
}

type MLProjectRef struct {
	Name        string            `yaml:"name"`
	Type        string            `yaml:"type"`
	Ownership   Ownership         `yaml:"ownership,omitempty"`
	Source      MLSourceRef       `yaml:"source"`
	Models      []MLModelRef      `yaml:"models,omitempty"`
	Experiments []MLExperimentRef `yaml:"experiments,omitempty"`
}

type MLSourceRef struct {
	Type  string `yaml:"type"`
	URL   string `yaml:"url"`
	Token string `yaml:"token,omitempty"`
}

type MLModelRef struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

type MLExperimentRef struct {
	Name string `yaml:"name"`
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

type Project struct {
	Name        string `yaml:"name" json:"name"`
	Environment string `yaml:"environment,omitempty" json:"environment,omitempty"`
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

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	for i := range cfg.TestPacks {
		if cfg.TestPacks[i].TestCasesPath == "" {
			continue
		}
		p := cfg.TestPacks[i].TestCasesPath
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("failed to read testCases file %q: %w", p, err)
		}
		var f testCasesFile
		if err := yaml.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("failed to parse testCases file %q: %w", p, err)
		}
		cfg.TestPacks[i].TestCases = append(cfg.TestPacks[i].TestCases, f.TestCases...)
	}

	for _, p := range cfg.QueryFiles {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("failed to read queries file %q: %w", p, err)
		}
		var f queriesFile
		if err := yaml.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("failed to parse queries file %q: %w", p, err)
		}
		cfg.Queries = append(cfg.Queries, f.Queries...)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version: %d (expected 1)", c.Version)
	}

	if c.Project.Name == "" {
		return fmt.Errorf("project.name is required")
	}

	if c.Service.Name != "" {
		if c.Service.Category == "" {
			return fmt.Errorf("service.category is required")
		}
		if c.Service.Description == "" {
			return fmt.Errorf("service.description is required")
		}

		if c.Service.Repository.Provider == "" {
			return fmt.Errorf("service.repository.provider is required")
		}
		validProviders := map[string]bool{"github": true, "gitlab": true, "bitbucket": true}
		if !validProviders[c.Service.Repository.Provider] {
			return fmt.Errorf("service.repository.provider must be one of: github, gitlab, bitbucket")
		}
		if c.Service.Repository.URL == "" {
			return fmt.Errorf("service.repository.url is required")
		}

		if c.Service.Ownership.Team == "" {
			return fmt.Errorf("service.ownership.team is required")
		}
	} else {
		if len(c.APIs) > 0 {
			return fmt.Errorf("service is required to sync apis; configs without a service may only sync maps and frames")
		}
		if len(c.Databases) > 0 {
			return fmt.Errorf("service is required to sync databases; configs without a service may only sync maps and frames")
		}
		if len(c.Queries) > 0 {
			return fmt.Errorf("service is required to sync queries; configs without a service may only sync maps and frames")
		}
		if len(c.ArchitectureDiagrams) > 0 {
			return fmt.Errorf("service is required to sync architectureDiagrams; configs without a service may only sync maps and frames")
		}
		if len(c.TestPacks) > 0 {
			return fmt.Errorf("service is required to sync testPacks; configs without a service may only sync maps and frames")
		}
		if len(c.Docs) > 0 {
			return fmt.Errorf("service is required to sync docs; configs without a service may only sync maps and frames")
		}
		if len(c.Dependencies) > 0 {
			return fmt.Errorf("service is required to sync dependencies; configs without a service may only sync maps and frames")
		}
	}

	for i, api := range c.APIs {
		if api.Name == "" {
			return fmt.Errorf("apis[%d].name is required", i)
		}
		if api.Type == "" {
			return fmt.Errorf("apis[%d].type is required", i)
		}
		validTypes := map[string]bool{"openapi": true, "graphql": true, "grpc": true}
		if !validTypes[api.Type] {
			return fmt.Errorf("apis[%d].type must be one of: openapi, graphql, grpc", i)
		}
		if api.Path == "" {
			return fmt.Errorf("apis[%d].path is required", i)
		}
		if _, err := os.Stat(api.Path); os.IsNotExist(err) {
			return fmt.Errorf("apis[%d].path file does not exist: %s", i, api.Path)
		}
	}

	validDependencyTypes := map[string]bool{"http": true, "graphql": true, "grpc": true, "database": true}
	validCriticalities := map[string]bool{"hard": true, "soft": true}
	dependencyNames := map[string]bool{}
	for i, dependency := range c.Dependencies {
		if dependency.Name == "" {
			return fmt.Errorf("dependencies[%d].name is required", i)
		}
		if dependency.Service == "" {
			return fmt.Errorf("dependencies[%d].service is required", i)
		}
		if dependency.Service == c.Service.Name {
			return fmt.Errorf("dependencies[%d].service must not reference the current service", i)
		}
		if dependencyNames[dependency.Name] {
			return fmt.Errorf("dependencies[%d].name must be unique", i)
		}
		dependencyNames[dependency.Name] = true
		if dependency.Type != "" && !validDependencyTypes[dependency.Type] {
			return fmt.Errorf("dependencies[%d].type must be one of: http, graphql, grpc, database", i)
		}
		if dependency.Criticality == "" {
			return fmt.Errorf("dependencies[%d].criticality is required", i)
		}
		if !validCriticalities[dependency.Criticality] {
			return fmt.Errorf("dependencies[%d].criticality must be one of: hard, soft", i)
		}
		endpointNames := map[string]bool{}
		for j, endpointName := range dependency.APIEndpointNames {
			if endpointName == "" {
				return fmt.Errorf("dependencies[%d].apiEndpointNames[%d] is required", i, j)
			}
			if endpointNames[endpointName] {
				return fmt.Errorf("dependencies[%d].apiEndpointNames[%d] must be unique", i, j)
			}
			endpointNames[endpointName] = true
		}
	}

	for i, ad := range c.ArchitectureDiagrams {
		if ad.Name == "" {
			return fmt.Errorf("architectureDiagrams[%d].name is required", i)
		}
		if ad.Path == "" {
			return fmt.Errorf("architectureDiagrams[%d].path is required", i)
		}
		if _, err := os.Stat(ad.Path); os.IsNotExist(err) {
			return fmt.Errorf("architectureDiagrams[%d].path file does not exist: %s", i, ad.Path)
		}
		if ad.ContextPath != "" {
			if _, err := os.Stat(ad.ContextPath); os.IsNotExist(err) {
				return fmt.Errorf("architectureDiagrams[%d].contextPath file does not exist: %s", i, ad.ContextPath)
			}
		}
	}

	validTestPackTypes := map[string]bool{"smoke": true, "regression": true, "manual": true}
	validTestCaseTypes := map[string]bool{"api": true, "manual": true}
	for i, pack := range c.TestPacks {
		if pack.Name == "" {
			return fmt.Errorf("testPacks[%d].name is required", i)
		}
		if pack.Type == "" {
			return fmt.Errorf("testPacks[%d].type is required", i)
		}
		if !validTestPackTypes[pack.Type] {
			return fmt.Errorf("testPacks[%d].type must be one of: smoke, regression, manual", i)
		}
		for j, tc := range pack.TestCases {
			if tc.Type == "" {
				return fmt.Errorf("testPacks[%d].testCases[%d].type is required", i, j)
			}
			if !validTestCaseTypes[tc.Type] {
				return fmt.Errorf("testPacks[%d].testCases[%d].type must be one of: api, manual", i, j)
			}
			if tc.Title == "" {
				return fmt.Errorf("testPacks[%d].testCases[%d].title is required", i, j)
			}
		}
	}

	validFileTypes := map[string]bool{"pdf": true, "html": true, "markdown": true, "doc": true, "txt": true, "image": true, "video": true, "audio": true, "other": true}
	for i, doc := range c.Docs {
		if doc.Name == "" {
			return fmt.Errorf("docs[%d].name is required", i)
		}
		if doc.Path == "" {
			return fmt.Errorf("docs[%d].path is required", i)
		}
		if _, err := os.Stat(doc.Path); os.IsNotExist(err) {
			return fmt.Errorf("docs[%d].path file does not exist: %s", i, doc.Path)
		}
		if doc.FileType != "" && !validFileTypes[doc.FileType] {
			return fmt.Errorf("docs[%d].fileType must be one of: pdf, html, markdown, doc, txt, image, video, audio, other", i)
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
			return fmt.Errorf("maps[%d].name is required", i)
		}
		for j, frame := range m.Frames {
			if frame.Name == "" {
				return fmt.Errorf("maps[%d].frames[%d].name is required", i, j)
			}
			if frame.ImagePath != "" {
				if _, err := os.Stat(frame.ImagePath); os.IsNotExist(err) {
					return fmt.Errorf("maps[%d].frames[%d].imagePath file does not exist: %s", i, j, frame.ImagePath)
				}
			}
			for k, fp := range frame.FocalPoints {
				if fp.Name == "" {
					return fmt.Errorf("maps[%d].frames[%d].focalPoints[%d].name is required", i, j, k)
				}
				for l, comp := range fp.Components {
					if comp.ComponentID == "" {
						return fmt.Errorf("maps[%d].frames[%d].focalPoints[%d].components[%d].componentId is required", i, j, k, l)
					}
					if !validComponentIDs[comp.ComponentID] {
						return fmt.Errorf("maps[%d].frames[%d].focalPoints[%d].components[%d].componentId '%s' is not valid", i, j, k, l, comp.ComponentID)
					}
					if comp.ComponentLinkID == "" && comp.ServiceName == "" && len(comp.ModalFields) == 0 {
						return fmt.Errorf("maps[%d].frames[%d].focalPoints[%d].components[%d]: either componentLinkId, serviceName, or modalFields is required", i, j, k, l)
					}
					if comp.ComponentID == "component_backend-flow-diagram" && comp.ComponentLinkID == "" {
						if comp.ServiceName == "" || comp.ArchitectureDiagramName == "" {
							return fmt.Errorf("maps[%d].frames[%d].focalPoints[%d].components[%d]: component_backend-flow-diagram requires componentLinkId, or both serviceName and architectureDiagramName", i, j, k, l)
						}
					}
					if comp.ComponentID == "component_api-contract" && comp.ComponentLinkID == "" {
						if comp.ServiceName == "" || comp.APIGroupName == "" || comp.OperationID == "" {
							return fmt.Errorf("maps[%d].frames[%d].focalPoints[%d].components[%d]: component_api-contract requires componentLinkId, or serviceName, apiGroupName, and operationId", i, j, k, l)
						}
					}
				}
			}
		}
	}

	validDialects := map[string]bool{"postgres": true, "mysql": true, "sqlite": true, "dynamodb": true, "mongodb": true, "other": true}
	for i, db := range c.Databases {
		if db.Name == "" {
			return fmt.Errorf("databases[%d].name is required", i)
		}
		if db.Dialect == "" {
			return fmt.Errorf("databases[%d].dialect is required", i)
		}
		if !validDialects[db.Dialect] {
			return fmt.Errorf("databases[%d].dialect must be one of: postgres, mysql, sqlite, dynamodb, mongodb, other", i)
		}
		if db.SchemaPath == "" {
			return fmt.Errorf("databases[%d].schemaPath is required", i)
		}
		if _, err := os.Stat(db.SchemaPath); os.IsNotExist(err) {
			return fmt.Errorf("databases[%d].schemaPath file does not exist: %s", i, db.SchemaPath)
		}
	}

	dbNames := map[string]bool{}
	for _, db := range c.Databases {
		dbNames[db.Name] = true
	}
	for i, q := range c.Queries {
		if q.Name == "" {
			return fmt.Errorf("queries[%d].name is required", i)
		}
		if q.Database == "" {
			return fmt.Errorf("queries[%d].database is required", i)
		}
		if !dbNames[q.Database] {
			return fmt.Errorf("queries[%d].database %q does not match any databases[].name", i, q.Database)
		}
		hasPath := q.Path != ""
		hasInline := q.QueryText != ""
		if hasPath == hasInline {
			return fmt.Errorf("queries[%d]: exactly one of path or queryText is required", i)
		}
		if hasPath {
			if _, err := os.Stat(q.Path); os.IsNotExist(err) {
				return fmt.Errorf("queries[%d].path file does not exist: %s", i, q.Path)
			}
		}
	}

	for i, p := range c.ML {
		if p.Name == "" {
			return fmt.Errorf("ml[%d].name is required", i)
		}
		if p.Type != "model" && p.Type != "training" {
			return fmt.Errorf("ml[%d].type must be one of: model, training", i)
		}
		if p.Source.Type != "mlflow" {
			return fmt.Errorf("ml[%d].source.type must be: mlflow", i)
		}
		if p.Source.URL == "" {
			return fmt.Errorf("ml[%d].source.url is required", i)
		}
		if p.Type == "model" {
			if len(p.Models) == 0 {
				return fmt.Errorf("ml[%d]: a model project must declare models", i)
			}
			if len(p.Experiments) > 0 {
				return fmt.Errorf("ml[%d]: a model project must not declare experiments", i)
			}
			for j, m := range p.Models {
				if m.Name == "" {
					return fmt.Errorf("ml[%d].models[%d].name is required", i, j)
				}
			}
		}
		if p.Type == "training" {
			if len(p.Experiments) == 0 {
				return fmt.Errorf("ml[%d]: a training project must declare experiments", i)
			}
			if len(p.Models) > 0 {
				return fmt.Errorf("ml[%d]: a training project must not declare models", i)
			}
			for j, e := range p.Experiments {
				if e.Name == "" {
					return fmt.Errorf("ml[%d].experiments[%d].name is required", i, j)
				}
			}
		}
	}

	return nil
}
