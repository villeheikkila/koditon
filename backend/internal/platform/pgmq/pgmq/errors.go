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
