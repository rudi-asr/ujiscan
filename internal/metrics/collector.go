// Package metrics provides performance monitoring and metrics collection
package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Collector tracks performance metrics in real-time
type Collector struct {
	mu sync.RWMutex

	// Request metrics
	TotalRequests   atomic.Int64
	SuccessRequests atomic.Int64
	ErrorRequests   atomic.Int64

	// Latency tracking (microseconds)
	TotalLatency   atomic.Int64 // sum of all request latencies
	MaxLatency     atomic.Int64
	MinLatency     atomic.Int64

	// Database metrics
	QueryCount     atomic.Int64
	QueryTime      atomic.Int64 // microseconds
	QueryErrors    atomic.Int64
	SlowQueryCount atomic.Int64 // queries > 100ms

	// Cache metrics (if implemented)
	CacheHits      atomic.Int64
	CacheMisses    atomic.Int64
	CacheEvictions atomic.Int64

	// Per-endpoint tracking
	EndpointStats map[string]*EndpointStat

	// System metrics
	StartTime      time.Time
	LastResetTime  time.Time
	MemoryUsage    atomic.Int64 // bytes
	GoroutineCount atomic.Int64

	// Connection pool metrics
	OpenConnections atomic.Int64
	IdleConnections atomic.Int64
	MaxConnections  int
}

// EndpointStat tracks statistics per API endpoint
type EndpointStat struct {
	Path           string
	Method         string
	RequestCount   atomic.Int64
	SuccessCount   atomic.Int64
	ErrorCount     atomic.Int64
	TotalLatency   atomic.Int64 // microseconds
	MaxLatency     atomic.Int64
	MinLatency     atomic.Int64
	LastAccessTime atomic.Value // time.Time
}

// New creates a new metrics collector
func New(maxConnections int) *Collector {
	c := &Collector{
		EndpointStats:  make(map[string]*EndpointStat),
		StartTime:      time.Now(),
		LastResetTime:  time.Now(),
		MaxConnections: maxConnections,
	}

	// Initialize min latency to max int64 for proper comparison
	c.MinLatency.Store(^int64(0) >> 1) // max int64

	return c
}

// RecordRequest records a successful API request
func (c *Collector) RecordRequest(path, method string, latency time.Duration) {
	latencyMicros := int64(latency.Microseconds())

	c.TotalRequests.Add(1)
	c.SuccessRequests.Add(1)
	c.TotalLatency.Add(latencyMicros)

	// Update max/min latency
	for {
		maxLatency := c.MaxLatency.Load()
		if latencyMicros <= maxLatency || c.MaxLatency.CompareAndSwap(maxLatency, latencyMicros) {
			break
		}
	}

	for {
		minLatency := c.MinLatency.Load()
		if latencyMicros >= minLatency || c.MinLatency.CompareAndSwap(minLatency, latencyMicros) {
			break
		}
	}

	// Record endpoint stats
	c.recordEndpointStat(path, method, latency, true)
}

// RecordError records a failed API request
func (c *Collector) RecordError(path, method string, latency time.Duration) {
	c.TotalRequests.Add(1)
	c.ErrorRequests.Add(1)

	latencyMicros := int64(latency.Microseconds())
	c.TotalLatency.Add(latencyMicros)

	c.recordEndpointStat(path, method, latency, false)
}

// RecordQuery records a database query execution
func (c *Collector) RecordQuery(latency time.Duration, isError bool) {
	latencyMicros := int64(latency.Microseconds())

	c.QueryCount.Add(1)
	c.QueryTime.Add(latencyMicros)

	if isError {
		c.QueryErrors.Add(1)
	}

	// Track slow queries (>100ms)
	if latency > 100*time.Millisecond {
		c.SlowQueryCount.Add(1)
	}
}

// RecordCacheHit records a cache hit
func (c *Collector) RecordCacheHit() {
	c.CacheHits.Add(1)
}

// RecordCacheMiss records a cache miss
func (c *Collector) RecordCacheMiss() {
	c.CacheMisses.Add(1)
}

// RecordCacheEviction records a cache eviction
func (c *Collector) RecordCacheEviction() {
	c.CacheEvictions.Add(1)
}

// recordEndpointStat updates per-endpoint statistics
func (c *Collector) recordEndpointStat(path, method string, latency time.Duration, success bool) {
	key := method + " " + path

	c.mu.Lock()
	stat, exists := c.EndpointStats[key]
	if !exists {
		stat = &EndpointStat{
			Path:   path,
			Method: method,
		}
		// Initialize min latency
		stat.MinLatency.Store(^int64(0) >> 1)
		c.EndpointStats[key] = stat
	}
	c.mu.Unlock()

	latencyMicros := int64(latency.Microseconds())

	stat.RequestCount.Add(1)
	if success {
		stat.SuccessCount.Add(1)
	} else {
		stat.ErrorCount.Add(1)
	}

	stat.TotalLatency.Add(latencyMicros)

	// Update max latency
	for {
		maxLatency := stat.MaxLatency.Load()
		if latencyMicros <= maxLatency || stat.MaxLatency.CompareAndSwap(maxLatency, latencyMicros) {
			break
		}
	}

	// Update min latency
	for {
		minLatency := stat.MinLatency.Load()
		if latencyMicros >= minLatency || stat.MinLatency.CompareAndSwap(minLatency, latencyMicros) {
			break
		}
	}

	stat.LastAccessTime.Store(time.Now())
}

// UpdateConnectionPoolStats updates database connection pool statistics
func (c *Collector) UpdateConnectionPoolStats(open, idle int) {
	c.OpenConnections.Store(int64(open))
	c.IdleConnections.Store(int64(idle))
}

// UpdateMemoryUsage updates current memory usage
func (c *Collector) UpdateMemoryUsage(bytes int64) {
	c.MemoryUsage.Store(bytes)
}

// UpdateGoroutineCount updates goroutine count
func (c *Collector) UpdateGoroutineCount(count int) {
	c.GoroutineCount.Store(int64(count))
}

// GetSnapshot returns a point-in-time snapshot of all metrics
func (c *Collector) GetSnapshot() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalRequests := c.TotalRequests.Load()
	successRequests := c.SuccessRequests.Load()
	errorRequests := c.ErrorRequests.Load()
	totalLatency := c.TotalLatency.Load()

	var avgLatency int64
	if totalRequests > 0 {
		avgLatency = totalLatency / totalRequests
	}

	queryCount := c.QueryCount.Load()
	queryTime := c.QueryTime.Load()
	var avgQueryTime int64
	if queryCount > 0 {
		avgQueryTime = queryTime / queryCount
	}

	cacheHits := c.CacheHits.Load()
	cacheMisses := c.CacheMisses.Load()
	var cacheHitRate float64
	if (cacheHits + cacheMisses) > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
	}

	// Copy endpoint stats
	endpointSnapshots := make([]EndpointSnapshot, 0, len(c.EndpointStats))
	for _, stat := range c.EndpointStats {
		requests := stat.RequestCount.Load()
		var avgLatency int64
		if requests > 0 {
			avgLatency = stat.TotalLatency.Load() / requests
		}

		lastAccess := stat.LastAccessTime.Load()
		var lastAccessTime time.Time
		if lastAccess != nil {
			lastAccessTime = lastAccess.(time.Time)
		}

		endpointSnapshots = append(endpointSnapshots, EndpointSnapshot{
			Path:           stat.Path,
			Method:         stat.Method,
			RequestCount:   requests,
			SuccessCount:   stat.SuccessCount.Load(),
			ErrorCount:     stat.ErrorCount.Load(),
			AvgLatencyMs:   float64(avgLatency) / 1000,
			MaxLatencyMs:   float64(stat.MaxLatency.Load()) / 1000,
			MinLatencyMs:   float64(stat.MinLatency.Load()) / 1000,
			LastAccessTime: lastAccessTime,
		})
	}

	uptime := time.Since(c.StartTime)

	return Snapshot{
		Timestamp:         time.Now(),
		UptimeSec:         uptime.Seconds(),
		TotalRequests:     totalRequests,
		SuccessRequests:   successRequests,
		ErrorRequests:     errorRequests,
		SuccessRate:       float64(successRequests) / float64(totalRequests) * 100,
		AvgLatencyMs:      float64(avgLatency) / 1000,
		MaxLatencyMs:      float64(c.MaxLatency.Load()) / 1000,
		MinLatencyMs:      float64(c.MinLatency.Load()) / 1000,
		QueryCount:        queryCount,
		AvgQueryTimeMs:    float64(avgQueryTime) / 1000,
		SlowQueryCount:    c.SlowQueryCount.Load(),
		CacheHits:         cacheHits,
		CacheMisses:       cacheMisses,
		CacheHitRate:      cacheHitRate,
		CacheEvictions:    c.CacheEvictions.Load(),
		OpenConnections:   c.OpenConnections.Load(),
		IdleConnections:   c.IdleConnections.Load(),
		MaxConnections:    int64(c.MaxConnections),
		MemoryUsageMB:     float64(c.MemoryUsage.Load()) / 1024 / 1024,
		GoroutineCount:    c.GoroutineCount.Load(),
		EndpointStats:     endpointSnapshots,
	}
}

// Reset clears all metrics (useful for testing)
func (c *Collector) Reset() {
	c.TotalRequests.Store(0)
	c.SuccessRequests.Store(0)
	c.ErrorRequests.Store(0)
	c.TotalLatency.Store(0)
	c.QueryCount.Store(0)
	c.QueryTime.Store(0)
	c.QueryErrors.Store(0)
	c.SlowQueryCount.Store(0)
	c.CacheHits.Store(0)
	c.CacheMisses.Store(0)
	c.CacheEvictions.Store(0)

	c.mu.Lock()
	c.EndpointStats = make(map[string]*EndpointStat)
	c.mu.Unlock()

	c.LastResetTime = time.Now()
}
