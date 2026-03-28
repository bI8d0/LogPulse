package ui

import (
	"strings"

	"LogPulse/internal/config"
	"LogPulse/internal/parser"
	"LogPulse/internal/watcher"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// App is the main LogPulse application
type App struct {
	tviewApp *tview.Application
	cfg      *config.Config
	sysPanel *Panel
	appPanel *Panel
	webPanel *Panel
	dbPanel  *Panel
	menuBar  *MenuBar
	tailers  [4]*watcher.Tailer
}

// NewApp creates and initializes the application
func NewApp(cfg *config.Config) *App {
	tviewApp := tview.NewApplication()

	sysPanel := NewPanel("🖥  System", cfg.MaxLines, tcell.ColorTeal)
	appPanel := NewPanel("📦 Application", cfg.MaxLines, tcell.ColorPurple)
	webPanel := NewPanel("🌐 Web", cfg.MaxLines, tcell.ColorGreen)
	dbPanel := NewPanel("🗄️ Database", cfg.MaxLines, tcell.ColorOrange)

	app := &App{
		tviewApp: tviewApp,
		cfg:      cfg,
		sysPanel: sysPanel,
		appPanel: appPanel,
		webPanel: webPanel,
		dbPanel:  dbPanel,
	}
	app.menuBar = NewMenuBar(cfg, func() {})
	return app
}

// initPanel sets up a panel BEFORE Run() — called from main goroutine, no event loop yet.
func (a *App) initPanel(panels [4]*Panel, idx int, path string) {
	if strings.TrimSpace(path) == "" {
		panels[idx].ShowDisabledDirect()
		return
	}
	ch := make(chan parser.LogEntry, 200)
	panels[idx].StartListening(a.tviewApp, ch)
	t := watcher.NewTailer(path, ch)
	t.Start()
	a.tailers[idx] = t
}

// restartPanel is called from the tview form "Apply" button callback —
// we are INSIDE the event loop thread, so write directly (no QueueUpdateDraw).
func (a *App) restartPanel(panels [4]*Panel, idx int, path string) {
	if idx >= 4 {
		return
	}
	if a.tailers[idx] != nil {
		a.tailers[idx].Stop()
		a.tailers[idx] = nil
	}

	if strings.TrimSpace(path) == "" {
		panels[idx].ShowDisabledDirect()
		return
	}
	ch := make(chan parser.LogEntry, 200)
	panels[idx].StartListening(a.tviewApp, ch)
	t := watcher.NewTailer(path, ch)
	t.Start()
	a.tailers[idx] = t
}

// Run starts the watchers and the UI
func (a *App) Run() error {
	panels := [4]*Panel{a.sysPanel, a.appPanel, a.webPanel, a.dbPanel}

	// Initialise all panels synchronously before Run() — safe, no QueueUpdateDraw.
	for i := 0; i < 4; i++ {
		a.initPanel(panels, i, a.cfg.Sources[i].Path)
	}

	pages := tview.NewPages()

	tailerRestart := func(panelIdx int, newPath string) {
		a.restartPanel(panels, panelIdx, newPath)
	}

	focusTargets := []tview.Primitive{
		a.sysPanel.GetWidget(),
		a.appPanel.GetWidget(),
		a.webPanel.GetWidget(),
		a.dbPanel.GetWidget(),
	}
	focusIndex := 0

	configForm, onConfigOpen := BuildConfigModal(a.tviewApp, pages, a.cfg, a.menuBar, tailerRestart, func() {
		a.tviewApp.SetFocus(focusTargets[focusIndex])
	})
	flex := BuildLayout(a.menuBar, a.sysPanel, a.appPanel, a.webPanel, a.dbPanel)

	pages.AddPage("main", flex, true, true)
	pages.AddPage("config",
		tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(
				tview.NewFlex().SetDirection(tview.FlexRow).
					AddItem(nil, 0, 1, false).
					AddItem(configForm, 20, 1, true).
					AddItem(nil, 0, 1, false),
				82, 1, true).
			AddItem(nil, 0, 1, false),
		true, false,
	)

	var globalKeys func(*tcell.EventKey) *tcell.EventKey
	globalKeys = func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			a.tviewApp.Stop()
			return nil
		case tcell.KeyF2:
			name, _ := pages.GetFrontPage()
			if name == "config" {
				pages.HidePage("config")
				a.tviewApp.SetFocus(focusTargets[focusIndex])
			} else {
				onConfigOpen()
				pages.ShowPage("config")
				a.tviewApp.SetFocus(configForm)
			}
			return nil
		case tcell.KeyTab:
			name, _ := pages.GetFrontPage()
			if name != "config" {
				focusIndex = (focusIndex + 1) % len(focusTargets)
				a.tviewApp.SetFocus(focusTargets[focusIndex])
				return nil
			}
		case tcell.KeyRune:
			if event.Rune() == 'q' || event.Rune() == 'Q' {
				name, _ := pages.GetFrontPage()
				if name != "config" {
					a.tviewApp.Stop()
					return nil
				}
			}
		default:
		}
		return event
	}

	a.tviewApp.SetInputCapture(globalKeys)
	a.sysPanel.SetKeyHandler(globalKeys)
	a.appPanel.SetKeyHandler(globalKeys)
	a.webPanel.SetKeyHandler(globalKeys)
	a.dbPanel.SetKeyHandler(globalKeys)

	a.tviewApp.SetRoot(pages, true)

	// Set focus once the event loop is running.
	go func() {
		a.tviewApp.QueueUpdateDraw(func() {
			pages.SwitchToPage("main")
			focusIndex = 0
			a.tviewApp.SetFocus(focusTargets[focusIndex])
		})
	}()

	return a.tviewApp.Run()
}
