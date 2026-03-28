package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// BuildLayout builds the main layout: top bar + 2x2 panel grid
func BuildLayout(
	menuBar *MenuBar,
	sysPanel *Panel,
	appPanel *Panel,
	webPanel *Panel,
	dbPanel *Panel,
) *tview.Flex {
	grid := tview.NewGrid()
	grid.SetRows(0, 0)
	grid.SetColumns(0, 0)
	grid.SetBorders(false)
	grid.SetBackgroundColor(tcell.ColorBlack)

	grid.AddItem(sysPanel.GetWidget(), 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(appPanel.GetWidget(), 0, 1, 1, 1, 0, 0, false)
	grid.AddItem(webPanel.GetWidget(), 1, 0, 1, 1, 0, 0, false)
	grid.AddItem(dbPanel.GetWidget(), 1, 1, 1, 1, 0, 0, false)

	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)
	flex.AddItem(menuBar.GetWidget(), 1, 0, false)
	flex.AddItem(grid, 0, 1, true)

	return flex
}
