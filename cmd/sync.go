package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/uigraph-oss/uigraph-cli/pkg/config"
	"github.com/uigraph-oss/uigraph-cli/pkg/gateway"
	"github.com/uigraph-oss/uigraph-cli/pkg/git"
	"github.com/uigraph-oss/uigraph-cli/pkg/mlflow"
	"github.com/uigraph-oss/uigraph-cli/pkg/timeline"
)

var (
	configPath    string
	apiURL        string
	enterpriseEnv string
	dryRun        bool
	verbose       bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync service and APIs to UiGraph Gateway",
	Long: `Reads .uigraph.yaml, captures git metadata, and syncs service, API groups, architecture diagrams, test packs/test cases, and database schemas (when configured) to the gateway.
This command is designed to run in CI/CD environments and requires UIGRAPH_TOKEN environment variable.
A dry run reaches no gateway, so it needs neither UIGRAPH_TOKEN nor a gateway URL.`,
	Args: cobra.NoArgs,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().StringVar(&configPath, "config", ".uigraph.yaml", "Path to config file")
	syncCmd.Flags().StringVar(&apiURL, "api-url", "", "Gateway API URL (defaults to UIGRAPH_GATEWAY_URL env var)")
	syncCmd.Flags().StringVar(&enterpriseEnv, "enterprise", "", "Sync to the UiGraph gateway ("+enterpriseGatewayURL+")")
	syncCmd.Flags().Lookup("enterprise").NoOptDefVal = "DEFAULT"
	syncCmd.MarkFlagsMutuallyExclusive("api-url", "enterprise")
	syncCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print payloads without sending to gateway (no token or gateway URL needed)")
	syncCmd.Flags().BoolVar(&verbose, "verbose", false, "Log every MLflow and gateway request with its duration")
}

func exitGatewayError(action string) {
	fmt.Fprintf(os.Stderr, "Error: failed to %s. Please try again.\n", action)
	os.Exit(2)
}

func exitGatewayErrorErr(action string, err error) {
	fmt.Fprintf(os.Stderr, "Error: failed to %s: %v\n", action, err)
	os.Exit(2)
}

// pluralize returns singular or plural form based on count
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// mlNoColumn marks a column that has no meaning for an entity
const mlNoColumn = "-"

// printMLTable prints an aligned table under the given header
func printMLTable(indent string, header []string, rows [][]string) {
	widths := make([]int, len(header))
	for i, cell := range header {
		widths[i] = len(cell)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	for _, row := range append([][]string{header}, rows...) {
		line := indent + fmt.Sprintf("%-*s", widths[0], row[0])
		for i := 1; i < len(row); i++ {
			line += fmt.Sprintf("  %*s", widths[i], row[i])
		}
		fmt.Println(line)
	}
}

// formatDuration formats duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func runSync(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := cmd.Context()

	// 1. Authenticate using UIGRAPH_TOKEN
	token := os.Getenv("UIGRAPH_TOKEN")
	if token == "" && !dryRun {
		fmt.Fprintln(os.Stderr, "Error: UIGRAPH_TOKEN environment variable is required")
		os.Exit(1)
	}

	if enterpriseEnv == "DEFAULT" {
		apiURL = enterpriseGatewayURL
	}
	if enterpriseEnv == "DEV" {
		apiURL = enterpriseDevGatewayURL
	}
	if enterpriseEnv != "" && apiURL == "" {
		fmt.Fprintf(os.Stderr, "Error: --enterprise accepts no value or =DEV, got %q\n", enterpriseEnv)
		os.Exit(1)
	}
	if apiURL == "" {
		apiURL = os.Getenv("UIGRAPH_GATEWAY_URL")
	}
	if apiURL == "" && !dryRun {
		fmt.Fprintln(os.Stderr, "Error: gateway URL is required (set --enterprise, --api-url or UIGRAPH_GATEWAY_URL environment variable)")
		os.Exit(1)
	}

	// 2. Load and validate config
	fmt.Printf("📦 Loading config from: %s\n", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
		os.Exit(1)
	}

	// 3. Capture git metadata
	fmt.Println("🔍 Capturing git metadata...")
	gitMeta := git.CaptureMetadata()
	if gitMeta.CommitHash == "" {
		fmt.Println("  ⚠️  Warning: Git metadata unavailable, continuing without it")
	} else {
		fmt.Printf("  • Commit: %s\n", gitMeta.CommitHash)
		fmt.Printf("  • Branch: %s\n", gitMeta.Branch)
		if gitMeta.IsDirty {
			fmt.Println("  ⚠️  Status: dirty (uncommitted changes)")
		}
	}

	// 4. Initialize gateway client
	client := gateway.NewClient(apiURL, token)
	client.Verbose = verbose

	// 5. Sync service
	if cfg.Service.Name != "" {
		fmt.Printf("\n🚀 Syncing service: %s\n", cfg.Service.Name)

		syncReq := gateway.ServiceSyncRequest{
			Service: cfg.Service,
			Git:     gitMeta,
			Source: gateway.Source{
				Type: "ci",
				Tool: "uigraph-cli",
			},
		}

		if dryRun {
			fmt.Println("\n=== DRY RUN: Service Sync Request ===")
			syncReq.Print()
		} else {
			serviceResp, err := client.SyncService(ctx, syncReq)
			if err != nil {
				exitGatewayErrorErr("sync service", err)
			}
			// We intentionally ignore service ID; UX only cares about the name.
			_ = serviceResp
			fmt.Printf("✓ Service synced: %s\n", cfg.Service.Name)
		}
	} else {
		fmt.Println("\nℹ️  No service defined — syncing maps/frames only")
	}

	// 6. Sync service databases
	if len(cfg.Databases) > 0 {
		fmt.Printf("\n🗄️  Syncing %d database %s...\n", len(cfg.Databases), pluralize(len(cfg.Databases), "schema", "schemas"))
		gitMinimal := gateway.GitMetadataMinimal{CommitHash: gitMeta.CommitHash}
		serviceName := cfg.Service.Name

		for _, db := range cfg.Databases {
			fmt.Printf("  • %s (%s)\n", db.Name, db.Dialect)

			schemaBytes, err := os.ReadFile(db.SchemaPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Error reading schema file %s: %v\n", db.SchemaPath, err)
				os.Exit(1)
			}

			req := gateway.ServiceDatabaseSyncRequest{
				ServiceName:       serviceName,
				DBName:            db.Name,
				Dialect:           db.Dialect,
				DBType:            db.DBType,
				SchemaFileContent: string(schemaBytes),
				Git:               gitMinimal,
			}

			if dryRun {
				fmt.Printf("\n=== DRY RUN: Service Database (%s) ===\n", db.Name)
				fmt.Printf("  SchemaPath: %s, size: %d bytes\n", db.SchemaPath, len(schemaBytes))
			} else {
				resp, err := client.SyncServiceDatabase(ctx, req)
				if err != nil {
					exitGatewayError(fmt.Sprintf("sync database schema %q", db.Name))
				}
				versionNote := ""
				if resp.VersionCreated {
					versionNote = " (new version)"
				}
				fmt.Printf("    ✓ Database schema synced: %s%s\n", db.Name, versionNote)
			}
		}
	}

	// 6b. Sync saved queries
	if len(cfg.Queries) > 0 {
		fmt.Printf("\n📝 Syncing %d saved %s...\n", len(cfg.Queries), pluralize(len(cfg.Queries), "query", "queries"))
		serviceName := cfg.Service.Name

		for _, q := range cfg.Queries {
			fmt.Printf("  • %s (db: %s)\n", q.Name, q.Database)

			queryText := q.QueryText
			if q.Path != "" {
				b, err := os.ReadFile(q.Path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "    Error reading query file %s: %v\n", q.Path, err)
					os.Exit(1)
				}
				queryText = string(b)
			}

			req := gateway.SavedQuerySyncRequest{
				ServiceName: serviceName,
				DBName:      q.Database,
				SourceRef:   q.Name,
				Title:       q.Name,
				Description: q.Description,
				QueryText:   queryText,
				Tags:        q.Tags,
				Git:         gateway.GitMetadataMinimal{CommitHash: gitMeta.CommitHash},
			}

			if dryRun {
				fmt.Printf("\n=== DRY RUN: Saved Query (%s) ===\n", q.Name)
				data, _ := json.MarshalIndent(req, "", "  ")
				fmt.Println(string(data))
			} else {
				resp, err := client.SyncSavedQuery(ctx, req)
				if err != nil {
					exitGatewayError(fmt.Sprintf("sync saved query %q", q.Name))
				}
				note := ""
				if resp.Created {
					note = " (new)"
				}
				fmt.Printf("    ✓ Query synced: %s%s\n", q.Name, note)
			}
		}
	}

	// 7. Sync API groups
	if len(cfg.APIs) > 0 {
		fmt.Printf("\n📡 Syncing %d API %s...\n", len(cfg.APIs), pluralize(len(cfg.APIs), "group", "groups"))

		for _, api := range cfg.APIs {
			fmt.Printf("  • %s (%s)\n", api.Name, api.Type)

			// Read spec file
			specContent, err := os.ReadFile(api.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Error reading spec file %s: %v\n", api.Path, err)
				os.Exit(1)
			}

			apiReq := gateway.APIGroupSyncRequest{
				APIGroup: gateway.APIGroup{
					Name: api.Name,
					Type: api.Type,
				},
				Spec: gateway.Spec{
					Content: string(specContent),
					Path:    api.Path,
				},
				Git: gateway.GitMetadataMinimal{
					CommitHash: gitMeta.CommitHash,
				},
				// Let the gateway resolve serviceId from service name
				ServiceName: cfg.Service.Name,
			}

			if dryRun {
				fmt.Printf("\n=== DRY RUN: API Group Sync Request (%s) ===\n", api.Name)
				apiReq.Print()
			} else {
				apiResp, err := client.SyncAPIGroup(ctx, apiReq)
				if err != nil {
					exitGatewayError(fmt.Sprintf("sync API group %q", api.Name))
				}
				fmt.Printf("    ✓ API group synced\n")
				_ = apiResp
			}
		}
	}

	// 8. Sync service dependencies
	if len(cfg.Dependencies) > 0 {
		fmt.Printf("\n🔗 Syncing %d service %s...\n", len(cfg.Dependencies), pluralize(len(cfg.Dependencies), "dependency", "dependencies"))
		req := gateway.ServiceDependenciesSyncRequest{
			ServiceName:  cfg.Service.Name,
			Dependencies: cfg.Dependencies,
		}
		if dryRun {
			fmt.Println("\n=== DRY RUN: Service Dependencies ===")
			data, _ := json.MarshalIndent(req, "", "  ")
			fmt.Println(string(data))
		} else {
			_, err := client.SyncServiceDependencies(ctx, req)
			if err != nil {
				exitGatewayError(fmt.Sprintf("sync service dependencies: %v", err))
			}
			fmt.Printf("✓ Synced %d service %s\n", len(cfg.Dependencies), pluralize(len(cfg.Dependencies), "dependency", "dependencies"))
		}
	}

	// 8b. Sync cost category tags
	costTagsCreated := 0
	costTagsDeleted := 0
	costTagsUnchanged := 0
	if cfg.CostTags != nil {
		fmt.Printf("\n💰 Syncing %d cost %s...\n", len(cfg.CostTags), pluralize(len(cfg.CostTags), "tag", "tags"))
		req := gateway.CostTagsSyncRequest{
			ServiceName: cfg.Service.Name,
			Tags:        cfg.CostTags,
		}
		if dryRun {
			fmt.Println("\n=== DRY RUN: Cost Tags ===")
			fmt.Println("  (config owns the full set; rules not listed here are deleted)")
			for _, tag := range cfg.CostTags {
				fmt.Printf("  • %s=%s\n", tag.Key, tag.Value)
			}
		} else {
			resp, err := client.SyncCostTags(ctx, req)
			if err != nil {
				exitGatewayErrorErr("sync cost tags", err)
			}
			costTagsCreated = resp.Created
			costTagsDeleted = resp.Deleted
			costTagsUnchanged = resp.Unchanged
			fmt.Printf("✓ Cost tags synced: %d created, %d deleted, %d unchanged\n", resp.Created, resp.Deleted, resp.Unchanged)
		}
	}

	// 9. Sync architecture diagrams
	if len(cfg.ArchitectureDiagrams) > 0 {
		fmt.Printf("\n📊 Syncing %d architecture %s...\n", len(cfg.ArchitectureDiagrams), pluralize(len(cfg.ArchitectureDiagrams), "diagram", "diagrams"))

		for _, ad := range cfg.ArchitectureDiagrams {
			fmt.Printf("  • %s\n", ad.Name)

			mermaidContent, err := os.ReadFile(ad.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Error reading mermaid file %s: %v\n", ad.Path, err)
				os.Exit(1)
			}

			req := gateway.ArchitectureDiagramSyncRequest{
				ServiceName:    cfg.Service.Name,
				Name:           ad.Name,
				MermaidContent: string(mermaidContent),
				GitCommitHash:  gitMeta.CommitHash,
			}
			if ad.ContextPath != "" {
				contextContent, err := os.ReadFile(ad.ContextPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "    Error reading context file %s: %v\n", ad.ContextPath, err)
					os.Exit(1)
				}
				req.ContextContent = string(contextContent)
			}

			if dryRun {
				fmt.Printf("\n=== DRY RUN: Architecture Diagram (%s) ===\n", ad.Name)
				fmt.Printf("  Path: %s, ContextPath: %s, Mermaid size: %d bytes\n", ad.Path, ad.ContextPath, len(req.MermaidContent))
			} else {
				archResp, err := client.SyncArchitectureDiagram(ctx, req)
				if err != nil {
					exitGatewayErrorErr(fmt.Sprintf("sync architecture diagram %q", ad.Name), err)
				}
				versionNote := ""
				if archResp.VersionCreated {
					versionNote = " (new version)"
				}
				fmt.Printf("    ✓ Architecture diagram synced: %s%s\n", ad.Name, versionNote)
				_ = archResp
			}
		}
	}

	// 10. Sync test packs and test cases
	screenshotsUploaded := 0
	screenshotsSkipped := 0
	totalTestCases := 0
	for _, pack := range cfg.TestPacks {
		totalTestCases += len(pack.TestCases)
	}
	if len(cfg.TestPacks) > 0 {
		fmt.Printf("\n🧪 Syncing %d test %s (%d test %s)...\n",
			len(cfg.TestPacks), pluralize(len(cfg.TestPacks), "pack", "packs"),
			totalTestCases, pluralize(totalTestCases, "case", "cases"))

		gitMinimal := gateway.GitMetadataMinimal{CommitHash: gitMeta.CommitHash}
		serviceName := cfg.Service.Name

		for _, pack := range cfg.TestPacks {
			fmt.Printf("  • %s (%s)\n", pack.Name, pack.Type)

			// Build test pack payload with optional pointers
			packPayload := gateway.TestPackInfoPayload{
				Name: pack.Name,
				Type: pack.Type,
			}
			if pack.Environment != "" {
				packPayload.Environment = &pack.Environment
			}
			if pack.ReleaseLabel != "" {
				packPayload.ReleaseLabel = &pack.ReleaseLabel
			}

			packReq := gateway.TestPackSyncRequest{
				TestPack:    packPayload,
				Git:         gitMinimal,
				ServiceName: serviceName,
			}

			if dryRun {
				fmt.Printf("\n=== DRY RUN: Test Pack (%s) ===\n", pack.Name)
				fmt.Printf("  Name: %s, Type: %s, TestCases: %d\n", pack.Name, pack.Type, len(pack.TestCases))
				for _, tc := range pack.TestCases {
					fmt.Printf("    - %s (type: %s, order: %g)\n", tc.Title, tc.Type, tc.Order)
					if len(tc.Screenshots) > 0 {
						fmt.Printf("      %d %s\n", len(tc.Screenshots), pluralize(len(tc.Screenshots), "screenshot", "screenshots"))
					}
				}
			} else {
				packResp, err := client.SyncTestPack(ctx, packReq)
				if err != nil {
					exitGatewayError(fmt.Sprintf("sync test pack %q", pack.Name))
				}
				fmt.Printf("    ✓ Test pack synced: %s\n", pack.Name)

				// Sync test cases for this pack
				for _, tc := range pack.TestCases {
					tcPayload := gateway.TestCaseInfoPayload{
						Type:             tc.Type,
						Title:            tc.Title,
						Order:            tc.Order,
						RequiresEvidence: tc.RequiresEvidence,
						IsCritical:       tc.IsCritical,
					}
					if tc.Description != "" {
						tcPayload.Description = &tc.Description
					}
					if tc.Priority != "" {
						tcPayload.Priority = &tc.Priority
					}
					if len(tc.Tags) > 0 {
						tcPayload.Labels = tc.Tags
					}
					if tc.LinkedTicket != "" {
						tcPayload.LinkedTicket = &tc.LinkedTicket
					}
					if tc.EstimatedDurationMins > 0 {
						mins := tc.EstimatedDurationMins
						tcPayload.EstimatedDurationMins = &mins
					}
					if tc.TestOwner != "" {
						tcPayload.TestOwner = &tc.TestOwner
					}
					if tc.MapName != "" {
						tcPayload.MapName = &tc.MapName
					}
					if tc.FrameName != "" {
						tcPayload.FrameName = &tc.FrameName
					}
					if tc.FocalPointName != "" {
						tcPayload.FocalPointName = &tc.FocalPointName
					}
					if tc.APIGroupName != "" {
						tcPayload.APIGroupName = &tc.APIGroupName
					}
					if tc.OperationID != "" {
						tcPayload.OperationID = &tc.OperationID
					}
					if tc.ExpectedStatusCode != 0 {
						tcPayload.ExpectedStatusCode = &tc.ExpectedStatusCode
					}
					if tc.RequestTemplate != "" {
						tcPayload.RequestTemplate = &tc.RequestTemplate
					}
					if tc.ResponseTimeMs > 0 {
						ms := tc.ResponseTimeMs
						tcPayload.MaxResponseTimeMs = &ms
					}
					if tc.ResponseBody != "" {
						body := tc.ResponseBody
						tcPayload.ResponseBody = &body
					}
					if len(tc.Assertions) > 0 {
						tcPayload.Assertions = make([]gateway.AssertionPayload, len(tc.Assertions))
						for i, a := range tc.Assertions {
							tcPayload.Assertions[i] = gateway.AssertionPayload{
								Field: a.Field,
								Type:  a.Type,
								Value: a.Value,
							}
						}
					}
					if len(tc.StepsList) > 0 {
						tcPayload.StepsList = make([]gateway.TestCaseStepPayload, len(tc.StepsList))
						for i, s := range tc.StepsList {
							tcPayload.StepsList[i] = gateway.TestCaseStepPayload{Action: s.Action, ExpectedResult: s.ExpectedResult}
						}
					}
					if tc.ExpectedOutcome != "" {
						tcPayload.ExpectedOutcome = &tc.ExpectedOutcome
					}
					if tc.Preconditions != "" {
						tcPayload.Preconditions = &tc.Preconditions
					}
					if tc.TestData != "" {
						tcPayload.TestData = &tc.TestData
					}
					if tc.Postconditions != "" {
						tcPayload.Postconditions = &tc.Postconditions
					}

					// Screenshots resolve to asset IDs before the test case is
					// written, so one sync call carries them — no follow-up update.
					for _, shot := range tc.Screenshots {
						content, err := os.ReadFile(shot)
						if err != nil {
							fmt.Fprintf(os.Stderr, "    Error reading screenshot %s: %v\n", shot, err)
							os.Exit(1)
						}
						sum := sha256.Sum256(content)
						prepResp, err := client.PrepareTestCaseScreenshot(ctx, gateway.TestCaseScreenshotPrepareRequest{
							ServiceName:   serviceName,
							TestPackID:    packResp.TestPackID,
							TestCaseTitle: tc.Title,
							ContentHash:   hex.EncodeToString(sum[:]),
							FileName:      filepath.Base(shot),
						})
						if err != nil {
							exitGatewayErrorErr(fmt.Sprintf("prepare screenshot %q", shot), err)
						}

						switch prepResp.Action {
						case "skip":
							screenshotsSkipped++
						case "upload":
							if err := uploadToS3(ctx, prepResp.UploadURL, content, "image", shot); err != nil {
								fmt.Fprintf(os.Stderr, "    Error uploading screenshot %s: %v\n", shot, err)
								os.Exit(1)
							}
							screenshotsUploaded++
						default:
							fmt.Fprintf(os.Stderr, "    Error: unexpected screenshot prepare action %q for %s\n", prepResp.Action, shot)
							os.Exit(1)
						}
						tcPayload.ScreenshotURLs = append(tcPayload.ScreenshotURLs, prepResp.AssetID)
					}

					tcReq := gateway.TestCaseSyncRequest{
						TestCase:     tcPayload,
						Git:          gitMinimal,
						TestPackName: pack.Name,
						TestPackID:   packResp.TestPackID,
						ServiceName:  serviceName,
					}

					tcResp, err := client.SyncTestCase(ctx, tcReq)
					if err != nil {
						exitGatewayError(fmt.Sprintf("sync test case %q", tc.Title))
					}
					fmt.Printf("      ✓ Test case synced: %s\n", tc.Title)
					_ = tcResp
				}
			}
		}
		if dryRun {
			fmt.Printf("\n=== DRY RUN: would sync %d test pack(s), %d test case(s)\n", len(cfg.TestPacks), totalTestCases)
		}
	}

	// 11. Sync service docs
	if len(cfg.Docs) > 0 {
		fmt.Printf("\n📄 Syncing %d documentation %s...\n", len(cfg.Docs), pluralize(len(cfg.Docs), "file", "files"))
		serviceName := cfg.Service.Name

		for _, doc := range cfg.Docs {
			fmt.Printf("  • %s (%s)\n", doc.Name, doc.Path)

			fileContent, err := os.ReadFile(doc.Path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Error reading doc file %s: %v\n", doc.Path, err)
				os.Exit(1)
			}

			sum := sha256.Sum256(fileContent)
			contentHash := hex.EncodeToString(sum[:])
			fileSize := int64(len(fileContent))

			prepReq := gateway.ServiceDocPrepareRequest{
				ServiceName: serviceName,
				DocName:     doc.Name,
				ContentHash: contentHash,
				FileSize:    fileSize,
				FilePath:    doc.Path,
				FileType:    doc.FileType,
				Description: doc.Description,
			}

			if dryRun {
				fmt.Printf("\n=== DRY RUN: Service Doc (%s) ===\n", doc.Name)
				fmt.Printf("  Path: %s, size: %d bytes, hash: %s\n", doc.Path, fileSize, contentHash)
				continue
			}

			prepResp, err := client.PrepareServiceDocUpload(ctx, prepReq)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    ✗ Prepare failed: %v\n", err)
				exitGatewayError(fmt.Sprintf("prepare doc upload for %q", doc.Name))
			}

			if prepResp.Action == "skip" {
				fmt.Printf("    ✓ Unchanged (skipped)\n")
				continue
			}

			if prepResp.Action == "upload" {
				if prepResp.UploadURL == nil || prepResp.FileID == nil {
					fmt.Fprintf(os.Stderr, "    ✗ Invalid prepare response (missing uploadUrl or fileId)\n")
					exitGatewayError(fmt.Sprintf("prepare doc upload for %q", doc.Name))
				}

				if err := uploadToS3(ctx, *prepResp.UploadURL, fileContent, doc.FileType, doc.Path); err != nil {
					fmt.Fprintf(os.Stderr, "    ✗ Upload failed: %v\n", err)
					exitGatewayError(fmt.Sprintf("upload doc to S3 for %q", doc.Name))
				}

				completeReq := gateway.ServiceDocCompleteRequest{
					ServiceName: serviceName,
					DocName:     doc.Name,
					FileID:      *prepResp.FileID,
					ContentHash: contentHash,
					FileType:    doc.FileType,
					Description: doc.Description,
					CommitHash:  gitMeta.CommitHash,
				}

				_, err := client.CompleteServiceDocUpload(ctx, completeReq)
				if err != nil {
					fmt.Fprintf(os.Stderr, "    ✗ Finalize failed: %v\n", err)
					exitGatewayError(fmt.Sprintf("complete doc upload for %q", doc.Name))
				}

				fmt.Printf("    ✓ Synced\n")
			}
		}
	}

	// 11b. Sync timeline events scanned from the repo
	timelineEvents := []timeline.Event{}
	timelineCreated := 0
	timelineUpdated := 0
	if cfg.Timeline != nil {
		scanned, err := timeline.Scan(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Timeline scan error: %v\n", err)
			os.Exit(1)
		}
		timelineEvents = scanned

		fmt.Printf("\n🕓 Syncing %d timeline %s...\n", len(timelineEvents), pluralize(len(timelineEvents), "event", "events"))
		if dryRun {
			fmt.Println("\n=== DRY RUN: Timeline Events ===")
			for _, event := range timelineEvents {
				fmt.Printf("  • %s · %s · %s\n", event.Type, event.Title, event.EventDate.Format("2006-01-02"))
			}
		} else if len(timelineEvents) > 0 {
			resp, err := client.SyncTimeline(ctx, gateway.TimelineSyncRequest{
				ServiceName: cfg.Service.Name,
				CommitHash:  gitMeta.CommitHash,
				Events:      timelineEvents,
			})
			if err != nil {
				exitGatewayErrorErr("sync timeline events", err)
			}
			timelineCreated = resp.Created
			timelineUpdated = resp.Updated
			fmt.Printf("✓ Timeline synced: %d created, %d updated\n", resp.Created, resp.Updated)
		}
	}

	// 12. Sync maps (focal points and component links)
	totalFrames := 0
	totalFocalPoints := 0
	totalComponents := 0
	for _, m := range cfg.Maps {
		totalFrames += len(m.Frames)
		for _, frame := range m.Frames {
			totalFocalPoints += len(frame.FocalPoints)
			for _, fp := range frame.FocalPoints {
				totalComponents += len(fp.Components)
			}
		}
	}
	if len(cfg.Maps) > 0 {
		fmt.Printf("\n🗺️  Syncing %d %s (%d %s, %d focal %s, %d %s)...\n",
			len(cfg.Maps), pluralize(len(cfg.Maps), "map", "maps"),
			totalFrames, pluralize(totalFrames, "frame", "frames"),
			totalFocalPoints, pluralize(totalFocalPoints, "point", "points"),
			totalComponents, pluralize(totalComponents, "component", "components"))

		for _, m := range cfg.Maps {
			fmt.Printf("  • Map: %s\n", m.Name)

			if dryRun {
				fmt.Printf("    [DRY RUN] Would sync map: %s\n", m.Name)
				for _, frame := range m.Frames {
					imgNote := ""
					if frame.ImagePath != "" {
						imgNote = fmt.Sprintf(" (image: %s)", frame.ImagePath)
					}
					fmt.Printf("      Frame: %s%s\n", frame.Name, imgNote)
					for _, fp := range frame.FocalPoints {
						fmt.Printf("        Focal point: %s (x:%.0f y:%.0f)\n", fp.Name, fp.X, fp.Y)
						for _, comp := range fp.Components {
							fmt.Printf("          - %s (link:%s svc:%s)\n", comp.ComponentID, comp.ComponentLinkID, comp.ServiceName)
						}
					}
				}
				continue
			}

			// Sync map
			_, err := client.SyncMap(ctx, gateway.MapSyncRequest{
				MapName:     m.Name,
				Description: m.Description,
				CommitHash:  gitMeta.CommitHash,
			})
			if err != nil {
				exitGatewayError(fmt.Sprintf("sync map %q", m.Name))
			}
			fmt.Printf("    ✓ Map synced: %s\n", m.Name)

			for _, frame := range m.Frames {
				fmt.Printf("    Frame: %s\n", frame.Name)

				// Sync frame (with optional image SHA check)
				prepReq := gateway.FramePrepareRequest{
					MapName:     m.Name,
					FrameName:   frame.Name,
					Description: frame.Description,
					ImagePath:   frame.ImagePath,
					CommitHash:  gitMeta.CommitHash,
				}

				if frame.ImagePath != "" {
					imageBytes, err := os.ReadFile(frame.ImagePath)
					if err != nil {
						fmt.Fprintf(os.Stderr, "      Error reading image file %s: %v\n", frame.ImagePath, err)
						os.Exit(1)
					}
					sum := sha256.Sum256(imageBytes)
					prepReq.ContentHash = hex.EncodeToString(sum[:])
					prepReq.FileSize = int64(len(imageBytes))

					prepResp, err := client.PrepareFrameSync(ctx, prepReq)
					if err != nil {
						exitGatewayError(fmt.Sprintf("prepare frame %q", frame.Name))
					}

					switch prepResp.Action {
					case "skip":
						fmt.Printf("      ✓ Frame synced (image unchanged): %s\n", frame.Name)
					case "upload":
						if prepResp.UploadURL == nil || prepResp.FileID == nil {
							fmt.Fprintf(os.Stderr, "      ✗ Invalid prepare response for frame %s\n", frame.Name)
							os.Exit(1)
						}
						if err := uploadToS3(ctx, *prepResp.UploadURL, imageBytes, "image", frame.ImagePath); err != nil {
							fmt.Fprintf(os.Stderr, "      ✗ Image upload failed: %v\n", err)
							exitGatewayError(fmt.Sprintf("upload image for frame %q", frame.Name))
						}
						_, err := client.CompleteFrameSync(ctx, gateway.FrameCompleteRequest{
							MapName:     m.Name,
							FrameName:   frame.Name,
							FileID:      *prepResp.FileID,
							ContentHash: prepReq.ContentHash,
							Description: frame.Description,
							CommitHash:  gitMeta.CommitHash,
						})
						if err != nil {
							exitGatewayError(fmt.Sprintf("complete frame image sync for %q", frame.Name))
						}
						fmt.Printf("      ✓ Frame synced (image updated): %s\n", frame.Name)
					case "done":
						fmt.Printf("      ✓ Frame synced: %s\n", frame.Name)
					}
				} else {
					// No image — metadata-only sync
					prepResp, err := client.PrepareFrameSync(ctx, prepReq)
					if err != nil {
						exitGatewayError(fmt.Sprintf("sync frame %q", frame.Name))
					}
					_ = prepResp
					fmt.Printf("      ✓ Frame synced: %s\n", frame.Name)
				}

				// Sync focal points within this frame
				for _, fp := range frame.FocalPoints {
					visibility := fp.Visibility
					if visibility == "" {
						visibility = "public"
					}

					fpResp, err := client.SyncFocalPoint(ctx, gateway.FocalPointSyncRequest{
						MapName:        m.Name,
						FrameName:      frame.Name,
						FocalPointName: fp.Name,
						X:              fp.X,
						Y:              fp.Y,
						Visibility:     visibility,
						CommitHash:     gitMeta.CommitHash,
					})
					if err != nil {
						exitGatewayError(fmt.Sprintf("sync focal point %q in frame %q", fp.Name, frame.Name))
					}
					_ = fpResp
					fmt.Printf("        ✓ Focal point: %s\n", fp.Name)

					for _, comp := range fp.Components {
						var modalFields []gateway.ComponentFieldItem
						for _, f := range comp.ModalFields {
							modalFields = append(modalFields, gateway.ComponentFieldItem{
								ComponentFieldID: f.ComponentFieldID,
								Label:            f.Label,
								Type:             f.Type,
								Data:             f.Data,
							})
						}

						_, err := client.SyncFocalPointMeta(ctx, gateway.FocalPointMetaSyncRequest{
							MapName:                 m.Name,
							FrameName:               frame.Name,
							FocalPointName:          fp.Name,
							ComponentID:             comp.ComponentID,
							ComponentLinkID:         comp.ComponentLinkID,
							ComponentModalFields:    modalFields,
							ServiceName:             comp.ServiceName,
							APIGroupName:            comp.APIGroupName,
							OperationID:             comp.OperationID,
							TestPackName:            comp.TestPackName,
							DocName:                 comp.DocName,
							ArchitectureDiagramName: comp.ArchitectureDiagramName,
							CommitHash:              gitMeta.CommitHash,
						})
						if err != nil {
							exitGatewayError(fmt.Sprintf("sync component %q for focal point %q", comp.ComponentID, fp.Name))
						}
						fmt.Printf("          ✓ Component: %s\n", comp.ComponentID)
					}
				}
			}
		}
	}

	// 13. Sync ML projects (models & experiments from MLflow)
	totalMLProjects := mlflow.EntityCount{}
	totalMLExperiments := mlflow.EntityCount{}
	totalMLDatasets := mlflow.EntityCount{}
	totalMLRuns := mlflow.EntityCount{}
	totalMLArtifacts := mlflow.EntityCount{}
	totalMLModels := mlflow.EntityCount{}
	totalMLVersions := mlflow.EntityCount{}
	totalMLEvaluations := mlflow.EntityCount{}
	if len(cfg.ML) > 0 {
		fmt.Printf("\n🤖 Syncing %d ML %s...\n", len(cfg.ML), pluralize(len(cfg.ML), "project", "projects"))

		watermarks := map[string]time.Time{}
		deletedProjects := map[string]bool{}
		if !dryRun {
			states, err := client.ListMLProjects(ctx, true)
			if err != nil {
				exitGatewayErrorErr("read last ML sync time", err)
			}
			for _, state := range states {
				if state.DeletedAt != nil {
					deletedProjects[state.Name] = true
				}
				if state.SyncedAt == nil {
					continue
				}
				watermarks[state.Name] = *state.SyncedAt
			}
		}

		orderedML := make([]config.MLProjectRef, 0, len(cfg.ML))
		for _, project := range cfg.ML {
			if project.Type == "training" {
				orderedML = append(orderedML, project)
			}
		}
		for _, project := range cfg.ML {
			if project.Type != "training" {
				orderedML = append(orderedML, project)
			}
		}

		syncedExperiments := map[string]bool{}
		evaluatedModelIDs := map[string]bool{}

		for _, project := range orderedML {
			fmt.Printf("  • %s (%s)\n", project.Name, project.Type)

			if deletedProjects[project.Name] {
				if _, err := client.RestoreMLProject(ctx, project.Name); err != nil {
					exitGatewayErrorErr(fmt.Sprintf("restore ML project %q", project.Name), err)
				}
				fmt.Println("    ↺ restored")
			}

			since := watermarks[project.Name]

			sourceURL, err := project.Source.ResolveURL()
			if err != nil {
				exitGatewayErrorErr(fmt.Sprintf("resolve MLflow url for %q", project.Name), err)
			}
			mlflowToken, err := project.Source.ResolveToken()
			if err != nil {
				exitGatewayErrorErr(fmt.Sprintf("resolve MLflow token for %q", project.Name), err)
			}

			projectItem := gateway.MLProjectItem{
				Name:        project.Name,
				Type:        project.Type,
				Description: project.Description,
				SourceType:  project.Source.Type,
				SourceURL:   sourceURL,
				Team:        project.Ownership.Team,
			}

			mlflowClient := mlflow.NewClient(sourceURL, mlflowToken)
			mlflowClient.Verbose = verbose

			if project.Type == "training" {
				if dryRun {
					fmt.Printf("\n=== DRY RUN: ML Training Project (%s) ===\n", project.Name)
				}
				projectStart := time.Now()
				var projectTime, experimentTime, datasetTime, runTime, artifactTime time.Duration
				var projectReqs, experimentReqs, datasetReqs, runReqs, artifactReqs int

				totalMLProjects.Found++
				if !dryRun {
					start := time.Now()
					res, err := client.SyncMLProject(ctx, projectItem)
					projectTime += time.Since(start)
					projectReqs++
					if err != nil {
						exitGatewayErrorErr(fmt.Sprintf("sync ML project %q", project.Name), err)
					}
					totalMLProjects.Created += res.Created
					totalMLProjects.Updated += res.Updated
				}

				runProgress := 0
				sink := mlflow.TrainingSink{
					Experiment: func(item gateway.MLExperimentItem) (gateway.SyncResult, error) {
						if dryRun {
							data, _ := json.MarshalIndent(item, "", "  ")
							fmt.Println(string(data))
							return gateway.SyncResult{}, nil
						}
						start := time.Now()
						res, err := client.SyncMLExperiment(ctx, item)
						experimentTime += time.Since(start)
						experimentReqs++
						return res, err
					},
					Dataset: func(item gateway.MLDatasetItem) (gateway.SyncResult, error) {
						if dryRun {
							return gateway.SyncResult{}, nil
						}
						start := time.Now()
						res, err := client.SyncMLDataset(ctx, item)
						datasetTime += time.Since(start)
						datasetReqs++
						return res, err
					},
					Run: func(item gateway.MLRunItem) (gateway.SyncResult, error) {
						runProgress++
						if runProgress%1000 == 0 {
							fmt.Printf("      … %d runs\n", runProgress)
						}
						if dryRun {
							return gateway.SyncResult{}, nil
						}
						start := time.Now()
						res, err := client.SyncMLRun(ctx, item)
						runTime += time.Since(start)
						runReqs++
						return res, err
					},
					Artifact: func(item gateway.MLArtifactItem) (gateway.SyncResult, error) {
						if dryRun {
							return gateway.SyncResult{}, nil
						}
						start := time.Now()
						res, err := client.SyncMLArtifact(ctx, item)
						artifactTime += time.Since(start)
						artifactReqs++
						return res, err
					},
				}

				ts, err := mlflow.SyncTraining(ctx, mlflowClient, project, since, sink)
				if err != nil {
					exitGatewayErrorErr(fmt.Sprintf("sync ML project %q from MLflow", project.Name), err)
				}
				totalMLExperiments.Merge(ts.Experiments)
				totalMLDatasets.Merge(ts.Datasets)
				totalMLRuns.Merge(ts.Runs)
				totalMLArtifacts.Merge(ts.Artifacts)
				for _, id := range ts.ExperimentMLflowIDs {
					syncedExperiments[sourceURL+"\x00"+id] = true
				}
				for _, modelID := range ts.EvaluatedModelIDs {
					evaluatedModelIDs[sourceURL+"\x00"+modelID] = true
				}

				rows := [][]string{
					{"experiments", strconv.Itoa(ts.Experiments.Found), strconv.Itoa(ts.Experiments.Created), strconv.Itoa(ts.Experiments.Updated)},
					{"datasets", strconv.Itoa(ts.Datasets.Found), strconv.Itoa(ts.Datasets.Created), strconv.Itoa(ts.Datasets.Updated)},
					{"runs", strconv.Itoa(ts.Runs.Found), strconv.Itoa(ts.Runs.Created), strconv.Itoa(ts.Runs.Updated)},
					{"artifacts", strconv.Itoa(ts.Artifacts.Found), strconv.Itoa(ts.Artifacts.Created), strconv.Itoa(ts.Artifacts.Updated)},
				}
				if dryRun {
					for i := range rows {
						rows[i][2] = mlNoColumn
						rows[i][3] = mlNoColumn
					}
				}
				printMLTable("      ", []string{"entity", "found", "created", "updated"}, rows)

				totalTime := time.Since(projectStart)
				if verbose {
					printMLTable("      ", []string{"step", "time", "requests"}, [][]string{
						{"mlflow reads", formatDuration(mlflowClient.Elapsed), strconv.Itoa(mlflowClient.Requests)},
						{"project", formatDuration(projectTime), strconv.Itoa(projectReqs)},
						{"experiments", formatDuration(experimentTime), strconv.Itoa(experimentReqs)},
						{"datasets", formatDuration(datasetTime), strconv.Itoa(datasetReqs)},
						{"runs", formatDuration(runTime), strconv.Itoa(runReqs)},
						{"artifacts", formatDuration(artifactTime), strconv.Itoa(artifactReqs)},
						{"total", formatDuration(totalTime), mlNoColumn},
					})
				}
				if !dryRun {
					fmt.Println("    ✓ synced")
				}
			}

			if project.Type == "model" {
				sourceEvaluatedModelIDs := map[string]bool{}
				for key := range evaluatedModelIDs {
					if strings.HasPrefix(key, sourceURL+"\x00") {
						sourceEvaluatedModelIDs[strings.TrimPrefix(key, sourceURL+"\x00")] = true
					}
				}

				if dryRun {
					fmt.Printf("\n=== DRY RUN: ML Model Project (%s) ===\n", project.Name)
				}
				projectStart := time.Now()
				var projectTime, modelTime, versionTime, evaluationTime, productionTime time.Duration
				var projectReqs, modelReqs, versionReqs, evaluationReqs, productionReqs int

				totalMLProjects.Found++
				if !dryRun {
					start := time.Now()
					res, err := client.SyncMLProject(ctx, projectItem)
					projectTime += time.Since(start)
					projectReqs++
					if err != nil {
						exitGatewayErrorErr(fmt.Sprintf("sync ML project %q", project.Name), err)
					}
					totalMLProjects.Created += res.Created
					totalMLProjects.Updated += res.Updated
				}

				versionProgress := 0
				skippedEvaluations := 0
				sink := mlflow.ModelSink{
					Model: func(item gateway.MLModelItem) (gateway.SyncResult, error) {
						if dryRun {
							return gateway.SyncResult{}, nil
						}
						start := time.Now()
						res, err := client.SyncMLModel(ctx, item)
						modelTime += time.Since(start)
						modelReqs++
						return res, err
					},
					Version: func(item gateway.MLVersionItem) (gateway.SyncResult, error) {
						versionProgress++
						if versionProgress%1000 == 0 {
							fmt.Printf("      … %d versions\n", versionProgress)
						}
						if dryRun {
							return gateway.SyncResult{}, nil
						}
						start := time.Now()
						res, err := client.SyncMLVersion(ctx, item)
						versionTime += time.Since(start)
						versionReqs++
						return res, err
					},
					Evaluation: func(item gateway.MLEvaluationItem) (gateway.SyncResult, error) {
						if !syncedExperiments[sourceURL+"\x00"+item.ExperimentMLflowID] {
							skippedEvaluations++
							return gateway.SyncResult{}, nil
						}
						if dryRun {
							return gateway.SyncResult{}, nil
						}
						start := time.Now()
						res, err := client.SyncMLEvaluation(ctx, item)
						evaluationTime += time.Since(start)
						evaluationReqs++
						return res, err
					},
					ProductionModel: func(item gateway.MLModelItem) (gateway.SyncResult, error) {
						if dryRun {
							data, _ := json.MarshalIndent(item, "", "  ")
							fmt.Println(string(data))
							return gateway.SyncResult{}, nil
						}
						start := time.Now()
						res, err := client.SyncMLModel(ctx, item)
						productionTime += time.Since(start)
						productionReqs++
						return res, err
					},
				}

				ms, err := mlflow.SyncModels(ctx, mlflowClient, project, since, sourceEvaluatedModelIDs, sink)
				if err != nil {
					exitGatewayErrorErr(fmt.Sprintf("sync ML project %q from MLflow", project.Name), err)
				}
				totalMLModels.Merge(ms.Models)
				totalMLVersions.Merge(ms.Versions)
				totalMLEvaluations.Merge(ms.Evaluations)

				rows := [][]string{
					{"models", strconv.Itoa(ms.Models.Found), strconv.Itoa(ms.Models.Created), strconv.Itoa(ms.Models.Updated)},
					{"versions", strconv.Itoa(ms.Versions.Found), strconv.Itoa(ms.Versions.Created), strconv.Itoa(ms.Versions.Updated)},
					{"evaluations", strconv.Itoa(ms.Evaluations.Found), strconv.Itoa(ms.Evaluations.Created), strconv.Itoa(ms.Evaluations.Updated)},
				}
				if dryRun {
					for i := range rows {
						rows[i][2] = mlNoColumn
						rows[i][3] = mlNoColumn
					}
				}
				printMLTable("      ", []string{"entity", "found", "created", "updated"}, rows)

				totalTime := time.Since(projectStart)
				if verbose {
					printMLTable("      ", []string{"step", "time", "requests"}, [][]string{
						{"mlflow reads", formatDuration(mlflowClient.Elapsed), strconv.Itoa(mlflowClient.Requests)},
						{"project", formatDuration(projectTime), strconv.Itoa(projectReqs)},
						{"models", formatDuration(modelTime), strconv.Itoa(modelReqs)},
						{"versions", formatDuration(versionTime), strconv.Itoa(versionReqs)},
						{"evaluations", formatDuration(evaluationTime), strconv.Itoa(evaluationReqs)},
						{"production", formatDuration(productionTime), strconv.Itoa(productionReqs)},
						{"total", formatDuration(totalTime), mlNoColumn},
					})
				}
				if skippedEvaluations > 0 {
					fmt.Printf("      %d evaluations skipped\n", skippedEvaluations)
				}
				if !dryRun {
					fmt.Println("    ✓ synced")
				}
			}
		}
	}

	// 14. Print summary
	elapsed := time.Since(startTime)
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 Sync Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if cfg.Service.Name != "" {
		fmt.Printf("Service: %s\n", cfg.Service.Name)
	} else {
		fmt.Println("Service: (none)")
	}
	if gitMeta.CommitHash != "" {
		fmt.Printf("Commit: %s\n", gitMeta.CommitHash)
	}
	fmt.Printf("API Groups: %d\n", len(cfg.APIs))
	fmt.Printf("Architecture Diagrams: %d\n", len(cfg.ArchitectureDiagrams))
	fmt.Printf("Service Dependencies: %d\n", len(cfg.Dependencies))
	fmt.Printf("Test Packs: %d\n", len(cfg.TestPacks))
	fmt.Printf("Test Cases: %d\n", totalTestCases)
	fmt.Printf("Screenshots: %d uploaded, %d skipped\n", screenshotsUploaded, screenshotsSkipped)
	fmt.Printf("Databases: %d\n", len(cfg.Databases))
	fmt.Printf("Queries: %d\n", len(cfg.Queries))
	fmt.Printf("Docs: %d\n", len(cfg.Docs))
	fmt.Printf("Cost Tags: %d (%d created, %d deleted, %d unchanged)\n", len(cfg.CostTags), costTagsCreated, costTagsDeleted, costTagsUnchanged)
	fmt.Printf("Timeline Events: %d (%d created, %d updated)\n", len(timelineEvents), timelineCreated, timelineUpdated)
	fmt.Printf("Maps: %d (Frames: %d, Focal Points: %d, Components: %d)\n", len(cfg.Maps), totalFrames, totalFocalPoints, totalComponents)
	fmt.Printf("ML Projects: %d\n", len(cfg.ML))
	if len(cfg.ML) > 0 {
		rows := [][]string{
			{"projects", strconv.Itoa(totalMLProjects.Found), strconv.Itoa(totalMLProjects.Created), strconv.Itoa(totalMLProjects.Updated)},
			{"experiments", strconv.Itoa(totalMLExperiments.Found), strconv.Itoa(totalMLExperiments.Created), strconv.Itoa(totalMLExperiments.Updated)},
			{"datasets", strconv.Itoa(totalMLDatasets.Found), strconv.Itoa(totalMLDatasets.Created), strconv.Itoa(totalMLDatasets.Updated)},
			{"runs", strconv.Itoa(totalMLRuns.Found), strconv.Itoa(totalMLRuns.Created), strconv.Itoa(totalMLRuns.Updated)},
			{"artifacts", strconv.Itoa(totalMLArtifacts.Found), strconv.Itoa(totalMLArtifacts.Created), strconv.Itoa(totalMLArtifacts.Updated)},
			{"models", strconv.Itoa(totalMLModels.Found), strconv.Itoa(totalMLModels.Created), strconv.Itoa(totalMLModels.Updated)},
			{"versions", strconv.Itoa(totalMLVersions.Found), strconv.Itoa(totalMLVersions.Created), strconv.Itoa(totalMLVersions.Updated)},
			{"evaluations", strconv.Itoa(totalMLEvaluations.Found), strconv.Itoa(totalMLEvaluations.Created), strconv.Itoa(totalMLEvaluations.Updated)},
		}
		if dryRun {
			for i := range rows {
				rows[i][2] = mlNoColumn
				rows[i][3] = mlNoColumn
			}
		}
		printMLTable("  ", []string{"entity", "found", "created", "updated"}, rows)
	}
	fmt.Printf("Duration: %s\n", formatDuration(elapsed))
	if dryRun {
		fmt.Println("\n(Dry run - no data sent to gateway)")
	}

	return nil
}

// uploadToS3 uploads file content to S3 using a presigned URL
func uploadToS3(ctx context.Context, presignedURL string, content []byte, fileType, filePath string) error {
	contentType := resolveContentTypeForUpload(fileType, filePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to create S3 upload request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("S3 upload request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("S3 upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// resolveContentTypeForUpload determines the content type for S3 upload.
// It must match what uigraph-gateway sends when presigning (see v1SyncFramePrepare.resolveContentType
// and v1SyncServiceDocPrepare.resolveContentType); otherwise S3 returns 403 SignatureDoesNotMatch.
func resolveContentTypeForUpload(fileType, filePath string) string {
	if fileType != "" {
		switch fileType {
		case "pdf":
			return "application/pdf"
		case "html":
			return "text/html"
		case "markdown":
			return "text/markdown"
		case "doc":
			return "application/msword"
		case "txt":
			return "text/plain"
		case "image":
			return contentTypeForImagePath(filePath)
		case "video":
			return contentTypeForVideoPath(filePath)
		case "audio":
			return contentTypeForAudioPath(filePath)
		}
	}

	if filePath != "" {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
		switch ext {
		case "pdf":
			return "application/pdf"
		case "html", "htm":
			return "text/html"
		case "md", "markdown":
			return "text/markdown"
		case "doc", "docx":
			return "application/msword"
		case "txt":
			return "text/plain"
		case "png":
			return "image/png"
		case "jpg", "jpeg":
			return "image/jpeg"
		case "gif":
			return "image/gif"
		case "webp":
			return "image/webp"
		case "svg":
			return "image/svg+xml"
		case "mp4":
			return "video/mp4"
		case "mov":
			return "video/quicktime"
		case "webm":
			return "video/webm"
		case "mp3":
			return "audio/mpeg"
		case "wav":
			return "audio/wav"
		case "ogg":
			return "audio/ogg"
		case "m4a":
			return "audio/mp4"
		}
	}

	return "application/octet-stream"
}

// contentTypeForImagePath matches gateway endpoints/sync/v1SyncFramePrepare resolveContentType (default image/png).
func contentTypeForImagePath(filePath string) string {
	ext := ""
	if filePath != "" {
		ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	}
	switch ext {
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "svg":
		return "image/svg+xml"
	default:
		return "image/png"
	}
}

func contentTypeForVideoPath(filePath string) string {
	ext := ""
	if filePath != "" {
		ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	}
	switch ext {
	case "mp4":
		return "video/mp4"
	case "mov":
		return "video/quicktime"
	case "webm":
		return "video/webm"
	default:
		return "video/mp4"
	}
}

func contentTypeForAudioPath(filePath string) string {
	ext := ""
	if filePath != "" {
		ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	}
	switch ext {
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "ogg":
		return "audio/ogg"
	case "m4a":
		return "audio/mp4"
	default:
		return "audio/mpeg"
	}
}
