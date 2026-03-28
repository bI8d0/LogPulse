package ui

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"LogPulse/internal/parser"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Panel represents a dashboard log panel
type Panel struct {
	view       *tview.TextView
	title      string
	mu         sync.Mutex
	count      int
	errors     int
	warns      int
	lineCount  int // number of log lines currently displayed
	maxLines   int
	generation atomic.Int64 // incremented on every source change
	disabled   atomic.Bool  // true when the panel is in disabled state
	autoScroll atomic.Bool  // true when the panel should auto-scroll to the end
}

// NewPanel creates a new panel with the given title
func NewPanel(title string, maxLines int, borderColor tcell.Color) *Panel {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetScrollable(true)
	tv.SetWrap(true)
	tv.SetBorder(true)
	tv.SetTitle(fmt.Sprintf(" %s ", title))
	tv.SetTitleColor(tcell.ColorYellow)
	tv.SetBorderColor(borderColor)
	tv.SetTextColor(tcell.ColorWhite)
	tv.SetBackgroundColor(tcell.ColorBlack)
	p := &Panel{
		view:     tv,
		title:    title,
		maxLines: maxLines,
	}
	p.autoScroll.Store(true)

	// Capture navigation keys to toggle autoScroll:
	// When the user scrolls up, disable autoScroll so new lines don't jump to the end.
	// When the user reaches the end (pressing End or scrolling down past the last row), re-enable it.
	tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyPgUp:
			p.autoScroll.Store(false)
		case tcell.KeyDown, tcell.KeyPgDn:
			// After tview processes the key, check if we reached the end.
			// We do it asynchronously so tview updates the offset first.
			go func() {
				// Small yield so tview processes the scroll first
				row, _ := tv.GetScrollOffset()
				_, _, _, h := tv.GetInnerRect()
				lineCount := strings.Count(tv.GetText(false), "\n")
				if row+h >= lineCount {
					p.autoScroll.Store(true)
				}
			}()
		case tcell.KeyEnd:
			p.autoScroll.Store(true)
			tv.ScrollToEnd()
		case tcell.KeyHome:
			p.autoScroll.Store(false)
		}
		return event
	})

	// SetDrawFunc is called by tview on every repaint with the REAL screen
	// coordinates. We let tview draw the widget normally (returning GetInnerRect)
	// and then, if the panel is disabled, we overdraw the centred message
	// directly onto the screen using those real dimensions.
	tv.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		ix, iy, iw, ih := tv.GetInnerRect()
		if p.disabled.Load() && screen != nil && iw > 0 && ih > 0 {
			drawDisabledOverlay(screen, ix, iy, iw, ih)
		}
		return ix, iy, iw, ih
	})

	return p
}

// drawDisabledOverlay paints the centred disabled message directly on screen.
// Called from inside SetDrawFunc so coordinates are always real.
func drawDisabledOverlay(screen tcell.Screen, x, y, w, h int) {
	// Fill background black
	bgStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			screen.SetContent(x+col, y+row, ' ', nil, bgStyle)
		}
	}

	line1 := []rune("⚠  Panel disabled")
	line2 := []rune("Press F2 and assign a path to enable it.")

	styleYellow := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	styleGray := tcell.StyleDefault.Foreground(tcell.ColorGray).Background(tcell.ColorBlack)

	// Centre vertically: 3 lines (line1 + blank + line2)
	startRow := y + (h-3)/2
	if startRow < y {
		startRow = y
	}

	printCentered := func(row int, runes []rune, style tcell.Style) {
		startCol := x + (w-len(runes))/2
		if startCol < x {
			startCol = x
		}
		for i, r := range runes {
			screen.SetContent(startCol+i, row, r, nil, style)
		}
	}

	printCentered(startRow, line1, styleYellow)
	printCentered(startRow+2, line2, styleGray)
}

// GetWidget returns the tview widget for this panel
func (p *Panel) GetWidget() *tview.TextView {
	return p.view
}

// refreshTitle updates the panel border title with the current line count.
// Must be called from within a tview draw context (QueueUpdateDraw callback).
func (p *Panel) refreshTitle() {
	p.mu.Lock()
	lc := p.lineCount
	p.mu.Unlock()
	if lc == 0 {
		p.view.SetTitle(fmt.Sprintf(" %s ", p.title))
	} else {
		p.view.SetTitle(fmt.Sprintf(" %s [gray](%d lines)[-] ", p.title, lc))
	}
}

// SetKeyHandler installs a global keyboard handler on the panel widget,
// chaining it AFTER the panel's own scroll-tracking handler.
func (p *Panel) SetKeyHandler(fn func(*tcell.EventKey) *tcell.EventKey) {
	existing := p.view.GetInputCapture()
	p.view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if existing != nil {
			event = existing(event)
		}
		if event == nil {
			return nil
		}
		return fn(event)
	})
}

// AppendEntry adds a log entry to the panel with colour based on level.
// Called from background goroutines — uses QueueUpdateDraw.
func (p *Panel) AppendEntry(app *tview.Application, entry parser.LogEntry) {
	p.mu.Lock()
	p.count++
	p.lineCount++
	if entry.Level == parser.LevelError {
		p.errors++
	}
	if entry.Level == parser.LevelWarn {
		p.warns++
	}
	p.mu.Unlock()

	color := levelColor(entry.Level)
	timestamp := entry.Timestamp.Format("15:04:05")
	logLine := fmt.Sprintf("[gray]%s[white] [%s]%s[-] %s",
		timestamp, color, entry.Level.String(), tview.Escape(entry.Message))

	app.QueueUpdateDraw(func() {
		// Build separator with exact inner width to avoid wrapping.
		_, _, w, _ := p.view.GetInnerRect()
		if w <= 0 {
			w = 60
		}
		sep := "[#2a2a2a]" + strings.Repeat("─", w) + "[-]"
		_, _ = fmt.Fprintf(p.view, "%s\n%s\n", logLine, sep)
		p.refreshTitle()
		if p.autoScroll.Load() {
			p.view.ScrollToEnd()
		}
	})
}

func levelColor(l parser.Level) string {
	switch l {
	case parser.LevelError:
		return "red"
	case parser.LevelWarn:
		return "yellow"
	case parser.LevelDebug:
		return "blue"
	default:
		return "green"
	}
}

// StartListening clears the panel, disables the disabled state, and starts
// a goroutine reading from ch.
func (p *Panel) StartListening(app *tview.Application, ch <-chan parser.LogEntry) {
	gen := p.generation.Add(1)
	p.disabled.Store(false)
	p.autoScroll.Store(true)
	p.mu.Lock()
	p.lineCount = 0
	p.mu.Unlock()
	p.view.Clear()
	p.view.SetTitle(fmt.Sprintf(" %s ", p.title))

	go func() {
		for entry := range ch {
			if p.generation.Load() != gen {
				go func() {
					for range ch {
					}
				}()
				return
			}
			p.AppendEntry(app, entry)
		}
		// Channel closed by tailer — show disabled if still active generation.
		if p.generation.Load() == gen {
			app.QueueUpdateDraw(func() {
				if p.generation.Load() == gen {
					p.disabled.Store(true)
					p.view.Clear()
				}
			})
		}
	}()
}

// ShowDisabledDirect marks the panel as disabled.
// Safe to call before Run() (main goroutine) or inside a tview callback.
func (p *Panel) ShowDisabledDirect() {
	p.generation.Add(1)
	p.disabled.Store(true)
	p.mu.Lock()
	p.lineCount = 0
	p.mu.Unlock()
	p.view.Clear()
	p.view.SetTitle(fmt.Sprintf(" %s ", p.title))
}

// GetStats returns the panel counters: total entries, errors and warnings
func (p *Panel) GetStats() (total, errors, warns int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.count, p.errors, p.warns
}
