package utils

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
)

// LogEntry is the bounded, operator-facing projection of a zap entry.
type LogEntry struct {
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// LogBuffer keeps the most recent log entries in memory for the admin panel.
// It is intentionally small: Render remains the durable log sink.
type LogBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	max     int
}

// LogFilter is applied to the in-memory buffer before pagination. User is the
// raw query (username or numeric id); UserID is the resolved account id when
// the caller has it.
type LogFilter struct {
	Query  string
	Level  string
	User   string
	UserID uint32
}

func NewLogBuffer(max int) *LogBuffer {
	if max <= 0 {
		max = 500
	}
	return &LogBuffer{max: max}
}

func (b *LogBuffer) Add(entry LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.entries) >= b.max {
		copy(b.entries, b.entries[1:])
		b.entries[len(b.entries)-1] = entry
		return
	}
	b.entries = append(b.entries, entry)
}

func (b *LogBuffer) Snapshot(limit int) []LogEntry {
	entries, _ := b.Search(LogFilter{}, limit, 0)
	return entries
}

// Search filters newest-first and returns the requested page plus the total
// number of matching entries. The buffer is bounded, so this never scans an
// unbounded source.
func (b *LogBuffer) Search(filter LogFilter, limit, offset int) ([]LogEntry, int) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	if limit > b.max {
		limit = b.max
	}
	if offset < 0 {
		offset = 0
	}
	out := make([]LogEntry, 0, limit)
	total := 0
	for i := len(b.entries) - 1; i >= 0; i-- {
		entry := b.entries[i]
		if !matchesLog(entry, filter) {
			continue
		}
		total++
		if total <= offset || len(out) >= limit {
			continue
		}
		out = append(out, entry)
	}
	return out, total
}

func matchesLog(entry LogEntry, filter LogFilter) bool {
	level := strings.ToLower(strings.TrimSpace(filter.Level))
	if level != "" && strings.ToLower(entry.Level) != level {
		return false
	}
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	if query != "" &&
		!strings.Contains(strings.ToLower(entry.Message), query) &&
		!fieldsContain(entry.Fields, query) {
		return false
	}
	user := strings.ToLower(strings.TrimSpace(filter.User))
	if user == "" && filter.UserID == 0 {
		return true
	}
	userID := ""
	if filter.UserID != 0 {
		userID = strconv.FormatUint(uint64(filter.UserID), 10)
	}
	if userID != "" && fieldValue(entry.Fields, "user_id") == userID {
		return true
	}
	for _, key := range []string{"username", "actor", "target"} {
		if user != "" && strings.Contains(strings.ToLower(fieldValue(entry.Fields, key)), user) {
			return true
		}
	}
	if user != "" && strings.Contains(strings.ToLower(entry.Message), user) {
		return true
	}
	if userID != "" && strings.Contains(strings.ToLower(entry.Message), userID) {
		return true
	}
	return false
}

func fieldsContain(fields map[string]any, query string) bool {
	for key, value := range fields {
		if strings.Contains(strings.ToLower(key), query) ||
			strings.Contains(strings.ToLower(fmt.Sprint(value)), query) {
			return true
		}
	}
	return false
}

func fieldValue(fields map[string]any, key string) string {
	if fields == nil {
		return ""
	}
	value, ok := fields[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(value)
}

var processLogs = NewLogBuffer(500)

func Logs() *LogBuffer { return processLogs }

// LogBufferCore copies entries into a LogBuffer before forwarding them to the
// normal zap core.
type LogBufferCore struct {
	zapcore.Core
	buf *LogBuffer
}

func NewLogBufferCore(core zapcore.Core, buf *LogBuffer) zapcore.Core {
	return &LogBufferCore{Core: core, buf: buf}
}

func (c *LogBufferCore) With(fields []zapcore.Field) zapcore.Core {
	return &LogBufferCore{Core: c.Core.With(fields), buf: c.buf}
}

func (c *LogBufferCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Core.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *LogBufferCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(encoder)
	}
	c.buf.Add(LogEntry{
		Time:    entry.Time,
		Level:   entry.Level.String(),
		Message: entry.Message,
		Fields:  encoder.Fields,
	})
	return c.Core.Write(entry, fields)
}
