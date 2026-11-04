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

// focusedText returns the state of the text input the Tester focused.
func focusedText(t *testing.T, tt *Tester) *state {
	t.Helper()
	s := tt.rt.states[tt.rt.focused]
	if s == nil || s.editor == nil {
		t.Fatal("no text input has the focus")
	}
	return s
}

// TestDrawMenusEditMenuAsksForAFrame checks that the editing menu asked
// for in the last pass allowed, where the frame ends with it, still has a
// frame requested, which builds it. A frame runs here (inFrame), where
// requestFrame does nothing.
func TestDrawMenusEditMenuAsksForAFrame(t *testing.T) {
	a := &drawApp{draw: true, value: "hello"}
	tt := coreNewTester(a.view, 400, 300)
	tt.Click("Name")
	s := focusedText(t, tt)
	tt.h.requested.Store(false)
	tt.rt.inFrame, tt.rt.pass = true, 2
	tt.rt.editMenu(s)
	tt.rt.inFrame = false
	if tt.rt.menu.edit == nil {
		t.Fatal("the editing menu was not kept for a pass")
	}
	if !tt.h.requested.Load() {
		t.Fatal("no frame was requested: the menu waits for an unrelated event")
	}
	tt.settle()
	if !tt.HasText("Select All") {
		t.Errorf("the next frame drew no editing menu: %q", tt.Texts())
	}
}

// TestDrawMenusEditMenuKeepsWhereAsked checks that a menu asked for again
// before the editing menu opens neither moves nor retargets it.
func TestDrawMenusEditMenuKeepsWhereAsked(t *testing.T) {
	a := &drawApp{draw: true, value: "hello"}
	tt := coreNewTester(a.view, 400, 300)
	tt.Click("Name")
	s := focusedText(t, tt)
	rt := tt.rt
	b := transformRect(s.world, Rect{s.x, s.y, s.w, s.h})
	rt.menu.x, rt.menu.y, rt.menu.text, rt.menu.key = b.X+13, b.Y+7, s.id, false
	rt.editMenu(s)
	// Another menu is asked for before the pass that builds this one.
	rt.menu.x, rt.menu.y, rt.menu.text, rt.menu.key, rt.menu.below = b.X+99, b.Y+50, 12345, true, true
	tt.Frame()
	d := rt.menu.drawn
	if d == nil {
		t.Fatal("no drawn menu")
	}
	if d.menu.dx != 13 || d.menu.dy != 7 || d.text != s.id || d.menu.below {
		t.Errorf("menu opened at (%v, %v) on text %d below %v, want (13, 7) on %d, not below", d.menu.dx, d.menu.dy, d.text, d.menu.below, s.id)
	}
	if d.menu.levels[0].hi != 0 {
		t.Errorf("the keyboard's start applied: hi %d", d.menu.levels[0].hi)
	}
}

// TestDrawMenusEditMenuOfAnElementNotBuiltYet checks that the editing
// menu of a text input that the view's tree does not hold when the menu
// opens, as a toast's or the inspector's, built after the view, falls back
// to the system's menu rather than vanishing. The element is simulated by
// an id the tree lacks.
func TestDrawMenusEditMenuOfAnElementNotBuiltYet(t *testing.T) {
	a := &drawApp{draw: true, value: "hello"}
	tt := coreNewTester(a.view, 400, 300)
	tt.Click("Name")
	s := focusedText(t, tt)
	rt := tt.rt
	rt.editMenu(s)
	rt.menu.edit.id = 987654321 // not in the view or its overlays
	tt.Frame()
	if got := tt.Menu(); len(got) == 0 {
		t.Fatal("no menu at all: the editing menu was dropped")
	}
	if tt.HasText("Select All") {
		t.Error("a drawn menu shows for an element that is not built")
	}
}

// TestMenuStylePanelRunsAfterTheLayout checks that Panel may change what
// MyGo set for the menu's layout, its margin included.
func TestMenuStylePanelRunsAfterTheLayout(t *testing.T) {
	style := &MenuStyle{Panel: func(p Element, level int) { p.Margin(33) }}
	a := &buttonMenuApp{}
	tt := coreNewTester(func(c *context) {
		c.Theme().Menu = style
		a.view(c)
	}, 400, 300)
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	panel, _ := menuNodes(tt, "Open")
	if panel == nil {
		t.Fatal("no drawn menu")
	}
	if panel.margin != [4]float32{33, 33, 33, 33} {
		t.Errorf("panel margin %v, want Panel's 33 on every side", panel.margin)
	}
}
