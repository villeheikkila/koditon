package pgmq

import (
	"errors"
)

var (
	ErrNoRows           = errors.New("no rows in result set")
	ErrInvalidQueueName = errors.New("invalid queue name")
)

func IsNoRows(err error) bool {
	return errors.Is(err, ErrNoRows)
}
