package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"koditon/internal/domain/ads"
	"koditon/internal/sync/consumers"
	"koditon/internal/sync/workflows"
)

type sourceRefreshInput struct {
	Body struct {
		URL string `json:"url" required:"true" doc:"Frontdoor or Shortcut listing/building URL"`
	}
}

type sourceRefreshOutput struct {
	Body struct {
		URL               string `json:"url"`
		CanonicalSourceID string `json:"canonical_source_id"`
		Provider          string `json:"provider"`
		Kind              string `json:"kind"`
		NativeID          string `json:"native_id"`
		TaskName          string `json:"task_name"`
		TaskID            string `json:"task_id"`
		Created           bool   `json:"created"`
	}
}

type sourceRefreshTask struct {
	TaskName   string
	SourceType string
	SourceID   string
}

func (a *API) sourceRefreshHandler(ctx context.Context, input *sourceRefreshInput) (*sourceRefreshOutput, error) {
	urlText := strings.TrimSpace(input.Body.URL)
	if !strings.HasPrefix(urlText, "http://") && !strings.HasPrefix(urlText, "https://") {
		return nil, huma.Error400BadRequest("url must be an http or https source URL")
	}
	canonicalID, err := ads.ResolveInput(urlText, a.cfg.Shortcut.SitemapBase, a.cfg.Frontdoor.SitemapBase)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid source URL")
	}
	source, kind, nativeID, err := ads.ParseCanonicalID(canonicalID)
	if err != nil {
		return nil, huma.Error400BadRequest("resolved invalid canonical ID")
	}
	task, err := sourceRefreshTaskFor(source, kind, nativeID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	params, err := json.Marshal(map[string]string{"source_type": task.SourceType, "source_id": task.SourceID})
	if err != nil {
		return nil, fmt.Errorf("marshal source refresh params: %w", err)
	}
	taskID, created, err := a.spawnSyncWorkflowRequest(ctx, workflows.SpawnTaskRequest{
		TaskName:       task.TaskName,
		Params:         params,
		IdempotencyKey: fmt.Sprintf("manual-source-refresh:%s:%d", canonicalID, time.Now().UTC().UnixNano()),
	})
	if err != nil {
		a.logger.ErrorContext(ctx, "source refresh enqueue failed", "canonical_source_id", canonicalID, "task_name", task.TaskName, "error", err)
		return nil, huma.Error500InternalServerError("source refresh enqueue failed")
	}
	out := &sourceRefreshOutput{}
	out.Body.URL = urlText
	out.Body.CanonicalSourceID = canonicalID
	out.Body.Provider = source
	out.Body.Kind = kind
	out.Body.NativeID = nativeID
	out.Body.TaskName = task.TaskName
	out.Body.TaskID = taskID
	out.Body.Created = created
	return out, nil
}

func sourceRefreshTaskFor(source string, kind string, nativeID string) (sourceRefreshTask, error) {
	switch source + ":" + kind {
	case "frontdoor:ad":
		return sourceRefreshTask{TaskName: consumers.TaskTypeFrontdoorSync, SourceType: "ad", SourceID: nativeID}, nil
	case "frontdoor:building":
		return sourceRefreshTask{TaskName: consumers.TaskTypeFrontdoorSync, SourceType: "building", SourceID: nativeID}, nil
	case "shortcut:ad":
		return sourceRefreshTask{TaskName: consumers.TaskTypeShortcutAPISync, SourceType: "ad", SourceID: nativeID}, nil
	case "shortcut:building":
		return sourceRefreshTask{TaskName: consumers.TaskTypeShortcutScraperSync, SourceType: "building", SourceID: nativeID}, nil
	default:
		return sourceRefreshTask{}, fmt.Errorf("source URL resolves to unsupported refresh target %s:%s", source, kind)
	}
}
