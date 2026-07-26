package mlflow

import (
	"context"
	"fmt"

	"github.com/uigraph-oss/uigraph-cli/pkg/config"
	"github.com/uigraph-oss/uigraph-cli/pkg/gateway"
)

type TrainingPayload struct {
	Experiments []gateway.MLExperimentItem
	Datasets    []gateway.MLDatasetItem
	Runs        []gateway.MLRunItem
	Artifacts   []gateway.MLArtifactItem
}

type ModelPayload struct {
	Models           []gateway.MLModelItem
	Versions         []gateway.MLVersionItem
	ModelsProduction []gateway.MLModelItem
	Evaluations      []gateway.MLEvaluationItem
}

func BuildTraining(ctx context.Context, client *Client, project config.MLProjectRef) (*TrainingPayload, error) {
	payload := &TrainingPayload{
		Experiments: []gateway.MLExperimentItem{},
		Datasets:    []gateway.MLDatasetItem{},
		Runs:        []gateway.MLRunItem{},
		Artifacts:   []gateway.MLArtifactItem{},
	}
	datasetSeen := map[string]bool{}

	for _, ref := range project.Experiments {
		exp, err := client.GetExperimentByName(ctx, ref.Name)
		if err != nil {
			return nil, fmt.Errorf("experiment %q: %w", ref.Name, err)
		}
		payload.Experiments = append(payload.Experiments, experimentToItem(exp, project.Name))

		runs, err := client.SearchRuns(ctx, exp.ExperimentID)
		if err != nil {
			return nil, fmt.Errorf("experiment %q runs: %w", ref.Name, err)
		}

		for _, run := range runs {
			if IsEvaluationRun(run) {
				continue
			}
			payload.Runs = append(payload.Runs, runToItem(run))

			if run.Inputs != nil {
				for _, di := range run.Inputs.DatasetInputs {
					item := datasetToItem(di.Dataset, run.Info.ExperimentID, inputContext(di))
					key := run.Info.ExperimentID + "\x00" + item.MLflowID
					if datasetSeen[key] {
						continue
					}
					datasetSeen[key] = true
					payload.Datasets = append(payload.Datasets, item)
				}
			}

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
						return nil, fmt.Errorf("run %q logged model %q artifacts: %w", run.Info.RunID, modelID, err)
					}
					for _, f := range artifacts {
						payload.Artifacts = append(payload.Artifacts, loggedModelArtifactToItem(client.baseURL, run.Info.RunID, modelID, f))
					}
				}
			} else {
				artifacts, err := client.Artifacts(ctx, run.Info.RunID)
				if err != nil {
					return nil, fmt.Errorf("run %q artifacts: %w", run.Info.RunID, err)
				}
				for _, f := range artifacts {
					payload.Artifacts = append(payload.Artifacts, artifactToItem(client.baseURL, run.Info.RunID, f))
				}
			}
		}
	}

	return payload, nil
}

func versionEvaluations(ctx context.Context, client *Client, version ModelVersion, versionMLflowID string, runCache map[string]*Run) ([]gateway.MLEvaluationItem, error) {
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
		items = append(items, evaluationToItem(*run, versionMLflowID, byRun[runID]))
	}
	return items, nil
}

func BuildModels(ctx context.Context, client *Client, project config.MLProjectRef) (*ModelPayload, error) {
	payload := &ModelPayload{
		Models:           []gateway.MLModelItem{},
		Versions:         []gateway.MLVersionItem{},
		ModelsProduction: []gateway.MLModelItem{},
		Evaluations:      []gateway.MLEvaluationItem{},
	}
	runCache := map[string]*Run{}

	for _, ref := range project.Models {
		model, err := client.GetRegisteredModel(ctx, ref.Name)
		if err != nil {
			return nil, fmt.Errorf("model %q: %w", ref.Name, err)
		}
		if ref.Description != "" {
			model.Description = ref.Description
		}

		versions, err := client.ModelVersions(ctx, ref.Name)
		if err != nil {
			return nil, fmt.Errorf("model %q versions: %w", ref.Name, err)
		}

		var productionVersion *string
		for _, v := range versions {
			payload.Versions = append(payload.Versions, versionToItem(v))
			versionMLflowID := fmt.Sprintf("%s/%s", v.Name, v.Version)
			if v.CurrentStage == "Production" {
				id := versionMLflowID
				productionVersion = &id
			}

			evaluations, err := versionEvaluations(ctx, client, v, versionMLflowID, runCache)
			if err != nil {
				return nil, fmt.Errorf("model %q version %q evaluations: %w", ref.Name, v.Version, err)
			}
			payload.Evaluations = append(payload.Evaluations, evaluations...)
		}

		item := modelToItem(model, project.Name, nil)
		item.ProblemType = ref.ProblemType
		item.Domain = ref.Domain
		item.License = ref.License
		item.IntendedUse = ref.IntendedUse
		item.Limitations = ref.Limitations
		item.Recommendations = ref.Recommendations
		item.Considerations = ref.Considerations
		payload.Models = append(payload.Models, item)

		production := item
		production.ProductionVersionMLflowID = productionVersion
		payload.ModelsProduction = append(payload.ModelsProduction, production)
	}

	return payload, nil
}
