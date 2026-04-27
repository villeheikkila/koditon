package consumers

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"koditon/internal/db"
	"koditon/internal/platform/taskqueue"
	syncjobs "koditon/internal/sync/jobs"
)

type syncJobRunner func(ctx context.Context, logger *slog.Logger, job db.SyncJob) error

func (c *Consumer) handleSyncJobTask(ctx context.Context, queueName string, logger *slog.Logger, msg taskqueue.Message, run syncJobRunner) error {
	if msg.Data.SyncJobID == nil {
		return taskqueue.NewPermanentError(errors.New("missing sync job id"), "invalid sync job message")
	}
	decision, err := c.syncJobs.ClaimForDispatch(ctx, *msg.Data.SyncJobID)
	if err != nil {
		return err
	}
	if decision.Delete {
		return nil
	}
	if decision.Retry {
		return taskqueue.NewRetryableErrorWithDelay(errors.New(decision.ErrorCode), int(durationSeconds(decision.RetryAfter)))
	}
	if decision.Claim == nil {
		return taskqueue.NewRetryableErrorWithDelay(errors.New("sync job not claimed"), int(durationSeconds(30*time.Second)))
	}
	claim := *decision.Claim
	attemptID, err := c.syncJobs.InsertAttemptRunning(ctx, claim.Job, queueName, msg.MessageID)
	if err != nil {
		retryAt := time.Now().Add(30 * time.Second)
		_ = c.syncJobs.TransitionToRetry(ctx, claim.Job, syncjobs.RetryUpdate{RunAfter: retryAt, ErrorCode: stringPtr("attempt_insert_failed"), Error: stringPtr(err.Error())})
		return taskqueue.NewRetryableErrorWithDelay(err, 30)
	}
	runErr := run(ctx, logger, claim.Job)
	if runErr != nil {
		classified := classifyError(runErr)
		if taskqueue.IsRetryable(classified) && claim.AttemptNo < claim.Job.SyncJobMaxAttempts {
			delay := retryDelay(classified)
			transitionErr := c.syncJobs.TransitionToRetry(ctx, claim.Job, syncjobs.RetryUpdate{RunAfter: time.Now().Add(delay), ErrorCode: stringPtr("sync_retry"), Error: stringPtr(runErr.Error())})
			if transitionErr != nil {
				return transitionErr
			}
			_ = c.syncJobs.FinalizeAttempt(ctx, attemptID, "retry", stringPtr("sync_retry"), stringPtr(runErr.Error()))
			return taskqueue.NewRetryableErrorWithDelay(runErr, int(durationSeconds(delay)))
		}
		status := syncjobs.StatusFailed
		code := "sync_failed"
		if taskqueue.IsPermanent(classified) {
			code = "sync_permanent_failure"
		}
		if err := c.syncJobs.TransitionToFinal(ctx, claim.Job, syncjobs.FinalUpdate{Status: status, ErrorCode: stringPtr(code), Error: stringPtr(runErr.Error())}); err != nil {
			return err
		}
		_ = c.syncJobs.FinalizeAttempt(ctx, attemptID, status, stringPtr(code), stringPtr(runErr.Error()))
		return nil
	}
	if err := c.syncJobs.TransitionToFinal(ctx, claim.Job, syncjobs.FinalUpdate{Status: syncjobs.StatusSucceeded}); err != nil {
		return err
	}
	_ = c.syncJobs.FinalizeAttempt(ctx, attemptID, syncjobs.StatusSucceeded, nil, nil)
	return nil
}

func retryDelay(err error) time.Duration {
	if seconds := taskqueue.GetRetryDelay(err); seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return 30 * time.Second
}

func durationSeconds(value time.Duration) int64 {
	seconds := int64(value / time.Second)
	if seconds <= 0 {
		return 1
	}
	return seconds
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
