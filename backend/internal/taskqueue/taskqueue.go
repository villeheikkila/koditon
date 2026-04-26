package taskqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"koditon-go/internal/pgmq"
)

const (
	EntityPrefixAd       = "ad:"
	EntityPrefixBuilding = "building:"
	EntityPrefixCity     = "city:"
)

const (
	PriorityLow      = -10
	PriorityNormal   = 0
	PriorityHigh     = 10
	PriorityCritical = 100
)

var ErrNoRows = pgmq.ErrNoRows

type MessageData struct {
	SyncTaskID int64  `json:"sync_task_id"`
	EntityID   string `json:"entity_id"`
	TaskType   string `json:"task_type"`
	Attempt    int32  `json:"attempt"`
}

type Message struct {
	MessageID  int64
	ReadCount  int32
	EnqueuedAt time.Time
	VT         time.Time
	Data       MessageData
}

type Queue struct {
	name       string
	pool       *pgxpool.Pool
	pgmqClient *pgmq.Client
}

func NewQueue(pool *pgxpool.Pool, name string) *Queue {
	return &Queue{
		name:       name,
		pool:       pool,
		pgmqClient: pgmq.NewWithPool(pool),
	}
}

func (q *Queue) Name() string {
	return q.name
}

func (q *Queue) EnsureQueue(ctx context.Context) error {
	if err := q.pgmqClient.CreateQueue(ctx, q.name); err != nil {
		return fmt.Errorf("create queue %s: %w", q.name, err)
	}
	return nil
}

func (q *Queue) Send(ctx context.Context, data MessageData) (int64, error) {
	return q.SendWithDelay(ctx, data, 0)
}

func (q *Queue) SendWithDelay(ctx context.Context, data MessageData, delay time.Duration) (int64, error) {
	msgJSON, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("marshal message: %w", err)
	}
	delaySecs := max(int(delay.Seconds()), 0)
	msgID, err := q.pgmqClient.SendWithDelay(ctx, q.name, json.RawMessage(msgJSON), delaySecs)
	if err != nil {
		return 0, fmt.Errorf("send to queue %s: %w", q.name, err)
	}
	return msgID, nil
}

func (q *Queue) Read(ctx context.Context, visibilityTimeout time.Duration) (*Message, error) {
	vtSeconds := int64(visibilityTimeout.Seconds())
	msg, err := q.pgmqClient.Read(ctx, q.name, vtSeconds)
	if err != nil {
		if pgmq.IsNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read from queue %s: %w", q.name, err)
	}
	var data MessageData
	if err := json.Unmarshal(msg.Message, &data); err != nil {
		return nil, fmt.Errorf("unmarshal message from queue %s: %w", q.name, err)
	}
	return &Message{
		MessageID:  msg.MsgID,
		ReadCount:  int32(msg.ReadCount),
		EnqueuedAt: msg.EnqueuedAt,
		VT:         msg.VT,
		Data:       data,
	}, nil
}

func (q *Queue) Delete(ctx context.Context, messageID int64) error {
	deleted, err := q.pgmqClient.Delete(ctx, q.name, messageID)
	if err != nil {
		return fmt.Errorf("delete from queue %s: %w", q.name, err)
	}
	if !deleted {
		return fmt.Errorf("message %d not found in queue %s", messageID, q.name)
	}
	return nil
}
