package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"koditon/internal/db"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/taskqueue"
	syncjobs "koditon/internal/sync/jobs"
	"koditon/internal/sync/prices"
)

const (
	TaskTypePricesCitiesInit                 = "prices_cities_init"
	TaskTypePricesSync                       = "prices_sync"
	TaskTypePricesPostalCodeSync             = "prices_postal_code_sync"
	TaskTypePricesPostalCodePageSync         = "prices_postal_code_page_sync"
	TaskTypePricesNeighborhoodPostalCodeSync = "prices_neighborhood_postal_code_sync"
	TaskTypePricesSyncAll                    = "prices_sync_all"
)

type pricesPostalCodePayload struct {
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Page       int    `json:"page,omitempty"`
}

func (c *Consumer) handlePricesTask(ctx context.Context, msg taskqueue.Message) error {
	logger := logging.With(c.logger,
		slog.String("task_type", msg.Data.TaskType),
		slog.String("entity_id", msg.Data.EntityID),
	)
	return c.handleSyncJobTask(ctx, "prices", logger, msg, c.runPricesSyncJob)
}

func (c *Consumer) handlePricesCitiesInit(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.prices.cities_init"))
	logger.InfoContext(ctx, "prices cities initialization started")
	cities, err := c.syncRunner.PricesFetchCities(ctx)
	if err != nil {
		return err
	}
	if len(cities) > 0 {
		pricesQueue := taskqueue.NewQueue(c.pool, "prices")
		var enqueueErrors int
		for _, city := range cities {
			if enqErr := c.enqueuePricesTask(ctx, pricesQueue, taskqueue.EntityPrefixCity+city, TaskTypePricesSync); enqErr != nil {
				enqueueErrors++
			}
		}
		logger.InfoContext(ctx, "prices city entities enqueued", "count", len(cities), "enqueue_errors", enqueueErrors, "outcome", logging.OutcomeSuccess)
	}
	return nil
}

func (c *Consumer) handlePricesSync(ctx context.Context, logger *slog.Logger, msg taskqueue.Message) error {
	logger = logging.With(logger, logging.Op("consumer.prices.city_sync"))
	logger.InfoContext(ctx, "prices city sync started")
	if err := c.syncRunner.PricesSyncCityEntity(ctx, msg.Data.EntityID); err != nil {
		return err
	}
	logger.InfoContext(ctx, "prices city sync completed", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handlePricesNeighborhoodPostalCodeSync(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.prices.neighborhood_postal_code_sync"))
	logger.InfoContext(ctx, "prices neighborhood postal code sync started")
	err := c.syncRunner.PricesSyncNeighborhoodPostalCodes(ctx, func(p prices.SyncNeighborhoodPostalCodesProgress) {
		if p.Page > 0 {
			logger.DebugContext(ctx, "postal code transactions fetch started", "city", p.City, "postal_code", p.PostalCode, "page", p.Page)
		} else if p.Updated > 0 {
			logger.InfoContext(ctx, "neighborhood postal code mappings updated", "city", p.City, "postal_code", p.PostalCode, "updated", p.Updated)
		}
	})
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "prices neighborhood postal code sync completed", "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handlePricesSyncAll(ctx context.Context, logger *slog.Logger) error {
	logger = logging.With(logger, logging.Op("consumer.prices.sync_all"))
	logger.InfoContext(ctx, "prices sync all fanout started")
	cities, err := c.syncRunner.PricesFetchCities(ctx)
	if err != nil {
		return err
	}
	var enqueueErrors int
	for _, city := range cities {
		if err := c.enqueuePricesTask(ctx, nil, taskqueue.EntityPrefixCity+city, TaskTypePricesSync); err != nil {
			enqueueErrors++
		}
	}
	if enqueueErrors > 0 {
		return fmt.Errorf("enqueue prices sync all city jobs: %d errors", enqueueErrors)
	}
	logger.InfoContext(ctx, "prices sync all fanout completed", "cities", len(cities), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueuePricesTask(ctx context.Context, _ *taskqueue.Queue, entityID, taskType string) error {
	_, err := c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "prices",
		Kind:        taskType,
		EntityID:    entityID,
		Priority:    int32(taskqueue.PriorityNormal),
		MaxAttempts: 3,
	})
	return err
}

func (c *Consumer) runPricesSyncJob(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	switch job.SyncJobKind {
	case TaskTypePricesCitiesInit:
		return c.handlePricesCitiesInit(ctx, logger)
	case TaskTypePricesSync:
		return c.handleDurablePricesCitySync(ctx, logger, job)
	case TaskTypePricesPostalCodeSync:
		return c.handleDurablePricesPostalCodeSync(ctx, logger, job)
	case TaskTypePricesPostalCodePageSync:
		return c.handleDurablePricesPostalCodePageSync(ctx, logger, job)
	case TaskTypePricesNeighborhoodPostalCodeSync:
		return c.handlePricesNeighborhoodPostalCodeSync(ctx, logger)
	case TaskTypePricesSyncAll:
		return c.handlePricesSyncAll(ctx, logger)
	default:
		return taskqueue.NewPermanentError(fmt.Errorf("unknown prices sync job kind: %s", job.SyncJobKind), "unrecognized sync job kind")
	}
}

func (c *Consumer) handleDurablePricesCitySync(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.prices.city_index"))
	logger.InfoContext(ctx, "prices city index started")
	entityType, cityName, err := parseJobEntity(job.SyncJobEntityID)
	if err != nil {
		return taskqueue.NewPermanentError(err, "invalid prices city entity")
	}
	if entityType != "city" {
		return taskqueue.NewPermanentError(fmt.Errorf("expected city entity, got %s", entityType), "invalid prices city entity")
	}
	result, err := c.syncRunner.PricesSyncCityIndex(ctx, cityName, func(p prices.SyncCityProgress) {
		c.updatePricesCheckpoint(ctx, job, map[string]any{
			"city":       p.City,
			"step":       p.Step,
			"count":      p.Count,
			"details":    p.Details,
			"updated_at": time.Now().UTC(),
		})
	})
	if err != nil {
		return err
	}
	var enqueueErrors int
	for _, postalCode := range result.PostalCodes {
		payload, err := json.Marshal(pricesPostalCodePayload{City: result.City, PostalCode: postalCode, Page: 0})
		if err != nil {
			return fmt.Errorf("marshal prices postal code page payload: %w", err)
		}
		_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
			Provider:    "prices",
			Kind:        TaskTypePricesPostalCodePageSync,
			EntityID:    pricesPostalCodePageEntityID(result.City, postalCode, 0),
			Priority:    int32(taskqueue.PriorityNormal),
			MaxAttempts: 3,
			Payload:     payload,
		})
		if err != nil {
			enqueueErrors++
		}
	}
	if enqueueErrors > 0 {
		return fmt.Errorf("enqueue prices postal code jobs: %d errors", enqueueErrors)
	}
	logger.InfoContext(ctx, "prices city index completed", "city", result.City, "postal_codes", len(result.PostalCodes), "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) handleDurablePricesPostalCodeSync(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	var payload pricesPostalCodePayload
	if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
		return taskqueue.NewPermanentError(fmt.Errorf("decode prices postal code payload: %w", err), "invalid payload")
	}
	payload.Page = 0
	return c.enqueuePricesPostalCodePage(ctx, payload)
}

func (c *Consumer) handleDurablePricesPostalCodePageSync(ctx context.Context, logger *slog.Logger, job db.SyncJob) error {
	logger = logging.With(logger, logging.Op("consumer.prices.postal_code_sync"))
	var payload pricesPostalCodePayload
	if err := json.Unmarshal(job.SyncJobPayload, &payload); err != nil {
		return taskqueue.NewPermanentError(fmt.Errorf("decode prices postal code page payload: %w", err), "invalid payload")
	}
	if payload.City == "" || payload.PostalCode == "" {
		return taskqueue.NewPermanentError(fmt.Errorf("city and postal_code are required"), "invalid payload")
	}
	logger.InfoContext(ctx, "prices postal code page sync started", "city", payload.City, "postal_code", payload.PostalCode, "page", payload.Page)
	result, err := c.syncRunner.PricesSyncPostalCodeTransactionPage(ctx, payload.City, payload.PostalCode, payload.Page, func(p prices.SyncPostalCodeProgress) {
		c.updatePricesCheckpoint(ctx, job, map[string]any{
			"city":         p.City,
			"postal_code":  p.PostalCode,
			"page":         p.Page,
			"transactions": p.Transactions,
			"upserted":     p.Upserted,
			"updated_at":   time.Now().UTC(),
		})
	})
	if err != nil {
		return err
	}
	if result.NextPage != nil {
		payload.Page = *result.NextPage
		if err := c.enqueuePricesPostalCodePage(ctx, payload); err != nil {
			return err
		}
	}
	logger.InfoContext(ctx, "prices postal code page sync completed", "city", payload.City, "postal_code", payload.PostalCode, "page", result.Page, "next_page", result.NextPage, "outcome", logging.OutcomeSuccess)
	return nil
}

func (c *Consumer) enqueuePricesPostalCodePage(ctx context.Context, payload pricesPostalCodePayload) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal prices postal code page payload: %w", err)
	}
	_, err = c.syncJobs.Enqueue(ctx, syncjobs.EnqueueRequest{
		Provider:    "prices",
		Kind:        TaskTypePricesPostalCodePageSync,
		EntityID:    pricesPostalCodePageEntityID(payload.City, payload.PostalCode, payload.Page),
		Priority:    int32(taskqueue.PriorityNormal),
		MaxAttempts: 3,
		Payload:     raw,
	})
	return err
}

func pricesPostalCodePageEntityID(city, postalCode string, page int) string {
	return fmt.Sprintf("city:%s:postal_code:%s:page:%d", city, postalCode, page)
}

func parseJobEntity(entityID string) (string, string, error) {
	entityType, value, ok := strings.Cut(entityID, ":")
	if !ok || entityType == "" || value == "" {
		return "", "", fmt.Errorf("expected type:value entity id")
	}
	return entityType, value, nil
}

func (c *Consumer) updatePricesCheckpoint(ctx context.Context, job db.SyncJob, value map[string]any) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.syncJobs.UpdateCheckpoint(ctx, job, raw)
}
