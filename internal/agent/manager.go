package agent

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Manager manages all agents and the task queue
type Manager struct {
	mu       sync.RWMutex
	agents   map[AgentType]Agent
	queue    *TaskQueue
	workers  int
	running  bool
	stopChan chan struct{}
	workerWg sync.WaitGroup
}

// NewManager creates a new agent manager
func NewManager(maxWorkers int, maxQueueSize int) *Manager {
	if maxWorkers < 1 {
		maxWorkers = 4
	}

	return &Manager{
		agents:   make(map[AgentType]Agent),
		queue:    NewTaskQueue(3, maxQueueSize),
		workers:  maxWorkers,
		stopChan: make(chan struct{}),
	}
}

// RegisterAgent registers an agent
func (m *Manager) RegisterAgent(agentType AgentType, agent Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if agent == nil {
		return errors.New("agent cannot be nil")
	}

	m.agents[agentType] = agent
	return nil
}

// GetAgent retrieves an agent by type
func (m *Manager) GetAgent(agentType AgentType) Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents[agentType]
}

// SubmitTask adds a task to the queue
func (m *Manager) SubmitTask(task *Task) error {
	if !m.IsRunning() {
		return errors.New("agent manager not running")
	}

	if task.AgentType == "" {
		return errors.New("agent type required")
	}

	agent := m.GetAgent(task.AgentType)
	if agent == nil {
		return errors.New("agent type not registered: " + string(task.AgentType))
	}

	if err := agent.Validate(task); err != nil {
		return err
	}

	return m.queue.Enqueue(task)
}

// Start starts the agent manager and worker goroutines
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return errors.New("agent manager already running")
	}
	m.running = true
	m.mu.Unlock()

	// Start worker goroutines
	for i := 0; i < m.workers; i++ {
		m.workerWg.Add(1)
		go m.worker(ctx)
	}

	return nil
}

// Stop stops the agent manager and all workers
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return errors.New("agent manager not running")
	}
	m.running = false
	m.mu.Unlock()

	close(m.stopChan)
	m.workerWg.Wait()

	// Stop all agents
	m.mu.RLock()
	agents := make([]Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	m.mu.RUnlock()

	for _, agent := range agents {
		agent.Stop()
	}

	return nil
}

// IsRunning returns whether the manager is running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// worker processes tasks from the queue
func (m *Manager) worker(ctx context.Context) {
	defer m.workerWg.Done()

	for {
		select {
		case <-m.stopChan:
			return
		default:
			// Dequeue next task
			task := m.queue.Dequeue()
			if task == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Get the agent for this task
			agent := m.GetAgent(task.AgentType)
			if agent == nil {
				m.queue.MarkFailed(task.ID, "agent not found: "+string(task.AgentType))
				continue
			}

			// Create timeout context if needed
			execCtx := ctx
			if task.TimeoutSeconds > 0 {
				var cancel context.CancelFunc
				execCtx, cancel = context.WithTimeout(ctx, time.Duration(task.TimeoutSeconds)*time.Second)
				defer cancel()
			}

			// Execute the task
			result, err := agent.Execute(execCtx, task)
			if err != nil {
				m.queue.MarkFailed(task.ID, err.Error())
			} else {
				m.queue.MarkCompleted(task.ID, result)
			}
		}
	}
}

// GetStatus returns the status of all agents and queue
func (m *Manager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	agents := make([]map[string]interface{}, 0)
	for agentType, agent := range m.agents {
		metrics := agent.GetMetrics()
		agents = append(agents, map[string]interface{}{
			"type":    agentType,
			"status":  agent.GetStatus(),
			"metrics": metrics,
		})
	}
	m.mu.RUnlock()

	return map[string]interface{}{
		"running": m.IsRunning(),
		"workers": m.workers,
		"queue":   m.queue.GetStats(),
		"agents":  agents,
	}
}

// GetTask retrieves a task by ID
func (m *Manager) GetTask(taskID string) *Task {
	return m.queue.GetTask(taskID)
}

// GetTasks retrieves all tasks, optionally filtered by status
func (m *Manager) GetTasks(status ...AgentStatus) []*Task {
	return m.queue.GetTasks(status...)
}

// WaitForTask waits for a task to complete
func (m *Manager) WaitForTask(ctx context.Context, taskID string, timeout time.Duration) (*Task, error) {
	return m.queue.WaitForTask(ctx, taskID, timeout)
}

// CancelTask attempts to cancel a task (if not yet started)
func (m *Manager) CancelTask(taskID string) error {
	task := m.queue.GetTask(taskID)
	if task == nil {
		return errors.New("task not found")
	}

	if task.Status != StatusIdle {
		return errors.New("can only cancel idle tasks")
	}

	task.Status = StatusTerminated
	return nil
}
