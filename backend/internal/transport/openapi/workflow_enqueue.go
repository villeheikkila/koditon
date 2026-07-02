package api

import (
	"context"
	"time"

	"koditon/internal/sync/workflows"
)

func (a *API) spawnSyncWorkflow(ctx context.Context, taskName string, params []byte) (string, bool, error) {
	return a.spawnSyncWorkflowRequest(ctx, workflows.SpawnTaskRequest{TaskName: taskName, Params: params})
}

func (a *API) spawnSyncWorkflowRequest(ctx context.Context, req workflows.SpawnTaskRequest) (string, bool, error) {
	spawnCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	spawner, err := workflows.NewDatabaseTaskSpawner(a.cfg.DatabaseURL)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = spawner.Close() }()
	result, err := spawner.SpawnRaw(spawnCtx, req)
	if err != nil {
		return "", false, err
	}
	return result.TaskID, result.Created, nil
}
