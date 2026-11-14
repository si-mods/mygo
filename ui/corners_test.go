package ui

import (
	"image"
	"testing"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/surface"
)

// cornerSurface is a surface whose content rounds the window's corners
// (platform.CornerSurface), keeping the frames it is given in memory.
type cornerSurface struct {
	testSurface
	radius        float64
	width, height float64
	scale         float64
	pix           []byte
	stride        int
	damage        []image.Rectangle
}

func (s *cornerSurface) Native() platform.SurfaceNative    { return platform.SurfaceNative{} }
func (s *cornerSurface) Size() (float64, float64, float64) { return s.width, s.height, s.scale }
func (s *cornerSurface) WindowCorners() float64            { return s.radius }
func (s *cornerSurface) PresentDamage(pix []byte, stride, width, height int, damage []image.Rectangle) {
	s.pix, s.stride, s.damage = append(s.pix[:0], pix...), stride, damage
}

// alpha returns the frame's alpha at the pixel x, y.
func (s *cornerSurface) alpha(x, y int) uint8 { return s.pix[y*s.stride+x*4+3] }

// TestWindowCorners cuts a frame in memory to the window's rounded
// corners: transparent outside them, antialiased along their curves, and
// opaque inside, the same in the four corners, with the radius at most
// half the window's sides. A change of the radius paints the corners
// again.
func TestWindowCorners(t *testing.T) {
	t.Setenv("MYGO_GPU", "0")
	for _, tc := range []struct {
		name          string
		width, height float64
		scale, radius float64
		r             int // the radius in pixels the frame takes
	}{
		{"radius 16", 200, 100, 1, 16, 16},
		{"radius 16 at scale 2", 200, 100, 2, 16, 32},
		{"radius past half the height", 200, 100, 1, 500, 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &cornerSurface{width: tc.width, height: tc.height, scale: tc.scale, radius: tc.radius}
			h := &windowHost{conn: &surface.Conn{Surface: s}}
			h.rt = newRuntime(func(c *context) {}, h) // the theme's background alone
			h.event(platform.SurfaceEvent{Kind: platform.SurfaceFrame})
			pw, ph := int(tc.width*tc.scale), int(tc.height*tc.scale)
			if s.pix == nil || s.stride < pw*4 {
				t.Fatal("no frame in memory")
			}
			r := tc.r
			if a := s.alpha(0, 0); a != 0 {
				t.Errorf("corner's alpha %d, want 0", a)
			}
			if a := s.alpha(pw/2, ph/2); a != 255 {
				t.Errorf("alpha inside = %d, want 255", a)
			}
			// The edges between the corners, where a pill's curves meet
			// them in a point.
			for _, p := range [][2]int{{pw / 2, 0}, {0, ph / 2}, {pw - 1, ph / 2}, {pw / 2, ph - 1}, {r, r}} {
				if a := s.alpha(p[0], p[1]); a < 250 {
					t.Errorf("alpha at %d, %d = %d, want 255 or nearly", p[0], p[1], a)
				}
			}
			// Along the diagonal from the corner, the alpha rises from 0
			// to 255 through a pixel or two in between: the curve's
			// antialiased edge.
			partial, last := 0, uint8(0)
			for i := range r {
				a := s.alpha(i, i)
				if a < last {
					t.Errorf("alpha falls along the diagonal at %d: %d after %d", i, a, last)
				}
				if a > 0 && a < 255 {
					partial++
				}
				last = a
			}
			if partial == 0 || partial > 2 || last != 255 {
				t.Errorf("%d pixels partly covered along the diagonal, ending with %d; want 1 or 2, then 255", partial, last)
			}
			// The four corners are mirrors of each other.
			for y := range r {
				for x := range r {
					a := s.alpha(x, y)
					if b, c, d := s.alpha(pw-1-x, y), s.alpha(x, ph-1-y), s.alpha(pw-1-x, ph-1-y); max(a, b, c, d)-min(a, b, c, d) > 1 {
						t.Fatalf("pixel %d, %d of the corners: alpha %d, %d, %d, %d", x, y, a, b, c, d)
					}
				}
			}

			// Square again, as when maximized: the corners are painted
			// again, opaque.
			s.radius = 0
			h.event(platform.SurfaceEvent{Kind: platform.SurfaceResize})
			h.event(platform.SurfaceEvent{Kind: platform.SurfaceFrame})
			if !image.Pt(0, 0).In(damageBox(s.damage)) || !image.Pt(pw-1, ph-1).In(damageBox(s.damage)) {
				t.Errorf("damage %v leaves out the corners", s.damage)
			}
			if a := s.alpha(0, 0); a != 255 {
				t.Errorf("square corner's alpha %d, want 255", a)
			}
		})
	}
}

// damageBox returns the box of rects.
func damageBox(rects []image.Rectangle) image.Rectangle {
	var u image.Rectangle
	for _, r := range rects {
		u = u.Union(r)
	}
	return u
}
