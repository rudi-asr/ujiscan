// Package metrics defines data structures for monitoring
package metrics

import "time"

// Snapshot represents a point-in-time capture of all metrics
type Snapshot struct {
	Timestamp         time.Time            `json:"timestamp"`
	UptimeSec         float64              `json:"uptime_sec"`
	TotalRequests     int64                `json:"total_requests"`
	SuccessRequests   int64                `json:"success_requests"`
	ErrorRequests     int64                `json:"error_requests"`
	SuccessRate       float64              `json:"success_rate"`
	AvgLatencyMs      float64              `json:"avg_latency_ms"`
	MaxLatencyMs      float64              `json:"max_latency_ms"`
	MinLatencyMs      float64              `json:"min_latency_ms"`
	QueryCount        int64                `json:"query_count"`
	AvgQueryTimeMs    float64              `json:"avg_query_time_ms"`
	SlowQueryCount    int64                `json:"slow_query_count"`
	CacheHits         int64                `json:"cache_hits"`
	CacheMisses       int64                `json:"cache_misses"`
	CacheHitRate      float64              `json:"cache_hit_rate"`
	CacheEvictions    int64                `json:"cache_evictions"`
	OpenConnections   int64                `json:"open_connections"`
	IdleConnections   int64                `json:"idle_connections"`
	MaxConnections    int64                `json:"max_connections"`
	MemoryUsageMB     float64              `json:"memory_usage_mb"`
	GoroutineCount    int64                `json:"goroutine_count"`
	EndpointStats     []EndpointSnapshot   `json:"endpoint_stats"`
}

// EndpointSnapshot represents metrics for a single API endpoint
type EndpointSnapshot struct {
	Path           string    `json:"path"`
	Method         string    `json:"method"`
	RequestCount   int64     `json:"request_count"`
	SuccessCount   int64     `json:"success_count"`
	ErrorCount     int64     `json:"error_count"`
	AvgLatencyMs   float64   `json:"avg_latency_ms"`
	MaxLatencyMs   float64   `json:"max_latency_ms"`
	MinLatencyMs   float64   `json:"min_latency_ms"`
	LastAccessTime time.Time `json:"last_access_time"`
}

// HealthStatus represents the health status of the system
type HealthStatus struct {
	Status             string  `json:"status"`
	Timestamp          time.Time `json:"timestamp"`
	DatabaseConnected  bool    `json:"database_connected"`
	DatabasePoolHealth string  `json:"database_pool_health"`
	MemoryUsageMB      float64 `json:"memory_usage_mb"`
	MemoryHealthy      bool    `json:"memory_healthy"`
	UptimeSec          float64 `json:"uptime_sec"`
	ErrorRate          float64 `json:"error_rate"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
}

// PoolStats represents database connection pool statistics
type PoolStats struct {
	Timestamp       time.Time `json:"timestamp"`
	OpenConnections int64     `json:"open_connections"`
	IdleConnections int64     `json:"idle_connections"`
	MaxConnections  int64     `json:"max_connections"`
	PoolHealth      string    `json:"pool_health"`
	UtilizationRate float64   `json:"utilization_rate"`
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	Timestamp      time.Time `json:"timestamp"`
	Hits           int64     `json:"hits"`
	Misses         int64     `json:"misses"`
	HitRate        float64   `json:"hit_rate"`
	Evictions      int64     `json:"evictions"`
	TotalOperations int64    `json:"total_operations"`
}

// RequestStats represents HTTP request statistics
type RequestStats struct {
	Timestamp         time.Time          `json:"timestamp"`
	TotalRequests     int64              `json:"total_requests"`
	SuccessRequests   int64              `json:"success_requests"`
	ErrorRequests     int64              `json:"error_requests"`
	SuccessRate       float64            `json:"success_rate"`
	AvgLatencyMs      float64            `json:"avg_latency_ms"`
	MaxLatencyMs      float64            `json:"max_latency_ms"`
	MinLatencyMs      float64            `json:"min_latency_ms"`
	RequestsPerSecond float64            `json:"requests_per_second"`
	TopEndpoints      []EndpointSnapshot `json:"top_endpoints"`
}
