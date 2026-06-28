package consumers

import "encoding/json"

type syncJobEnvelope struct {
	SyncJobEntityID   string
	SyncJobPayload    json.RawMessage
	SyncJobCheckpoint json.RawMessage
}
