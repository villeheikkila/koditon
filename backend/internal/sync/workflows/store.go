package workflows

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/earendil-works/absurd/sdks/go/absurd"
)

type Store struct {
	app absurdClient
}

type absurdClient interface {
	FetchTaskResult(context.Context, string, string) (*absurd.TaskResultSnapshot, error)
	Spawn(context.Context, string, any, ...absurd.SpawnOptions) (absurd.SpawnResult, error)
}

type EnqueueResult struct {
	TaskID   string
	RunID    string
	Attempt  int
	Created  bool
	Queue    string
	TaskName string
}

type Snapshot struct {
	TaskID  string
	Queue   string
	State   absurd.TaskResultState
	Result  []byte
	Failure []byte
}

type WatchOptions struct {
	Interval time.Duration
	Timeout  time.Duration
}

var ErrTaskNotFound = errors.New("absurd task not found")

func NewStore(app *absurd.Client) *Store {
	return &Store{app: app}
}

func (s *Store) Enqueue(ctx context.Context, req SpawnTaskRequest) (EnqueueResult, error) {
	if s == nil || s.app == nil {
		return EnqueueResult{}, errors.New("absurd store is not configured")
	}
	def, ok := FindDefinition(req.TaskName)
	if !ok {
		return EnqueueResult{}, fmt.Errorf("%w: %s", ErrUnknownTask, req.TaskName)
	}
	spawned, err := Spawn(ctx, s.app, req)
	if err != nil {
		return EnqueueResult{}, err
	}
	return EnqueueResult{
		TaskID:   spawned.TaskID,
		RunID:    spawned.RunID,
		Attempt:  spawned.Attempt,
		Created:  spawned.Created,
		Queue:    def.Queue,
		TaskName: strings.TrimSpace(req.TaskName),
	}, nil
}

func (s *Store) GetSnapshot(ctx context.Context, taskID string) (Snapshot, error) {
	if s == nil || s.app == nil {
		return Snapshot{}, errors.New("absurd store is not configured")
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return Snapshot{}, errors.New("task id is required")
	}
	for _, queue := range QueueNames() {
		snapshot, err := s.app.FetchTaskResult(ctx, queue, taskID)
		if err != nil {
			return Snapshot{}, fmt.Errorf("fetch absurd task %s from queue %s: %w", taskID, queue, err)
		}
		if snapshot != nil {
			return Snapshot{TaskID: taskID, Queue: queue, State: snapshot.State, Result: snapshot.Result, Failure: snapshot.Failure}, nil
		}
	}
	return Snapshot{}, fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
}

func (s *Store) Watch(ctx context.Context, taskID string, options WatchOptions, onSnapshot func(Snapshot) error) error {
	if options.Interval <= 0 {
		options.Interval = 2 * time.Second
	}
	watchCtx := ctx
	cancel := func() {}
	if options.Timeout > 0 {
		watchCtx, cancel = context.WithTimeout(ctx, options.Timeout)
	}
	defer cancel()
	ticker := time.NewTicker(options.Interval)
	defer ticker.Stop()
	for {
		snapshot, err := s.GetSnapshot(watchCtx, taskID)
		if err != nil {
			return err
		}
		if onSnapshot != nil {
			if err := onSnapshot(snapshot); err != nil {
				return err
			}
		}
		if snapshot.IsTerminal() {
			return nil
		}
		select {
		case <-watchCtx.Done():
			return watchCtx.Err()
		case <-ticker.C:
		}
	}
}

func (s Snapshot) IsTerminal() bool {
	switch s.State {
	case absurd.TaskCompleted, absurd.TaskFailed, absurd.TaskCancelled:
		return true
	default:
		return false
	}
}
