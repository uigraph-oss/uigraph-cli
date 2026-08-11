package mlflow

import (
	"context"
	"fmt"
	"time"

	"github.com/uigraph-oss/uigraph-cli/pkg/config"
	"github.com/uigraph-oss/uigraph-cli/pkg/gateway"
)

type TrainingSink struct {
	Experiment func(gateway.MLExperimentItem) error
	Dataset    func(gateway.MLDatasetItem) error
	Run        func(gateway.MLRunItem) error
	Artifact   func(gateway.MLArtifactItem) error
}

type TrainingSummary struct {
	Experiments         int
	Datasets            int
	Runs                int
	Artifacts           int
	ExperimentMLflowIDs []string
	EvaluatedModelIDs   []string
}

type ModelSink struct {
	Model           func(gateway.MLModelItem) error
	Version         func(gateway.MLVersionItem) error
	Evaluation      func(gateway.MLEvaluationItem) error
	ProductionModel func(gateway.MLModelItem) error
}

type ModelSummary struct {
	Models            int
	Versions          int
	UnchangedVersions int
}

func SyncTraining(ctx context.Context, client *Client, project config.MLProjectRef, since time.Time, sink TrainingSink) (TrainingSummary, error) {
	summary := TrainingSummary{
		ExperimentMLflowIDs: []string{},
		EvaluatedModelIDs:   []string{},
	}
	datasetSeen := map[string]bool{}
	evaluatedSeen := map[string]bool{}

	for _, ref := range project.Experiments {
		exp, err := client.GetExperimentByName(ctx, ref.Name)
		if err != nil {
			return summary, fmt.Errorf("experiment %q: %w", ref.Name, err)
		}
		experimentItem := experimentToItem(exp, project.Name)
		if err := sink.Experiment(experimentItem); err != nil {
			return summary, fmt.Errorf("experiment %q: %w", ref.Name, err)
		}
		summary.Experiments++
		summary.ExperimentMLflowIDs = append(summary.ExperimentMLflowIDs, experimentItem.MLflowID)
		experimentEmail := tagValue(exp.Tags, userTagKey)

		err = client.SearchRuns(ctx, exp.ExperimentID, since, func(runs []Run) error {
			for _, run := range runs {
				for _, modelID := range evaluatedModelIDs(run) {
					if evaluatedSeen[modelID] {
						continue
					}
					evaluatedSeen[modelID] = true
					summary.EvaluatedModelIDs = append(summary.EvaluatedModelIDs, modelID)
				}

				if IsEvaluationRun(run) {
					continue
				}
				runEmail := tagValue(run.Data.Tags, userTagKey)
				if runEmail == "" {
					runEmail = experimentEmail
				}

				if run.Inputs != nil {
					for _, di := range run.Inputs.DatasetInputs {
						item := datasetToItem(di, run.Info.ExperimentID, runEmail)
						key := run.Info.ExperimentID + "\x00" + item.MLflowID
						if datasetSeen[key] {
							continue
						}
						datasetSeen[key] = true
						if err := sink.Dataset(item); err != nil {
							return err
						}
						summary.Datasets++
					}
				}

				if err := sink.Run(runToItem(run, experimentEmail)); err != nil {
					return err
				}
				summary.Runs++

				var modelIDs []string
				if run.Outputs != nil {
					for _, out := range run.Outputs.ModelOutputs {
						if out.ModelID != "" {
							modelIDs = append(modelIDs, out.ModelID)
						}
					}
				}
				if len(modelIDs) > 0 {
					for _, modelID := range modelIDs {
						artifacts, err := client.LoggedModelArtifacts(ctx, modelID)
						if err != nil {
							return fmt.Errorf("run %q logged model %q artifacts: %w", run.Info.RunID, modelID, err)
						}
						for _, f := range artifacts {
							if err := sink.Artifact(loggedModelArtifactToItem(client.baseURL, run.Info.RunID, modelID, f, runEmail)); err != nil {
								return err
							}
							summary.Artifacts++
						}
					}
				} else {
					artifacts, err := client.Artifacts(ctx, run.Info.RunID)
					if err != nil {
						return fmt.Errorf("run %q artifacts: %w", run.Info.RunID, err)
					}
					for _, f := range artifacts {
						if err := sink.Artifact(artifactToItem(client.baseURL, run.Info.RunID, f, runEmail)); err != nil {
							return err
						}
						summary.Artifacts++
					}
				}
			}
			return nil
		})
		if err != nil {
			return summary, fmt.Errorf("experiment %q runs: %w", ref.Name, err)
		}
	}

	return summary, nil
}

func versionChangedSince(v ModelVersion, since time.Time) bool {
	if since.IsZero() {
		return true
	}
	if v.LastUpdatedTimestamp == nil {
		return true
	}
	return *v.LastUpdatedTimestamp > since.UnixMilli()
}

func versionEvaluations(ctx context.Context, client *Client, version ModelVersion, versionMLflowID, versionEmail string, runCache map[string]*Run) ([]gateway.MLEvaluationItem, error) {
	modelID := loggedModelIDFromSource(version.Source)
	if modelID == "" {
		return nil, nil
	}

	logged, err := client.GetLoggedModel(ctx, modelID)
	if err != nil {
		return nil, err
	}

	byRun := map[string][]Metric{}
	var order []string
	for _, m := range logged.Data.Metrics {
		if m.RunID == "" || m.RunID == logged.Info.SourceRunID {
			continue
		}
		if _, seen := byRun[m.RunID]; !seen {
			order = append(order, m.RunID)
		}
		byRun[m.RunID] = append(byRun[m.RunID], m)
	}

	items := make([]gateway.MLEvaluationItem, 0, len(order))
	for _, runID := range order {
		run, cached := runCache[runID]
		if !cached {
			run, err = client.GetRun(ctx, runID)
			if err != nil {
				return nil, fmt.Errorf("evaluation run %q: %w", runID, err)
			}
			runCache[runID] = run
		}
		items = append(items, evaluationToItem(*run, versionMLflowID, byRun[runID], versionEmail))
	}
	return items, nil
}

func SyncModels(ctx context.Context, client *Client, project config.MLProjectRef, since time.Time, changedEvaluatedModelIDs map[string]bool, sink ModelSink) (ModelSummary, error) {
	summary := ModelSummary{}
	runCache := map[string]*Run{}

	for _, ref := range project.Models {
		model, err := client.GetRegisteredModel(ctx, ref.Name)
		if err != nil {
			return summary, fmt.Errorf("model %q: %w", ref.Name, err)
		}
		if ref.Description != "" {
			model.Description = ref.Description
		}
		modelEmail := tagValue(model.Tags, userTagKey)

		item := modelToItem(model, project.Name, nil)
		item.ProblemType = ref.ProblemType
		item.Domain = ref.Domain
		item.License = ref.License
		item.IntendedUse = ref.IntendedUse
		item.Limitations = ref.Limitations
		item.Recommendations = ref.Recommendations
		item.Considerations = ref.Considerations
		if err := sink.Model(item); err != nil {
			return summary, fmt.Errorf("model %q: %w", ref.Name, err)
		}
		summary.Models++

		var productionVersion *string
		err = client.ModelVersions(ctx, ref.Name, func(versions []ModelVersion) error {
			for _, v := range versions {
				versionMLflowID := fmt.Sprintf("%s/%s", v.Name, v.Version)
				if v.CurrentStage == "Production" {
					id := versionMLflowID
					productionVersion = &id
				}

				changed := versionChangedSince(v, since)
				if changed {
					if err := sink.Version(versionToItem(v, modelEmail)); err != nil {
						return err
					}
					summary.Versions++
				}
				if !changed {
					summary.UnchangedVersions++
				}

				if !changed && !changedEvaluatedModelIDs[loggedModelIDFromSource(v.Source)] {
					continue
				}

				versionEmail := tagValue(v.Tags, userTagKey)
				if versionEmail == "" {
					versionEmail = modelEmail
				}

				evaluations, err := versionEvaluations(ctx, client, v, versionMLflowID, versionEmail, runCache)
				if err != nil {
					return fmt.Errorf("version %q evaluations: %w", v.Version, err)
				}
				for _, e := range evaluations {
					if err := sink.Evaluation(e); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return summary, fmt.Errorf("model %q versions: %w", ref.Name, err)
		}

		production := item
		production.ProductionVersionMLflowID = productionVersion
		if err := sink.ProductionModel(production); err != nil {
			return summary, fmt.Errorf("model %q production version: %w", ref.Name, err)
		}
	}

	return summary, nil
}
