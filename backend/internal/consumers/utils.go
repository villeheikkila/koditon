package consumers

import (
	"strings"
)

func parseEntityID(entityID string) (entityType, value string, err error) {
	idx := strings.Index(entityID, ":")
	if idx == -1 || idx == 0 || idx == len(entityID)-1 {
		return "", "", &EntityParseError{
			EntityID: entityID,
			Reason:   "expected 'type:value' format",
		}
	}
	return entityID[:idx], entityID[idx+1:], nil
}
