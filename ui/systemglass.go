package ui

import (
	"fmt"
	"math"
	"slices"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/scene"
)

// SystemGlassStyle is the style of the system's Liquid Glass
// (SystemGlass), as NSGlassEffectViewStyle.
type SystemGlassStyle uint8

const (
	// SystemGlassRegular is the regular glass, frosted enough to read
	// over anything.
	SystemGlassRegular SystemGlassStyle = iota
	// SystemGlassClear is the clear glass, which shows more of what is
	// behind it.
	SystemGlassClear
)

// SystemGlass is a material that shows the system's own Liquid Glass
// behind the element: AppKit's NSGlassEffectView on macOS 26 and later,
// what SwiftUI's glassEffect draws, rather than the glass plugin's shader,
// which only refracts what the window's content painted. It is the one to
// use over the window's material (Vibrancy), which the content does not
// paint:
//
//	ui.Row(c).Padding(6, 10).Radius(16.5).Material(ui.SystemGlass{
//		Fallback: glass.Glass{},
//	})
//
// Where the system's glass can show, the element paints a hole through
// the window's content, shaped by its box and Radius, and the window puts
// a glass view under the content there, which follows the element every
// frame it paints: as it is laid out, scrolled, clipped, moved, scaled and
// faded. The element's border, children and what paints after it show
// over the glass. The glass goes when the element stops painting, as when
// it is no longer built, hidden, or scrolled out of sight.
//
// Elsewhere it paints Fallback, or the Tint as a color without one:
// before macOS 26, on Windows, Linux and in browsers, in ui.Tester and
// Render, inside an element that renders into a bitmap (Effect, Cache,
// Blur, a Canvas), and in a Draw callback. So tests see Fallback.
//
// The native glass stays upright: a rotated element shows it over the
// bounds of its box. It is clipped to the rectangle the scroll containers
// and clips around the element leave, not to their rounded corners, and
// has one radius, the first corner's. It shows under the web views of the
// window (WebView), not over their pages.
type SystemGlass struct {
	// Style is the glass's style: regular unless clear.
	Style SystemGlassStyle
	// Tint tints the glass, as NSGlassEffectView's tintColor; none when
	// transparent.
	Tint Color
	// Fallback paints the element where the system's glass cannot show.
	Fallback Material
}

// BuildMaterial builds the fallback, when it follows its element
// (MaterialBuilder).
func (g SystemGlass) BuildMaterial(e Element) Material {
	if b, ok := g.Fallback.(MaterialBuilder); ok {
		g.Fallback = b.BuildMaterial(e)
	}
	return g
}

// PaintMaterial shows the system's glass over box, rounded by radii, or
// paints the fallback.
func (g SystemGlass) PaintMaterial(p *Painter, box Rect, radii [4]float32) {
	// The frames send the glass they paint once a view paints some: apps
	// painting none don't link the backends' side of it.
	sendGlass = sendSystemGlass
	if !p.systemGlassShows() {
		switch {
		case g.Fallback != nil:
			g.Fallback.PaintMaterial(p, box, radii)
		case g.Tint.A > 0:
			p.add(scene.Op{Kind: scene.OpFill, Rect: p.snap(box), Radii: p.radii(radii), Continuous: continuousCorners,
				Color: g.Tint.scene(), Opacity: p.opacity, Wide: p.wide(g.Tint, Color{}, Color{})})
		}
		return
	}
	p.systemGlass(box, radii, g)
}

// String describes the glass, as the inspector shows it.
func (g SystemGlass) String() string {
	s := "system-glass"
	if g.Style == SystemGlassClear {
		s += " clear"
	}
	if g.Tint.A > 0 {
		s += fmt.Sprintf(" tint(#%02x%02x%02x%02x)", g.Tint.R, g.Tint.G, g.Tint.B, g.Tint.A)
	}
	return s
}

// systemGlassHost is a host whose window can show ui.SystemGlass, under
// web views. It is separate from the host's placeGlass, which places
// native panes above web views (Glass{Native: true}).
type systemGlassHost interface {
	supportsSystemGlass() bool
	placeSystemGlass(glass []platform.SystemGlassPlacement)
}

// systemGlassShows reports whether the glass the painter paints shows as
// the system's: into the window's frame, outside a Draw callback, in a
// window that can show it.
func (p *Painter) systemGlassShows() bool {
	if p.s != &p.rt.scene || p.drawing {
		return false
	}
	h, ok := p.rt.host.(systemGlassHost)
	return ok && h.supportsSystemGlass()
}

// systemGlass paints the hole the system's glass shows through, and notes
// where the frame shows it.
func (p *Painter) systemGlass(box Rect, radii [4]float32, g SystemGlass) {
	if box.W <= 0 || box.H <= 0 {
		return
	}
	p.add(scene.Op{Kind: scene.OpHole, Rect: p.snap(box), Radii: p.radii(radii), Continuous: continuousCorners, Opacity: p.opacity})
	// The glass stays upright, over the bounds of the transformed box, on
	// whole pixels as the hole.
	b := transformRect(p.transform, box)
	if s := p.scale; s > 0 {
		x0, y0 := round(b.X*s)/s, round(b.Y*s)/s
		x1, y1 := round((b.X+b.W)*s)/s, round((b.Y+b.H)*s)/s
		b = Rect{x0, y0, x1 - x0, y1 - y0}
	}
	c := intersect(b, p.clip)
	if b.W <= 0 || b.H <= 0 || c.W <= 0 || c.H <= 0 {
		return
	}
	// Nothing under the hole is opaque: text over it is smoothed in
	// grayscale.
	p.holes = append(p.holes, c)
	radius := max(radii[0], 0)
	if t := p.transform; t.Set {
		radius *= float32(math.Sqrt(math.Abs(float64(t.A*t.D - t.B*t.C))))
	}
	rt := p.rt
	rt.systemGlassPlaces = append(rt.systemGlassPlaces, platform.SystemGlassPlacement{
		Frame:   rt.toWindow(rectF(b)),
		Clip:    rt.toWindow(rectF(c)),
		Radius:  float64(radius) * float64(rt.zoomFactor()),
		Clear:   g.Style == SystemGlassClear,
		Tint:    platform.Color{R: g.Tint.R, G: g.Tint.G, B: g.Tint.B, A: g.Tint.A},
		Opacity: float64(min(max(p.opacity, 0), 1)),
	})
}

// sendGlass is sendSystemGlass once a view painted ui.SystemGlass.
var sendGlass func(rt *engine, glass []platform.SystemGlassPlacement)

func sendSystemGlass(rt *engine, glass []platform.SystemGlassPlacement) {
	if h, ok := rt.host.(systemGlassHost); ok {
		h.placeSystemGlass(glass)
	}
}

// placeSystemGlass tells the host where the frame painted ui.SystemGlass,
// unless it is where the last frame sent did: it hides the glass no frame
// paints any longer. The call goes through sendGlass, set by
// PaintMaterial, so apps that paint none do not link the surface's side.
func (rt *engine) placeSystemGlass() {
	if sendGlass == nil || slices.Equal(rt.systemGlassPlaces, rt.systemGlassSent) {
		return
	}
	rt.systemGlassSent = append(rt.systemGlassSent[:0], rt.systemGlassPlaces...)
	sendGlass(rt, rt.systemGlassSent)
}

func (h *windowHost) supportsSystemGlass() bool {
	s, ok := h.conn.Surface.(platform.SystemGlassSurface)
	return ok && s.SupportsGlass()
}

func (h *windowHost) placeSystemGlass(glass []platform.SystemGlassPlacement) {
	if s, ok := h.conn.Surface.(platform.SystemGlassSurface); ok {
		s.PlaceSystemGlass(glass)
	}
}
