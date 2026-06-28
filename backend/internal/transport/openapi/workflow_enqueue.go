package api

import (
	"context"
	"fmt"

	"koditon/internal/sync/workflows"
)

func (a *API) spawnSyncWorkflow(ctx context.Context, provider, kind, entityID string, payload []byte) (string, bool, error) {
	def, ok := workflows.FindDefinition(provider, kind)
	if !ok {
		return "", false, fmt.Errorf("unknown sync workflow: %s/%s", provider, kind)
	}
	app, err := workflows.NewClient(a.cfg.DatabaseURL, def.Queue)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = app.Close() }()
	result, err := workflows.Spawn(ctx, app, workflows.SpawnRequest{
		Provider: provider,
		Kind:     kind,
		EntityID: entityID,
		Payload:  payload,
	})
	if err != nil {
		return "", false, err
	}
	return result.TaskID, result.Created, nil
}
