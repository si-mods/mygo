//go:build linux && (amd64 || arm64)

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// windowShape is a window's bounding shape on the X server, in its
// pixels.
type windowShape [][4]int

func shapeOf(w *mygo.Window) (s windowShape, ok bool) {
	mygo.RunOnMain(func() { s, ok = linux.TestWindowShape(w.NativeHandle()) })
	return s, ok
}

// has reports whether the shape shows the pixel at x, y.
func (s windowShape) has(x, y int) bool {
	for _, r := range s {
		if x >= r[0] && x < r[0]+r[2] && y >= r[1] && y < r[1]+r[3] {
			return true
		}
	}
	return false
}

// size returns the width and height the shape spans.
func (s windowShape) size() (width, height int) {
	for _, r := range s {
		width, height = max(width, r[0]+r[2]), max(height, r[1]+r[3])
	}
	return width, height
}

// TestCornerRadiusX11 rounds the corners of undecorated windows on X11,
// a window with a hidden title bar's as a frameless one's. Without a
// compositor, the X window's bounding shape cuts them, which the X server
// shows the window inside; with one, the window has an alpha channel and
// its content cuts its frames to them, with the window unshaped. Either
// follows the window's size, goes while it is maximized or in full
// screen, and comes back after; other windows keep square corners.
func TestCornerRadiusX11(t *testing.T) {
	content := func() *ui.Content { return ui.View(func(c *ui.Context) { ui.Text(c, "corners") }) }
	// show shows a window, on an X server with SHAPE.
	show := func(t *testing.T, opts mygo.WindowOptions) *mygo.Window {
		t.Helper()
		opts.Content = content()
		w := newWindow(t, opts)
		eventually(t, "the window shown", w.IsVisible)
		if _, ok := shapeOf(w); !ok || !decorations(w).OnX11 {
			t.Skip("needs an X server with SHAPE")
		}
		return w
	}
	corners := func(w *mygo.Window) (alpha bool, radius float64) {
		mygo.RunOnMain(func() { alpha, radius = linux.TestWindowCorners(w.NativeHandle()) })
		return alpha, radius
	}

	// cut tells whether the shape spans width by height, scaled, with its
	// corners cut by radius and its edges kept past them.
	cut := func(s windowShape, width, height, radius int) bool {
		sw, sh := s.size()
		if sw == 0 || sw%width != 0 || sh != sw/width*height {
			return false
		}
		k := sw / width // GDK_SCALE
		r := radius * k
		for _, corner := range [][2]int{{0, 0}, {sw - 1, 0}, {0, sh - 1}, {sw - 1, sh - 1}} {
			if s.has(corner[0], corner[1]) {
				return false
			}
		}
		return s.has(r, 0) && s.has(sw-1-r, sh-1) && s.has(0, r) && s.has(sw-1, sh-1-r) && s.has(sw/2, sh/2)
	}
	// whole tells that the window shows to its corners.
	whole := func(s windowShape) bool {
		sw, sh := s.size()
		return sw > 0 && s.has(0, 0) && s.has(sw-1, 0) && s.has(0, sh-1) && s.has(sw-1, sh-1)
	}
	is := func(what string, cond func() bool) {
		t.Helper()
		if !wait.Poll(cond) {
			t.Fatalf("%s never happened", what)
		}
	}
	// rounded tells that the window of width by height has its corners
	// rounded by 12, by its shape or by its content; square that it has
	// them square.
	rounded := func(w *mygo.Window, width, height int) func() bool {
		return func() bool {
			s, _ := shapeOf(w)
			if alpha, radius := corners(w); alpha {
				sw, _ := s.size()
				return radius == 12 && whole(s) && sw%width == 0
			}
			return cut(s, width, height, 12)
		}
	}
	square := func(w *mygo.Window) func() bool {
		return func() bool {
			s, _ := shapeOf(w)
			_, radius := corners(w)
			return radius == 0 && whole(s)
		}
	}

	for _, tc := range []struct {
		name string
		opts mygo.WindowOptions
	}{
		{"hidden title bar", mygo.WindowOptions{TitleBarStyle: mygo.TitleBarHidden}},
		{"frameless", mygo.WindowOptions{Frameless: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts
			opts.Width, opts.Height, opts.CornerRadius = 400, 300, 12
			w := show(t, opts)
			alpha, _ := corners(w)
			t.Logf("content cuts the corners (a compositor runs): %v", alpha)
			is("corners of 12 on 400x300", rounded(w, 400, 300))

			w.SetSize(500, 360)
			is("corners of 12 on 500x360", rounded(w, 500, 360))

			// The input region, the input shape, leaves the corners be.
			w.SetInputRegion([]mygo.Rectangle{{X: 0, Y: 0, Width: 100, Height: 100}})
			is("corners kept with an input region", rounded(w, 500, 360))
			w.SetInputRegion(nil)

			var managed bool
			mygo.RunOnMain(func() { managed = linux.TestWindowManager(w.NativeHandle()) })
			if !managed {
				t.Log("no window manager to maximize the window, as under Xvfb alone: states not checked")
				return
			}
			w.Maximize()
			eventually(t, "a maximized window", w.IsMaximized)
			is("a maximized window with square corners", square(w))
			w.Unmaximize()
			eventually(t, "an unmaximized window", func() bool { return !w.IsMaximized() })
			is("the corners rounded again", rounded(w, 500, 360))

			w.SetFullScreen(true)
			eventually(t, "a window in full screen", w.IsFullScreen)
			is("a window in full screen with square corners", square(w))
			w.SetFullScreen(false)
			eventually(t, "a window out of full screen", func() bool { return !w.IsFullScreen() })
			is("the corners rounded again", rounded(w, 500, 360))
		})
	}

	// Windows with a title bar, transparent ones and those without a
	// radius keep their corners.
	for _, tc := range []struct {
		name string
		opts mygo.WindowOptions
	}{
		{"title bar", mygo.WindowOptions{CornerRadius: 12}},
		{"transparent", mygo.WindowOptions{Frameless: true, Transparent: true, CornerRadius: 12}},
		{"no radius", mygo.WindowOptions{Frameless: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts
			opts.Width, opts.Height = 400, 300
			w := show(t, opts)
			is("a window shown", func() bool { s, _ := shapeOf(w); width, _ := s.size(); return width > 0 })
			if s, _ := shapeOf(w); !whole(s) {
				t.Fatalf("shape %v, want the whole window", s)
			}
			if alpha, radius := corners(w); alpha || radius != 0 {
				t.Fatalf("content cuts corners: %v, radius %g; want square ones", alpha, radius)
			}
		})
	}
}
