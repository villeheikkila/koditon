package pgmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"koditon-go/internal/pgmq/db"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Client struct {
	queries *db.Queries
}

func New(dbtx DBTX) *Client {
	return &Client{
		queries: db.New(dbtx),
	}
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

func (c *Client) CreateUnloggedQueue(ctx context.Context, queueName string) error {
	if err := ValidateQueueName(queueName); err != nil {
		return fmt.Errorf("create unlogged queue %s: %w", queueName, err)
	}
	err := c.queries.CreateUnloggedQueue(ctx, queueName)
	if err != nil {
		return fmt.Errorf("create unlogged queue %s: %w", queueName, err)
	}
	return nil
}

func (c *Client) CreatePartitionedQueue(ctx context.Context, queueName, partitionInterval, retentionInterval string) error {
	if err := ValidateQueueName(queueName); err != nil {
		return fmt.Errorf("create partitioned queue %s: %w", queueName, err)
	}
	err := c.queries.CreatePartitionedQueue(ctx, queueName, partitionInterval, retentionInterval)
	if err != nil {
		return fmt.Errorf("create partitioned queue %s: %w", queueName, err)
	}
	return nil
}

func (c *Client) DropQueue(ctx context.Context, queueName string) error {
	if err := ValidateQueueName(queueName); err != nil {
		return fmt.Errorf("drop queue %s: %w", queueName, err)
	}
	dropped, err := c.queries.DropQueue(ctx, queueName)
	if err != nil {
		return fmt.Errorf("drop queue %s: %w", queueName, err)
	}
	if !dropped {
		return fmt.Errorf("drop queue %s: %w", queueName, ErrQueueNotFound)
	}
	return nil
}

func (c *Client) PurgeQueue(ctx context.Context, queueName string) (int64, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return 0, fmt.Errorf("purge queue %s: %w", queueName, err)
	}
	count, err := c.queries.PurgeQueue(ctx, queueName)
	if err != nil {
		return 0, fmt.Errorf("purge queue %s: %w", queueName, err)
	}
	return count, nil
}

func (c *Client) ListQueues(ctx context.Context) ([]QueueMeta, error) {
	rows, err := c.queries.ListQueues(ctx)
	if err != nil {
		return nil, fmt.Errorf("list queues: %w", err)
	}
	queues := make([]QueueMeta, 0, len(rows))
	for _, row := range rows {
		q := QueueMeta{
			QueueName:     row.QueueName,
			IsPartitioned: row.IsPartitioned,
			IsUnlogged:    row.IsUnlogged,
			CreatedAt:     row.CreatedAt,
		}
		queues = append(queues, q)
	}
	return queues, nil
}

func (c *Client) Send(ctx context.Context, queueName string, msg json.RawMessage) (int64, error) {
	return c.SendWithDelay(ctx, queueName, msg, 0)
}

func (c *Client) SendWithDelay(ctx context.Context, queueName string, msg json.RawMessage, delaySecs int) (int64, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return 0, fmt.Errorf("send message to queue %s: %w", queueName, err)
	}
	msgID, err := c.queries.Send(ctx, queueName, msg, int32(delaySecs))
	if err != nil {
		return 0, fmt.Errorf("send message to queue %s: %w", queueName, err)
	}
	return msgID, nil
}

func (c *Client) SendBatch(ctx context.Context, queueName string, msgs []json.RawMessage) ([]int64, error) {
	return c.SendBatchWithDelay(ctx, queueName, msgs, 0)
}

func (c *Client) SendBatchWithDelay(ctx context.Context, queueName string, msgs []json.RawMessage, delaySecs int) ([]int64, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return nil, fmt.Errorf("send batch to queue %s: %w", queueName, err)
	}
	msgIDs, err := c.queries.SendBatch(ctx, queueName, msgs, int32(delaySecs))
	if err != nil {
		return nil, fmt.Errorf("send batch to queue %s: %w", queueName, err)
	}
	return msgIDs, nil
}

func (c *Client) Read(ctx context.Context, queueName string, vtSecs int64) (*Message, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return nil, fmt.Errorf("read from queue %s: %w", queueName, err)
	}
	vtSecs = withDefaultVT(vtSecs)
	rows, err := c.queries.Read(ctx, queueName, int32(vtSecs), 1)
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

func (c *Client) ReadBatch(ctx context.Context, queueName string, vtSecs int64, numMsgs int64) ([]*Message, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return nil, fmt.Errorf("read batch from queue %s: %w", queueName, err)
	}
	vtSecs = withDefaultVT(vtSecs)
	rows, err := c.queries.Read(ctx, queueName, int32(vtSecs), int32(numMsgs))
	if err != nil {
		return nil, fmt.Errorf("read batch from queue %s: %w", queueName, err)
	}
	msgs := make([]*Message, len(rows))
	for i, row := range rows {
		msgs[i] = &Message{
			MsgID:      row.MsgID,
			ReadCount:  int64(row.ReadCt),
			EnqueuedAt: row.EnqueuedAt,
			VT:         row.Vt,
			Message:    row.Message,
			Headers:    row.Headers,
		}
	}
	return msgs, nil
}

func (c *Client) Pop(ctx context.Context, queueName string) (*Message, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return nil, fmt.Errorf("pop from queue %s: %w", queueName, err)
	}
	rows, err := c.queries.Pop(ctx, queueName)
	if err != nil {
		return nil, fmt.Errorf("pop from queue %s: %w", queueName, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("pop from queue %s: %w", queueName, ErrNoRows)
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

func (c *Client) Archive(ctx context.Context, queueName string, msgID int64) (bool, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return false, fmt.Errorf("archive message %d from queue %s: %w", msgID, queueName, err)
	}
	archived, err := c.queries.Archive(ctx, queueName, msgID)
	if err != nil {
		return false, fmt.Errorf("archive message %d from queue %s: %w", msgID, queueName, err)
	}
	return archived, nil
}

func (c *Client) ArchiveBatch(ctx context.Context, queueName string, msgIDs []int64) ([]int64, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return nil, fmt.Errorf("archive batch from queue %s: %w", queueName, err)
	}
	archivedIDs, err := c.queries.ArchiveBatch(ctx, queueName, msgIDs)
	if err != nil {
		return nil, fmt.Errorf("archive batch from queue %s: %w", queueName, err)
	}
	return archivedIDs, nil
}

func (c *Client) Delete(ctx context.Context, queueName string, msgID int64) (bool, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return false, fmt.Errorf("delete message %d from queue %s: %w", msgID, queueName, err)
	}
	deleted, err := c.queries.Delete(ctx, queueName, msgID)
	if err != nil {
		return false, fmt.Errorf("delete message %d from queue %s: %w", msgID, queueName, err)
	}
	return deleted, nil
}

func (c *Client) DeleteBatch(ctx context.Context, queueName string, msgIDs []int64) ([]int64, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return nil, fmt.Errorf("delete batch from queue %s: %w", queueName, err)
	}
	deletedIDs, err := c.queries.DeleteBatch(ctx, queueName, msgIDs)
	if err != nil {
		return nil, fmt.Errorf("delete batch from queue %s: %w", queueName, err)
	}
	return deletedIDs, nil
}

func (c *Client) SetVisibilityTimeout(ctx context.Context, queueName string, msgID int64, vtSecs int64) (*Message, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return nil, fmt.Errorf("set visibility timeout for message %d in queue %s: %w", msgID, queueName, err)
	}

	rows, err := c.queries.SetVT(ctx, queueName, msgID, int32(vtSecs))
	if err != nil {
		return nil, fmt.Errorf("set visibility timeout for message %d in queue %s: %w", msgID, queueName, err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("set visibility timeout for message %d in queue %s: %w", msgID, queueName, ErrMessageNotFound)
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

func (c *Client) Metrics(ctx context.Context, queueName string) (*QueueMetrics, error) {
	if err := ValidateQueueName(queueName); err != nil {
		return nil, fmt.Errorf("get metrics for queue %s: %w", queueName, err)
	}

	row, err := c.queries.GetQueueMetrics(ctx, queueName)
	if err != nil {
		return nil, fmt.Errorf("get metrics for queue %s: %w", queueName, err)
	}

	m := &QueueMetrics{
		QueueName:       row.QueueName,
		QueueLength:     row.QueueLength,
		NewestMsgAgeSec: row.NewestMsgAgeSec,
		OldestMsgAgeSec: row.OldestMsgAgeSec,
		TotalMessages:   row.TotalMessages,
		ScrapeTime:      row.ScrapeTime,
	}

	return m, nil
}

func (c *Client) MetricsAll(ctx context.Context) ([]QueueMetrics, error) {
	rows, err := c.queries.GetAllQueueMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all queue metrics: %w", err)
	}
	metrics := make([]QueueMetrics, 0, len(rows))
	for _, row := range rows {
		m := QueueMetrics{
			QueueName:       row.QueueName,
			QueueLength:     row.QueueLength,
			NewestMsgAgeSec: row.NewestMsgAgeSec,
			OldestMsgAgeSec: row.OldestMsgAgeSec,
			TotalMessages:   row.TotalMessages,
			ScrapeTime:      row.ScrapeTime,
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}
