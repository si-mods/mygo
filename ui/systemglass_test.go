package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/scene"
)

// glassWindows are the Testers whose window shows the system's glass
// (withSystemGlass), with what it was told of it.
var glassWindows = map[*headless]*glassLog{}

// glassLog is what a Tester's window was told of the system's glass: each
// list the frames sent.
type glassLog struct{ sent [][]platform.GlassPlacement }

func (h *headless) supportsGlass() bool { return glassWindows[h] != nil }

func (h *headless) placeGlass(glass []platform.GlassPlacement) {
	if l := glassWindows[h]; l != nil {
		l.sent = append(l.sent, append([]platform.GlassPlacement(nil), glass...))
	}
}

// withSystemGlass has tt's window show the system's glass, as macOS 26's
// does, from its next frame.
func withSystemGlass(t *testing.T, tt *Tester) *glassLog {
	l := &glassLog{}
	glassWindows[tt.h] = l
	t.Cleanup(func() { delete(glassWindows, tt.h) })
	tt.Frame()
	return l
}

// last returns what the window shows of the glass: the list sent last.
func (l *glassLog) last() []platform.GlassPlacement {
	if len(l.sent) == 0 {
		return nil
	}
	return l.sent[len(l.sent)-1]
}

// paintedMaterial notes that it painted, as a fallback.
type paintedMaterial struct{ n *int }

func (m paintedMaterial) PaintMaterial(p *Painter, box Rect, radii [4]float32) {
	*m.n++
	p.Fill(box, RGB(0, 0, 255), radii[0])
}

func holes(tt *Tester) []scene.Op {
	var out []scene.Op
	for _, op := range tt.rt.scene.Ops {
		if op.Kind == scene.OpHole {
			out = append(out, op)
		}
	}
	return out
}

// Where the system's glass cannot show, as in a Tester, the element paints
// its fallback, and nothing goes to the window.
func TestSystemGlassFallback(t *testing.T) {
	n := 0
	tt := coreNewTester(func(c *context) {
		c.Root().Background(RGB(255, 255, 255))
		coreBox(c).Absolute().Left(10).Top(10).Size(100, 40).Radius(8).Material(SystemGlass{Fallback: paintedMaterial{&n}})
	}, 200, 100)
	if n == 0 {
		t.Fatal("the fallback was not painted")
	}
	if len(holes(tt)) != 0 || len(tt.rt.glassPlaces) != 0 {
		t.Error("the fallback paints a hole or places the system's glass")
	}
	o := tt.Image().PixOffset(50, 30)
	if c := [4]uint8(tt.Image().Pix[o : o+4]); c != [4]uint8{0, 0, 255, 255} {
		t.Errorf("the fallback shows %v: want blue", c)
	}

	// Without a fallback, the tint paints as a color.
	tt = coreNewTester(func(c *context) {
		c.Root().Background(RGB(255, 255, 255))
		coreBox(c).Absolute().Left(10).Top(10).Size(100, 40).Material(SystemGlass{Tint: RGB(255, 0, 0)})
	}, 200, 100)
	o = tt.Image().PixOffset(50, 30)
	if c := [4]uint8(tt.Image().Pix[o : o+4]); c != [4]uint8{255, 0, 0, 255} {
		t.Errorf("the tint without a fallback shows %v: want red", c)
	}
}

// Where it can, the element paints a hole through the content in its box,
// rounded by its Radius, under its children, and the window shows the
// glass there, with its style, tint and opacity.
func TestSystemGlassPlacement(t *testing.T) {
	n := 0
	tt := coreNewTester(func(c *context) {
		c.Root().Background(RGB(255, 255, 255))
		coreBox(c).Absolute().Left(10).Top(20).Size(100, 40).Radius(16.5).Opacity(0.5).
			Material(SystemGlass{Style: SystemGlassClear, Tint: RGBA(29, 29, 29, 0.25), Fallback: paintedMaterial{&n}}).
			Children(func() {
				coreBox(c).Absolute().Left(4).Top(4).Size(10, 10).Background(RGB(255, 0, 0))
			})
	}, 200, 100)
	l := withSystemGlass(t, tt)
	n = 0
	tt.Frame()
	if n != 0 {
		t.Error("the fallback paints where the system's glass shows")
	}
	got := l.last()
	want := platform.GlassPlacement{
		Frame: platform.RectF{X: 10, Y: 20, W: 100, H: 40}, Clip: platform.RectF{X: 10, Y: 20, W: 100, H: 40},
		Radius: 16.5, Clear: true, Tint: platform.Color{R: 29, G: 29, B: 29, A: 64}, Opacity: 0.5,
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("the glass shows at %+v: want %+v", got, want)
	}
	h := holes(tt)
	if len(h) != 1 || h[0].Rect != (scene.Rect{X: 10, Y: 20, W: 100, H: 40}) || h[0].Radii[0] != 16.5 || h[0].Opacity != 0.5 {
		t.Fatalf("holes %+v: want one at the box, rounded by 16.5, half opaque", h)
	}
	img := tt.Image()
	o := img.PixOffset(18, 28)
	if c := img.Pix[o : o+4]; c[0] < 100 || c[3] < 100 {
		t.Errorf("the child over the glass shows %v: want red", c)
	}
	// The white under the half-opaque hole fades by half.
	o = img.PixOffset(60, 40)
	if c := img.Pix[o : o+4]; c[3] < 120 || c[3] > 135 {
		t.Errorf("the hole shows %v: want half the white under it", c)
	}

	// A frame that paints the same sends nothing.
	sent := len(l.sent)
	tt.Frame()
	if len(l.sent) != sent {
		t.Error("a frame showing the glass where the last did sent it again")
	}
}

// The glass follows its element: as it moves, as a scroll container clips
// it, and goes once it scrolls out of sight or is no longer built.
func TestSystemGlassFollows(t *testing.T) {
	shown := true
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Absolute().Left(0).Top(0).Size(200, 100).Children(func() {
			coreBox(c).Height(30)
			if shown {
				coreBox(c).Height(40).Radius(8).Material(SystemGlass{})
			}
			coreBox(c).Height(400)
		})
	}, 200, 100)
	l := withSystemGlass(t, tt)
	if g := l.last(); len(g) != 1 || g[0].Frame != (platform.RectF{X: 0, Y: 30, W: 200, H: 40}) || g[0].Clip != g[0].Frame {
		t.Fatalf("the glass shows at %+v: want the element's box", g)
	}
	tt.Scroll(100, 50, 0, 50)
	g := l.last()
	if len(g) != 1 || g[0].Frame != (platform.RectF{X: 0, Y: -20, W: 200, H: 40}) || g[0].Clip != (platform.RectF{X: 0, Y: 0, W: 200, H: 20}) {
		t.Fatalf("scrolled by 50, the glass shows at %+v: want at -20, clipped to the top 20", g)
	}
	tt.Scroll(100, 50, 0, 50)
	if g := l.last(); len(g) != 0 {
		t.Fatalf("scrolled out of sight, the glass shows at %+v", g)
	}
	tt.Scroll(100, 50, 0, -100)
	if g := l.last(); len(g) != 1 {
		t.Fatalf("scrolled back, the glass shows at %+v: want one", g)
	}
	shown = false
	tt.Frame()
	if g := l.last(); len(g) != 0 {
		t.Errorf("no longer built, the glass shows at %+v", g)
	}
}

// A scaled element shows the glass over its scaled box, with its corners
// scaled; the window's zoom maps it to the window's DIPs.
func TestSystemGlassTransformAndZoom(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreBox(c).Absolute().Left(50).Top(20).Size(100, 40).Radius(10).Scale(2, 2).Material(SystemGlass{})
	}, 300, 100)
	l := withSystemGlass(t, tt)
	if g := l.last(); len(g) != 1 || g[0].Frame != (platform.RectF{X: 0, Y: 0, W: 200, H: 80}) || g[0].Radius != 20 {
		t.Fatalf("scaled twice, the glass shows at %+v: want twice the box and radius", g)
	}

	tt = coreNewTester(func(c *context) {
		coreBox(c).Absolute().Left(10).Top(10).Size(50, 20).Radius(5).Material(SystemGlass{})
	}, 300, 100)
	l = withSystemGlass(t, tt)
	tt.SetZoom(2)
	if g := l.last(); len(g) != 1 || g[0].Frame != (platform.RectF{X: 20, Y: 20, W: 100, H: 40}) || g[0].Radius != 10 {
		t.Fatalf("zoomed twice, the glass shows at %+v: want twice the box and radius", g)
	}
}

// Inside an element rendering into a bitmap, the glass cannot show: the
// fallback paints there.
func TestSystemGlassInCache(t *testing.T) {
	n := 0
	tt := coreNewTester(func(c *context) {
		coreBox(c).Absolute().Left(10).Top(10).Size(120, 60).Cache().Children(func() {
			coreBox(c).Size(100, 40).Material(SystemGlass{Fallback: paintedMaterial{&n}})
		})
	}, 200, 100)
	l := withSystemGlass(t, tt)
	n = 0
	tt.Frame()
	if n == 0 || len(l.last()) != 0 || len(holes(tt)) != 0 {
		t.Errorf("in a cached element: fallback painted %d times, glass %+v, %d holes; want the fallback alone", n, l.last(), len(holes(tt)))
	}
}

// A selected element whose material paints in place of its selection's
// background, as a sidebar's active tab with the system's glass, fades no
// background as it is deselected: Selected's accent, which the material
// hid, must not show as the colors transition from it.
func TestMaterialHidesColorsTransition(t *testing.T) {
	for _, native := range []bool{false, true} {
		active := 0
		tt := coreNewTester(func(c *context) {
			c.Root().Background(Transparent)
			for i := range 2 {
				e := coreRow(c).Key(i).Absolute().Left(10).Top(float32(10+50*i)).Size(100, 40).Radius(16).
					Selected(active == i).Transition(ElementTransition{Colors: true, Duration: 160 * time.Millisecond})
				if active == i {
					e.Material(SystemGlass{Fallback: paintedMaterial{new(int)}})
				}
			}
		}, 200, 120)
		if native {
			withSystemGlass(t, tt)
		}
		active = 1
		tt.Frame()
		for _, d := range []time.Duration{0, 16 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond} {
			tt.Advance(d)
			img := tt.Image()
			o := img.PixOffset(60, 30)
			if c := [4]uint8(img.Pix[o : o+4]); c[3] != 0 {
				t.Errorf("native %v: %v after deselecting, the old element shows %v: want nothing", native, d, c)
			}
		}
	}
}
