package workflows

import (
	"context"
	"errors"
	"testing"

	"github.com/earendil-works/absurd/sdks/go/absurd"
)

type fakeAbsurdClient struct {
	spawnedTask absurd.SpawnResult
	spawnName   string
	spawnParams any
	spawnOpts   absurd.SpawnOptions
	snapshots   map[string]*absurd.TaskResultSnapshot
	queues      []string
}

func (f *fakeAbsurdClient) Spawn(_ context.Context, taskName string, params any, options ...absurd.SpawnOptions) (absurd.SpawnResult, error) {
	f.spawnName = taskName
	f.spawnParams = params
	if len(options) > 0 {
		f.spawnOpts = options[0]
	}
	return f.spawnedTask, nil
}

func (f *fakeAbsurdClient) FetchTaskResult(_ context.Context, queueName, taskID string) (*absurd.TaskResultSnapshot, error) {
	f.queues = append(f.queues, queueName)
	return f.snapshots[queueName+"/"+taskID], nil
}

func TestStoreEnqueueUsesAbsurdSpawnContract(t *testing.T) {
	t.Parallel()
	client := &fakeAbsurdClient{spawnedTask: absurd.SpawnResult{TaskID: "task-1", RunID: "run-1", Attempt: 1, Created: true}}
	store := &Store{app: client}
	result, err := store.Enqueue(context.Background(), SpawnTaskRequest{TaskName: "frontdoor_sync", Params: []byte(`{"source_type":"ad","source_id":"123"}`)})
	if err != nil {
		t.Fatalf("Enqueue returned error: %v", err)
	}
	if result.TaskID != "task-1" || result.Queue != QueueFrontdoor || !result.Created {
		t.Fatalf("result = %#v", result)
	}
	if client.spawnName != "frontdoor_sync" {
		t.Fatalf("spawn task = %q", client.spawnName)
	}
	if client.spawnOpts.QueueName != QueueFrontdoor {
		t.Fatalf("queue = %q", client.spawnOpts.QueueName)
	}
	if client.spawnOpts.IdempotencyKey == "" || client.spawnOpts.IdempotencyKey == "frontdoor_sync" {
		t.Fatalf("idempotency = %q", client.spawnOpts.IdempotencyKey)
	}
}

func TestStoreGetSnapshotProbesKnownQueues(t *testing.T) {
	t.Parallel()
	client := &fakeAbsurdClient{snapshots: map[string]*absurd.TaskResultSnapshot{
		QueuePrices + "/task-1": {State: absurd.TaskCompleted, Result: []byte(`{"ok":true}`)},
	}}
	store := &Store{app: client}
	snapshot, err := store.GetSnapshot(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if snapshot.Queue != QueuePrices || snapshot.State != absurd.TaskCompleted {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if len(client.queues) < 2 {
		t.Fatalf("expected queue probing, got %v", client.queues)
	}
}

func TestStoreGetSnapshotNotFound(t *testing.T) {
	t.Parallel()
	store := &Store{app: &fakeAbsurdClient{snapshots: map[string]*absurd.TaskResultSnapshot{}}}
	_, err := store.GetSnapshot(context.Background(), "missing")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("GetSnapshot error = %v, want ErrTaskNotFound", err)
	}
}
