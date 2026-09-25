package agent

import (
	"context"
	"testing"
	"time"
)

// MockAgent is a test agent implementation
type MockAgent struct {
	BaseAgent
	executeFunc func(ctx context.Context, task *Task) (interface{}, error)
}

func (m *MockAgent) Execute(ctx context.Context, task *Task) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, task)
	}
	return map[string]interface{}{"result": "success"}, nil
}

func (m *MockAgent) Validate(task *Task) error {
	return nil
}

func (m *MockAgent) Stop() error {
	m.Status = StatusTerminated
	return nil
}

// Test TaskQueue.Enqueue
func TestEnqueue(t *testing.T) {
	q := NewTaskQueue(3, 100)

	task := &Task{
		ID:        "test1",
		AgentType: AgentTypeReconnaissance,
		Priority:  50,
	}

	err := q.Enqueue(task)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	if q.GetQueueSize() != 1 {
		t.Fatalf("Queue size should be 1, got %d", q.GetQueueSize())
	}
}

// Test TaskQueue.Dequeue
func TestDequeue(t *testing.T) {
	q := NewTaskQueue(3, 100)

	task1 := &Task{ID: "test1", AgentType: AgentTypeReconnaissance, Priority: 50}
	task2 := &Task{ID: "test2", AgentType: AgentTypeScanner, Priority: 100}

	q.Enqueue(task1)
	q.Enqueue(task2)

	// Should dequeue task2 first (higher priority)
	dequeued := q.Dequeue()
	if dequeued.ID != "test2" {
		t.Fatalf("Expected task2, got %s", dequeued.ID)
	}

	if dequeued.Status != StatusRunning {
		t.Fatalf("Task status should be Running, got %v", dequeued.Status)
	}
}

// Test TaskQueue.MarkCompleted
func TestMarkCompleted(t *testing.T) {
	q := NewTaskQueue(3, 100)
	task := &Task{ID: "test1", AgentType: AgentTypeReconnaissance}
	q.Enqueue(task)

	dequeued := q.Dequeue()
	result := map[string]interface{}{"hosts": []string{"192.168.1.1"}}

	err := q.MarkCompleted(dequeued.ID, result)
	if err != nil {
		t.Fatalf("MarkCompleted failed: %v", err)
	}

	completed := q.GetTask(dequeued.ID)
	if completed.Status != StatusCompleted {
		t.Fatalf("Task status should be Completed, got %v", completed.Status)
	}
}

// Test TaskQueue.MarkFailed with retry
func TestMarkFailedWithRetry(t *testing.T) {
	q := NewTaskQueue(3, 100)
	task := &Task{ID: "test1", AgentType: AgentTypeReconnaissance, MaxRetries: 3}
	q.Enqueue(task)

	dequeued := q.Dequeue()
	q.MarkFailed(dequeued.ID, "connection timeout")

	// Task should be re-queued for retry
	requeued := q.GetTask(dequeued.ID)
	if requeued.Status != StatusIdle {
		t.Fatalf("Task should be re-queued, status is %v", requeued.Status)
	}

	if requeued.Retries != 1 {
		t.Fatalf("Retries should be 1, got %d", requeued.Retries)
	}
}

// Test Agent Manager registration
func TestManagerRegisterAgent(t *testing.T) {
	m := NewManager(2, 100)

	agent := &MockAgent{
		BaseAgent: BaseAgent{
			AgentType: AgentTypeReconnaissance,
			Status:    StatusIdle,
			Metrics: &Metrics{},
		},
	}

	err := m.RegisterAgent(AgentTypeReconnaissance, agent)
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	retrieved := m.GetAgent(AgentTypeReconnaissance)
	if retrieved == nil {
		t.Fatalf("Agent not found after registration")
	}
}

// Test Agent Manager task submission
func TestManagerSubmitTask(t *testing.T) {
	m := NewManager(2, 100)

	agent := &MockAgent{
		BaseAgent: BaseAgent{
			AgentType: AgentTypeReconnaissance,
			Status:    StatusIdle,
			Metrics: &Metrics{},
		},
	}

	m.RegisterAgent(AgentTypeReconnaissance, agent)
	m.Start(context.Background())
	defer m.Stop()

	task := &Task{
		ID:        "test1",
		AgentType: AgentTypeReconnaissance,
		EngagementID: "eng1",
	}

	err := m.SubmitTask(task)
	if err != nil {
		t.Fatalf("SubmitTask failed: %v", err)
	}
}

// Test Agent Manager start/stop
func TestManagerStartStop(t *testing.T) {
	m := NewManager(2, 100)

	if m.IsRunning() {
		t.Fatalf("Manager should not be running initially")
	}

	ctx := context.Background()
	err := m.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !m.IsRunning() {
		t.Fatalf("Manager should be running")
	}

	err = m.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if m.IsRunning() {
		t.Fatalf("Manager should be stopped")
	}
}

// Test task execution through manager
func TestManagerExecution(t *testing.T) {
	m := NewManager(2, 100)

	agent := &MockAgent{
		BaseAgent: BaseAgent{
			AgentType: AgentTypeReconnaissance,
			Status:    StatusIdle,
			Metrics: &Metrics{},
		},
		executeFunc: func(ctx context.Context, task *Task) (interface{}, error) {
			return map[string]interface{}{"hosts": []string{"192.168.1.1"}}, nil
		},
	}

	m.RegisterAgent(AgentTypeReconnaissance, agent)
	m.Start(context.Background())
	defer m.Stop()

	task := &Task{
		ID:        "test1",
		AgentType: AgentTypeReconnaissance,
		EngagementID: "eng1",
	}

	m.SubmitTask(task)

	// Wait for task completion
	completed, err := m.WaitForTask(context.Background(), "test1", 5*time.Second)
	if err != nil {
		t.Fatalf("WaitForTask failed: %v", err)
	}

	if completed.Status != StatusCompleted {
		t.Fatalf("Task should be completed, got %v", completed.Status)
	}
}
