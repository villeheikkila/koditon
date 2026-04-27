package tui

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	syncflows "koditon/internal/sync/flows"

	tea "charm.land/bubbletea/v2"
)

var ErrJobAlreadyRunning = errors.New("job already running")

type runFinishedMsg struct {
	actionTitle string
	result      actionResult
	err         error
	duration    time.Duration
}

type runProgressMsg struct {
	message string
	current int
	total   int
}

type jobRuntime struct {
	mu     sync.Mutex
	nextID int64
	active *activeJob
}

type activeJob struct {
	id     string
	cancel context.CancelFunc
}

func newJobRuntime() *jobRuntime {
	return &jobRuntime{}
}

func (r *jobRuntime) Start(runner *syncflows.Runner, a action, inputs []string, events chan<- tea.Msg) (string, error) {
	r.mu.Lock()
	if r.active != nil {
		r.mu.Unlock()
		return "", ErrJobAlreadyRunning
	}
	r.nextID++
	jobID := fmt.Sprintf("job-%d", r.nextID)
	ctx, cancel := context.WithCancel(context.Background())
	r.active = &activeJob{id: jobID, cancel: cancel}
	r.mu.Unlock()
	go func() {
		start := time.Now()
		report := func(update progressUpdate) {
			events <- runProgressMsg{message: update.Message, current: update.Current, total: update.Total}
		}
		result, err := a.Run(ctx, runner, inputs, report)
		events <- runFinishedMsg{actionTitle: a.Title, result: result, err: err, duration: time.Since(start)}
		r.mu.Lock()
		if r.active != nil && r.active.id == jobID {
			r.active = nil
		}
		r.mu.Unlock()
	}()
	return jobID, nil
}

func (r *jobRuntime) CancelActive() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == nil {
		return false
	}
	r.active.cancel()
	return true
}

func (r *jobRuntime) HasActive() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active != nil
}
