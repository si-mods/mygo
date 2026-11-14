//go:build linux && (amd64 || arm64)

package linux

import (
	"fmt"
	"testing"
)

// TestCornerCuts checks the pixels rounded corners cut off a window on
// X11: in each corner, those a part of which is outside the corner's
// quarter of a circle, and no other, each once, in rows a pixel high.
func TestCornerCuts(t *testing.T) {
	for _, tc := range []struct {
		width, height int32
		radius        float64
		r             int32 // the radius the window takes
	}{
		{400, 300, 8, 8},
		{400, 300, 1, 1},
		{37, 23, 5, 5},
		{20, 30, 50, 10}, // at most half the width
		{40, 16, 12, 8},  // and the height
	} {
		t.Run(fmt.Sprintf("%dx%d r%g", tc.width, tc.height, tc.radius), func(t *testing.T) {
			cut := map[[2]int32]bool{}
			for _, c := range cornerCuts(tc.width, tc.height, tc.radius) {
				if c.Height != 1 || c.Width <= 0 {
					t.Fatalf("cut %+v, want a row of pixels", c)
				}
				for x := c.X; x < c.X+c.Width; x++ {
					if cut[[2]int32{x, c.Y}] {
						t.Fatalf("pixel %d, %d cut twice", x, c.Y)
					}
					cut[[2]int32{x, c.Y}] = true
				}
			}
			r := tc.r
			for y := range tc.height {
				for x := range tc.width {
					// The pixel's distance from the window's nearest
					// corner, to its own corner furthest from the curve's
					// center.
					dx, dy := x, y
					if x >= tc.width/2 {
						dx = tc.width - 1 - x
					}
					if y >= tc.height/2 {
						dy = tc.height - 1 - y
					}
					want := dx < r && dy < r && (r-dx)*(r-dx)+(r-dy)*(r-dy) > r*r
					if cut[[2]int32{x, y}] != want {
						t.Errorf("pixel %d, %d cut: %v, want %v", x, y, !want, want)
					}
				}
			}
		})
	}

	// A radius of 8: the corner's pixel and its diagonal neighbours out to
	// the curve go, the edges stay past it.
	cut := map[[2]int32]bool{}
	for _, c := range cornerCuts(400, 300, 8) {
		for x := c.X; x < c.X+c.Width; x++ {
			cut[[2]int32{x, c.Y}] = true
		}
	}
	for _, c := range []struct {
		x, y int32
		want bool
	}{
		{0, 0, true},
		{2, 2, true},
		{3, 3, false},
		{7, 0, true}, // the top edge's last pixel the curve crosses
		{8, 0, false},
		{0, 8, false},
		{399, 0, true},
		{399, 299, true},
		{0, 299, true},
		{200, 0, false},
		{392, 299, true},
		{391, 299, false},
	} {
		if cut[[2]int32{c.x, c.y}] != c.want {
			t.Errorf("pixel %d, %d cut: %v, want %v", c.x, c.y, !c.want, c.want)
		}
	}

	if c := cornerCuts(400, 300, 0); c != nil {
		t.Errorf("no radius cuts %v, want nothing", c)
	}
	if c := cornerCuts(0, 0, 8); c != nil {
		t.Errorf("an empty window cuts %v, want nothing", c)
	}
}
