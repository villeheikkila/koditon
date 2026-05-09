package syncjobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/internal/db"
	"koditon/internal/platform/pgmq"
	"koditon/internal/platform/taskqueue"
)

const (
	StatusPending     = "pending"
	StatusInProgress  = "in_progress"
	StatusSucceeded   = "succeeded"
	StatusFailed      = "failed"
	StatusNotFound    = "not_found"
	StatusNoop        = "noop"
	StatusSkippedLock = "skipped_lock"

	CapacityClassDefault          = "default"
	CapacityClassFrontdoor        = "provider_frontdoor"
	CapacityClassShortcutScraper  = "provider_shortcut_scraper"
	CapacityClassShortcutAPI      = "provider_shortcut_api"
	CapacityClassPrices           = "provider_prices"
	CapacityClassPostal           = "provider_postal"
	CapacityClassInternalDB       = "internal_db"
	defaultDispatchRetryDelay     = 30 * time.Second
	defaultStaleClaimAfter        = 35 * time.Minute
	dispatchAdmissionLockClassID  = int32(174201)
	dispatchAdmissionLockObjectID = int32(1)
)

var ErrJobStateConflict = errors.New("sync job state conflict")

type Store struct {
	logger  *slog.Logger
	pool    *pgxpool.Pool
	queries *db.Queries
	policy  ExecutionPolicy
}

type ExecutionPolicy struct {
	GlobalMaxInProgress int
	ClassMaxInProgress  map[string]int
	KindMaxInProgress   map[string]int
	BaseDeferDelay      time.Duration
	MaxDeferDelay       time.Duration
}

type EnqueueRequest struct {
	Provider      string
	Kind          string
	EntityID      string
	Priority      int32
	MaxAttempts   int32
	RunAfter      time.Time
	CapacityClass string
	Payload       json.RawMessage
}

type EnqueueResult struct {
	Job       db.SyncJob
	Enqueued  bool
	MessageID *int64
}

type ClaimResult struct {
	Job       db.SyncJob
	AttemptNo int32
}

type ClaimDecision struct {
	Claim      *ClaimResult
	Delete     bool
	Retry      bool
	Status     string
	RetryAfter time.Duration
	ErrorCode  string
}

type RetryUpdate struct {
	RunAfter   time.Time
	Checkpoint json.RawMessage
	ErrorCode  *string
	Error      *string
	HTTPStatus *int32
}

type FinalUpdate struct {
	Status     string
	Result     json.RawMessage
	Checkpoint json.RawMessage
	ErrorCode  *string
	Error      *string
	HTTPStatus *int32
}

type JobSnapshot struct {
	Job      db.SyncJob
	Attempts []db.SyncJobAttempt
}

type ReconcileResult struct {
	Scanned    int
	Reenqueued int
}

type ReapResult struct {
	RecoveredJobs     []db.SyncJob
	FinalizedAttempts int64
}

type ListFilter struct {
	Status   string
	Provider string
	Kind     string
	Limit    int32
}

type JobDefinition struct {
	Provider        string
	Kind            string
	Queue           string
	CapacityClass   string
	MaxInProgress   int
	DefaultPriority int32
	DefaultAttempts int32
}

var jobDefinitions = []JobDefinition{
	{Provider: "frontdoor", Kind: "frontdoor_sitemap_sync", Queue: "frontdoor", CapacityClass: CapacityClassFrontdoor, MaxInProgress: 1},
	{Provider: "frontdoor", Kind: "frontdoor_buildings_sitemap_sync", Queue: "frontdoor", CapacityClass: CapacityClassFrontdoor, MaxInProgress: 1},
	{Provider: "frontdoor", Kind: "frontdoor_sync", Queue: "frontdoor", CapacityClass: CapacityClassFrontdoor, MaxInProgress: 2},
	{Provider: "frontdoor", Kind: "frontdoor_ad_data_hash_backfill", Queue: "frontdoor", CapacityClass: CapacityClassInternalDB, MaxInProgress: 1},
	{Provider: "shortcut", Kind: "shortcut_sitemap_sync", Queue: "shortcut", CapacityClass: CapacityClassShortcutAPI, MaxInProgress: 1},
	{Provider: "shortcut", Kind: "shortcut_buildings_sitemap_sync", Queue: "shortcut", CapacityClass: CapacityClassShortcutAPI, MaxInProgress: 1},
	{Provider: "shortcut", Kind: "shortcut_scraper_sync", Queue: "shortcut", CapacityClass: CapacityClassShortcutScraper, MaxInProgress: 1},
	{Provider: "shortcut", Kind: "shortcut_api_sync", Queue: "shortcut", CapacityClass: CapacityClassShortcutAPI, MaxInProgress: 2},
	{Provider: "shortcut", Kind: "shortcut_ad_data_hash_backfill", Queue: "shortcut", CapacityClass: CapacityClassInternalDB, MaxInProgress: 1},
	{Provider: "prices", Kind: "prices_cities_init", Queue: "prices", CapacityClass: CapacityClassPrices, MaxInProgress: 1},
	{Provider: "prices", Kind: "prices_sync", Queue: "prices", CapacityClass: CapacityClassPrices, MaxInProgress: 1},
	{Provider: "prices", Kind: "prices_postal_code_sync", Queue: "prices", CapacityClass: CapacityClassPrices, MaxInProgress: 1},
	{Provider: "prices", Kind: "prices_postal_code_page_sync", Queue: "prices", CapacityClass: CapacityClassPrices, MaxInProgress: 1},
	{Provider: "prices", Kind: "prices_neighborhood_postal_code_sync", Queue: "prices", CapacityClass: CapacityClassPrices, MaxInProgress: 1},
	{Provider: "prices", Kind: "prices_sync_all", Queue: "prices", CapacityClass: CapacityClassPrices, MaxInProgress: 1},
	{Provider: "prices", Kind: "prices_match_sale_listings_backfill", Queue: "prices", CapacityClass: CapacityClassInternalDB, MaxInProgress: 1},
	{Provider: "prices", Kind: "prices_match_sale_listings_fanout", Queue: "prices", CapacityClass: CapacityClassInternalDB, MaxInProgress: 1},
	{Provider: "prices", Kind: "prices_match_sale_listing", Queue: "prices", CapacityClass: CapacityClassInternalDB, MaxInProgress: 8},
	{Provider: "canonical", Kind: "canonicalize_source_ads_fanout", Queue: "canonical", CapacityClass: CapacityClassInternalDB, MaxInProgress: 1},
	{Provider: "canonical", Kind: "canonicalize_source_ad", Queue: "canonical", CapacityClass: CapacityClassInternalDB, MaxInProgress: 8},
	{Provider: "canonical", Kind: "canonical_match_sale_listing_sources_backfill", Queue: "canonical", CapacityClass: CapacityClassInternalDB, MaxInProgress: 1},
	{Provider: "canonical", Kind: "canonical_match_sale_listing_sources_fanout", Queue: "canonical", CapacityClass: CapacityClassInternalDB, MaxInProgress: 1},
	{Provider: "canonical", Kind: "canonical_match_sale_listing_source", Queue: "canonical", CapacityClass: CapacityClassInternalDB, MaxInProgress: 8},
	{Provider: "canonical", Kind: "canonical_rebuild_dimension_layer_backfill", Queue: "canonical", CapacityClass: CapacityClassInternalDB, MaxInProgress: 1},
	{Provider: "canonical", Kind: "canonical_rebuild_dimension_layer_listing", Queue: "canonical", CapacityClass: CapacityClassInternalDB, MaxInProgress: 2},
	{Provider: "canonical", Kind: "canonical_resolve_dirty_dimension_targets", Queue: "canonical", CapacityClass: CapacityClassInternalDB, MaxInProgress: 1},
	{Provider: "canonical", Kind: "canonical_resolve_dimension_target", Queue: "canonical", CapacityClass: CapacityClassInternalDB, MaxInProgress: 8},
	{Provider: "postal", Kind: "postal_sync", Queue: "postal", CapacityClass: CapacityClassPostal, MaxInProgress: 1},
}

func FindJobDefinition(provider, kind string) (JobDefinition, bool) {
	provider = strings.TrimSpace(provider)
	kind = strings.TrimSpace(kind)
	for _, def := range jobDefinitions {
		if def.Provider == provider && def.Kind == kind {
			return def, true
		}
	}
	return JobDefinition{}, false
}

func IsKnownJobKind(provider, kind string) bool {
	_, ok := FindJobDefinition(provider, kind)
	return ok
}

func NewStore(logger *slog.Logger, pool *pgxpool.Pool) *Store {
	if logger == nil {
		logger = slog.Default()
	}
	return &Store{logger: logger.With("component", "syncjobs"), pool: pool, queries: db.New(pool), policy: DefaultExecutionPolicy()}
}

func DefaultExecutionPolicy() ExecutionPolicy {
	kindMaxInProgress := make(map[string]int, len(jobDefinitions))
	for _, def := range jobDefinitions {
		kindMaxInProgress[def.Kind] = def.MaxInProgress
	}
	return ExecutionPolicy{
		GlobalMaxInProgress: 8,
		ClassMaxInProgress: map[string]int{
			CapacityClassDefault:         2,
			CapacityClassFrontdoor:       2,
			CapacityClassShortcutScraper: 1,
			CapacityClassShortcutAPI:     2,
			CapacityClassPrices:          1,
			CapacityClassPostal:          1,
			CapacityClassInternalDB:      8,
		},
		KindMaxInProgress: kindMaxInProgress,
		BaseDeferDelay:    5 * time.Second,
		MaxDeferDelay:     5 * time.Minute,
	}
}

func QueueNameForProvider(provider string) string {
	switch strings.TrimSpace(provider) {
	case "frontdoor":
		return "frontdoor"
	case "shortcut":
		return "shortcut"
	case "prices":
		return "prices"
	case "canonical":
		return "canonical"
	case "postal":
		return "postal"
	default:
		return ""
	}
}

func CapacityClassForJob(provider, kind string) string {
	if def, ok := FindJobDefinition(provider, kind); ok {
		return def.CapacityClass
	}
	switch strings.TrimSpace(provider) {
	case "frontdoor":
		if kind == "frontdoor_ad_data_hash_backfill" {
			return CapacityClassInternalDB
		}
		return CapacityClassFrontdoor
	case "shortcut":
		if kind == "shortcut_ad_data_hash_backfill" {
			return CapacityClassInternalDB
		}
		if kind == "shortcut_scraper_sync" {
			return CapacityClassShortcutScraper
		}
		return CapacityClassShortcutAPI
	case "prices":
		if kind == "prices_match_sale_listings_backfill" || kind == "prices_match_sale_listings_fanout" || kind == "prices_match_sale_listing" {
			return CapacityClassInternalDB
		}
		return CapacityClassPrices
	case "canonical":
		return CapacityClassInternalDB
	case "postal":
		return CapacityClassPostal
	default:
		return CapacityClassDefault
	}
}

func DedupKey(provider, kind, entityID string) string {
	return strings.Join([]string{strings.TrimSpace(provider), strings.TrimSpace(kind), strings.TrimSpace(entityID)}, ":")
}

func (s *Store) Enqueue(ctx context.Context, req EnqueueRequest) (EnqueueResult, error) {
	queueName := QueueNameForProvider(req.Provider)
	if queueName == "" {
		return EnqueueResult{}, fmt.Errorf("unknown sync provider %q", req.Provider)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return EnqueueResult{}, fmt.Errorf("begin sync job enqueue tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.queries.WithTx(tx)
	job, err := qtx.UpsertSyncJobForEnqueue(ctx, db.UpsertSyncJobForEnqueueParams{
		SyncJobProvider:      strings.TrimSpace(req.Provider),
		SyncJobKind:          strings.TrimSpace(req.Kind),
		SyncJobEntityID:      strings.TrimSpace(req.EntityID),
		SyncJobDedupKey:      DedupKey(req.Provider, req.Kind, req.EntityID),
		SyncJobPriority:      req.Priority,
		SyncJobMaxAttempts:   max(req.MaxAttempts, 1),
		SyncJobRunAfter:      resolveRunAfter(req.RunAfter),
		SyncJobCapacityClass: resolveCapacityClass(req),
		SyncJobPayload:       resolvePayload(req.Payload),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, getErr := qtx.GetSyncJobByDedupKey(ctx, DedupKey(req.Provider, req.Kind, req.EntityID))
			if getErr != nil {
				return EnqueueResult{}, fmt.Errorf("get existing sync job after enqueue conflict: %w", getErr)
			}
			if err := tx.Commit(ctx); err != nil {
				return EnqueueResult{}, fmt.Errorf("commit sync job enqueue conflict tx: %w", err)
			}
			return EnqueueResult{Job: existing, Enqueued: false, MessageID: existing.SyncJobLastPgmqMessageID}, nil
		}
		return EnqueueResult{}, fmt.Errorf("upsert sync job: %w", err)
	}
	msgID, err := sendJobMessage(ctx, qtx, queueName, job, delaySecondsUntil(job.SyncJobRunAfter))
	if err != nil {
		return EnqueueResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EnqueueResult{}, fmt.Errorf("commit sync job enqueue tx: %w", err)
	}
	return EnqueueResult{Job: job, Enqueued: true, MessageID: &msgID}, nil
}

func (s *Store) ClaimForDispatch(ctx context.Context, jobID uuid.UUID) (ClaimDecision, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ClaimDecision{}, fmt.Errorf("begin sync job claim tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.queries.WithTx(tx)
	job, err := qtx.GetSyncJobByID(ctx, jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ClaimDecision{Delete: true, Status: StatusNoop, ErrorCode: "job_missing"}, tx.Commit(ctx)
		}
		return ClaimDecision{}, fmt.Errorf("get sync job: %w", err)
	}
	now := time.Now()
	if job.SyncJobStatus != StatusPending {
		decision := decisionForCurrentJob(job, now)
		return decision, tx.Commit(ctx)
	}
	if job.SyncJobRunAfter.After(now) {
		return ClaimDecision{Retry: true, Status: "retry", RetryAfter: atLeastDuration(time.Until(job.SyncJobRunAfter), time.Second), ErrorCode: "job_not_ready"}, tx.Commit(ctx)
	}
	if err := qtx.AcquireSyncDispatchAdmissionLock(ctx, db.AcquireSyncDispatchAdmissionLockParams{ClassID: dispatchAdmissionLockClassID, ObjectID: dispatchAdmissionLockObjectID}); err != nil {
		return ClaimDecision{}, fmt.Errorf("acquire sync dispatch lock: %w", err)
	}
	if blockReason := s.dispatchBlockReason(ctx, qtx, job); blockReason != "" {
		retryAfter := s.nextDeferDelay(job)
		rows, err := qtx.DeferPendingSyncJobForCapacity(ctx, db.DeferPendingSyncJobForCapacityParams{
			SyncJobRunAfter:      now.Add(retryAfter),
			SyncJobLastErrorCode: stringPtr(blockReason),
			SyncJobLastError:     stringPtr(blockReason),
			SyncJobID:            job.SyncJobID,
		})
		if err != nil {
			return ClaimDecision{}, fmt.Errorf("defer sync job for capacity: %w", err)
		}
		if rows == 0 {
			return ClaimDecision{}, ErrJobStateConflict
		}
		return ClaimDecision{Retry: true, Status: "retry", RetryAfter: retryAfter, ErrorCode: blockReason}, tx.Commit(ctx)
	}
	claimToken := uuid.New()
	claimed, err := qtx.ClaimSyncJob(ctx, db.ClaimSyncJobParams{SyncJobClaimToken: &claimToken, SyncJobID: jobID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ClaimDecision{Retry: true, Status: "retry", RetryAfter: defaultDispatchRetryDelay, ErrorCode: "job_pending_unclaimed"}, tx.Commit(ctx)
		}
		return ClaimDecision{}, fmt.Errorf("claim sync job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ClaimDecision{}, fmt.Errorf("commit sync job claim tx: %w", err)
	}
	return ClaimDecision{Claim: &ClaimResult{Job: claimed, AttemptNo: claimed.SyncJobAttemptCount}}, nil
}

func (s *Store) InsertAttemptRunning(ctx context.Context, job db.SyncJob, queueName string, msgID int64) (int64, error) {
	id, err := s.queries.InsertSyncJobAttemptRunning(ctx, db.InsertSyncJobAttemptRunningParams{
		SyncJobID:                     job.SyncJobID,
		SyncJobAttemptQueueName:       queueName,
		SyncJobAttemptMsgID:           &msgID,
		SyncJobAttemptNo:              job.SyncJobAttemptCount,
		SyncJobAttemptPayloadSnapshot: job.SyncJobPayload,
	})
	if err != nil {
		return 0, fmt.Errorf("insert sync job attempt: %w", err)
	}
	return id, nil
}

func (s *Store) FinalizeAttempt(ctx context.Context, attemptID int64, status string, errorCode, errorDetail *string) error {
	err := s.queries.FinalizeSyncJobAttempt(ctx, db.FinalizeSyncJobAttemptParams{
		SyncJobAttemptStatus:      status,
		SyncJobAttemptErrorCode:   errorCode,
		SyncJobAttemptErrorDetail: errorDetail,
		SyncJobAttemptID:          attemptID,
	})
	if err != nil {
		return fmt.Errorf("finalize sync job attempt: %w", err)
	}
	return nil
}

func (s *Store) TransitionToRetry(ctx context.Context, job db.SyncJob, update RetryUpdate) error {
	rows, err := s.queries.MarkSyncJobPendingRetry(ctx, db.MarkSyncJobPendingRetryParams{
		SyncJobRunAfter:           update.RunAfter,
		SyncJobCheckpoint:         update.Checkpoint,
		SyncJobLastErrorCode:      update.ErrorCode,
		SyncJobLastError:          update.Error,
		SyncJobLastHttpStatus:     update.HTTPStatus,
		SyncJobID:                 job.SyncJobID,
		ExpectedSyncJobClaimToken: job.SyncJobClaimToken,
	})
	if err != nil {
		return fmt.Errorf("transition sync job to retry: %w", err)
	}
	if rows == 0 {
		return ErrJobStateConflict
	}
	return nil
}

func (s *Store) UpdateCheckpoint(ctx context.Context, job db.SyncJob, checkpoint json.RawMessage) error {
	rows, err := s.queries.UpdateSyncJobCheckpoint(ctx, db.UpdateSyncJobCheckpointParams{
		SyncJobCheckpoint:         checkpoint,
		SyncJobID:                 job.SyncJobID,
		ExpectedSyncJobClaimToken: job.SyncJobClaimToken,
	})
	if err != nil {
		return fmt.Errorf("update sync job checkpoint: %w", err)
	}
	if rows == 0 {
		return ErrJobStateConflict
	}
	return nil
}

func (s *Store) TransitionToFinal(ctx context.Context, job db.SyncJob, update FinalUpdate) error {
	rows, err := s.queries.MarkSyncJobFinal(ctx, db.MarkSyncJobFinalParams{
		SyncJobStatus:             update.Status,
		SyncJobResult:             update.Result,
		SyncJobCheckpoint:         update.Checkpoint,
		SyncJobLastErrorCode:      update.ErrorCode,
		SyncJobLastError:          update.Error,
		SyncJobLastHttpStatus:     update.HTTPStatus,
		SyncJobID:                 job.SyncJobID,
		ExpectedSyncJobClaimToken: job.SyncJobClaimToken,
	})
	if err != nil {
		return fmt.Errorf("transition sync job to final: %w", err)
	}
	if rows == 0 {
		return ErrJobStateConflict
	}
	return nil
}

func (s *Store) ReapStaleClaims(ctx context.Context, staleAfter time.Duration, limit int32) ([]db.SyncJob, error) {
	result, err := s.ReapStaleClaimsWithAttempts(ctx, staleAfter, limit)
	if err != nil {
		return nil, err
	}
	return result.RecoveredJobs, nil
}

func (s *Store) ReapStaleClaimsWithAttempts(ctx context.Context, staleAfter time.Duration, limit int32) (ReapResult, error) {
	if staleAfter <= 0 {
		staleAfter = defaultStaleClaimAfter
	}
	if limit <= 0 {
		limit = 25
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ReapResult{}, fmt.Errorf("begin stale sync job reap tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.queries.WithTx(tx)
	staleBefore := time.Now().Add(-staleAfter)
	jobs, err := qtx.ReapStaleSyncJobs(ctx, db.ReapStaleSyncJobsParams{StaleBefore: &staleBefore, LimitCount: limit})
	if err != nil {
		return ReapResult{}, fmt.Errorf("reap stale sync jobs: %w", err)
	}
	if len(jobs) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return ReapResult{}, fmt.Errorf("commit empty stale sync job reap tx: %w", err)
		}
		return ReapResult{}, nil
	}
	ids := make([]uuid.UUID, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.SyncJobID)
	}
	rows, err := qtx.FinalizeRunningSyncJobAttemptsForJobs(ctx, ids)
	if err != nil {
		return ReapResult{}, fmt.Errorf("finalize stale sync job attempts: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ReapResult{}, fmt.Errorf("commit stale sync job reap tx: %w", err)
	}
	return ReapResult{RecoveredJobs: jobs, FinalizedAttempts: rows}, nil
}

func (s *Store) ReconcilePendingJobs(ctx context.Context, limit int32) (ReconcileResult, error) {
	if limit <= 0 {
		limit = 100
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ReconcileResult{}, fmt.Errorf("begin sync job reconcile tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.queries.WithTx(tx)
	now := time.Now()
	jobs, err := qtx.ListDuePendingSyncJobsForReconcile(ctx, db.ListDuePendingSyncJobsForReconcileParams{NowAt: now, LimitCount: limit})
	if err != nil {
		return ReconcileResult{}, fmt.Errorf("list due pending sync jobs: %w", err)
	}
	result := ReconcileResult{Scanned: len(jobs)}
	for _, job := range jobs {
		queueName := QueueNameForProvider(job.SyncJobProvider)
		if queueName == "" {
			continue
		}
		if _, err := sendJobMessage(ctx, qtx, queueName, job, 0); err != nil {
			return ReconcileResult{}, err
		}
		result.Reenqueued++
	}
	if err := tx.Commit(ctx); err != nil {
		return ReconcileResult{}, fmt.Errorf("commit sync job reconcile tx: %w", err)
	}
	return result, nil
}

func (s *Store) GetSnapshot(ctx context.Context, jobID uuid.UUID) (JobSnapshot, error) {
	job, err := s.queries.GetSyncJobByID(ctx, jobID)
	if err != nil {
		return JobSnapshot{}, fmt.Errorf("get sync job: %w", err)
	}
	attempts, err := s.queries.ListSyncJobAttempts(ctx, jobID)
	if err != nil {
		return JobSnapshot{}, fmt.Errorf("list sync job attempts: %w", err)
	}
	return JobSnapshot{Job: job, Attempts: attempts}, nil
}

func (s *Store) List(ctx context.Context, filter ListFilter) ([]db.SyncJob, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 500 {
		limit = 500
	}
	jobs, err := s.queries.ListSyncJobs(ctx, db.ListSyncJobsParams{
		SyncJobStatus:   optionalString(filter.Status),
		SyncJobProvider: optionalString(filter.Provider),
		SyncJobKind:     optionalString(filter.Kind),
		LimitCount:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list sync jobs: %w", err)
	}
	return jobs, nil
}

func sendJobMessage(ctx context.Context, qtx *db.Queries, queueName string, job db.SyncJob, delaySeconds int) (int64, error) {
	msg, err := json.Marshal(taskqueue.MessageData{SyncJobID: &job.SyncJobID, EntityID: job.SyncJobEntityID, TaskType: job.SyncJobKind})
	if err != nil {
		return 0, fmt.Errorf("marshal sync job message: %w", err)
	}
	if err := pgmq.ValidateQueueName(queueName); err != nil {
		return 0, fmt.Errorf("validate sync job queue name: %w", err)
	}
	msgID, err := qtx.Send(ctx, db.SendParams{QueueName: queueName, Message: json.RawMessage(msg), DelaySeconds: int32(delaySeconds)})
	if err != nil {
		return 0, fmt.Errorf("send sync job message: %w", err)
	}
	if err := qtx.UpdateSyncJobEnqueueMetadata(ctx, db.UpdateSyncJobEnqueueMetadataParams{SyncJobLastPgmqMessageID: &msgID, SyncJobID: job.SyncJobID}); err != nil {
		return 0, fmt.Errorf("update sync job enqueue metadata: %w", err)
	}
	return msgID, nil
}

func delaySecondsUntil(runAfter time.Time) int {
	delay := time.Until(runAfter)
	if delay <= 0 {
		return 0
	}
	return int((delay + time.Second - time.Nanosecond) / time.Second)
}

func resolveRunAfter(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now()
	}
	return value
}

func resolveCapacityClass(req EnqueueRequest) string {
	if strings.TrimSpace(req.CapacityClass) != "" {
		return strings.TrimSpace(req.CapacityClass)
	}
	return CapacityClassForJob(req.Provider, req.Kind)
}

func resolvePayload(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func decisionForCurrentJob(job db.SyncJob, now time.Time) ClaimDecision {
	switch job.SyncJobStatus {
	case StatusPending:
		if job.SyncJobRunAfter.After(now) {
			return ClaimDecision{Retry: true, Status: "retry", RetryAfter: atLeastDuration(time.Until(job.SyncJobRunAfter), time.Second), ErrorCode: "job_not_ready"}
		}
		return ClaimDecision{Retry: true, Status: "retry", RetryAfter: defaultDispatchRetryDelay, ErrorCode: "job_pending_unclaimed"}
	case StatusInProgress:
		return ClaimDecision{Retry: true, Status: "retry", RetryAfter: defaultDispatchRetryDelay, ErrorCode: "job_redelivered_in_progress"}
	default:
		return ClaimDecision{Delete: true, Status: StatusNoop, ErrorCode: "job_not_pending"}
	}
}

func (s *Store) dispatchBlockReason(ctx context.Context, qtx *db.Queries, job db.SyncJob) string {
	if s.policy.GlobalMaxInProgress > 0 {
		count, err := qtx.CountSyncJobsInProgress(ctx)
		if err == nil && count >= int64(s.policy.GlobalMaxInProgress) {
			return "global_concurrency_saturated"
		}
	}
	if limit := s.policy.ClassMaxInProgress[job.SyncJobCapacityClass]; limit > 0 {
		count, err := qtx.CountSyncJobsInProgressByCapacityClass(ctx, job.SyncJobCapacityClass)
		if err == nil && count >= int64(limit) {
			return "capacity_class_saturated"
		}
	}
	if limit := s.policy.KindMaxInProgress[job.SyncJobKind]; limit > 0 {
		count, err := qtx.CountSyncJobsInProgressByKind(ctx, job.SyncJobKind)
		if err == nil && count >= int64(limit) {
			return "job_kind_saturated"
		}
	}
	return ""
}

func (s *Store) nextDeferDelay(job db.SyncJob) time.Duration {
	base := s.policy.BaseDeferDelay
	if base <= 0 {
		base = 5 * time.Second
	}
	maxDelay := s.policy.MaxDeferDelay
	if maxDelay <= 0 {
		maxDelay = 5 * time.Minute
	}
	delay := base * time.Duration(max(job.SyncJobAttemptCount, 1))
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func atLeastDuration(value, floor time.Duration) time.Duration {
	if value < floor {
		return floor
	}
	return value
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
