package tuilog

import "time"

// Level represents a log severity level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the full name of the level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ShortString returns a single-character abbreviation.
func (l Level) ShortString() string {
	switch l {
	case LevelDebug:
		return "D"
	case LevelInfo:
		return "I"
	case LevelWarn:
		return "W"
	case LevelError:
		return "E"
	default:
		return "?"
	}
}

// Entry represents a single log entry.
type Entry struct {
	Time    time.Time
	Level   Level
	Source  string
	Message string
}
