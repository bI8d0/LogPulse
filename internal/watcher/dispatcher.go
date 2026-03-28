package watcher

import (
	"LogPulse/internal/parser"
)

// PanelChannel holds the channels to each UI panel
type PanelChannel struct {
	System chan parser.LogEntry
	App    chan parser.LogEntry
	Errors chan parser.LogEntry
}

// Dispatcher routes log entries to the corresponding panel
type Dispatcher struct {
	input  chan parser.LogEntry
	panels PanelChannel
	done   chan struct{}
}

// NewDispatcher creates a dispatcher with the panel channels already initialised
func NewDispatcher(input chan parser.LogEntry, panels PanelChannel) *Dispatcher {
	return &Dispatcher{
		input:  input,
		panels: panels,
		done:   make(chan struct{}),
	}
}

// Start launches the dispatcher in a goroutine
func (d *Dispatcher) Start() {
	go d.dispatch()
}

// Stop halts the dispatcher
func (d *Dispatcher) Stop() {
	close(d.done)
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case <-d.done:
			return
		case entry, ok := <-d.input:
			if !ok {
				return
			}
			// Route to the errors panel if the level is Error
			if entry.Level == parser.LevelError {
				select {
				case d.panels.Errors <- entry:
				default: // drop if the channel is full
				}
			}
			// Always send to the system panel
			select {
			case d.panels.System <- entry:
			default:
			}
		}
	}
}
