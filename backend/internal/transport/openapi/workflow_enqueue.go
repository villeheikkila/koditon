package api

import (
	"context"

	"koditon/internal/sync/workflows"
)

func (a *API) spawnSyncWorkflow(ctx context.Context, taskName string, params []byte) (string, bool, error) {
	def, ok := workflows.FindDefinition(taskName)
	if !ok {
		return "", false, workflows.ErrUnknownTask
	}
	app, err := workflows.NewClient(a.cfg.DatabaseURL, def.Queue)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = app.Close() }()
	result, err := workflows.Spawn(ctx, app, workflows.SpawnTaskRequest{
		TaskName: taskName,
		Params:   params,
	})
	if err != nil {
		return "", false, err
	}
	return result.TaskID, result.Created, nil
}
