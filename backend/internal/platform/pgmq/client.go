package pgmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/internal/db"
)

type Client struct {
	queries *db.Queries
}

func NewWithPool(pool *pgxpool.Pool) *Client {
	return &Client{
		queries: db.New(pool),
	}
}

func (c *Client) CreateQueue(ctx context.Context, queueName string) error {
	if err := ValidateQueueName(queueName); err != nil {
		return fmt.Errorf("create queue %s: %w", queueName, err)
	}
	err := c.queries.CreateQueue(ctx, queueName)
	if err != nil {
		return fmt.Errorf("create queue %s: %w", queueName, err)
	}
	return nil
}

func (c *Client) SendWithDelay(ctx context.Context, queueName string, msg json.RawMessage, delaySecs int) (int64, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return 0, fmt.Errorf("send message to queue %s: %w", queueName, err)
	}
	msgID, err := c.queries.Send(ctx, db.SendParams{
		QueueName:    queueName,
		Message:      msg,
		DelaySeconds: int32(delaySecs),
	})
	if err != nil {
		return 0, fmt.Errorf("send message to queue %s: %w", queueName, err)
	}
	return msgID, nil
}

func (c *Client) Read(ctx context.Context, queueName string, vtSecs int64) (*Message, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return nil, fmt.Errorf("read from queue %s: %w", queueName, err)
	}
	vtSecs = withDefaultVT(vtSecs)
	rows, err := c.queries.Read(ctx, db.ReadParams{
		QueueName:   queueName,
		VtSeconds:   int32(vtSecs),
		NumMessages: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("read from queue %s: %w", queueName, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("read from queue %s: %w", queueName, ErrNoRows)
	}
	row := rows[0]
	msg := &Message{
		MsgID:      row.MsgID,
		ReadCount:  int64(row.ReadCt),
		EnqueuedAt: row.EnqueuedAt,
		VT:         row.Vt,
		Message:    row.Message,
		Headers:    row.Headers,
	}
	return msg, nil
}

func (c *Client) Delete(ctx context.Context, queueName string, msgID int64) (bool, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return false, fmt.Errorf("delete message %d from queue %s: %w", msgID, queueName, err)
	}
	deleted, err := c.queries.Delete(ctx, db.DeleteParams{
		QueueName: queueName,
		MsgID:     msgID,
	})
	if err != nil {
		return false, fmt.Errorf("delete message %d from queue %s: %w", msgID, queueName, err)
	}
	return deleted, nil
}
