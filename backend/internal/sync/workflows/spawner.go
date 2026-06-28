package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/earendil-works/absurd/sdks/go/absurd"
)

type TaskSpawner struct {
	app     spawnClient
	closeFn func() error
}

func NewTaskSpawner(app spawnClient) *TaskSpawner {
	return &TaskSpawner{app: app}
}

func NewDatabaseTaskSpawner(databaseURL string) (*TaskSpawner, error) {
	app, err := NewClient(databaseURL, QueueCanonicalDB)
	if err != nil {
		return nil, err
	}
	return &TaskSpawner{app: app, closeFn: app.Close}, nil
}

func (s *TaskSpawner) Close() error {
	if s == nil || s.closeFn == nil {
		return nil
	}
	return s.closeFn()
}

func (s *TaskSpawner) Spawn(ctx context.Context, taskName string, params any) (absurd.SpawnResult, error) {
	raw, err := MarshalParams(params)
	if err != nil {
		return absurd.SpawnResult{}, err
	}
	return s.SpawnRaw(ctx, SpawnTaskRequest{TaskName: taskName, Params: raw})
}

func (s *TaskSpawner) SpawnRaw(ctx context.Context, req SpawnTaskRequest) (absurd.SpawnResult, error) {
	if s == nil || s.app == nil {
		return absurd.SpawnResult{}, errors.New("task spawner is not configured")
	}
	return Spawn(ctx, s.app, req)
}

func (s *TaskSpawner) SpawnCronSlot(ctx context.Context, taskName string, params any, scheduleName string, slot time.Time) (absurd.SpawnResult, error) {
	raw, err := MarshalParams(params)
	if err != nil {
		return absurd.SpawnResult{}, err
	}
	if scheduleName == "" {
		return absurd.SpawnResult{}, errors.New("schedule name is required")
	}
	return s.SpawnRaw(ctx, SpawnTaskRequest{
		TaskName:       taskName,
		Params:         raw,
		IdempotencyKey: CronSlotIdempotencyKey(taskName, scheduleName, slot),
	})
}

func MarshalParams(params any) (json.RawMessage, error) {
	if params == nil {
		return json.RawMessage(`{}`), nil
	}
	switch value := params.(type) {
	case json.RawMessage:
		return normalizeParams(value), ValidateRawParams(value)
	case []byte:
		raw := json.RawMessage(value)
		return normalizeParams(raw), ValidateRawParams(raw)
	case string:
		raw := json.RawMessage(value)
		return normalizeParams(raw), ValidateRawParams(raw)
	default:
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal task params: %w", err)
		}
		return normalizeParams(raw), nil
	}
}

func ValidateRawParams(params json.RawMessage) error {
	if len(params) == 0 {
		return nil
	}
	if !json.Valid(params) {
		return errors.New("params must be valid JSON")
	}
	return nil
}
