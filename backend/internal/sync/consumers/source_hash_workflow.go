package consumers

import (
	"encoding/json"
	"fmt"
)

func decodeSourceAdDataHashBackfillPayload(raw json.RawMessage, provider string) (sourceAdDataHashBackfillPayload, error) {
	payload := sourceAdDataHashBackfillPayload{Limit: 1000}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return sourceAdDataHashBackfillPayload{}, fmt.Errorf("decode %s ad data hash backfill payload: %w", provider, err)
		}
	}
	if payload.Limit <= 0 {
		payload.Limit = 1000
	}
	return payload, nil
}
