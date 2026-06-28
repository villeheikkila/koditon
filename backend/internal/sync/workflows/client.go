package workflows

import (
	"context"
	"errors"
	"fmt"

	"github.com/earendil-works/absurd/sdks/go/absurd"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// NewClient creates an Absurd client for sync workflows.
func NewClient(databaseURL, defaultQueue string) (*absurd.Client, error) {
	if databaseURL == "" {
		return nil, errors.New("database URL is required")
	}
	if defaultQueue == "" {
		defaultQueue = QueueCanonicalDB
	}
	app, err := absurd.New(absurd.Options{DriverName: "pgx", DatabaseURL: databaseURL, QueueName: defaultQueue})
	if err != nil {
		return nil, fmt.Errorf("create absurd client: %w", err)
	}
	return app, nil
}

// EnsureQueues creates every Absurd queue used by sync workflows.
func EnsureQueues(ctx context.Context, app *absurd.Client) error {
	if app == nil {
		return errors.New("absurd client is required")
	}
	for _, queue := range QueueNames() {
		if err := app.CreateQueue(ctx, queue); err != nil {
			return fmt.Errorf("create absurd queue %s: %w", queue, err)
		}
	}
	return nil
}
