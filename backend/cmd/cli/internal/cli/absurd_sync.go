package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/earendil-works/absurd/sdks/go/absurd"

	"koditon/internal/sync/workflows"
)

type AbsurdSyncFlags struct {
	TaskName string
	Params   json.RawMessage
	Watch    bool
	Interval time.Duration
	JSON     bool
	Out      io.Writer
}

type absurdSyncTaskOutput struct {
	ID       string          `json:"id"`
	Queue    string          `json:"queue,omitempty"`
	TaskName string          `json:"task_name,omitempty"`
	State    string          `json:"state,omitempty"`
	Result   json.RawMessage `json:"result,omitempty"`
	Failure  json.RawMessage `json:"failure,omitempty"`
	RunID    string          `json:"run_id,omitempty"`
	Attempt  int             `json:"attempt,omitempty"`
}

func RunAbsurdSync(ctx context.Context, store *workflows.Store, f AbsurdSyncFlags) error {
	out := resolveOutput(f.Out)
	result, err := store.Enqueue(ctx, workflows.SpawnTaskRequest{
		TaskName:    f.TaskName,
		Params:      f.Params,
		MaxAttempts: 3,
	})
	if err != nil {
		return err
	}
	task := mapAbsurdEnqueueResult(result)
	if f.JSON {
		if err := writeJSON(out, map[string]any{"event": "queued", "created": result.Created, "task": task}); err != nil {
			return err
		}
	} else if result.Created {
		fmt.Fprintf(out, "queued absurd task %s id=%s queue=%s\n", result.TaskName, result.TaskID, result.Queue)
	} else {
		fmt.Fprintf(out, "existing absurd task %s id=%s queue=%s\n", result.TaskName, result.TaskID, result.Queue)
	}
	if !f.Watch {
		return nil
	}
	return WatchAbsurdSyncTask(ctx, store, result.TaskID, f.Interval, f.JSON, out)
}

func RunAbsurdSyncStatus(ctx context.Context, store *workflows.Store, taskID string, jsonOutput bool, out io.Writer) error {
	snapshot, err := store.GetSnapshot(ctx, taskID)
	if err != nil {
		return err
	}
	out = resolveOutput(out)
	if jsonOutput {
		return writeJSON(out, map[string]any{"task": mapAbsurdSnapshot(snapshot)})
	}
	fmt.Fprintln(out, FormatAbsurdSyncSnapshot(snapshot))
	return nil
}

func WatchAbsurdSyncTask(ctx context.Context, store *workflows.Store, taskID string, interval time.Duration, jsonOutput bool, out io.Writer) error {
	out = resolveOutput(out)
	lastLine := ""
	err := store.Watch(ctx, taskID, workflows.WatchOptions{Interval: interval}, func(snapshot workflows.Snapshot) error {
		line := FormatAbsurdSyncSnapshot(snapshot)
		if line == lastLine {
			return nil
		}
		if jsonOutput {
			if err := writeJSON(out, map[string]any{"event": "status", "task": mapAbsurdSnapshot(snapshot)}); err != nil {
				return err
			}
		} else {
			fmt.Fprintln(out, line)
		}
		lastLine = line
		return nil
	})
	if err != nil {
		return err
	}
	snapshot, err := store.GetSnapshot(ctx, taskID)
	if err != nil {
		return err
	}
	switch snapshot.State {
	case absurd.TaskCompleted:
		return nil
	case absurd.TaskFailed, absurd.TaskCancelled:
		return fmt.Errorf("absurd task finished with state %s: %s", snapshot.State, string(snapshot.Failure))
	default:
		return nil
	}
}

func FormatAbsurdSyncSnapshot(snapshot workflows.Snapshot) string {
	parts := []string{
		fmt.Sprintf("state=%s", snapshot.State),
		fmt.Sprintf("queue=%s", snapshot.Queue),
		fmt.Sprintf("task=%s", snapshot.TaskID),
	}
	if len(snapshot.Result) > 0 {
		parts = append(parts, "result="+compactJSON(snapshot.Result))
	}
	if len(snapshot.Failure) > 0 {
		parts = append(parts, "failure="+compactJSON(snapshot.Failure))
	}
	return strings.Join(parts, " ")
}

func mapAbsurdEnqueueResult(result workflows.EnqueueResult) absurdSyncTaskOutput {
	return absurdSyncTaskOutput{
		ID:       result.TaskID,
		Queue:    result.Queue,
		TaskName: result.TaskName,
		RunID:    result.RunID,
		Attempt:  result.Attempt,
	}
}

func mapAbsurdSnapshot(snapshot workflows.Snapshot) absurdSyncTaskOutput {
	return absurdSyncTaskOutput{
		ID:      snapshot.TaskID,
		Queue:   snapshot.Queue,
		State:   string(snapshot.State),
		Result:  snapshot.Result,
		Failure: snapshot.Failure,
	}
}
