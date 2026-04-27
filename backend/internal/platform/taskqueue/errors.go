package taskqueue

import (
	"errors"
	"fmt"
)

type RetryableError struct {
	Err        error
	RetryAfter int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable: %v", e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func NewRetryableError(err error) *RetryableError {
	return &RetryableError{Err: err}
}

func NewRetryableErrorWithDelay(err error, retryAfterSeconds int) *RetryableError {
	return &RetryableError{Err: err, RetryAfter: retryAfterSeconds}
}

type PermanentError struct {
	Err    error
	Reason string
}

func (e *PermanentError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("permanent (%s): %v", e.Reason, e.Err)
	}
	return fmt.Sprintf("permanent: %v", e.Err)
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

func NewPermanentError(err error, reason string) *PermanentError {
	return &PermanentError{Err: err, Reason: reason}
}

func IsRetryable(err error) bool {
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return true
	}
	var permanent *PermanentError
	return !errors.As(err, &permanent)
}

func IsPermanent(err error) bool {
	var permanent *PermanentError
	return errors.As(err, &permanent)
}

func GetRetryDelay(err error) int {
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return retryable.RetryAfter
	}
	return 0
}
