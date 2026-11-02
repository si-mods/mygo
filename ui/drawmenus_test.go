package ui

import (
	"testing"
)

// drawApp is a view of a row with a context menu and a text input, in a
// window that draws its menus when draw is set.
type drawApp struct {
	draw   bool
	style  *MenuStyle
	chosen []string
	value  string
}

func (a *drawApp) view(c *context) {
	c.SetDrawMenus(a.draw)
	if a.style != nil {
		c.Theme().Menu = a.style
	}
	coreColumn(c).Fill().Padding(20).Gap(10).Children(func() {
		row := coreRow(c).Padding(8).Focusable().Label("Report").Children(func() {
			coreText(c, "Report.pdf")
		})
		row.ContextMenu(func(m *Menu) {
			if m.Item("Open").Chosen() {
				a.chosen = append(a.chosen, "Open")
			}
			m.Separator()
			if m.Item("Delete").Chosen() {
				a.chosen = append(a.chosen, "Delete")
			}
		})
		coreTextInput(c, &a.value).Label("Name")
	})
}

func TestDrawMenusContextMenu(t *testing.T) {
	a := &drawApp{draw: true}
	tt := coreNewTester(a.view, 400, 300)
	r, _ := tt.Find("Report.pdf")
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	if !tt.HasText("Open") || !tt.HasText("Delete") {
		t.Fatalf("no drawn menu: %q", tt.Texts())
	}
	if tt.Menu() != nil {
		t.Errorf("the system shows a menu too: %q", tt.Menu())
	}
	if err := tt.Click("Delete"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.chosen) != 1 || a.chosen[0] != "Delete" {
		t.Errorf("chosen %q, want Delete", a.chosen)
	}
	if tt.HasText("Open") {
		t.Error("the menu stays after a choice")
	}
}

func TestDrawMenusMenuKey(t *testing.T) {
	a := &drawApp{draw: true}
	tt := coreNewTester(a.view, 400, 300)
	r, _ := tt.Find("Report.pdf")
	tt.ClickAt(r.X+10, r.Y+r.H/2) // focuses the row
	tt.Key(0, KeyContextMenu)
	if !tt.HasText("Open") {
		t.Fatalf("the menu key opened no drawn menu: %q", tt.Texts())
	}
	if open, _ := tt.Find("Open"); open.Y < r.Y {
		t.Errorf("the menu at %+v is above the element %+v", open, r)
	}
	tt.Key(0, KeyEnter)
	tt.Frame()
	if len(a.chosen) != 1 || a.chosen[0] != "Open" {
		t.Errorf("chosen %q, want Open", a.chosen)
	}
}

func TestDrawMenusEditingMenu(t *testing.T) {
	a := &drawApp{draw: true, value: "hello"}
	tt := coreNewTester(a.view, 400, 300)
	tt.RightClick("Name")
	if tt.Menu() != nil {
		t.Errorf("the system shows a menu: %q", tt.Menu())
	}
	if !tt.HasText("Select All") || !tt.HasText("Paste") {
		t.Fatalf("no drawn editing menu: %q", tt.Texts())
	}
	if err := tt.Click("Select All"); err != nil {
		t.Fatal(err)
	}
	tt.RightClick("Name")
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "hello" {
		t.Errorf("Select All, Copy: clipboard %q", tt.Clipboard())
	}
	tt.SetClipboard("world")
	tt.RightClick("Name")
	if err := tt.Click("Paste"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.value != "world" {
		t.Errorf("Paste left %q, want the selection replaced", a.value)
	}
	if tt.HasText("Select All") {
		t.Error("the editing menu stays after a choice")
	}
}

func TestDrawMenusEditingMenuKey(t *testing.T) {
	a := &drawApp{draw: true, value: "hello"}
	tt := coreNewTester(a.view, 400, 300)
	tt.Click("Name")
	tt.Key(0, KeyContextMenu)
	if !tt.HasText("Select All") {
		t.Fatalf("the menu key opened no drawn editing menu: %q", tt.Texts())
	}
	tt.Key(0, KeyEscape)
	if tt.HasText("Select All") {
		t.Error("Escape left the menu")
	}
}

func TestDrawMenusOff(t *testing.T) {
	a := &drawApp{}
	tt := coreNewTester(a.view, 400, 300)
	r, _ := tt.Find("Report.pdf")
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	if got := tt.Menu(); len(got) == 0 {
		t.Fatal("the host's popupMenu was not called")
	}
	if tt.HasText("Open") {
		t.Error("a menu is drawn with the switch off")
	}
	tt.RightClick("Name")
	if len(tt.Menu()) == 0 || tt.HasText("Select All") {
		t.Errorf("the editing menu: system %q, texts %q", tt.Menu(), tt.Texts())
	}
}

func TestDrawMenusSwitchedOffClosesMenu(t *testing.T) {
	a := &drawApp{draw: true}
	tt := coreNewTester(a.view, 400, 300)
	tt.RightClick("Name")
	if !tt.HasText("Select All") {
		t.Fatal("no menu")
	}
	a.draw = false
	tt.Frame()
	tt.Frame()
	if tt.HasText("Select All") {
		t.Error("the menu stays with the switch off")
	}
}

// menuNodes returns the panel of the drawn menu, and the row labeled
// label inside it.
func menuNodes(tt *Tester, label string) (panel, row *node) {
	var walk func(e *node)
	walk = func(e *node) {
		if e.widget == "PopoverMenu" && panel == nil {
			panel = e
		}
		if e.role == RoleMenuItem && e.label == label {
			row = e
		}
		for c := e.first; c != nil; c = c.next {
			walk(c)
		}
	}
	if o := tt.rt.c.overlay; o != nil {
		walk(o)
	}
	return panel, row
}

func TestMenuStyleZeroKeepsTheLook(t *testing.T) {
	look := func(style *MenuStyle) (panel, row Rect, bg Color, radius, pad [4]float32) {
		a := &menuApp{}
		tt := coreNewTester(func(c *context) {
			c.Theme().Menu = style
			a.view(c)
		}, 400, 300)
		r, _ := tt.Find("Report.pdf")
		tt.RightClickAt(r.X+10, r.Y+r.H/2)
		p, w := menuNodes(tt, "Open")
		if p == nil || w == nil {
			t.Fatal("no drawn menu")
		}
		return p.Bounds(), w.Bounds(), w.bg, w.radius, w.pad
	}
	p0, r0, bg0, rad0, pad0 := look(nil)
	p1, r1, bg1, rad1, pad1 := look(&MenuStyle{})
	if p0 != p1 || r0 != r1 || bg0 != bg1 || rad0 != rad1 || pad0 != pad1 {
		t.Errorf("a zero MenuStyle changed the look: %+v %+v, was %+v %+v", p1, r1, p0, r0)
	}
}

func TestMenuStyleReachesPanelAndRows(t *testing.T) {
	var levels []int
	hi, hiText := RGB(10, 20, 30), RGB(250, 240, 230)
	style := &MenuStyle{
		Panel: func(p Element, level int) {
			levels = append(levels, level)
			p.Padding(9)
		},
		RowHeight: 40, RowRadius: 11, RowPadding: 17, Highlight: hi, HighlightText: hiText,
		Separator: RGB(1, 2, 3), TextSize: 20,
	}
	a := &menuApp{}
	tt := coreNewTester(func(c *context) {
		c.Theme().Menu = style
		a.view(c)
	}, 400, 300)
	r, _ := tt.Find("Report.pdf")
	tt.RightClickAt(r.X+10, r.Y+r.H/2)
	tt.Key(0, KeyDown) // the keys highlight the first item
	panel, row := menuNodes(tt, "Open")
	if panel == nil || row == nil {
		t.Fatal("no drawn menu")
	}
	if len(levels) == 0 || levels[0] != 0 {
		t.Errorf("Panel called for levels %v", levels)
	}
	if panel.pad != [4]float32{9, 9, 9, 9} {
		t.Errorf("panel padding %v, want Panel's 9", panel.pad)
	}
	if row.Bounds().H < 40 {
		t.Errorf("row height %v with RowHeight 40", row.Bounds().H)
	}
	if row.radius != [4]float32{11, 11, 11, 11} || row.pad[1] != 17 || row.pad[3] != 17 {
		t.Errorf("row radius %v, padding %v", row.radius, row.pad)
	}
	// Open or the item after it is highlighted: some row carries the color.
	found := false
	var walk func(e *node)
	walk = func(e *node) {
		if e.role == RoleMenuItem && e.bg == hi && e.ts.color == hiText {
			found = true
		}
		for c := e.first; c != nil; c = c.next {
			walk(c)
		}
	}
	walk(tt.rt.c.overlay)
	if !found {
		t.Error("no row has the Highlight and HighlightText")
	}
	sep := false
	var walkSep func(e *node)
	walkSep = func(e *node) {
		if e.bg == RGB(1, 2, 3) {
			sep = true
		}
		for c := e.first; c != nil; c = c.next {
			walkSep(c)
		}
	}
	walkSep(tt.rt.c.overlay)
	if !sep {
		t.Error("no separator line has the Separator color")
	}
}
