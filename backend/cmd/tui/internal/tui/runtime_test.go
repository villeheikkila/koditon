package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"koditon/internal/sync/flows"
)

func TestJobRuntimeStartAndFinish(t *testing.T) {
	runtime := newJobRuntime()
	events := make(chan tea.Msg, 8)
	a := action{Title: "test", Run: func(_ context.Context, _ *syncflows.Runner, _ []string, report reportFn) (actionResult, error) {
		report(progressUpdate{Message: "step", Current: 1, Total: 1})
		return actionResult{Output: "ok"}, nil
	}}
	_, err := runtime.Start(nil, a, nil, events)
	if err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	first := <-events
	if _, ok := first.(runProgressMsg); !ok {
		t.Fatalf("expected first event runProgressMsg, got %T", first)
	}
	second := <-events
	finished, ok := second.(runFinishedMsg)
	if !ok {
		t.Fatalf("expected second event runFinishedMsg, got %T", second)
	}
	if finished.result.Output != "ok" {
		t.Fatalf("expected output ok, got %q", finished.result.Output)
	}
}

func TestJobRuntimeCancelActive(t *testing.T) {
	runtime := newJobRuntime()
	events := make(chan tea.Msg, 8)
	a := action{Title: "cancel", Run: func(ctx context.Context, _ *syncflows.Runner, _ []string, _ reportFn) (actionResult, error) {
		<-ctx.Done()
		return actionResult{}, ctx.Err()
	}}
	_, err := runtime.Start(nil, a, nil, events)
	if err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	if !runtime.HasActive() {
		t.Fatalf("expected active job")
	}
	if !runtime.CancelActive() {
		t.Fatalf("expected cancel to return true")
	}
	msg := <-events
	finished, ok := msg.(runFinishedMsg)
	if !ok {
		t.Fatalf("expected runFinishedMsg, got %T", msg)
	}
	if !errors.Is(finished.err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", finished.err)
	}
	deadline := time.After(2 * time.Second)
	for runtime.HasActive() {
		select {
		case <-deadline:
			t.Fatalf("runtime remained active after cancel")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestJobRuntimeRejectsConcurrentStart(t *testing.T) {
	runtime := newJobRuntime()
	events := make(chan tea.Msg, 8)
	wait := make(chan struct{})
	a := action{Title: "wait", Run: func(_ context.Context, _ *syncflows.Runner, _ []string, _ reportFn) (actionResult, error) {
		<-wait
		return actionResult{}, nil
	}}
	_, err := runtime.Start(nil, a, nil, events)
	if err != nil {
		t.Fatalf("first start returned error: %v", err)
	}
	_, secondErr := runtime.Start(nil, a, nil, events)
	if !errors.Is(secondErr, ErrJobAlreadyRunning) {
		t.Fatalf("expected ErrJobAlreadyRunning, got %v", secondErr)
	}
	close(wait)
	<-events
}
