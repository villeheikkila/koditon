package consumers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/earendil-works/absurd/sdks/go/absurd"
	"github.com/google/uuid"

	postalclient "koditon/internal/clients/postal"
	"koditon/internal/platform/logging"
	syncpostal "koditon/internal/sync/postal"
	"koditon/internal/sync/workflows"
)

func (c *Consumer) startPostalWorkflowWorker(ctx context.Context, cfg Config) error {
	if c.postalWorkflowClient == nil {
		return errors.New("postal absurd workflow client is not configured")
	}
	if c.postalService == nil {
		return errors.New("postal service is not configured")
	}
	task := absurd.Task[workflows.Params, workflows.Result](
		TaskTypePostalSync,
		c.handlePostalWorkflow,
		absurd.TaskOptions{QueueName: workflows.QueuePostal, DefaultMaxAttempts: 3},
	)
	if err := c.postalWorkflowClient.Register(task); err != nil {
		return fmt.Errorf("register postal workflow: %w", err)
	}
	logger := logging.With(c.logger, logging.Op("consumer.postal.workflow"))
	workerCtx, cancel := context.WithCancel(ctx)
	c.postalWorkflowCancel = cancel
	c.postalWorkflowDone = make(chan struct{})
	go func() {
		defer close(c.postalWorkflowDone)
		logger.InfoContext(workerCtx, "postal absurd worker starting", "worker_count", max(cfg.WorkerCount, 1), "queue", workflows.QueuePostal)
		err := c.postalWorkflowClient.RunWorker(workerCtx, absurd.WorkerOptions{
			WorkerID:     "postal",
			ClaimTimeout: 35 * time.Minute,
			Concurrency:  max(cfg.WorkerCount, 1),
			BatchSize:    max(cfg.WorkerCount, 1),
			OnError: func(err error) {
				if workerCtx.Err() == nil {
					logger.WarnContext(workerCtx, "postal absurd worker error", "error", err, "outcome", logging.OutcomeError)
				}
			},
		})
		if err != nil && !errors.Is(err, context.Canceled) && workerCtx.Err() == nil {
			logger.ErrorContext(context.Background(), "postal absurd worker stopped", "error", err, "outcome", logging.OutcomeError)
		}
	}()
	return nil
}

func (c *Consumer) handlePostalWorkflow(ctx context.Context, params workflows.Params) (workflows.Result, error) {
	if err := workflows.ValidateParams(params); err != nil {
		return workflows.Result{}, err
	}
	if params.Kind != TaskTypePostalSync {
		return workflows.Result{}, fmt.Errorf("unknown postal workflow kind: %s", params.Kind)
	}
	logger := logging.With(c.logger,
		logging.Op("consumer.postal.workflow"),
		slog.String("task_type", params.Kind),
		slog.String("entity_id", params.EntityID),
	)
	result, err := c.runPostalWorkflow(ctx, logger)
	if err != nil {
		return workflows.Result{}, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return workflows.Result{}, fmt.Errorf("marshal postal workflow result: %w", err)
	}
	return workflows.Result{Status: "succeeded", Result: raw}, nil
}

func (c *Consumer) runPostalWorkflow(ctx context.Context, logger *slog.Logger) (syncpostal.SyncResult, error) {
	records, err := absurd.Step(ctx, "fetch-postal-data", func(ctx context.Context) ([]*postalclient.PostalCodeRecord, error) {
		return c.postalService.FetchPostalData(ctx, logger)
	})
	if err != nil {
		return syncpostal.SyncResult{}, err
	}
	adAreaIDs, err := absurd.Step(ctx, "upsert-ad-areas", func(ctx context.Context) (map[string]uuid.UUID, error) {
		return c.postalService.UpsertAdAreas(ctx, records, logger)
	})
	if err != nil {
		return syncpostal.SyncResult{}, err
	}
	municipalityIDs, err := absurd.Step(ctx, "upsert-municipalities", func(ctx context.Context) (map[string]uuid.UUID, error) {
		return c.postalService.UpsertMunicipalities(ctx, records, logger)
	})
	if err != nil {
		return syncpostal.SyncResult{}, err
	}
	postalCodes, err := absurd.Step(ctx, "upsert-postal-codes", func(ctx context.Context) (syncpostal.PostalCodesUpsertResult, error) {
		return c.postalService.UpsertPostalCodes(ctx, records, adAreaIDs, municipalityIDs, logger)
	})
	if err != nil {
		return syncpostal.SyncResult{}, err
	}
	return syncpostal.SyncResult{
		TotalRecords:           len(records),
		AdAreasUpserted:        len(adAreaIDs),
		MunicipalitiesUpserted: len(municipalityIDs),
		PostalCodesUpserted:    postalCodes.Upserted,
		SkippedRecords:         postalCodes.Skipped,
	}, nil
}
