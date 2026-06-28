package consumers

import (
	"fmt"
)

const (
	priorityLow    = -10
	priorityNormal = 0
)

type permanentError struct {
	err    error
	reason string
}

func (e *permanentError) Error() string {
	if e.reason != "" {
		return fmt.Sprintf("permanent (%s): %v", e.reason, e.err)
	}
	return fmt.Sprintf("permanent: %v", e.err)
}

func (e *permanentError) Unwrap() error {
	return e.err
}

func newPermanentError(err error, reason string) *permanentError {
	return &permanentError{err: err, reason: reason}
}
