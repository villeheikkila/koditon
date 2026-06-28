package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"koditon/internal/sync/workflows"
)

type enqueuedSyncWork struct {
	tasks []workflows.EnqueueResult
}

func enqueueAndWatchSyncJobs(ctx context.Context, app *appContext, report reportFn, jobs []workflows.SpawnRequest) (actionResult, error) {
	if app.workflowStore == nil {
		return actionResult{}, fmt.Errorf("sync workflow store unavailable")
	}
	if len(jobs) == 0 {
		return actionResult{Output: "no sync jobs enqueued"}, nil
	}
	enqueued := enqueuedSyncWork{tasks: make([]workflows.EnqueueResult, 0, len(jobs))}
	for _, job := range jobs {
		if job.MaxAttempts == 0 {
			job.MaxAttempts = 3
		}
		if !workflows.IsImplemented(job.Provider, job.Kind) {
			return actionResult{}, fmt.Errorf("sync kind %q is not implemented for provider %q", job.Kind, job.Provider)
		}
		result, err := app.workflowStore.Enqueue(ctx, job)
		if err != nil {
			return actionResult{}, err
		}
		enqueued.tasks = append(enqueued.tasks, result)
	}
	total := len(enqueued.tasks)
	report(progressUpdate{Message: fmt.Sprintf("Queued %d sync task(s)", total), Current: 0, Total: total})
	return watchSyncJobs(ctx, app, report, enqueued)
}

func watchSyncJobs(ctx context.Context, app *appContext, report reportFn, enqueued enqueuedSyncWork) (actionResult, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	total := len(enqueued.tasks)
	last := make(map[string]string, total)
	for {
		done := 0
		var failed []string
		for _, result := range enqueued.tasks {
			snapshot, err := app.workflowStore.GetSnapshot(ctx, result.TaskID)
			if err != nil {
				return actionResult{}, err
			}
			statusLine := syncTaskStatusLine(snapshot)
			key := result.TaskID
			if last[key] != statusLine {
				report(progressUpdate{Message: statusLine, Current: done, Total: total})
				last[key] = statusLine
			}
			if snapshot.IsTerminal() {
				done++
				if snapshot.State != "completed" {
					failed = append(failed, fmt.Sprintf("%s %s/%s: %s", snapshot.State, result.Kind, result.EntityID, string(snapshot.Failure)))
				}
			}
		}
		if done == total {
			output := fmt.Sprintf("sync_tasks=%d completed=%d failed=%d", total, done, len(failed))
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

func syncTaskStatusLine(snapshot workflows.Snapshot) string {
	parts := []string{fmt.Sprintf("%s %s", snapshot.State, snapshot.TaskID)}
	if snapshot.Queue != "" {
		parts = append(parts, "queue="+snapshot.Queue)
	}
	if len(snapshot.Result) > 0 {
		parts = append(parts, "result="+compactJSON(snapshot.Result))
	}
	if len(snapshot.Failure) > 0 {
		parts = append(parts, "failure="+compactJSON(snapshot.Failure))
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
