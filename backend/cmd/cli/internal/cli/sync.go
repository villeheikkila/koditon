package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"koditon/internal/db"
	syncjobs "koditon/internal/sync/jobs"
)

type SyncFlags struct {
	Provider string
	Kind     string
	EntityID string
	Watch    bool
	Interval time.Duration
	JSON     bool
	Out      io.Writer
}

type SyncListFlags struct {
	Status   string
	Provider string
	Kind     string
	Limit    int
	JSON     bool
	Out      io.Writer
}

type SyncMaintenanceFlags struct {
	StaleAfter time.Duration
	Limit      int
	JSON       bool
	Out        io.Writer
}

type syncJobOutput struct {
	ID                uuid.UUID       `json:"id"`
	Provider          string          `json:"provider"`
	Kind              string          `json:"kind"`
	EntityID          string          `json:"entity_id"`
	Status            string          `json:"status"`
	AttemptCount      int32           `json:"attempt_count"`
	MaxAttempts       int32           `json:"max_attempts"`
	RunAfter          time.Time       `json:"run_after"`
	Checkpoint        json.RawMessage `json:"checkpoint,omitempty"`
	LastError         *string         `json:"last_error,omitempty"`
	LastErrorCode     *string         `json:"last_error_code,omitempty"`
	LastEnqueuedAt    *time.Time      `json:"last_enqueued_at,omitempty"`
	LastStartedAt     *time.Time      `json:"last_started_at,omitempty"`
	LastFinishedAt    *time.Time      `json:"last_finished_at,omitempty"`
	LastPgmqMessageID *int64          `json:"last_pgmq_message_id,omitempty"`
}

type syncAttemptOutput struct {
	ID         int64      `json:"id"`
	AttemptNo  int32      `json:"attempt_no"`
	Status     string     `json:"status"`
	Queue      string     `json:"queue"`
	MessageID  *int64     `json:"message_id,omitempty"`
	ErrorCode  *string    `json:"error_code,omitempty"`
	Error      *string    `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

func RunSync(ctx context.Context, store *syncjobs.Store, f SyncFlags) error {
	out := resolveOutput(f.Out)
	req, err := BuildSyncEnqueueRequest(f)
	if err != nil {
		return err
	}
	result, err := store.Enqueue(ctx, req)
	if err != nil {
		return err
	}
	if f.JSON {
		if err := writeJSON(out, map[string]any{"event": "queued", "enqueued": result.Enqueued, "job": mapSyncJob(result.Job)}); err != nil {
			return err
		}
	} else if result.Enqueued {
		fmt.Fprintf(out, "queued sync job %s %s %s id=%s\n", result.Job.SyncJobProvider, result.Job.SyncJobKind, result.Job.SyncJobEntityID, result.Job.SyncJobID)
	} else {
		fmt.Fprintf(out, "existing sync job %s %s %s id=%s status=%s\n", result.Job.SyncJobProvider, result.Job.SyncJobKind, result.Job.SyncJobEntityID, result.Job.SyncJobID, result.Job.SyncJobStatus)
	}
	if !f.Watch {
		return nil
	}
	return WatchSyncJob(ctx, store, result.Job.SyncJobID, f.Interval, f.JSON, out)
}

func RunSyncStatus(ctx context.Context, store *syncjobs.Store, jobID uuid.UUID, jsonOutput bool, out io.Writer) error {
	snapshot, err := store.GetSnapshot(ctx, jobID)
	if err != nil {
		return err
	}
	out = resolveOutput(out)
	if jsonOutput {
		return writeJSON(out, mapSyncSnapshot(snapshot))
	}
	fmt.Fprintln(out, FormatSyncJobSnapshot(snapshot))
	return nil
}

func RunSyncList(ctx context.Context, store *syncjobs.Store, f SyncListFlags) error {
	out := resolveOutput(f.Out)
	jobs, err := store.List(ctx, syncjobs.ListFilter{
		Status:   f.Status,
		Provider: f.Provider,
		Kind:     f.Kind,
		Limit:    int32(f.Limit),
	})
	if err != nil {
		return err
	}
	if f.JSON {
		items := make([]syncJobOutput, 0, len(jobs))
		for _, job := range jobs {
			items = append(items, mapSyncJob(job))
		}
		return writeJSON(out, map[string]any{"jobs": items})
	}
	for _, job := range jobs {
		fmt.Fprintf(out, "%s provider=%s kind=%s entity=%s status=%s attempts=%d/%d updated=%s\n", job.SyncJobID, job.SyncJobProvider, job.SyncJobKind, job.SyncJobEntityID, job.SyncJobStatus, job.SyncJobAttemptCount, job.SyncJobMaxAttempts, job.SyncJobUpdatedAt.Format(time.RFC3339))
	}
	return nil
}

func RunSyncMaintenance(ctx context.Context, store *syncjobs.Store, f SyncMaintenanceFlags) error {
	out := resolveOutput(f.Out)
	limit := int32(f.Limit)
	reaped, err := store.ReapStaleClaimsWithAttempts(ctx, f.StaleAfter, limit)
	if err != nil {
		return err
	}
	reconciled, err := store.ReconcilePendingJobs(ctx, limit)
	if err != nil {
		return err
	}
	if f.JSON {
		return writeJSON(out, map[string]any{
			"recovered_jobs":     len(reaped.RecoveredJobs),
			"finalized_attempts": reaped.FinalizedAttempts,
			"pending_scanned":    reconciled.Scanned,
			"pending_reenqueued": reconciled.Reenqueued,
		})
	}
	fmt.Fprintf(out, "recovered_jobs=%d finalized_attempts=%d pending_scanned=%d pending_reenqueued=%d\n", len(reaped.RecoveredJobs), reaped.FinalizedAttempts, reconciled.Scanned, reconciled.Reenqueued)
	return nil
}

func WatchSyncJob(ctx context.Context, store *syncjobs.Store, jobID uuid.UUID, interval time.Duration, jsonOutput bool, out io.Writer) error {
	out = resolveOutput(out)
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	lastLine := ""
	for {
		snapshot, err := store.GetSnapshot(ctx, jobID)
		if err != nil {
			return err
		}
		line := FormatSyncJobSnapshot(snapshot)
		if line != lastLine {
			if jsonOutput {
				if err := writeJSON(out, map[string]any{"event": "status", "snapshot": mapSyncSnapshot(snapshot)}); err != nil {
					return err
				}
			} else {
				fmt.Fprintln(out, line)
			}
			lastLine = line
		}
		if isFinalSyncJobStatus(snapshot.Job.SyncJobStatus) {
			if snapshot.Job.SyncJobStatus == syncjobs.StatusSucceeded || snapshot.Job.SyncJobStatus == syncjobs.StatusNoop {
				return nil
			}
			return fmt.Errorf("sync job finished with status %s: %s", snapshot.Job.SyncJobStatus, stringValue(snapshot.Job.SyncJobLastError))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func BuildSyncEnqueueRequest(f SyncFlags) (syncjobs.EnqueueRequest, error) {
	provider := strings.TrimSpace(f.Provider)
	kind := strings.TrimSpace(f.Kind)
	entityID := strings.TrimSpace(f.EntityID)
	if provider == "" || kind == "" || entityID == "" {
		return syncjobs.EnqueueRequest{}, errors.New("provider, kind, and entity id are required")
	}
	return syncjobs.EnqueueRequest{
		Provider:    provider,
		Kind:        kind,
		EntityID:    entityID,
		MaxAttempts: 3,
	}, nil
}

func FormatSyncJobSnapshot(snapshot syncjobs.JobSnapshot) string {
	job := snapshot.Job
	parts := []string{
		fmt.Sprintf("status=%s", job.SyncJobStatus),
		fmt.Sprintf("provider=%s", job.SyncJobProvider),
		fmt.Sprintf("kind=%s", job.SyncJobKind),
		fmt.Sprintf("entity=%s", job.SyncJobEntityID),
		fmt.Sprintf("attempts=%d/%d", job.SyncJobAttemptCount, job.SyncJobMaxAttempts),
	}
	if len(job.SyncJobCheckpoint) > 0 {
		parts = append(parts, "checkpoint="+compactJSON(job.SyncJobCheckpoint))
	}
	if job.SyncJobLastError != nil {
		parts = append(parts, "error="+*job.SyncJobLastError)
	}
	if len(snapshot.Attempts) > 0 {
		latest := snapshot.Attempts[0]
		parts = append(parts, fmt.Sprintf("latest_attempt=%s/%d", latest.SyncJobAttemptStatus, latest.SyncJobAttemptNo))
	}
	return strings.Join(parts, " ")
}

func mapSyncSnapshot(snapshot syncjobs.JobSnapshot) map[string]any {
	attempts := make([]syncAttemptOutput, 0, len(snapshot.Attempts))
	for _, attempt := range snapshot.Attempts {
		attempts = append(attempts, mapSyncAttempt(attempt))
	}
	return map[string]any{"job": mapSyncJob(snapshot.Job), "attempts": attempts}
}

func mapSyncJob(job db.SyncJob) syncJobOutput {
	return syncJobOutput{
		ID:                job.SyncJobID,
		Provider:          job.SyncJobProvider,
		Kind:              job.SyncJobKind,
		EntityID:          job.SyncJobEntityID,
		Status:            job.SyncJobStatus,
		AttemptCount:      job.SyncJobAttemptCount,
		MaxAttempts:       job.SyncJobMaxAttempts,
		RunAfter:          job.SyncJobRunAfter,
		Checkpoint:        job.SyncJobCheckpoint,
		LastError:         job.SyncJobLastError,
		LastErrorCode:     job.SyncJobLastErrorCode,
		LastEnqueuedAt:    job.SyncJobLastEnqueuedAt,
		LastStartedAt:     job.SyncJobLastStartedAt,
		LastFinishedAt:    job.SyncJobLastFinishedAt,
		LastPgmqMessageID: job.SyncJobLastPgmqMessageID,
	}
}

func mapSyncAttempt(attempt db.SyncJobAttempt) syncAttemptOutput {
	return syncAttemptOutput{
		ID:         attempt.SyncJobAttemptID,
		AttemptNo:  attempt.SyncJobAttemptNo,
		Status:     attempt.SyncJobAttemptStatus,
		Queue:      attempt.SyncJobAttemptQueueName,
		MessageID:  attempt.SyncJobAttemptMsgID,
		ErrorCode:  attempt.SyncJobAttemptErrorCode,
		Error:      attempt.SyncJobAttemptErrorDetail,
		CreatedAt:  attempt.SyncJobAttemptCreatedAt,
		FinishedAt: attempt.SyncJobAttemptFinishedAt,
	}
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return string(raw)
	}
	compact, err := json.Marshal(value)
	if err != nil {
		return string(raw)
	}
	return string(compact)
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(resolveOutput(out))
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func WriteJSON(out io.Writer, value any) error {
	return writeJSON(out, value)
}

func resolveOutput(out io.Writer) io.Writer {
	if out == nil {
		return io.Discard
	}
	return out
}

func isFinalSyncJobStatus(status string) bool {
	switch status {
	case syncjobs.StatusSucceeded, syncjobs.StatusFailed, syncjobs.StatusNotFound, syncjobs.StatusNoop, syncjobs.StatusSkippedLock:
		return true
	default:
		return false
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
