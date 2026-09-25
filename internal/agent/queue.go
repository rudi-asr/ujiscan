package agent

import (
	"context"
	"errors"
	"sync"
	"time"
)

// TaskQueue manages agent tasks with priority and dependencies
type TaskQueue struct {
	mu          sync.RWMutex
	tasks       map[string]*Task
	queue       []*Task // Priority queue
	completed   map[string]*Task
	failed      map[string]*Task
	maxRetries  int
	maxQueueLen int
}

// NewTaskQueue creates a new task queue
func NewTaskQueue(maxRetries, maxQueueLen int) *TaskQueue {
	if maxRetries < 0 {
		maxRetries = 3
	}
	if maxQueueLen < 0 {
		maxQueueLen = 1000
	}
	return &TaskQueue{
		tasks:       make(map[string]*Task),
		completed:   make(map[string]*Task),
		failed:      make(map[string]*Task),
		queue:       make([]*Task, 0, maxQueueLen),
		maxRetries:  maxRetries,
		maxQueueLen: maxQueueLen,
	}
}

// Enqueue adds a task to the queue
func (q *TaskQueue) Enqueue(task *Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) >= q.maxQueueLen {
		return errors.New("task queue full")
	}

	if task.ID == "" {
		return errors.New("task ID required")
	}

	// Prevent duplicate task IDs (a task may only be enqueued once;
	// retries re-queue internally via enqueueByPriority, not Enqueue)
	if _, exists := q.tasks[task.ID]; exists {
		return errors.New("task already queued or active")
	}

	if task.Status == "" {
		task.Status = StatusIdle
	}

	if task.CreatedAt == 0 {
		task.CreatedAt = time.Now().Unix()
	}

	if task.MaxRetries == 0 {
		task.MaxRetries = q.maxRetries
	}

	q.tasks[task.ID] = task
	q.enqueueByPriority(task)

	return nil
}

// enqueueByPriority inserts task in priority order (higher priority first)
func (q *TaskQueue) enqueueByPriority(task *Task) {
	inserted := false
	for i, t := range q.queue {
		if task.Priority > t.Priority {
			// Insert before lower priority task
			q.queue = append(q.queue[:i], append([]*Task{task}, q.queue[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.queue = append(q.queue, task)
	}
}

// Dequeue retrieves the next task from the queue
func (q *TaskQueue) Dequeue() *Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		return nil
	}

	task := q.queue[0]
	q.queue = q.queue[1:]
	task.Status = StatusRunning
	task.StartedAt = time.Now().Unix()

	return task
}

// MarkCompleted marks a task as completed
func (q *TaskQueue) MarkCompleted(taskID string, result interface{}) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, exists := q.tasks[taskID]
	if !exists {
		return errors.New("task not found")
	}

	task.Status = StatusCompleted
	task.CompletedAt = time.Now().Unix()
	task.Result = result

	delete(q.tasks, taskID)
	q.completed[taskID] = task

	return nil
}

// MarkFailed marks a task as failed and may retry
func (q *TaskQueue) MarkFailed(taskID string, errMsg string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, exists := q.tasks[taskID]
	if !exists {
		return errors.New("task not found")
	}

	task.Error = errMsg
	task.Retries++

	if task.Retries < task.MaxRetries {
		// Requeue for retry
		task.Status = StatusIdle
		task.StartedAt = 0
		q.enqueueByPriority(task)
		return nil
	}

	// Max retries exceeded, mark as failed
	task.Status = StatusError
	task.CompletedAt = time.Now().Unix()
	delete(q.tasks, taskID)
	q.failed[taskID] = task

	return nil
}

// GetTask retrieves a task by ID
func (q *TaskQueue) GetTask(taskID string) *Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Check active tasks
	if task, exists := q.tasks[taskID]; exists {
		return task
	}

	// Check completed tasks
	if task, exists := q.completed[taskID]; exists {
		return task
	}

	// Check failed tasks
	if task, exists := q.failed[taskID]; exists {
		return task
	}

	return nil
}

// GetTasks retrieves all tasks (optionally filtered by status)
func (q *TaskQueue) GetTasks(status ...AgentStatus) []*Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*Task

	if len(status) == 0 {
		// Return all tasks
		for _, task := range q.tasks {
			result = append(result, task)
		}
		for _, task := range q.completed {
			result = append(result, task)
		}
		for _, task := range q.failed {
			result = append(result, task)
		}
		return result
	}

	// Filter by status
	statusMap := make(map[AgentStatus]bool)
	for _, s := range status {
		statusMap[s] = true
	}

	for _, task := range q.tasks {
		if statusMap[task.Status] {
			result = append(result, task)
		}
	}
	for _, task := range q.completed {
		if statusMap[task.Status] {
			result = append(result, task)
		}
	}
	for _, task := range q.failed {
		if statusMap[task.Status] {
			result = append(result, task)
		}
	}

	return result
}

// GetQueueSize returns the number of pending tasks
func (q *TaskQueue) GetQueueSize() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.queue)
}

// GetStats returns queue statistics
func (q *TaskQueue) GetStats() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return map[string]interface{}{
		"pending":   len(q.tasks),
		"completed": len(q.completed),
		"failed":    len(q.failed),
		"queue_len": len(q.queue),
	}
}

// WaitForTask waits for a task to complete with timeout
func (q *TaskQueue) WaitForTask(ctx context.Context, taskID string, timeout time.Duration) (*Task, error) {
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if time.Now().After(deadline) {
				return nil, errors.New("timeout waiting for task")
			}

			task := q.GetTask(taskID)
			if task != nil && (task.Status == StatusCompleted || task.Status == StatusError) {
				return task, nil
			}

			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Clear removes all tasks (use with caution)
func (q *TaskQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.tasks = make(map[string]*Task)
	q.completed = make(map[string]*Task)
	q.failed = make(map[string]*Task)
	q.queue = make([]*Task, 0, q.maxQueueLen)
}
