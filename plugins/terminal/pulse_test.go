package terminal

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestSelectionPulse checks that a selection lights up as the drag that
// makes it ends, crosses it, and settles: a frame in the middle of the
// pulse is brighter over the selection than the plain selection is
// before and after it.
func TestSelectionPulse(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("hello world\r\nsecond line\r\nthird line"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	x0, y0 := cellCenter(term, 0, 0)
	x1, y1 := cellCenter(term, 8, 0)
	tt.Press(x0, y0)
	tt.Frame()
	tt.Move(x1, y1)
	tt.Frame()
	tt.Release(x1, y1)
	tt.Frame()
	plain := brightness(term, tt.Image())
	tt.Advance(pulseLife / 2)
	tt.Frame()
	lit := brightness(term, tt.Image())
	if lit <= plain {
		t.Fatalf("the selection is %.1f bright at rest and %.1f in the middle of the pulse", plain, lit)
	}
	tt.Advance(pulseLife)
	tt.Frame()
	after := brightness(term, tt.Image())
	if after != plain {
		t.Errorf("the selection is %.1f bright before the pulse and %.1f after it", plain, after)
	}
}

// TestSelectionPulseAfterCopy checks that copying a selection, which
// leaves it showing, starts the light too.
func TestSelectionPulseAfterCopy(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("hello world\r\nsecond line\r\nthird line"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	x0, y0 := cellCenter(term, 0, 0)
	x1, y1 := cellCenter(term, 8, 0)
	tt.Press(x0, y0)
	tt.Frame()
	tt.Move(x1, y1)
	tt.Frame()
	tt.Release(x1, y1)
	tt.Frame()
	// Past the light the drag started, so that the copy's is its own.
	tt.Advance(2 * pulseLife)
	tt.Frame()
	plain := brightness(term, tt.Image())
	tt.Command("copy")
	tt.Frame()
	tt.Advance(pulseLife / 2)
	tt.Frame()
	if got := brightness(term, tt.Image()); got <= plain {
		t.Fatalf("the selection is %.1f bright at rest and %.1f after the copy", plain, got)
	}
}

// brightness returns the mean brightness of the line the term's grid
// shows first, the row a selection covers in these tests.
func brightness(term *Terminal, img *image.RGBA) float64 {
	v := term.v
	if v == nil {
		return 0
	}
	sum, n := 0.0, 0
	for x := v.ox; x < v.ox+v.cellW*9; x++ {
		for y := v.oy + v.cellH/4; y < v.oy+v.cellH*3/4; y++ {
			c := img.RGBAAt(x, y)
			sum += float64(c.R) + float64(c.G) + float64(c.B)
			n += 3
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// TestSelectionPulseFollowsDrag checks that the light crosses a
// selection as the drag makes it, not only once the pointer lets go.
func TestSelectionPulseFollowsDrag(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("hello world\r\nsecond line\r\nthird line"))
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 400, 200)
	x0, y0 := cellCenter(term, 0, 0)
	x1, y1 := cellCenter(term, 8, 0)
	tt.Press(x0, y0)
	tt.Frame()
	tt.Move(x1, y1)
	tt.Frame()
	plain := brightness(term, tt.Image())
	tt.Advance(pulseLife / 2)
	tt.Frame()
	if got := brightness(term, tt.Image()); got <= plain {
		t.Fatalf("the selection is %.1f bright while the drag is held and %.1f halfway through the pulse", plain, got)
	}
	tt.Release(x1, y1)
	tt.Frame()
}
