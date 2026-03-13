package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func TestNewMenu(t *testing.T) {
	action1Called := false
	action2Called := false

	items := []MenuItem{
		{
			Label:       "Item 1",
			Description: "Desc 1",
			Action:      func() { action1Called = true },
			Disabled:    false,
		},
		{
			Label:       "Item 2",
			Description: "Desc 2",
			Action:      func() { action2Called = true },
			Disabled:    true,
		},
		{
			Label:       "Item 3",
			Description: "Desc 3",
			Action:      nil,
			Disabled:    false,
		},
	}

	title := "Test Menu Title"
	menu := NewMenu(title, items)

	// Verify basic properties
	assert.Equal(t, title, menu.GetTitle())

	selectableRows, selectableCols := menu.GetSelectable()
	assert.True(t, selectableRows)
	assert.False(t, selectableCols)

	assert.Equal(t, len(items), menu.GetRowCount())
	assert.Equal(t, len(items), len(menu.items))
	// applyItems makes 2 columns (label, description)
	assert.Equal(t, 2, menu.GetColumnCount())

	// Trigger selection on row 0 (enabled, has action)
	menu.Select(0, 0)
	handler := menu.InputHandler()
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})
	assert.True(t, action1Called, "Action 1 should have been called")
	action1Called = false // reset

	// Trigger selection on row 1 (disabled, has action)
	menu.Select(1, 0)
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})
	assert.False(t, action2Called, "Action 2 should not have been called because it's disabled")

	// Trigger selection on row 2 (enabled, nil action)
	menu.Select(2, 0)
	// Should not panic
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})

	// Trigger selection on out-of-bounds row (e.g. -1 or len(items))
	// We have to simulate the unexported selected function logic, but we can't easily trigger the exact SetSelectedFunc from outside other than InputHandler.
	// We'll test empty items to ensure it doesn't panic on empty.
	emptyMenu := NewMenu("Empty", []MenuItem{})
	emptyHandler := emptyMenu.InputHandler()
	emptyHandler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) {})
}

func TestMenuApplyItems(t *testing.T) {
	mainColor := tcell.ColorRed
	descColor := tcell.ColorBlue

	items := []MenuItem{
		{
			Label:       "Normal",
			Description: "Desc",
		},
		{
			Label:       "With Colors",
			Description: "Desc",
			MainColor:   &mainColor,
			DescColor:   &descColor,
		},
		{
			Label:       "Disabled",
			Description: "Desc",
			Disabled:    true,
		},
		{
			Label:       "",
			Description: "Empty Label Disabled",
			Disabled:    true,
		},
	}

	menu := NewMenu("Test", items)

	assert.Equal(t, len(items), menu.GetRowCount())

	// Check row 0: Normal
	cell00 := menu.GetCell(0, 0)
	cell01 := menu.GetCell(0, 1)
	assert.Equal(t, "Normal", cell00.Text)
	assert.Equal(t, "Desc", cell01.Text)
	// Right align for desc
	assert.Equal(t, tview.AlignRight, cell01.Align)

	// tview.TableCell in new versions uses `Style` object to store colors, not `Color` field when setting explicitly.
	// We can check style via cell01.Style
	fg, _, _ := cell01.Style.Decompose()
	assert.Equal(t, tview.Styles.TertiaryTextColor, fg)

	// Check row 1: With Colors
	cell10 := menu.GetCell(1, 0)
	cell11 := menu.GetCell(1, 1)
	assert.Equal(t, "With Colors", cell10.Text)
	fg, _, _ = cell10.Style.Decompose()
	assert.Equal(t, tcell.ColorRed, fg)
	fg, _, _ = cell11.Style.Decompose()
	assert.Equal(t, tcell.ColorBlue, fg)

	// Check row 2: Disabled
	cell20 := menu.GetCell(2, 0)
	cell21 := menu.GetCell(2, 1)
	assert.Equal(t, "Disabled (disabled)", cell20.Text)
	fg, _, _ = cell20.Style.Decompose()
	assert.Equal(t, tcell.ColorGray, fg)
	fg, _, _ = cell21.Style.Decompose()
	assert.Equal(t, tcell.ColorGray, fg)

	// Check row 3: Empty Label Disabled
	cell30 := menu.GetCell(3, 0)
	cell31 := menu.GetCell(3, 1)
	// Should not have the " (disabled)" suffix
	assert.Equal(t, "", cell30.Text)
	fg, _, _ = cell30.Style.Decompose()
	assert.Equal(t, tcell.ColorGray, fg)
	fg, _, _ = cell31.Style.Decompose()
	assert.Equal(t, tcell.ColorGray, fg)
}
