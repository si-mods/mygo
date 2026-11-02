package terminal

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A terminal sized by its server (Options.ResizeByConn) can have a screen
// taller than the view that shows it, as a session's does when another
// machine's window is larger. The view draws the rows around the cursor:
// where the cursor is, which a program with output at the top of a screen
// of its own keeps high, and the screen's last rows when it is at the
// bottom, where the prompts programs draw are. The rows above them stay
// out of sight.
func TestViewShowsTheRowsAroundTheCursorOfATallerScreen(t *testing.T) {
	loadLib(t)
	// A Conn told the size the view fits, which does not resize the screen.
	sizes := make(chan [2]int, 16)
	term, err := New(Options{Conn: sizedPipe{newPipe(), sizes}, ResizeByConn: true, Theme: LightTheme()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	const cols, screenRows = 20, 40
	term.Resize(cols, screenRows)
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 300, 200)
	tt.Frame()
	v := term.v
	if v.rows >= screenRows {
		t.Fatalf("the view shows %d rows of the screen's %d: it fits", v.rows, screenRows)
	}
	img := tt.Image()
	bg := img.RGBAAt(v.ox+v.cellW/2, v.oy+2)
	// ink reports whether the view's row has anything but the background.
	ink := func(row int) bool {
		y := v.oy + row*v.cellH
		for x := range v.cellW * cols {
			for dy := range v.cellH {
				if tt.Image().RGBAAt(v.ox+x, y+dy) != bg {
					return true
				}
			}
		}
		return false
	}

	// The cursor at the screen's top, the output a shell wrote there: the
	// view starts at the screen's own first row.
	term.Feed([]byte("\x1b[1;1HTOP"))
	tt.Frame()
	if v.top != 0 {
		t.Errorf("the cursor at the top, the view starts at row %d, want 0", v.top)
	}
	if !ink(0) {
		t.Error("the view's first row is empty: it drew below the cursor")
	}

	// The cursor at the screen's last row: the view ends with it.
	term.Feed([]byte("\x1b[40;1HBOTTOM"))
	tt.Frame()
	if want := screenRows - v.rows; v.top != want {
		t.Errorf("the cursor at the bottom, the view starts at row %d, want %d", v.top, want)
	}
	if ink(0) {
		t.Error("the view's first row shows ink: it drew the top of the screen")
	}
	if !ink(v.rows - 1) {
		t.Error("the view's last row is empty: the bottom of the screen is out of sight")
	}

	// The pointer maps to the rows the view shows: the view's last row is
	// the screen's last, with the cursor there.
	if _, row := v.cellAt(v.size[0]/2, v.size[1]-float32(v.cellH)/2); row != screenRows-1 {
		t.Errorf("the view's last row is the screen's %d, want %d", row, screenRows-1)
	}
}
