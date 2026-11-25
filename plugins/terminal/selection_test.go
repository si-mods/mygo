package terminal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestShiftClickExtendsSelection(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), CopyOnSelect: CopyOnSelectClipboard})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("hello world\r\nsecond line\r\nthird line"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	x, y := cellCenter(term, 0, 0)
	tt.Press(x-2, y)
	x, y = cellCenter(term, 4, 0)
	tt.Move(x+2, y)
	tt.Release(x+2, y)
	if got := tt.Clipboard(); got != "hello" {
		t.Fatalf("initial selection = %q", got)
	}
	for _, step := range []struct {
		col, row int
		want     string
	}{
		{5, 1, "hello world\nsecond"},
		{9, 2, "hello world\nsecond line\nthird line"},
		{2, 0, "hel"},
	} {
		x, y = cellCenter(term, step.col, step.row)
		tt.ClickAtWith(ui.Shift, x, y)
		if got := tt.Clipboard(); got != step.want {
			t.Errorf("Shift-click (%d,%d) copied %q, want %q", step.col, step.row, got, step.want)
		}
		tt.SetClipboard("sentinel")
		tt.Command("copy")
		if got := tt.Clipboard(); got != step.want {
			t.Errorf("Copy after Shift-click = %q, want %q", got, step.want)
		}
	}
}

func TestShiftClickKeepsSelectionAnchor(t *testing.T) {
	loadLib(t)
	for _, mode := range []string{"reverse", "word", "line", "copy off", "output", "mouse tracking"} {
		t.Run(mode, func(t *testing.T) {
			term, err := New(Options{Conn: newPipe()})
			if err != nil {
				t.Fatal(err)
			}
			defer term.Close()
			term.Feed([]byte("hello world\r\nsecond line\r\nthird line"))
			tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
			x, y := cellCenter(term, 0, 0)
			want := "hello world\nsecond"
			switch mode {
			case "reverse":
				x, y = cellCenter(term, 4, 1)
				tt.Press(x+2, y)
				x, y = cellCenter(term, 0, 1)
				tt.Move(x-2, y)
				tt.Release(x-2, y)
				want = "hello world\nsecon"
			case "word", "line":
				clicks := 2
				if mode == "line" {
					clicks = 3
				}
				for range clicks {
					tt.ClickAt(x, y)
				}
			default:
				tt.Press(x-2, y)
				x, y = cellCenter(term, 4, 0)
				tt.Move(x+2, y)
				tt.Release(x+2, y)
			}
			if mode == "output" {
				term.Feed([]byte("\r\nmore output"))
				tt.Frame()
			}
			if mode == "mouse tracking" {
				term.Feed([]byte("\x1b[?1000h\x1b[?1006h"))
			}
			x, y = cellCenter(term, 5, 1)
			if mode == "reverse" {
				x, y = cellCenter(term, 0, 0)
			}
			tt.SetClipboard("untouched")
			tt.ClickAtWith(ui.Shift, x, y)
			if got := tt.Clipboard(); got != "untouched" {
				t.Errorf("copy-on-select disabled: clipboard = %q", got)
			}
			tt.Command("copy")
			if got := tt.Clipboard(); got != want {
				t.Errorf("extended selection = %q, want %q", got, want)
			}
			term.Feed([]byte("\x1b[?1000l\x1b[?1006l"))
			// An ordinary click clears the selection. Shift-click must not
			// revive the previous selection's anchor after it is gone.
			x, y = cellCenter(term, 3, 2)
			tt.ClickAt(x, y)
			x, y = cellCenter(term, 8, 2)
			tt.ClickAtWith(ui.Shift, x, y)
			tt.SetClipboard("untouched")
			tt.Command("copy")
			if got := tt.Clipboard(); got != "untouched" {
				t.Errorf("Shift-click revived a cleared selection: %q", got)
			}
		})
	}
}

func TestShiftClickDragAutoscroll(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), CopyOnSelect: CopyOnSelectClipboard})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	for row := range 40 {
		term.Feed(fmt.Appendf(nil, "row%02d\r\n", row))
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	term.mu.Lock()
	term.term.ScrollToTop()
	term.mu.Unlock()
	tt.Frame()
	x, y := cellCenter(term, 0, 0)
	tt.Press(x-2, y)
	x, y = cellCenter(term, 4, 0)
	tt.Move(x+2, y)
	tt.Release(x+2, y)
	if got := tt.Clipboard(); got != "row00" {
		t.Fatalf("initial selection = %q", got)
	}
	v := term.v
	x, y = cellCenter(term, 4, 1)
	// A held Shift-click, followed by dragging beyond the bottom edge.
	// Use the view's real input handler so the button remains pressed.
	v.input(ui.InputEvent{Kind: ui.InputPointerDown, Mods: ui.Shift, X: x, Y: y})
	v.input(ui.InputEvent{Kind: ui.InputPointerMove, X: x, Y: 240})
	tt.Frame()
	tt.Advance(120 * time.Millisecond)
	v.input(ui.InputEvent{Kind: ui.InputPointerUp, X: x, Y: 240})
	got := tt.Clipboard()
	if !strings.HasPrefix(got, "row00\nrow01\n") || !strings.Contains(got, fmt.Sprintf("row%02d", v.rows)) {
		t.Errorf("drag did not keep the anchor and scroll to another row: %q", got)
	}
	if v.selecting || v.ticking {
		t.Error("release left selection dragging or autoscroll active")
	}
}

func TestWideCharSelectionCoversFullCells(t *testing.T) {
	loadLib(t)
	selColor := ui.RGB(0x20, 0x60, 0xc0)
	term, err := New(Options{
		Conn:    newPipe(),
		Theme:   &Theme{Background: ui.RGB(0x10, 0x10, 0x10), Foreground: ui.RGB(0xee, 0xee, 0xee), Selection: selColor},
		NoBlink: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	// Each CJK rune occupies 2 cells: 用(0,1) 绝(2,3) 对(4,5) 路(6,7) 径(8,9) 。(10,11).
	term.Feed([]byte("用绝对路径。\r\n用绝对路径。"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)

	clearSel := func() {
		x, y := cellCenter(term, 20, 0)
		tt.ClickAt(x, y)
	}

	assertRowSel := func(name string, row, wantA, wantB int, wantText string) {
		t.Helper()
		tt.Frame()
		term.mu.Lock()
		v := term.v
		line := v.lines[row]
		text, hasSel := term.term.SelectionText()
		term.mu.Unlock()
		if !hasSel || !line.sel {
			t.Fatalf("%s: expected selection, got hasSel=%v line.sel=%v", name, hasSel, line.sel)
		}
		if line.selA != wantA || line.selB != wantB {
			t.Errorf("%s: row %d selection = [%d..%d], want [%d..%d] (text=%q)", name, row, line.selA, line.selB, wantA, wantB, text)
		}
		if text != wantText {
			t.Errorf("%s: SelectionText = %q, want %q", name, text, wantText)
		}
		// Verify the painted selection covers the right half of the last wide cell too.
		img := tt.Image()
		py := v.oy + row*v.cellH + 1
		px := v.ox + wantB*v.cellW + v.cellW/2
		got := img.RGBAAt(px, py)
		if got.R != selColor.R || got.G != selColor.G || got.B != selColor.B {
			t.Errorf("%s: pixel at col %d (%d,%d) = %v, want selection color %v", name, wantB, px, py, got, selColor)
		}
	}

	cases := []struct {
		name     string
		c0       int
		dx0      float32
		c1       int
		dx1      float32
		wantA    int
		wantB    int
		wantText string
	}{
		{"forward left-to-head", 0, -2, 6, +2, 0, 7, "用绝对路"},
		{"forward left-to-tail", 0, -2, 7, -2, 0, 7, "用绝对路"},
		{"forward single char head", 6, -2, 6, +2, 6, 7, "路"},
		{"forward single char tail", 7, -2, 7, +2, 6, 7, "路"},
		{"forward from spacer", 1, +2, 6, +2, 2, 7, "绝对路"},
		{"reverse head-to-tail", 8, +2, 7, -2, 6, 9, "路径"},
		{"reverse single char head", 8, +2, 8, -2, 8, 9, "径"},
		{"reverse tail-to-head", 9, +2, 7, -2, 6, 9, "路径"},
		{"reverse single char tail", 7, +2, 6, -2, 6, 7, "路"},
	}
	for _, tc := range cases {
		clearSel()
		x0, y0 := cellCenter(term, tc.c0, 0)
		x1, y1 := cellCenter(term, tc.c1, 0)
		tt.Press(x0+tc.dx0, y0)
		tt.Move(x1+tc.dx1, y1)
		tt.Release(x1+tc.dx1, y1)
		assertRowSel(tc.name, 0, tc.wantA, tc.wantB, tc.wantText)
	}

	// Shift-click extend forward onto wide char head (col 6) -> covers [0..7].
	clearSel()
	x0, y0 := cellCenter(term, 0, 0)
	x1, y1 := cellCenter(term, 3, 0)
	tt.Press(x0-2, y0)
	tt.Move(x1+2, y1)
	tt.Release(x1+2, y1)
	x, y := cellCenter(term, 6, 0)
	tt.ClickAtWith(ui.Shift, x, y)
	assertRowSel("shift-click forward to head", 0, 0, 7, "用绝对路")

	// Shift-click extend backward onto spacer tail (col 7) -> covers [6..9].
	clearSel()
	x0, y0 = cellCenter(term, 8, 0)
	x1, y1 = cellCenter(term, 9, 0)
	tt.Press(x0-2, y0)
	tt.Move(x1+2, y1)
	tt.Release(x1+2, y1)
	x, y = cellCenter(term, 7, 0)
	tt.ClickAtWith(ui.Shift, x, y)
	assertRowSel("shift-click reverse to tail", 0, 6, 9, "路径")

	// Rectangle selection (Alt+drag) forward and reverse across 2 rows.
	clearSel()
	x0, y0 = cellCenter(term, 1, 0)
	x1, y1 = cellCenter(term, 6, 1)
	term.v.input(ui.InputEvent{Kind: ui.InputPointerDown, Mods: ui.Alt, X: x0 - 2, Y: y0})
	term.v.input(ui.InputEvent{Kind: ui.InputPointerMove, Mods: ui.Alt, X: x1 + 2, Y: y1})
	term.v.input(ui.InputEvent{Kind: ui.InputPointerUp, Mods: ui.Alt, X: x1 + 2, Y: y1})
	assertRowSel("rect forward", 0, 0, 7, "用绝对路\n用绝对路")
	assertRowSel("rect forward row 1", 1, 0, 7, "用绝对路\n用绝对路")

	clearSel()
	x0, y0 = cellCenter(term, 8, 1)
	x1, y1 = cellCenter(term, 1, 0)
	term.v.input(ui.InputEvent{Kind: ui.InputPointerDown, Mods: ui.Alt, X: x0 + 2, Y: y0})
	term.v.input(ui.InputEvent{Kind: ui.InputPointerMove, Mods: ui.Alt, X: x1 - 2, Y: y1})
	term.v.input(ui.InputEvent{Kind: ui.InputPointerUp, Mods: ui.Alt, X: x1 - 2, Y: y1})
	assertRowSel("rect reverse", 0, 0, 9, "用绝对路径\n用绝对路径")
	assertRowSel("rect reverse row 1", 1, 0, 9, "用绝对路径\n用绝对路径")

	// Search highlight on wide characters also covers the full trailing spacer.
	clearSel()
	term.Search("路径")
	tt.Frame()
	tt.Frame()
	term.mu.Lock()
	matches := append([]matchSpan(nil), term.v.matches...)
	term.mu.Unlock()
	if len(matches) != 2 {
		t.Fatalf("search matches = %+v, want 2", matches)
	}
	for _, m := range matches {
		if m.x0 != 6 || m.x1 != 9 {
			t.Errorf("search match on row %d = [%d..%d], want [6..9]", m.y, m.x0, m.x1)
		}
	}
}
