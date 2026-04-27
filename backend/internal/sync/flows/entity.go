package syncflows

import (
	"fmt"
	"strings"
)

type EntityParseError struct {
	EntityID string
	Reason   string
	Err      error
}

func (e *EntityParseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("parse entity %s: %s: %v", e.EntityID, e.Reason, e.Err)
	}
	return fmt.Sprintf("parse entity %s: %s", e.EntityID, e.Reason)
}

func (e *EntityParseError) Unwrap() error {
	return e.Err
}

func parseEntityID(entityID string) (entityType, value string, err error) {
	idx := strings.Index(entityID, ":")
	if idx == -1 || idx == 0 || idx == len(entityID)-1 {
		return "", "", &EntityParseError{EntityID: entityID, Reason: "expected 'type:value' format"}
	}
	return entityID[:idx], entityID[idx+1:], nil
}
