package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"LogPulse/internal/config"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// MenuBar is the top bar showing log names and keyboard shortcuts
type MenuBar struct {
	view      *tview.TextView
	cfg       *config.Config
	onRefresh func()
}

// NewMenuBar creates the top menu bar
func NewMenuBar(cfg *config.Config, onRefresh func()) *MenuBar {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(tcell.ColorNavy)
	tv.SetTextColor(tcell.ColorWhite)
	tv.SetScrollable(false)

	mb := &MenuBar{view: tv, cfg: cfg, onRefresh: onRefresh}
	mb.render()
	return mb
}

// GetWidget returns the menu bar widget
func (mb *MenuBar) GetWidget() *tview.TextView {
	return mb.view
}

// logName returns only the file name, or "(empty)" if the path is blank
func logName(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "[red](empty)[-]"
	}
	return "[white]" + filepath.Base(path) + "[-]"
}

// render refreshes the bar text with the current log file names
func (mb *MenuBar) render() {
	mb.view.Clear()
	icons := []string{"🖥", "📦", "🌐", "🗄 "}
	labels := []string{" SYS", "APP", "WEB", "DB"}
	line := " [yellow]LogPulse v1.0[-]  "
	for i, src := range mb.cfg.Sources {
		if i < 4 {
			line += fmt.Sprintf("[cyan]%s %s:[white] %s[-]   ", icons[i], labels[i], logName(src.Path))
		}
	}
	line += " [darkgray]F2[white]:Paths  [darkgray]Tab[white]:Focus  [darkgray]Q[white]:Quit  [darkgray]|  [yellow]By bI8d0[-]"
	_, _ = fmt.Fprint(mb.view, line)
}

// BuildConfigModal builds the modal form for editing the 4 log paths.
// Returns the form and an onOpen callback that MUST be called every time
// the modal is shown — it refreshes the fields and the change-detection snapshot.
func BuildConfigModal(
	app *tview.Application,
	pages *tview.Pages,
	cfg *config.Config,
	menuBar *MenuBar,
	tailerRestart func(panelIdx int, newPath string),
	onClose func(),
) (*tview.Form, func()) {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" ⚙  Configure log paths — F2 to open/close ")
	form.SetTitleColor(tcell.ColorYellow)
	form.SetBorderColor(tcell.ColorDarkCyan)
	form.SetBackgroundColor(tcell.ColorBlack)
	form.SetFieldBackgroundColor(tcell.ColorDarkBlue)
	form.SetFieldTextColor(tcell.ColorWhite)
	form.SetLabelColor(tcell.ColorAqua)

	formLabels := []string{
		"🖥 System (sys) ",
		"📦 Application (app) ",
		"🌐 Web (web) ",
		"🗄️ Database (db) ",
	}

	// Snapshot refreshed every time the modal opens — used to detect which
	// panels actually changed so we don't restart untouched panels.
	originalPaths := [4]string{}
	for i := range cfg.Sources {
		originalPaths[i] = cfg.Sources[i].Path
	}

	for i := range cfg.Sources {
		idx := i
		form.AddInputField(formLabels[idx], cfg.Sources[idx].Path, 55, nil, func(text string) {
			cfg.Sources[idx].Path = text
		})
	}

	restoreFields := func() {
		for i := range cfg.Sources {
			cfg.Sources[i].Path = originalPaths[i]
			if item := form.GetFormItemByLabel(formLabels[i]); item != nil {
				if field, ok := item.(*tview.InputField); ok {
					field.SetText(originalPaths[i])
				}
			}
		}
	}

	form.AddButton("✔ Apply", func() {
		_ = cfg.Save()
		menuBar.render()
		// Only restart panels whose path actually changed.
		for i := range cfg.Sources {
			if cfg.Sources[i].Path != originalPaths[i] {
				tailerRestart(i, cfg.Sources[i].Path)
			}
		}
		pages.HidePage("config")
		onClose()
	})

	form.AddButton("✘ Cancel", func() {
		restoreFields()
		pages.HidePage("config")
		onClose()
	})

	form.SetCancelFunc(func() {
		restoreFields()
		pages.HidePage("config")
		onClose()
	})

	// onOpen must be called every time the modal is shown.
	// It snapshots the current paths and refreshes the form fields.
	onOpen := func() {
		for i := range cfg.Sources {
			originalPaths[i] = cfg.Sources[i].Path
			if item := form.GetFormItemByLabel(formLabels[i]); item != nil {
				if field, ok := item.(*tview.InputField); ok {
					field.SetText(cfg.Sources[i].Path)
				}
			}
		}
	}

	return form, onOpen
}
