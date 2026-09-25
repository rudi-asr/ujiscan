// Package metrics provides HTTP handlers for metrics endpoints
package metrics

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Handler provides HTTP endpoints for metrics and monitoring
type Handler struct {
	collector *Collector
	getDBStats func() (open, idle int, err error) // Optional DB stats callback
}

// NewHandler creates a new metrics handler
func NewHandler(collector *Collector) *Handler {
	return &Handler{
		collector: collector,
	}
}

// SetDatabaseStatsCallback sets optional database stats fetcher
func (h *Handler) SetDatabaseStatsCallback(fn func() (open, idle int, err error)) {
	h.getDBStats = fn
}

// HandleMetrics returns all metrics as JSON
// GET /api/metrics
func (h *Handler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Update system metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	h.collector.UpdateMemoryUsage(int64(m.Alloc))
	h.collector.UpdateGoroutineCount(runtime.NumGoroutine())

	// Get snapshot
	snapshot := h.collector.GetSnapshot()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

// HandleHealth returns health check status
// GET /api/metrics/health
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.collector.GetSnapshot()

	status := "healthy"
	dbConnected := true
	dbPoolHealth := "good"

	// Check error rate
	errorRate := 0.0
	if snapshot.TotalRequests > 0 {
		errorRate = float64(snapshot.ErrorRequests) / float64(snapshot.TotalRequests) * 100
		if errorRate > 5 {
			status = "degraded"
		}
		if errorRate > 10 {
			status = "unhealthy"
		}
	}

	// Check memory
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryHealthy := true
	if m.Alloc > 500*1024*1024 { // >500MB
		memoryHealthy = false
		status = "degraded"
	}

	// Check latency
	avgLatency := snapshot.AvgLatencyMs
	if avgLatency > 500 { // >500ms
		status = "degraded"
	}

	// Check pool health
	if h.getDBStats != nil {
		open, idle, err := h.getDBStats()
		if err != nil {
			dbConnected = false
			dbPoolHealth = "error"
			status = "unhealthy"
		} else {
			poolUtilization := float64(open) / float64(open + idle)
			if poolUtilization > 0.9 {
				dbPoolHealth = "saturated"
				status = "degraded"
			}
		}
	}

	health := HealthStatus{
		Status:             status,
		Timestamp:          time.Now(),
		DatabaseConnected:  dbConnected,
		DatabasePoolHealth: dbPoolHealth,
		MemoryUsageMB:      float64(m.Alloc) / 1024 / 1024,
		MemoryHealthy:      memoryHealthy,
		UptimeSec:          snapshot.UptimeSec,
		ErrorRate:          errorRate,
		AvgLatencyMs:       avgLatency,
	}

	w.Header().Set("Content-Type", "application/json")
	if status == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else if status == "degraded" {
		w.WriteHeader(http.StatusOK) // Still OK but indicated in status
	}
	json.NewEncoder(w).Encode(health)
}

// HandleDatabaseStats returns database pool statistics
// GET /api/metrics/database
func (h *Handler) HandleDatabaseStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.collector.GetSnapshot()

	poolHealth := "good"
	if snapshot.OpenConnections > snapshot.MaxConnections {
		poolHealth = "error"
	} else if float64(snapshot.OpenConnections)/float64(snapshot.MaxConnections) > 0.9 {
		poolHealth = "saturated"
	}

	stats := PoolStats{
		Timestamp:       time.Now(),
		OpenConnections: snapshot.OpenConnections,
		IdleConnections: snapshot.IdleConnections,
		MaxConnections:  snapshot.MaxConnections,
		PoolHealth:      poolHealth,
		UtilizationRate: float64(snapshot.OpenConnections) / float64(snapshot.MaxConnections),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleCacheStats returns cache performance statistics
// GET /api/metrics/cache
func (h *Handler) HandleCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.collector.GetSnapshot()

	stats := CacheStats{
		Timestamp:       time.Now(),
		Hits:            snapshot.CacheHits,
		Misses:          snapshot.CacheMisses,
		HitRate:         snapshot.CacheHitRate,
		Evictions:       snapshot.CacheEvictions,
		TotalOperations: snapshot.CacheHits + snapshot.CacheMisses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleRequestStats returns HTTP request statistics
// GET /api/metrics/requests
func (h *Handler) HandleRequestStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.collector.GetSnapshot()

	// Sort endpoints by request count and take top 10
	topEndpoints := snapshot.EndpointStats
	if len(topEndpoints) > 10 {
		topEndpoints = topEndpoints[:10]
	}

	requestsPerSecond := 0.0
	if snapshot.UptimeSec > 0 {
		requestsPerSecond = float64(snapshot.TotalRequests) / snapshot.UptimeSec
	}

	stats := RequestStats{
		Timestamp:         time.Now(),
		TotalRequests:     snapshot.TotalRequests,
		SuccessRequests:   snapshot.SuccessRequests,
		ErrorRequests:     snapshot.ErrorRequests,
		SuccessRate:       snapshot.SuccessRate,
		AvgLatencyMs:      snapshot.AvgLatencyMs,
		MaxLatencyMs:      snapshot.MaxLatencyMs,
		MinLatencyMs:      snapshot.MinLatencyMs,
		RequestsPerSecond: requestsPerSecond,
		TopEndpoints:      topEndpoints,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleExport exports metrics in various formats
// GET /api/metrics/export?format=json|csv|prometheus
func (h *Handler) HandleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	snapshot := h.collector.GetSnapshot()

	switch format {
	case "csv":
		h.exportCSV(w, snapshot)
	case "prometheus":
		h.exportPrometheus(w, snapshot)
	case "json":
		fallthrough
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snapshot)
	}
}

// exportCSV exports metrics as CSV
func (h *Handler) exportCSV(w http.ResponseWriter, snapshot Snapshot) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=metrics.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{
		"Timestamp", "UptimeSec", "TotalRequests", "SuccessRequests", "ErrorRequests",
		"SuccessRate", "AvgLatencyMs", "MaxLatencyMs", "MinLatencyMs", "QueryCount",
		"AvgQueryTimeMs", "SlowQueryCount", "CacheHits", "CacheMisses", "CacheHitRate",
		"OpenConnections", "IdleConnections", "MaxConnections", "MemoryUsageMB", "GoroutineCount",
	})

	// Write data row
	writer.Write([]string{
		snapshot.Timestamp.String(),
		strconv.FormatFloat(snapshot.UptimeSec, 'f', 2, 64),
		strconv.FormatInt(snapshot.TotalRequests, 10),
		strconv.FormatInt(snapshot.SuccessRequests, 10),
		strconv.FormatInt(snapshot.ErrorRequests, 10),
		strconv.FormatFloat(snapshot.SuccessRate, 'f', 2, 64),
		strconv.FormatFloat(snapshot.AvgLatencyMs, 'f', 2, 64),
		strconv.FormatFloat(snapshot.MaxLatencyMs, 'f', 2, 64),
		strconv.FormatFloat(snapshot.MinLatencyMs, 'f', 2, 64),
		strconv.FormatInt(snapshot.QueryCount, 10),
		strconv.FormatFloat(snapshot.AvgQueryTimeMs, 'f', 2, 64),
		strconv.FormatInt(snapshot.SlowQueryCount, 10),
		strconv.FormatInt(snapshot.CacheHits, 10),
		strconv.FormatInt(snapshot.CacheMisses, 10),
		strconv.FormatFloat(snapshot.CacheHitRate, 'f', 2, 64),
		strconv.FormatInt(snapshot.OpenConnections, 10),
		strconv.FormatInt(snapshot.IdleConnections, 10),
		strconv.FormatInt(snapshot.MaxConnections, 10),
		strconv.FormatFloat(snapshot.MemoryUsageMB, 'f', 2, 64),
		strconv.FormatInt(snapshot.GoroutineCount, 10),
	})
}

// exportPrometheus exports metrics in Prometheus format
func (h *Handler) exportPrometheus(w http.ResponseWriter, snapshot Snapshot) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	lines := []string{
		"# HELP ujiscan_requests_total Total HTTP requests",
		"# TYPE ujiscan_requests_total counter",
		"ujiscan_requests_total " + strconv.FormatInt(snapshot.TotalRequests, 10),
		"",
		"# HELP ujiscan_requests_success Successful HTTP requests",
		"# TYPE ujiscan_requests_success counter",
		"ujiscan_requests_success " + strconv.FormatInt(snapshot.SuccessRequests, 10),
		"",
		"# HELP ujiscan_requests_error Failed HTTP requests",
		"# TYPE ujiscan_requests_error counter",
		"ujiscan_requests_error " + strconv.FormatInt(snapshot.ErrorRequests, 10),
		"",
		"# HELP ujiscan_request_latency_ms Average request latency in milliseconds",
		"# TYPE ujiscan_request_latency_ms gauge",
		"ujiscan_request_latency_ms " + strconv.FormatFloat(snapshot.AvgLatencyMs, 'f', 2, 64),
		"",
		"# HELP ujiscan_cache_hits Cache hits",
		"# TYPE ujiscan_cache_hits counter",
		"ujiscan_cache_hits " + strconv.FormatInt(snapshot.CacheHits, 10),
		"",
		"# HELP ujiscan_cache_hit_rate Cache hit rate percentage",
		"# TYPE ujiscan_cache_hit_rate gauge",
		"ujiscan_cache_hit_rate " + strconv.FormatFloat(snapshot.CacheHitRate, 'f', 2, 64),
		"",
		"# HELP ujiscan_db_connections_open Open database connections",
		"# TYPE ujiscan_db_connections_open gauge",
		"ujiscan_db_connections_open " + strconv.FormatInt(snapshot.OpenConnections, 10),
		"",
		"# HELP ujiscan_memory_usage_mb Memory usage in megabytes",
		"# TYPE ujiscan_memory_usage_mb gauge",
		"ujiscan_memory_usage_mb " + strconv.FormatFloat(snapshot.MemoryUsageMB, 'f', 2, 64),
	}

	w.Write([]byte(strings.Join(lines, "\n")))
}
