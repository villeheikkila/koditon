package taskqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"koditon-go/internal/db"
	"koditon-go/internal/pgmq"
)

type DLQEntry struct {
	DLQID             int64
	OriginalTaskID    int64
	EntityID          string
	TaskType          string
	Priority          int32
	TotalAttempts     int32
	FirstError        *string
	LastError         string
	ErrorHistory      json.RawMessage
	TaskMetadata      json.RawMessage
	OriginalCreatedAt time.Time
	FirstAttemptedAt  *time.Time
	LastAttemptedAt   time.Time
	MovedToDLQAt      time.Time
	RequeuedAt        *time.Time
	RequeueCount      int32
}

type DLQStats struct {
	Total    int64
	Pending  int64
	Requeued int64
}

type DLQByTaskType struct {
	TaskType string
	Count    int64
}

const (
	QueueName = "tasks"
)

var ErrNoRows = pgmq.ErrNoRows

type TaskMessageData struct {
	TaskID   int64  `json:"task_id"`
	EntityID string `json:"entity_id"`
	Attempt  int32  `json:"attempt"`
}

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusStopped    TaskStatus = "stopped"
)

type EntityStatus string

const (
	EntityStatusActive  EntityStatus = "active"
	EntityStatusStopped EntityStatus = "stopped"
)

type TaskMessage struct {
	MessageID  int64
	ReadCount  int32
	EnqueuedAt time.Time
	VT         time.Time
	Message    TaskMessageData
}

type Client struct {
	pool       *pgxpool.Pool
	queries    *db.Queries
	pgmqClient *pgmq.Client
}

func NewClient(pool *pgxpool.Pool) *Client {
	return &Client{
		pool:       pool,
		queries:    db.New(pool),
		pgmqClient: pgmq.NewWithPool(pool),
	}
}

func (c *Client) EnsureQueue(ctx context.Context) error {
	if err := c.pgmqClient.CreateQueue(ctx, QueueName); err != nil {
		return fmt.Errorf("create queue %s: %w", QueueName, err)
	}
	return nil
}

func (c *Client) ReadTask(ctx context.Context, visibilityTimeoutSeconds int) (*TaskMessage, error) {
	msg, err := c.pgmqClient.Read(ctx, QueueName, int64(visibilityTimeoutSeconds))
	if err != nil {
		if pgmq.IsNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read task from queue: %w", err)
	}
	var msgData TaskMessageData
	if err := json.Unmarshal(msg.Message, &msgData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task message: %w", err)
	}
	return &TaskMessage{
		MessageID:  msg.MsgID,
		ReadCount:  int32(msg.ReadCount),
		EnqueuedAt: msg.EnqueuedAt,
		VT:         msg.VT,
		Message:    msgData,
	}, nil
}

func (c *Client) DeleteTaskFromQueue(ctx context.Context, messageID int64) error {
	deleted, err := c.pgmqClient.Delete(ctx, QueueName, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete task from queue: %w", err)
	}
	if !deleted {
		return fmt.Errorf("task message %d not found in queue", messageID)
	}
	return nil
}

func (c *Client) ArchiveTaskFromQueue(ctx context.Context, messageID int64) error {
	archived, err := c.pgmqClient.Archive(ctx, QueueName, messageID)
	if err != nil {
		return fmt.Errorf("failed to archive task from queue: %w", err)
	}
	if !archived {
		return fmt.Errorf("task message %d not found in queue", messageID)
	}
	return nil
}

func (c *Client) EnqueueTask(ctx context.Context, taskID int64, entityID string, attempt int32, scheduledFor time.Time) (int64, error) {
	msgData := TaskMessageData{
		TaskID:   taskID,
		EntityID: entityID,
		Attempt:  attempt,
	}
	msgJSON, err := json.Marshal(msgData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal task message: %w", err)
	}
	delay := int(time.Until(scheduledFor).Seconds())
	delay = max(delay, 0)
	msgID, err := c.pgmqClient.SendWithDelay(ctx, QueueName, json.RawMessage(msgJSON), delay)
	if err != nil {
		return 0, fmt.Errorf("failed to enqueue task: %w", err)
	}
	return msgID, nil
}

func (c *Client) EnqueueTaskImmediate(ctx context.Context, taskID int64, entityID string, attempt int32) (int64, error) {
	return c.EnqueueTask(ctx, taskID, entityID, attempt, time.Now())
}

func (c *Client) GetQueueMetrics(ctx context.Context) (*QueueMetrics, error) {
	metrics, err := c.pgmqClient.Metrics(ctx, QueueName)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue metrics: %w", err)
	}
	return &QueueMetrics{
		QueueName:       metrics.QueueName,
		QueueLength:     metrics.QueueLength,
		NewestMsgAgeSec: &metrics.NewestMsgAgeSec,
		OldestMsgAgeSec: &metrics.OldestMsgAgeSec,
		TotalMessages:   metrics.TotalMessages,
		ScrapeTime:      metrics.ScrapeTime,
	}, nil
}

type QueueMetrics struct {
	QueueName       string
	QueueLength     int64
	NewestMsgAgeSec *int32
	OldestMsgAgeSec *int32
	TotalMessages   int64
	ScrapeTime      time.Time
}

func (c *Client) RegisterEntity(ctx context.Context, entityID, entityType, status, schedulingStrategy string) error {
	err := c.queries.CallRegisterEntity(ctx, db.CallRegisterEntityParams{
		Column1: entityID,
		Column2: entityType,
		Column3: status,
		Column4: schedulingStrategy,
		Column5: []byte("{}"),
	})
	if err != nil {
		return fmt.Errorf("failed to register entity: %w", err)
	}
	return nil
}

func (c *Client) RegisterEntities(ctx context.Context, entityIDs []string, entityType, schedulingStrategy string) (int, error) {
	count, err := c.queries.CallRegisterEntities(ctx, db.CallRegisterEntitiesParams{
		Column1: entityIDs,
		Column2: entityType,
		Column3: schedulingStrategy,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to register entities: %w", err)
	}
	return int(count), nil
}

func (c *Client) ScheduleDailySyncs(ctx context.Context, taskType string) (int, error) {
	count, err := c.queries.CallScheduleDailySyncs(ctx, taskType)
	if err != nil {
		return 0, fmt.Errorf("failed to schedule daily syncs: %w", err)
	}
	return int(count), nil
}

func (c *Client) RequeueStuckTasks(ctx context.Context) (int, error) {
	count, err := c.queries.CallRequeueStuckTasks(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to requeue stuck tasks: %w", err)
	}
	return int(count), nil
}

// CreateTaskWithPriority creates a new task with the specified priority
func (c *Client) CreateTaskWithPriority(ctx context.Context, entityID, taskType string, priority int, maxAttempts int, scheduledFor time.Time, runOn *time.Time) (int64, error) {
	task, err := c.queries.CreateTaskWithPriority(ctx, db.CreateTaskWithPriorityParams{
		EntityID:     entityID,
		TaskType:     taskType,
		Priority:     int32(priority),
		MaxAttempts:  int32(maxAttempts),
		ScheduledFor: scheduledFor,
		RunOn:        runOn,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create task with priority: %w", err)
	}
	return task.TaskID, nil
}

func (c *Client) UpdateTaskPriority(ctx context.Context, taskID int64, priority int) error {
	err := c.queries.UpdateTaskPriority(ctx, db.UpdateTaskPriorityParams{TaskID: taskID, Priority: int32(priority)})
	if err != nil {
		return fmt.Errorf("failed to update task priority: %w", err)
	}
	return nil
}

// DLQ Operations

func (c *Client) GetDLQEntry(ctx context.Context, dlqID int64) (*DLQEntry, error) {
	entry, err := c.queries.GetDLQEntry(ctx, dlqID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ entry: %w", err)
	}
	return convertDBDLQEntry(entry), nil
}

func (c *Client) ListDLQEntries(ctx context.Context, limit, offset int) ([]DLQEntry, error) {
	entries, err := c.queries.ListDLQEntries(ctx, db.ListDLQEntriesParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list DLQ entries: %w", err)
	}
	result := make([]DLQEntry, len(entries))
	for i, e := range entries {
		result[i] = *convertDBDLQEntry(e)
	}
	return result, nil
}

func (c *Client) ListDLQEntriesNotRequeued(ctx context.Context, limit, offset int) ([]DLQEntry, error) {
	entries, err := c.queries.ListDLQEntriesNotRequeued(ctx, db.ListDLQEntriesNotRequeuedParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pending DLQ entries: %w", err)
	}
	result := make([]DLQEntry, len(entries))
	for i, e := range entries {
		result[i] = *convertDBDLQEntry(e)
	}
	return result, nil
}

func (c *Client) ListDLQEntriesByTaskType(ctx context.Context, taskType string, limit, offset int) ([]DLQEntry, error) {
	entries, err := c.queries.ListDLQEntriesByTaskType(ctx, db.ListDLQEntriesByTaskTypeParams{
		TaskType: taskType,
		Limit:    int64(limit),
		Offset:   int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list DLQ entries by task type: %w", err)
	}
	result := make([]DLQEntry, len(entries))
	for i, e := range entries {
		result[i] = *convertDBDLQEntry(e)
	}
	return result, nil
}

func (c *Client) ListDLQEntriesByEntity(ctx context.Context, entityID string, limit, offset int) ([]DLQEntry, error) {
	entries, err := c.queries.ListDLQEntriesByEntity(ctx, db.ListDLQEntriesByEntityParams{
		EntityID: entityID,
		Limit:    int64(limit),
		Offset:   int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list DLQ entries by entity: %w", err)
	}
	result := make([]DLQEntry, len(entries))
	for i, e := range entries {
		result[i] = *convertDBDLQEntry(e)
	}
	return result, nil
}

func (c *Client) GetDLQStats(ctx context.Context) (*DLQStats, error) {
	stats, err := c.queries.CountDLQEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count DLQ entries: %w", err)
	}
	return &DLQStats{
		Total:    stats.Total,
		Pending:  stats.Pending,
		Requeued: stats.Requeued,
	}, nil
}

func (c *Client) GetDLQStatsByTaskType(ctx context.Context) ([]DLQByTaskType, error) {
	rows, err := c.queries.CountDLQEntriesByTaskType(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count DLQ entries by task type: %w", err)
	}
	result := make([]DLQByTaskType, len(rows))
	for i, r := range rows {
		result[i] = DLQByTaskType{
			TaskType: r.TaskType,
			Count:    r.Count,
		}
	}
	return result, nil
}

func (c *Client) RequeueFromDLQ(ctx context.Context, dlqID int64, priority *int, maxAttempts int) (int64, error) {
	var priorityVal int64
	if priority != nil {
		priorityVal = int64(*priority)
	}
	taskID, err := c.queries.CallRequeueFromDLQ(ctx, db.CallRequeueFromDLQParams{
		Column1: dlqID,
		Column2: int32(priorityVal),
		Column3: int32(maxAttempts),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to requeue from DLQ: %w", err)
	}
	msgID, err := c.EnqueueTaskImmediate(ctx, taskID, "", 0)
	if err != nil {
		return taskID, fmt.Errorf("task created but failed to enqueue: %w", err)
	}
	_ = c.queries.UpdateTaskQueueMessageId(ctx, db.UpdateTaskQueueMessageIdParams{TaskID: taskID, QueueMessageID: &msgID})
	return taskID, nil
}

func (c *Client) DeleteDLQEntry(ctx context.Context, dlqID int64) error {
	err := c.queries.DeleteDLQEntry(ctx, dlqID)
	if err != nil {
		return fmt.Errorf("failed to delete DLQ entry: %w", err)
	}
	return nil
}

func (c *Client) CleanupOldDLQEntries(ctx context.Context, olderThan time.Time) (int64, error) {
	count, err := c.queries.DeleteOldDLQEntries(ctx, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old DLQ entries: %w", err)
	}
	return count, nil
}

func convertDBDLQEntry(e db.TaskQueueDeadLetterQueue) *DLQEntry {
	entry := &DLQEntry{
		DLQID:             e.DlqID,
		OriginalTaskID:    e.OriginalTaskID,
		EntityID:          e.EntityID,
		TaskType:          e.TaskType,
		Priority:          e.Priority,
		TotalAttempts:     e.TotalAttempts,
		LastError:         e.LastError,
		ErrorHistory:      e.ErrorHistory,
		TaskMetadata:      e.TaskMetadata,
		OriginalCreatedAt: e.OriginalCreatedAt,
		LastAttemptedAt:   e.LastAttemptedAt,
		MovedToDLQAt:      e.MovedToDlqAt,
		RequeueCount:      e.RequeueCount,
	}
	entry.FirstError = e.FirstError
	entry.FirstAttemptedAt = e.FirstAttemptedAt
	entry.RequeuedAt = e.RequeuedAt
	return entry
}
