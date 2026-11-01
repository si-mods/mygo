//go:build darwin

package e2e

import (
	"math"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
	"github.com/egoist/mygo/ui"
)

// glassPanes returns the panes of the system's glass of w.
func glassPanes(w *mygo.Window) (panes []darwin.TestGlassPane) {
	mygo.RunOnMain(func() { panes = darwin.TestSystemGlass(w.NativeHandle()) })
	return panes
}

// TestSystemGlass shows the system's Liquid Glass behind an element over
// the window's material (macOS 26): an NSGlassEffectView under the
// surface, over the material, at the element's box, with its corners and
// tint, which moves with the element and hides once it is gone.
func TestSystemGlass(t *testing.T) {
	var left atomic.Int32
	var shown atomic.Bool
	left.Store(20)
	shown.Store(true)
	view := ui.View(func(c *ui.Context) {
		c.Root().Background(ui.Transparent)
		if shown.Load() {
			ui.Box(c).Absolute().Left(float32(left.Load())).Top(30).Size(120, 40).Radius(16.5).
				Material(ui.SystemGlass{Tint: ui.RGBA(29, 29, 29, 0.25)}).Children(func() {
				ui.Text(c, "Selected").Padding(10)
			})
		}
	})
	w := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200, Vibrancy: mygo.VibrancySidebar, Content: view})
	w.Show()
	if !darwin.TestSupportsGlass() {
		t.Skip("no NSGlassEffectView before macOS 26")
	}
	near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
	at := func(x float64) func() bool {
		return func() bool {
			p := glassPanes(w)
			return len(p) == 1 && p[0].Shown && near(p[0].Frame.X, x) && near(p[0].Frame.Y, 30) && near(p[0].Frame.W, 120) && near(p[0].Frame.H, 40)
		}
	}
	eventually(t, "the glass at the element's box", at(20))
	p := glassPanes(w)[0]
	if p.Class != "NSGlassEffectView" || !near(p.Radius, 16.5) || p.Style != 0 || !near(p.TintAlpha, 64.0/255) || !near(p.Opacity, 1) || !p.UnderSurface {
		t.Errorf("the glass is %+v: want an NSGlassEffectView, regular, rounded by 16.5, tinted, under the surface and over the material", p)
	}
	t.Logf("glass pane: %+v", p)

	left.Store(60)
	w.Invalidate()
	eventually(t, "the glass following the element", at(60))

	shown.Store(false)
	w.Invalidate()
	eventually(t, "the glass hidden", func() bool {
		p := glassPanes(w)
		return len(p) == 1 && !p[0].Shown
	})
}
