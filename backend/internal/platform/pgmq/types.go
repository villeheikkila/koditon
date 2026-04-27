package pgmq

import (
	"encoding/json"
	"time"
)

const (
	DefaultVT = 30
)

type Message struct {
	MsgID      int64           `json:"msg_id"`
	ReadCount  int64           `json:"read_ct"`
	EnqueuedAt time.Time       `json:"enqueued_at"`
	VT         time.Time       `json:"vt"`
	Message    json.RawMessage `json:"message"`
	Headers    json.RawMessage `json:"headers,omitempty"`
}
