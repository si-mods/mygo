package terminal

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// A terminal sized by its server (Options.ResizeByConn) can have a screen
// taller than the view that shows it, as a session's does when another
// machine's window is larger. The view draws the screen's last rows, not
// its first: the cursor and the prompts programs draw are at the bottom,
// and up there nothing of them would show.
func TestViewShowsTheBottomOfATallerScreen(t *testing.T) {
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
	if want := screenRows - v.rows; v.top != want {
		t.Errorf("the view starts at row %d of %d shown, want %d", v.top, v.rows, want)
	}

	// Ink on the screen's first row and on its last: the view shows the
	// last, at its bottom.
	term.Feed([]byte("\x1b[1;1H" + strings.Repeat(" ", cols) + "TOP"))
	term.Feed([]byte("\x1b[40;1H" + strings.Repeat(" ", cols) + "BOTTOM"))
	tt.Frame()
	img := tt.Image()
	bg := img.RGBAAt(v.ox+v.cellW/2, v.oy+2)
	ink := func(row int) bool {
		y := v.oy + row*v.cellH
		for x := range v.cellW * cols {
			for dy := range v.cellH {
				if img.RGBAAt(v.ox+x, y+dy) != bg {
					return true
				}
			}
		}
		return false
	}
	if ink(0) {
		t.Error("the view's first row shows ink: it drew the top of the screen")
	}
	if !ink(v.rows - 1) {
		t.Error("the view's last row is empty: the bottom of the screen is out of sight")
	}

	// The pointer maps to the rows it shows: the cell under the view's
	// bottom row is the screen's last.
	if _, row := v.cellAt(v.size[0]/2, v.size[1]-float32(v.cellH)/2); row != screenRows-1 {
		t.Errorf("the view's last row is the screen's %d, want %d", row, screenRows-1)
	}
}
