package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	syncjobs "koditon/internal/sync/jobs"
)

type SyncFlags struct {
	Provider string
	Kind     string
	EntityID string
	Watch    bool
	Interval time.Duration
}

func RunSync(ctx context.Context, store *syncjobs.Store, f SyncFlags) error {
	req, err := BuildSyncEnqueueRequest(f)
	if err != nil {
		return err
	}
	result, err := store.Enqueue(ctx, req)
	if err != nil {
		return err
	}
	if result.Enqueued {
		fmt.Printf("queued sync job %s %s %s id=%s\n", result.Job.SyncJobProvider, result.Job.SyncJobKind, result.Job.SyncJobEntityID, result.Job.SyncJobID)
	} else {
		fmt.Printf("existing sync job %s %s %s id=%s status=%s\n", result.Job.SyncJobProvider, result.Job.SyncJobKind, result.Job.SyncJobEntityID, result.Job.SyncJobID, result.Job.SyncJobStatus)
	}
	if !f.Watch {
		return nil
	}
	interval := f.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	lastLine := ""
	for {
		snapshot, err := store.GetSnapshot(ctx, result.Job.SyncJobID)
		if err != nil {
			return err
		}
		line := FormatSyncJobSnapshot(snapshot)
		if line != lastLine {
			fmt.Println(line)
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
