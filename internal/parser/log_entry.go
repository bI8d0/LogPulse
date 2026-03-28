package parser

import (
	"strings"
	"time"
)

// Level represents the severity level of a log entry
type Level int

const (
	LevelInfo Level = iota
	LevelWarn
	LevelError
	LevelDebug
	LevelUnknown
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelDebug:
		return "DEBUG"
	default:
		return "???"
	}
}

// LogEntry represents a parsed log entry
type LogEntry struct {
	Timestamp time.Time
	Level     Level
	Source    string
	Message   string
	Raw       string
}

// Parser is the interface for parsing log lines
type Parser interface {
	Parse(line string) LogEntry
}

// AutoParser automatically detects the format and parses the line
type AutoParser struct{}

// Parse detects the level based on keywords in the line
func (p *AutoParser) Parse(line string) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now(),
		Raw:       line,
		Level:     LevelInfo,
		Message:   line,
	}

	upper := strings.ToUpper(line)

	switch {
	case strings.Contains(upper, "ERROR") ||
		strings.Contains(upper, "FATAL") ||
		strings.Contains(upper, "CRITICAL") ||
		strings.Contains(upper, "CRIT"):
		entry.Level = LevelError
	case strings.Contains(upper, "WARN") ||
		strings.Contains(upper, "WARNING"):
		entry.Level = LevelWarn
	case strings.Contains(upper, "DEBUG") ||
		strings.Contains(upper, "TRACE"):
		entry.Level = LevelDebug
	default:
		entry.Level = LevelInfo
	}

	return entry
}

// NewAutoParser creates a new automatic parser
func NewAutoParser() *AutoParser {
	return &AutoParser{}
}
