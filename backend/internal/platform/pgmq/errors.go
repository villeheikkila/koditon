package pgmq

import (
	"errors"
)

var (
	ErrNoRows           = errors.New("no rows in result set")
	ErrInvalidQueueName = errors.New("invalid queue name")
	ErrQueueNotFound    = errors.New("queue not found")
	ErrMessageNotFound  = errors.New("message not found")
)

func IsNoRows(err error) bool {
	return errors.Is(err, ErrNoRows)
}

func IsInvalidQueueName(err error) bool {
	return errors.Is(err, ErrInvalidQueueName)
}

func IsQueueNotFound(err error) bool {
	return errors.Is(err, ErrQueueNotFound)
}

func IsMessageNotFound(err error) bool {
	return errors.Is(err, ErrMessageNotFound)
}
