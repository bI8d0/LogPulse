package watcher

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"

	"LogPulse/internal/parser"
)

// Tailer follows a log file in real time (equivalent to tail -f)
type Tailer struct {
	path   string
	ch     chan parser.LogEntry
	parser *parser.AutoParser
	done   chan struct{}
}

// NewTailer creates a new Tailer for the given file path
func NewTailer(path string, ch chan parser.LogEntry) *Tailer {
	return &Tailer{
		path:   path,
		ch:     ch,
		parser: parser.NewAutoParser(),
		done:   make(chan struct{}),
	}
}

// Start begins following the file in a separate goroutine.
// The channel is closed when the tailer exits so listeners unblock cleanly.
func (t *Tailer) Start() {
	go func() {
		defer close(t.ch)
		t.tail()
	}()
}

// Stop halts file following
func (t *Tailer) Stop() {
	select {
	case <-t.done: // already closed
	default:
		close(t.done)
	}
}

func (t *Tailer) send(entry parser.LogEntry) {
	select {
	case t.ch <- entry:
	case <-t.done:
	}
}

func (t *Tailer) tail() {
	// Empty path → panel disabled, do nothing
	if strings.TrimSpace(t.path) == "" {
		return
	}

	// Wait until the file exists
	for {
		if _, err := os.Stat(t.path); err == nil {
			break
		}
		t.send(parser.LogEntry{
			Timestamp: time.Now(),
			Level:     parser.LevelWarn,
			Source:    t.path,
			Message:   "⚠ Waiting for file: " + t.path,
		})
		select {
		case <-t.done:
			return
		case <-time.After(2 * time.Second):
		}
	}

	file, err := os.Open(t.path)
	if err != nil {
		t.send(parser.LogEntry{
			Timestamp: time.Now(),
			Level:     parser.LevelError,
			Source:    t.path,
			Message:   "✗ Error opening file: " + err.Error(),
		})
		return
	}
	defer func() { _ = file.Close() }()

	// Seek to the end so only new lines are read
	_, _ = file.Seek(0, io.SeekEnd)

	for {
		select {
		case <-t.done:
			return
		default:
		}

		// Create a fresh scanner at the current file descriptor position
		// to avoid stale buffered data between iterations
		scanner := bufio.NewScanner(file)
		gotLines := false
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			entry := t.parser.Parse(line)
			entry.Source = t.path
			t.send(entry)
			gotLines = true
		}

		// No new lines yet — wait before retrying
		if !gotLines {
			time.Sleep(250 * time.Millisecond)
		}
	}
}
