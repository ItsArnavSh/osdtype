package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

const requestIDContextKey = "request_id"

func newRequestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(b)
}

func validRequestID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if !validRequestID(id) {
			id = newRequestID()
		}
		c.Set(requestIDContextKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func requestID(c *gin.Context) string {
	return c.GetString(requestIDContextKey)
}

type requestMetrics struct {
	started        time.Time
	requests       atomic.Uint64
	inFlight       atomic.Int64
	durationNanos  atomic.Uint64
	durationCount  atomic.Uint64
	responses2xx   atomic.Uint64
	responses3xx   atomic.Uint64
	responses4xx   atomic.Uint64
	responses5xx   atomic.Uint64
	responsesOther atomic.Uint64
}

func newRequestMetrics() *requestMetrics {
	return &requestMetrics{started: time.Now()}
}

func (m *requestMetrics) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		m.requests.Add(1)
		m.inFlight.Add(1)
		start := time.Now()
		c.Next()
		m.inFlight.Add(-1)
		m.durationNanos.Add(uint64(time.Since(start)))
		m.durationCount.Add(1)
		switch status := c.Writer.Status(); {
		case status >= 200 && status < 300:
			m.responses2xx.Add(1)
		case status >= 300 && status < 400:
			m.responses3xx.Add(1)
		case status >= 400 && status < 500:
			m.responses4xx.Add(1)
		case status >= 500 && status < 600:
			m.responses5xx.Add(1)
		default:
			m.responsesOther.Add(1)
		}
	}
}

func (m *requestMetrics) handler(c *gin.Context) {
	if token := strings.TrimSpace(viper.GetString("Metrics.token")); token != "" {
		if c.GetHeader("Authorization") != "Bearer "+token {
			c.Header("WWW-Authenticate", `Bearer realm="metrics"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(c.Writer, `# HELP osdtyp_uptime_seconds Seconds since the process started.
# TYPE osdtyp_uptime_seconds gauge
osdtyp_uptime_seconds %.3f
# HELP osdtyp_http_requests_total Total HTTP requests handled.
# TYPE osdtyp_http_requests_total counter
osdtyp_http_requests_total %d
# HELP osdtyp_http_requests_in_flight HTTP requests currently being handled.
# TYPE osdtyp_http_requests_in_flight gauge
osdtyp_http_requests_in_flight %d
# HELP osdtyp_http_responses_total HTTP responses by status class.
# TYPE osdtyp_http_responses_total counter
osdtyp_http_responses_total{class="2xx"} %d
osdtyp_http_responses_total{class="3xx"} %d
osdtyp_http_responses_total{class="4xx"} %d
osdtyp_http_responses_total{class="5xx"} %d
osdtyp_http_responses_total{class="other"} %d
# HELP osdtyp_http_request_duration_seconds Total time spent handling HTTP requests.
# TYPE osdtyp_http_request_duration_seconds summary
osdtyp_http_request_duration_seconds_sum %.6f
osdtyp_http_request_duration_seconds_count %d
# HELP osdtyp_go_goroutines Current number of goroutines.
# TYPE osdtyp_go_goroutines gauge
osdtyp_go_goroutines %d
# HELP osdtyp_go_memory_alloc_bytes Bytes of allocated heap memory.
# TYPE osdtyp_go_memory_alloc_bytes gauge
osdtyp_go_memory_alloc_bytes %d
# HELP osdtyp_go_gc_cycles_total Completed garbage collection cycles.
# TYPE osdtyp_go_gc_cycles_total counter
osdtyp_go_gc_cycles_total %d
`,
		time.Since(m.started).Seconds(),
		m.requests.Load(),
		m.inFlight.Load(),
		m.responses2xx.Load(),
		m.responses3xx.Load(),
		m.responses4xx.Load(),
		m.responses5xx.Load(),
		m.responsesOther.Load(),
		float64(m.durationNanos.Load())/float64(time.Second),
		m.durationCount.Load(),
		runtime.NumGoroutine(),
		mem.Alloc,
		mem.NumGC,
	)
}
