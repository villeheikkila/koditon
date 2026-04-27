package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"koditon/internal/platform/taskqueue"
	syncjobs "koditon/internal/sync/jobs"
)

func enqueueAndWatchSyncJobs(ctx context.Context, app *appContext, report reportFn, jobs []syncjobs.EnqueueRequest) (actionResult, error) {
	if app.syncJobs == nil {
		return actionResult{}, fmt.Errorf("sync job store unavailable")
	}
	if len(jobs) == 0 {
		return actionResult{Output: "no sync jobs enqueued"}, nil
	}
	enqueued := make([]syncjobs.EnqueueResult, 0, len(jobs))
	for _, job := range jobs {
		if job.Priority == 0 {
			job.Priority = int32(taskqueue.PriorityNormal)
		}
		if job.MaxAttempts == 0 {
			job.MaxAttempts = 3
		}
		result, err := app.syncJobs.Enqueue(ctx, job)
		if err != nil {
			return actionResult{}, err
		}
		enqueued = append(enqueued, result)
	}
	report(progressUpdate{Message: fmt.Sprintf("Queued %d sync job(s)", len(enqueued)), Current: 0, Total: len(enqueued)})
	return watchSyncJobs(ctx, app, report, enqueued)
}

func watchSyncJobs(ctx context.Context, app *appContext, report reportFn, enqueued []syncjobs.EnqueueResult) (actionResult, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	last := make(map[string]string, len(enqueued))
	for {
		done := 0
		var failed []string
		for _, result := range enqueued {
			snapshot, err := app.syncJobs.GetSnapshot(ctx, result.Job.SyncJobID)
			if err != nil {
				return actionResult{}, err
			}
			job := snapshot.Job
			statusLine := syncJobStatusLine(snapshot)
			key := job.SyncJobID.String()
			if last[key] != statusLine {
				report(progressUpdate{Message: statusLine, Current: done, Total: len(enqueued)})
				last[key] = statusLine
			}
			if isFinalSyncJobStatus(job.SyncJobStatus) {
				done++
				if job.SyncJobStatus != syncjobs.StatusSucceeded && job.SyncJobStatus != syncjobs.StatusNoop {
					failed = append(failed, fmt.Sprintf("%s %s/%s: %s", job.SyncJobStatus, job.SyncJobKind, job.SyncJobEntityID, stringValue(job.SyncJobLastError)))
				}
			}
		}
		if done == len(enqueued) {
			output := fmt.Sprintf("sync_jobs=%d completed=%d failed=%d", len(enqueued), done, len(failed))
			if len(failed) > 0 {
				return actionResult{Output: output}, errors.New(strings.Join(failed, "\n"))
			}
			return actionResult{Output: output}, nil
		}
		select {
		case <-ctx.Done():
			return actionResult{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func syncJobStatusLine(snapshot syncjobs.JobSnapshot) string {
	job := snapshot.Job
	parts := []string{fmt.Sprintf("%s %s %s", job.SyncJobStatus, job.SyncJobKind, job.SyncJobEntityID)}
	if len(job.SyncJobCheckpoint) > 0 {
		parts = append(parts, "checkpoint="+compactJSON(job.SyncJobCheckpoint))
	}
	if job.SyncJobLastError != nil {
		parts = append(parts, "error="+*job.SyncJobLastError)
	}
	if len(snapshot.Attempts) > 0 {
		parts = append(parts, fmt.Sprintf("attempt=%d/%d", snapshot.Attempts[0].SyncJobAttemptNo, job.SyncJobMaxAttempts))
	}
	return strings.Join(parts, " ")
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	compact, err := json.Marshal(v)
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
