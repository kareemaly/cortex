package tuilog

import (
	"fmt"
	"sync"
	"time"
)

// DefaultCapacity is the default ring buffer capacity.
const DefaultCapacity = 1000

// Buffer is a thread-safe ring buffer for log entries.
type Buffer struct {
	mu         sync.Mutex
	entries    []Entry
	head       int // next write position
	count      int
	capacity   int
	errorCount int
	warnCount  int
}

// NewBuffer creates a new ring buffer with the given capacity.
func NewBuffer(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &Buffer{
		entries:  make([]Entry, capacity),
		capacity: capacity,
	}
}

// Log adds an entry to the buffer.
func (b *Buffer) Log(level Level, source, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// If overwriting an old entry, decrement its counts.
	if b.count == b.capacity {
		old := b.entries[b.head]
		switch old.Level {
		case LevelError:
			b.errorCount--
		case LevelWarn:
			b.warnCount--
		}
	}

	entry := Entry{
		Time:    time.Now(),
		Level:   level,
		Source:  source,
		Message: message,
	}
	b.entries[b.head] = entry
	b.head = (b.head + 1) % b.capacity
	if b.count < b.capacity {
		b.count++
	}

	// Increment counts for the new entry.
	switch level {
	case LevelError:
		b.errorCount++
	case LevelWarn:
		b.warnCount++
	}
}

// Entries returns a copy of all entries, newest first.
func (b *Buffer) Entries() []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return nil
	}

	result := make([]Entry, b.count)
	for i := 0; i < b.count; i++ {
		// Walk backwards from head-1.
		idx := (b.head - 1 - i + b.capacity) % b.capacity
		result[i] = b.entries[idx]
	}
	return result
}

// ErrorCount returns the current number of error entries.
func (b *Buffer) ErrorCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.errorCount
}

// WarnCount returns the current number of warning entries.
func (b *Buffer) WarnCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.warnCount
}

// Debug logs a debug message.
func (b *Buffer) Debug(source, message string) {
	b.Log(LevelDebug, source, message)
}

// Info logs an info message.
func (b *Buffer) Info(source, message string) {
	b.Log(LevelInfo, source, message)
}

// Warn logs a warning message.
func (b *Buffer) Warn(source, message string) {
	b.Log(LevelWarn, source, message)
}

// Error logs an error message.
func (b *Buffer) Error(source, message string) {
	b.Log(LevelError, source, message)
}

// Debugf logs a formatted debug message.
func (b *Buffer) Debugf(source, format string, args ...any) {
	b.Log(LevelDebug, source, fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message.
func (b *Buffer) Infof(source, format string, args ...any) {
	b.Log(LevelInfo, source, fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message.
func (b *Buffer) Warnf(source, format string, args ...any) {
	b.Log(LevelWarn, source, fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message.
func (b *Buffer) Errorf(source, format string, args ...any) {
	b.Log(LevelError, source, fmt.Sprintf(format, args...))
}
