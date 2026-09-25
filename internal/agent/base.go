package agent

import (
	"strconv"
	"time"
)

// parseInt parses an integer from a string, returning an error on failure.
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// NewBaseAgent initializes a BaseAgent with ready-to-use metrics.
func NewBaseAgent(agentType AgentType) BaseAgent {
	return BaseAgent{
		AgentType: agentType,
		Status:    StatusIdle,
		Metrics:   &Metrics{},
	}
}

// recordSuccess updates metrics after a successful execution.
func (b *BaseAgent) recordSuccess(start time.Time) {
	if b.Metrics == nil {
		b.Metrics = &Metrics{}
	}
	b.Status = StatusCompleted
	b.Metrics.TasksCompleted++
	ms := time.Since(start).Milliseconds()
	b.Metrics.TotalExecutionMS += ms
	b.Metrics.LastExecutionMS = ms
	if total := b.Metrics.TasksCompleted + b.Metrics.TasksFailed; total > 0 {
		b.Metrics.SuccessRate = float64(b.Metrics.TasksCompleted) / float64(total)
	}
	b.Metrics.AverageExecutionMS = b.Metrics.TotalExecutionMS / int64(b.Metrics.TasksCompleted)
	b.Metrics.LastUpdated = time.Now().Unix()
}

// recordFailure updates metrics after a failed execution.
func (b *BaseAgent) recordFailure(start time.Time) {
	if b.Metrics == nil {
		b.Metrics = &Metrics{}
	}
	b.Status = StatusError
	b.Metrics.TasksFailed++
	ms := time.Since(start).Milliseconds()
	b.Metrics.TotalExecutionMS += ms
	b.Metrics.LastExecutionMS = ms
	if total := b.Metrics.TasksCompleted + b.Metrics.TasksFailed; total > 0 {
		b.Metrics.SuccessRate = float64(b.Metrics.TasksCompleted) / float64(total)
	}
	b.Metrics.LastUpdated = time.Now().Unix()
}

// getStringParam returns a string parameter from task params.
func getStringParam(params map[string]interface{}, key string) string {
	if params == nil {
		return ""
	}
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

// getIntParam returns an int parameter from task params (accepts float64 from JSON decoding).
func getIntParam(params map[string]interface{}, key string, def int) int {
	if params == nil {
		return def
	}
	switch v := params[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

// getBoolParam returns a bool parameter from task params.
func getBoolParam(params map[string]interface{}, key string, def bool) bool {
	if params == nil {
		return def
	}
	if v, ok := params[key].(bool); ok {
		return v
	}
	return def
}
