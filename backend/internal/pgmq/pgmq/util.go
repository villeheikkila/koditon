package pgmq

import (
	"fmt"
	"regexp"
)

const (
	namedatalen      = 64
	maxIdentifierLen = namedatalen - 1
	biggestConcat    = "archived_at_idx_"
	maxQueueNameLen  = maxIdentifierLen - len(biggestConcat)
)

var queueNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func ValidateQueueName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: queue name cannot be empty", ErrInvalidQueueName)
	}
	if len(name) > maxQueueNameLen {
		return fmt.Errorf("%w: queue name too long (max %d characters, got %d)",
			ErrInvalidQueueName, maxQueueNameLen, len(name))
	}
	if !queueNameRegex.MatchString(name) {
		return fmt.Errorf("%w: queue name must contain only alphanumeric characters and underscores",
			ErrInvalidQueueName)
	}
	return nil
}

func withDefaultVT(vt int64) int64 {
	if vt == 0 {
		return DefaultVT
	}
	return vt
}
