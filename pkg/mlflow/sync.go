package mlflow

import (
	"context"
	"fmt"

	"github.com/uigraph-oss/uigraph-cli/pkg/config"
	"github.com/uigraph-oss/uigraph-cli/pkg/gateway"
)

type RunSeries struct {
	RunMLflowID string
	Points      []gateway.MLSeriesPoint
}

type TrainingPayload struct {
	Experiments []gateway.MLExperimentItem
	Datasets    []gateway.MLDatasetItem
	Runs        []gateway.MLRunItem
	Series      []RunSeries
	Artifacts   []gateway.MLArtifactItem
}

type ModelPayload struct {
	Models           []gateway.MLModelItem
	Versions         []gateway.MLVersionItem
	ModelsProduction []gateway.MLModelItem
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

			var points []gateway.MLSeriesPoint
			for _, metric := range run.Data.Metrics {
				history, err := client.MetricHistory(ctx, run.Info.RunID, metric.Key)
				if err != nil {
					return nil, fmt.Errorf("run %q metric %q history: %w", run.Info.RunID, metric.Key, err)
				}
				points = append(points, metricHistoryToPoints(history)...)
			}
			if len(points) > 0 {
				payload.Series = append(payload.Series, RunSeries{RunMLflowID: run.Info.RunID, Points: points})
			}

			artifacts, err := client.Artifacts(ctx, run.Info.RunID)
			if err != nil {
				return nil, fmt.Errorf("run %q artifacts: %w", run.Info.RunID, err)
			}
			for _, f := range artifacts {
				payload.Artifacts = append(payload.Artifacts, artifactToItem(run.Info.RunID, f))
			}
		}
	}

	return payload, nil
}

func BuildModels(ctx context.Context, client *Client, project config.MLProjectRef) (*ModelPayload, error) {
	payload := &ModelPayload{
		Models:           []gateway.MLModelItem{},
		Versions:         []gateway.MLVersionItem{},
		ModelsProduction: []gateway.MLModelItem{},
	}

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
			if v.CurrentStage == "Production" {
				id := fmt.Sprintf("%s/%s", v.Name, v.Version)
				productionVersion = &id
			}
		}

		item := modelToItem(model, project.Name, nil)
		item.ProblemType = ref.ProblemType
		item.Domain = ref.Domain
		item.License = ref.License
		item.Owners = ref.Owners
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
