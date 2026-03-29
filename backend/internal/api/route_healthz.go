package api

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
)

type livenessOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}

type readinessOutput struct {
	Body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
}

func (a *API) livezHandler(_ context.Context, _ *struct{}) (*livenessOutput, error) {
	out := &livenessOutput{}
	out.Body.Status = "ok"
	return out, nil
}

func (a *API) readyzHandler(ctx context.Context, _ *struct{}) (*readinessOutput, error) {
	checks := make(map[string]string, 2)
	var failed []string

	if err := a.pool.Ping(ctx); err != nil {
		checks["database"] = fmt.Sprintf("unavailable: %s", err.Error())
		failed = append(failed, "database")
	} else {
		checks["database"] = "ok"
	}

	if err := a.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = fmt.Sprintf("unavailable: %s", err.Error())
		failed = append(failed, "redis")
	} else {
		checks["redis"] = "ok"
	}

	if len(failed) > 0 {
		out := &readinessOutput{}
		out.Body.Status = "degraded"
		out.Body.Checks = checks
		return nil, huma.Error503ServiceUnavailable("service not ready", fmt.Errorf("failing checks: %v", failed))
	}

	out := &readinessOutput{}
	out.Body.Status = "ok"
	out.Body.Checks = checks
	return out, nil
}
