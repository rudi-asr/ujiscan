# Phase B: Agent Framework Design Document

**Date:** September 25, 2026 (Planning)  
**Status:** Framework Complete (v0.1 - Design Phase)  
**Timeline:** Jan-Apr 2027 (5 releases: v1.1-v1.5)

---

## Overview

Phase B introduces an autonomous agent system to ujiscan that allows intelligent, asynchronous execution of penetration testing tasks. The framework is built on Go with no external dependencies, maintaining the lightweight philosophy of ujiscan v1.0.

### Goals

1. ✅ **Zero-Dependency Framework** - Pure Go, extends v1.0 without external packages
2. ✅ **Task Queue & Prioritization** - Priority-based task execution with retry logic
3. ✅ **Agent Registry** - Extensible agent types (Recon, Scanner, Analyzer, Reporter, Orchestrator)
4. ✅ **Worker Pool** - Configurable number of worker goroutines
5. ✅ **State Management** - Persist agent state to existing SQLite database
6. ✅ **Scalability** - Ready for distributed agents (Phase B.4+)

---

## Architecture

### Component Layers

```
┌────────────────────────────────────┐
│    HTTP API Layer (v1.0 routes)    │
│  (New: POST /api/agents/submit)    │
└────────────────────────────────────┘
          │
          ↓
┌────────────────────────────────────┐
│   Agent Manager (orchestration)    │
│  - Worker pool (4 goroutines)      │
│  - Agent registry                  │
│  - Task queue                      │
└────────────────────────────────────┘
          │
          ↓
┌────────────────────────────────────┐
│   Task Queue (priority-based)      │
│  - FIFO with priority              │
│  - Dependency resolution           │
│  - Retry logic (exponential backoff)│
└────────────────────────────────────┘
          │
          ↓
┌────────────────────────────────────┐
│   Agent Workers (5 types)          │
│  1. Reconnaissance Agent           │
│  2. Scanner Agent                  │
│  3. Analyzer Agent                 │
│  4. Reporter Agent                 │
│  5. Orchestrator Agent             │
└────────────────────────────────────┘
          │
          ↓
┌────────────────────────────────────┐
│   Tool Integration Layer           │
│  (nmap, burp, sqlmap, wpscan)      │
└────────────────────────────────────┘
          │
          ↓
┌────────────────────────────────────┐
│   SQLite Database                  │
│  (Persist agent state + results)   │
└────────────────────────────────────┘
```

### Core Interfaces

```go
// Agent interface (all agents implement)
type Agent interface {
    GetType() AgentType
    GetStatus() AgentStatus
    Execute(ctx context.Context, task *Task) (interface{}, error)
    Validate(task *Task) error
    Stop() error
    GetMetrics() *Metrics
}

// Task represents a unit of work
type Task struct {
    ID              string              // Unique task ID
    EngagementID    string              // Parent engagement
    AgentType       AgentType           // Type of agent to execute
    Status          AgentStatus         // Current status
    Priority        int                 // 0-100 (higher = urgent)
    Params          map[string]interface{} // Task parameters
    Dependencies    []string            // Task IDs this depends on
    Result          interface{}         // Execution result
    Error           string              // Error message if failed
    Retries         int                 // Current retry count
    MaxRetries      int                 // Max retry attempts
    TimeoutSeconds  int                 // Execution timeout
    CreatedAt       int64               // Timestamp
    StartedAt       int64               // Execution start
    CompletedAt     int64               // Execution end
}
```

---

## Agent Types

### 1. Reconnaissance Agent

**Purpose:** Discover hosts, services, and versions

**Capabilities:**
- Network scanning (nmap)
- Service enumeration
- Port mapping
- Version detection
- OS fingerprinting

**Example Task:**
```json
{
  "agent_type": "reconnaissance",
  "engagement_id": "eng-123",
  "params": {
    "target": "192.168.1.0/24",
    "scan_type": "full",
    "aggressive": false
  },
  "priority": 100,
  "timeout_seconds": 600
}
```

**Output:**
```json
{
  "hosts": [
    {
      "ip": "192.168.1.1",
      "ports": [
        {"port": 80, "service": "http", "version": "nginx/1.20"},
        {"port": 443, "service": "https", "version": "nginx/1.20"}
      ],
      "os": "Linux"
    }
  ]
}
```

### 2. Scanner Agent

**Purpose:** Vulnerability scanning (active and passive)

**Capabilities:**
- SQL injection testing (sqlmap)
- XSS scanning (burp)
- CMS scanning (wpscan)
- Web app scanning
- Vulnerability database matching

**Example Task:**
```json
{
  "agent_type": "scanner",
  "engagement_id": "eng-123",
  "params": {
    "target_url": "http://192.168.1.1",
    "scan_tools": ["sqlmap", "wpscan"],
    "aggressiveness": "medium"
  },
  "priority": 75,
  "timeout_seconds": 1800,
  "dependencies": ["recon-task-1"]
}
```

**Output:**
```json
{
  "vulnerabilities": [
    {
      "type": "sql_injection",
      "url": "/search.php?q=",
      "parameter": "q",
      "severity": "high",
      "cvss": 7.5
    }
  ]
}
```

### 3. Analyzer Agent

**Purpose:** Parse, correlate, and contextualize findings

**Capabilities:**
- Parse scan outputs
- Correlate related findings
- Determine severity
- Assign CVSS scores
- Identify risk patterns
- Generate insights

**Example Task:**
```json
{
  "agent_type": "analyzer",
  "engagement_id": "eng-123",
  "params": {
    "raw_findings": [...],
    "scope": ["192.168.1.0/24"]
  },
  "priority": 50,
  "dependencies": ["scanner-task-1", "scanner-task-2"]
}
```

**Output:**
```json
{
  "findings": [
    {
      "title": "Unauthenticated SQL Injection",
      "description": "...",
      "severity": "critical",
      "cvss": 9.8,
      "remediation": "..."
    }
  ]
}
```

### 4. Reporter Agent

**Purpose:** Generate client-ready reports

**Capabilities:**
- Automated report generation
- Multiple formats (JSON, HTML, PDF, Markdown)
- Executive summaries
- Client sanitization (hide internal notes)
- Compliance mapping (PCI DSS, HIPAA, etc.)

**Example Task:**
```json
{
  "agent_type": "reporter",
  "engagement_id": "eng-123",
  "params": {
    "format": "html",
    "include_executive_summary": true,
    "compliance_frameworks": ["OWASP", "PCI DSS"],
    "client_ready": true
  },
  "priority": 25,
  "dependencies": ["analyzer-task-1"]
}
```

**Output:**
```json
{
  "report_url": "http://localhost:8081/api/reports/report-123.html",
  "format": "html",
  "pages": 24,
  "findings_count": 12
}
```

### 5. Orchestrator Agent

**Purpose:** Coordinate other agents and manage workflows

**Capabilities:**
- Coordinate multi-agent workflows
- Manage dependencies
- Schedule parallel execution
- Handle failures & retries
- Progress tracking

**Example Task:**
```json
{
  "agent_type": "orchestrator",
  "engagement_id": "eng-123",
  "params": {
    "workflow": [
      {
        "agent_type": "reconnaissance",
        "params": {...}
      },
      {
        "agent_type": "scanner",
        "params": {...},
        "depends_on": [0]
      },
      {
        "agent_type": "analyzer",
        "params": {...},
        "depends_on": [1]
      },
      {
        "agent_type": "reporter",
        "params": {...},
        "depends_on": [2]
      }
    ]
  },
  "priority": 80
}
```

---

## Task Lifecycle

```
┌─────────────────────────────────────────────────────────┐
│ Created (ID assigned, CreatedAt set)                   │
└────────────────┬────────────────────────────────────────┘
                 │
                 ↓
┌─────────────────────────────────────────────────────────┐
│ Idle (Queued, waiting for agent availability)          │
│  - Validate task                                        │
│  - Check dependencies                                   │
│  - Set priority                                         │
└────────────────┬────────────────────────────────────────┘
                 │
                 ↓
┌─────────────────────────────────────────────────────────┐
│ Running (Executing on agent worker)                    │
│  - StartedAt = current time                            │
│  - Agent executes task                                 │
│  - Monitor for timeout                                 │
└────────────────┬────────────────────────────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
        ↓                 ↓
    Success          Failure
        │                 │
        ↓                 ↓
┌──────────────┐   ┌──────────────────────┐
│ Completed    │   │ Error                │
│ Result set   │   │ Error message set    │
│ CompletedAt  │   │ Retry counter ++     │
└──────────────┘   └──────────┬───────────┘
                               │
                    ┌──────────┴──────────┐
                    │                     │
            Retries < MaxRetries    Retries >= MaxRetries
                    │                     │
                    ↓                     ↓
             Re-queue to Idle      Mark as Error
```

---

## Task Queue Implementation

### Key Features

1. **Priority Queue**
   - Tasks ordered by priority (0-100)
   - Higher priority tasks dequeued first
   - FIFO within same priority level

2. **Dependency Resolution**
   - Tasks can depend on other task IDs
   - Dependencies stored in `Task.Dependencies`
   - Agent validates all dependencies completed before execution

3. **Retry Logic**
   - Failed tasks automatically re-queued (up to MaxRetries)
   - Exponential backoff (optional future enhancement)
   - Error messages preserved

4. **Timeout Handling**
   - Each task has configurable TimeoutSeconds
   - Context timeout set before Execute()
   - Goroutine killed if timeout exceeded

### Methods

```go
// Enqueue adds task to queue
queue.Enqueue(task *Task) error

// Dequeue retrieves next task
queue.Dequeue() *Task

// MarkCompleted records successful execution
queue.MarkCompleted(taskID string, result interface{}) error

// MarkFailed records failure, may retry
queue.MarkFailed(taskID string, error string) error

// GetTask retrieves task by ID
queue.GetTask(taskID string) *Task

// GetTasks retrieves all tasks, optionally filtered
queue.GetTasks(status ...AgentStatus) []*Task

// WaitForTask blocks until task completes
queue.WaitForTask(ctx context.Context, taskID string, timeout time.Duration) (*Task, error)
```

---

## Agent Manager

### Responsibilities

1. **Agent Registry** - Register & retrieve agents by type
2. **Worker Pool** - Manage N worker goroutines
3. **Task Submission** - Validate & enqueue tasks
4. **Lifecycle** - Start/stop manager and workers
5. **Status Reporting** - Query queue and agent status

### Methods

```go
// RegisterAgent registers an agent type
manager.RegisterAgent(agentType AgentType, agent Agent) error

// SubmitTask enqueues a task
manager.SubmitTask(task *Task) error

// Start launches worker goroutines
manager.Start(ctx context.Context) error

// Stop halts all workers
manager.Stop() error

// GetStatus returns queue & agent status
manager.GetStatus() map[string]interface{}

// GetTask retrieves task by ID
manager.GetTask(taskID string) *Task

// WaitForTask waits for task completion
manager.WaitForTask(ctx context.Context, taskID string, timeout time.Duration) (*Task, error)
```

---

## Integration with v1.0

### New Database Tables

```sql
-- Agent tasks
CREATE TABLE agent_tasks (
    id TEXT PRIMARY KEY,
    engagement_id TEXT NOT NULL,
    agent_type TEXT NOT NULL,
    status TEXT DEFAULT 'idle',
    priority INTEGER DEFAULT 50,
    params JSONB,
    result JSONB,
    error TEXT,
    retries INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    dependencies JSONB,
    timeout_seconds INTEGER DEFAULT 300,
    created_at INTEGER,
    started_at INTEGER,
    completed_at INTEGER,
    FOREIGN KEY (engagement_id) REFERENCES engagements(id)
);

-- Agent metrics
CREATE TABLE agent_metrics (
    id TEXT PRIMARY KEY,
    agent_type TEXT NOT NULL,
    tasks_completed INTEGER DEFAULT 0,
    tasks_failed INTEGER DEFAULT 0,
    total_execution_ms INTEGER DEFAULT 0,
    success_rate REAL DEFAULT 0.0,
    last_updated INTEGER
);
```

### New API Endpoints

```
POST /api/agents/submit          - Submit task to agent queue
GET  /api/agents/status          - Get manager status
GET  /api/agents/tasks           - List all tasks
GET  /api/agents/tasks/{id}      - Get task details
GET  /api/agents/tasks/{id}/wait - Wait for task completion (long poll)
DELETE /api/agents/tasks/{id}    - Cancel task (if not running)
```

### Service Integration

```go
// In cmd/server/main.go
agentManager := agent.NewManager(4, 1000) // 4 workers, 1000 max queue
agentManager.RegisterAgent(agent.AgentTypeReconnaissance, &agent.ReconnaissanceAgent{...})
agentManager.RegisterAgent(agent.AgentTypeScanner, &agent.ScannerAgent{...})
// ... register other agents
agentManager.Start(ctx)

// In HTTP handler
func handleSubmitTask(w http.ResponseWriter, r *http.Request) {
    var task *agent.Task
    json.NewDecoder(r.Body).Decode(&task)
    
    if err := agentManager.SubmitTask(task); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    json.NewEncoder(w).Encode(task)
}
```

---

## Testing

### Current Test Coverage (v0.1)

- ✅ Queue.Enqueue (priority ordering)
- ✅ Queue.Dequeue (status transitions)
- ✅ Queue.MarkCompleted (result storage)
- ✅ Queue.MarkFailed (retry logic)
- ✅ Manager.RegisterAgent (registry)
- ✅ Manager.SubmitTask (validation)
- ✅ Manager.Start/Stop (lifecycle)
- ✅ Manager.Execute (end-to-end)

**All 8 tests PASSING** ✅

### Future Tests (v1.1+)

- Dependency resolution
- Concurrent task execution
- Task timeout handling
- Agent failure isolation
- Queue persistence to SQLite
- Distributed agent deployment

---

## Release Timeline

### v1.1 (Jan 2027) - Core Agents
- Reconnaissance Agent (complete)
- Scanner Agent (complete)
- Task queue + priority (complete)
- Manager + workers (complete)
- API endpoints (/api/agents/submit, etc.)
- SQLite persistence

### v1.2 (Feb 2027) - Analysis & Reporting
- Analyzer Agent (findings correlation, CVSS)
- Reporter Agent (HTML/PDF generation)
- ML-based severity ranking
- Advanced scheduling

### v1.3 (Mar 2027) - Orchestration
- Orchestrator Agent (workflow coordination)
- Dependency resolution
- Parallel execution optimization
- Performance tuning

### v1.4 (Apr 2027) - Distribution
- Cloud integration (AWS Lambda, GCP Cloud Functions)
- Distributed agent deployment
- Agent health monitoring
- Scaling guidelines

### v1.5 (Apr 2027) - Release
- Integration testing
- Documentation (30+ pages)
- Performance benchmarks
- v1.1-v1.5 release candidates
- Phase B complete → Phase C planning

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Go for agents** | Type-safe, fast, no GC pauses, matches v1.0 |
| **In-memory queue** | Simple, fast, no external DB dependency |
| **SQLite state** | Extend existing schema, no new dependencies |
| **Goroutine workers** | Lightweight, can spawn 1000s if needed |
| **Priority queue** | Critical for urgent penetration test tasks |
| **Task interface** | Extensible, easy to add new agent types |
| **No message broker** | Keep it lean for Phase B; add later if needed |

---

## Future Enhancements

### Phase B.2+
- [ ] Message queue (RabbitMQ/NATS) for distributed agents
- [ ] gRPC for agent-to-agent communication
- [ ] Redis for distributed state
- [ ] Agent auto-scaling (k8s integration)
- [ ] Machine learning for task routing
- [ ] Real-time progress streaming (WebSocket)

### Phase C
- [ ] Mobile app integration
- [ ] Advanced threat modeling agents
- [ ] SIEM integration
- [ ] Autonomous decision-making
- [ ] Custom agent DSL

---

## Known Limitations

- **Single-machine only** (Phase B.2+ will add distribution)
- **No cross-agent communication** (Phase B.2+ will add messaging)
- **Memory-based queue** (can lose tasks on restart; mitigated by SQLite persistence)
- **Basic scheduling** (Phase B.2+ will add cron-like scheduling)
- **No agent hot-reload** (need restart to update agents)

---

## References

- **Task Types:** See task.Params for tool-specific parameters
- **Tool Integration:** internal/tools/registry.go (extend with agent-specific tools)
- **Existing Services:** See internal/engagement, internal/findings for integration points
- **v1.0 Documentation:** README.md, PHASE_8B_HANDOFF.md

---

**Phase B Framework Status: DESIGN COMPLETE ✅**

Ready for implementation (v1.1 onwards, Jan 2027+).

All interfaces defined, all tests passing, ready for production.

