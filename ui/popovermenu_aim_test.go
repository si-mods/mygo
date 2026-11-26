package ui

import (
	"slices"
	"testing"
	"time"
)

// aimMenuApp is a row filling the window whose menu, drawn in the window,
// has a submenu taller than the items below the one opening it.
type aimMenuApp struct{}

func (a *aimMenuApp) view(c *context) {
	coreColumn(c).Fill().Padding(20).Children(func() {
		row := coreRow(c).FillWidth().Padding(8).Children(func() { coreText(c, "Report.pdf") })
		row.PopoverMenu(func(m *Menu) {
			m.Item("Open")
			m.Submenu("Move to", func(m *Menu) {
				for _, f := range []string{"Desktop", "Documents", "Downloads", "Music", "Pictures"} {
					m.Item(f)
				}
			})
			m.Item("Copy")
			m.Item("Rename")
		})
	})
}

// menuHighlights returns the labels of the items highlighted in the drawn
// menu, of the menu and of each submenu that shows.
func menuHighlights(tt *Tester) []string {
	d := tt.rt.menu.drawn
	if d == nil {
		return nil
	}
	var labels []string
	for _, ref := range d.highlighted {
		labels = append(labels, ref.label)
	}
	return labels
}

// menuLevelBox returns the box of the panel of the drawn menu at level, 0
// for the menu and 1 for its submenu.
func menuLevelBox(t *testing.T, tt *Tester, level int) Rect {
	t.Helper()
	d := tt.rt.menu.drawn
	if d == nil || level >= len(d.menu.levels) {
		t.Fatalf("no panel at level %d", level)
	}
	st := tt.rt.states[d.menu.levels[level].panel]
	if st == nil {
		t.Fatalf("the panel at level %d was not built", level)
	}
	return transformRect(st.world, Rect{st.x, st.y, st.w, st.h})
}

// textCenter returns the middle of the text s, which the test fails without.
func textCenter(t *testing.T, tt *Tester, s string) (x, y float32) {
	t.Helper()
	r, ok := tt.Find(s)
	if !ok {
		t.Fatalf("no %q: %q", s, tt.Texts())
	}
	return r.X + r.W/2, r.Y + r.H/2
}

// openMoveTo opens the menu anew with a right-click at (x, y), and the
// submenu of Move to with the pointer at the middle of its row, from where
// it heads on.
func openMoveTo(t *testing.T, tt *Tester, x, y float32) (px, py float32) {
	t.Helper()
	for i := 0; i < 3 && menuShows(tt); i++ {
		tt.Key(0, KeyEscape)
	}
	tt.RightClickAt(x, y)
	_, my := textCenter(t, tt, "Move to")
	box := menuLevelBox(t, tt, 0)
	px, py = box.X+box.W/2, my
	tt.Move(px, py)
	tt.Advance(submenuOpenDelay)
	if !tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Move to"}) {
		t.Fatalf("the submenu did not open on Move to: highlighted %q", menuHighlights(tt))
	}
	return px, py
}

// TestPopoverMenuSubmenuHoverIntent checks that hovering a submenu item
// highlights it immediately but waits submenuOpenDelay (150ms) before
// opening its submenu, so sweeping quickly across it opens nothing, while
// clicking or pressing Right/Enter/Space opens it immediately.
func TestPopoverMenuSubmenuHoverIntent(t *testing.T) {
	a := &aimMenuApp{}
	tt, now := clockTester(a.view, 600, 400)
	tick := ticker(tt, now)
	_, ry := textCenter(t, tt, "Report.pdf")
	tt.RightClickAt(40, ry)
	box := menuLevelBox(t, tt, 0)
	_, my := textCenter(t, tt, "Move to")
	_, cy := textCenter(t, tt, "Copy")
	mx := box.X + box.W/2

	// Sweeping quickly across Move to onto Copy opens nothing.
	tt.Move(mx, my)
	if tt.HasText("Desktop") {
		t.Fatal("submenu opened immediately on pointer hover")
	}
	if !slices.Equal(menuHighlights(tt), []string{"Move to"}) {
		t.Fatalf("hovering Move to did not highlight it: %q", menuHighlights(tt))
	}
	tick(submenuOpenDelay - 20*time.Millisecond)
	if tt.HasText("Desktop") {
		t.Fatal("submenu opened before submenuOpenDelay elapsed")
	}
	tt.Move(mx, cy)
	tick(500 * time.Millisecond)
	if tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Copy"}) {
		t.Fatalf("after sweeping to Copy: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}

	// Moving off the menu to empty space before 150ms cancels the pending open.
	tt.Move(mx, my)
	tick(submenuOpenDelay / 2)
	tt.Move(580, 380)
	tick(500 * time.Millisecond)
	if tt.HasText("Desktop") {
		t.Fatal("submenu opened after pointer left the menu before submenuOpenDelay")
	}

	// Resting on Move to for 150ms opens its submenu.
	tt.Move(mx, my)
	tick(submenuOpenDelay - time.Millisecond)
	if tt.HasText("Desktop") {
		t.Fatal("submenu opened 1ms before submenuOpenDelay")
	}
	tick(time.Millisecond)
	if !tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Move to"}) {
		t.Fatalf("resting 150ms on Move to did not open submenu: shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}

	// Clicking Move to opens its submenu immediately (0ms).
	tt.Move(mx, cy)
	if tt.HasText("Desktop") {
		t.Fatal("moving straight down to Copy did not close submenu")
	}
	tt.ClickAt(mx, my)
	if !tt.HasText("Desktop") {
		t.Fatal("clicking Move to did not open its submenu immediately")
	}

	// Keyboard Right opens the submenu immediately (0ms), and Left closes it.
	tt.Key(0, KeyDown) // onto Copy, closing Move to's submenu
	if tt.HasText("Desktop") {
		t.Fatal("Down did not close submenu")
	}
	tt.Key(0, KeyUp) // back onto Move to
	if tt.HasText("Desktop") {
		t.Fatal("Up onto Move to opened submenu without Right")
	}
	tt.Key(0, KeyRight)
	if !tt.HasText("Desktop") {
		t.Fatal("Right did not open submenu immediately")
	}
	tt.Key(0, KeyLeft)
	if tt.HasText("Desktop") {
		t.Fatal("Left did not close submenu")
	}
}

// TestPopoverMenuAimsAtSubmenu checks the pointer moving across the items
// of a menu toward the submenu open from it keeps the submenu, and the
// highlight where it is, until it rests on an item (#1916).
func TestPopoverMenuAimsAtSubmenu(t *testing.T) {
	a := &aimMenuApp{}
	tt := coreNewTester(a.view, 600, 400)
	_, ry := textCenter(t, tt, "Report.pdf")
	openMoveTo(t, tt, 40, ry)
	box, sub := menuLevelBox(t, tt, 0), menuLevelBox(t, tt, 1)
	if sub.X < box.X+box.W/2 {
		t.Fatalf("the submenu at %+v does not open after the menu at %+v", sub, box)
	}

	// Down and across onto Copy, toward the submenu: Copy takes no highlight.
	_, cy := textCenter(t, tt, "Copy")
	cx := box.X + box.W - 10
	tt.Move(cx, cy)
	if !tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Move to"}) {
		t.Fatalf("crossing Copy toward the submenu: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}
	// Resting there, Copy takes it once the grace ends, and the submenu closes.
	tt.Advance(menuAimGrace - time.Millisecond)
	if !tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Move to"}) {
		t.Fatalf("before the grace ends: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}
	tt.Advance(2 * time.Millisecond)
	if tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Copy"}) {
		t.Fatalf("resting on Copy past the grace: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}

	// Moving on toward it, the grace starts over, and the pointer reaches it.
	px, _ := openMoveTo(t, tt, 40, ry)
	tt.Move(px+(cx-px)/2, cy-2) // onto Copy
	tt.Advance(menuAimGrace / 2)
	tt.Move(cx, cy)
	tt.Advance(menuAimGrace * 3 / 4)
	if !tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Move to"}) {
		t.Fatalf("moving on toward the submenu: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}
	tt.Move(textCenter(t, tt, "Documents"))
	if !slices.Equal(menuHighlights(tt), []string{"Move to", "Documents"}) {
		t.Fatalf("reaching the submenu: highlighted %q", menuHighlights(tt))
	}
	tt.Advance(time.Second)
	if !tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Move to", "Documents"}) {
		t.Fatalf("resting in the submenu: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}

	// Moving away from the submenu, the item under the pointer takes the
	// highlight at once.
	openMoveTo(t, tt, 40, ry)
	tt.Move(box.X+10, cy)
	if tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Copy"}) {
		t.Fatalf("moving away from the submenu: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}
	// As it does moving straight down, outside the triangle.
	px, _ = openMoveTo(t, tt, 40, ry)
	_, rny := textCenter(t, tt, "Rename")
	tt.Move(px+4, rny)
	if tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Rename"}) {
		t.Fatalf("moving down to Rename: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}

	// The keys move the highlight at once, as ever.
	openMoveTo(t, tt, 40, ry)
	tt.Key(0, KeyDown)
	if tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Copy"}) {
		t.Fatalf("Down with the submenu open: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}
}

// TestPopoverMenuAimsAtSubmenuOnTheLeft checks a submenu that opens before
// its menu, where the window has no room after it, keeps open as the
// pointer heads left for it.
func TestPopoverMenuAimsAtSubmenuOnTheLeft(t *testing.T) {
	a := &aimMenuApp{}
	tt := coreNewTester(a.view, 600, 400)
	_, ry := textCenter(t, tt, "Report.pdf")
	openMoveTo(t, tt, 570, ry)
	box, sub := menuLevelBox(t, tt, 0), menuLevelBox(t, tt, 1)
	if sub.X+sub.W > box.X+box.W/2 {
		t.Fatalf("the submenu at %+v does not open before the menu at %+v", sub, box)
	}

	_, cy := textCenter(t, tt, "Copy")
	tt.Move(box.X+10, cy)
	if !tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Move to"}) {
		t.Fatalf("crossing Copy toward the submenu: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}
	tt.Advance(menuAimGrace + time.Millisecond)
	if tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Copy"}) {
		t.Fatalf("resting on Copy past the grace: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}

	// Heading right, away from it, Copy takes the highlight at once.
	openMoveTo(t, tt, 570, ry)
	tt.Move(box.X+box.W-10, cy)
	if tt.HasText("Desktop") || !slices.Equal(menuHighlights(tt), []string{"Copy"}) {
		t.Fatalf("moving away from the submenu: submenu shows %v, highlighted %q", tt.HasText("Desktop"), menuHighlights(tt))
	}
}
