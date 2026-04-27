package pgmq

import (
	"encoding/json"
	"time"
)

const (
	DefaultVT        = 30
	DefaultReadLimit = 1
	QueuePrefix      = "q"
	ArchivePrefix    = "a"
	PGMQSchema       = "pgmq"
)

type Message struct {
	MsgID      int64           `json:"msg_id"`
	ReadCount  int64           `json:"read_ct"`
	EnqueuedAt time.Time       `json:"enqueued_at"`
	VT         time.Time       `json:"vt"`
	Message    json.RawMessage `json:"message"`
	Headers    json.RawMessage `json:"headers,omitempty"`
}

type QueueMeta struct {
	QueueName     string    `json:"queue_name"`
	IsPartitioned bool      `json:"is_partitioned"`
	IsUnlogged    bool      `json:"is_unlogged"`
	CreatedAt     time.Time `json:"created_at"`
}

type QueueMetrics struct {
	QueueName       string    `json:"queue_name"`
	QueueLength     int64     `json:"queue_length"`
	NewestMsgAgeSec int32     `json:"newest_msg_age_sec"`
	OldestMsgAgeSec int32     `json:"oldest_msg_age_sec"`
	TotalMessages   int64     `json:"total_messages"`
	ScrapeTime      time.Time `json:"scrape_time"`
}
