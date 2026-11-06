package terminal

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// A selection pulse is the light that crosses a selection as it is made
// or copied, as Rex draws it: swift-ghostty's Shaders.metal sweeps three
// gaussian lobes (a bright core, a wide bloom and a rim) along a slanted
// line over the selection, fades them in over the first fifth of the
// pulse and out over the last, and lifts the light's strength with the
// selection's size (selection_pulse_light, selection_pulse_style).
const (
	// pulseLife is how long the light takes to cross the selection.
	pulseLife = 420 * time.Millisecond
	// pulseSlant leans the band: where a point sits along it is
	// x + pulseSlant*y, so the light runs down and to the right.
	pulseSlant = 0.35
	// pulseCore, pulseBloom and pulseRim are the lobes' strength where
	// they are brightest.
	pulseCore, pulseBloom, pulseRim = 0.72, 0.3, 0.55
	// pulseSpanScale and pulseCellScale size the band: its lobes are as
	// wide as a share of the selection's diagonal, or of its line
	// height, whichever is larger.
	pulseSpanScale, pulseCellScale = 0.3, 1.1
)

// pulseGauss is the light's falloff across a lobe, exp(-k²) sampled at
// k = -2.5, -2, …, 2.5: eleven stops from nothing, through the peak at
// the middle, to nothing.
var pulseGauss = [...]float32{0.002, 0.018, 0.105, 0.368, 0.779, 1, 0.779, 0.368, 0.105, 0.018, 0.002}

// pulseStarted asks the next frame to start the light over the
// selection. The clock it runs on is the frame's, so that a test
// advancing time advances the pulse.
func (v *view) pulseStarted() {
	v.pulseWanted = true
}

// pulseOn reports whether the light runs as of now.
func (v *view) pulseOn(now time.Time) bool {
	if v.pulseStart.IsZero() {
		return false
	}
	since := now.Sub(v.pulseStart)
	return since >= 0 && since < pulseLife
}

// startPulse starts the light now, unless one already runs or a
// selection is not shown.
func (v *view) startPulse(c *ui.Context) {
	v.pulseWanted = false
	if v.pulseOn(c.Now()) {
		// The light that runs crosses the selection it started on.
		return
	}
	if !v.hasSelection() {
		v.pulseStart = time.Time{}
		return
	}
	v.pulseStart = c.Now()
}

// advancePulse starts or keeps the light running as a frame builds: a
// selection that appears or changes crosses with it, as does one a frame
// asked for, and it asks for the frames the pulse still needs.
func (v *view) advancePulse(c *ui.Context) {
	b, ok := v.selectionBounds()
	if !ok {
		b = [4]int{}
	}
	if b != v.pulseSel {
		// The selection grew, shrank or went; the light follows the
		// selection as it is made.
		v.pulseSel = b
		if ok {
			v.pulseWanted = true
		}
	}
	if v.pulseWanted {
		v.startPulse(c)
	}
	if v.pulseOn(c.Now()) {
		c.After(8 * time.Millisecond)
	}
}

// selectionBounds returns the box the selection covers on the grid, as
// first and last column then first and last row, and whether it shows.
func (v *view) selectionBounds() ([4]int, bool) {
	x0, x1, y0, y1 := 0, -1, -1, -1
	for y := v.top; y < len(v.lines); y++ {
		line := &v.lines[y]
		if !line.sel {
			continue
		}
		if y0 < 0 {
			y0, x0, x1 = y, line.selA, line.selB
		} else {
			x0, x1 = min(x0, line.selA), max(x1, line.selB)
		}
		y1 = y
	}
	if y0 < 0 {
		return [4]int{}, false
	}
	return [4]int{x0, y0, x1, y1}, true
}

// pulseSheen returns how far across the selection the light has come,
// 0 to 1, and how bright it is there: the envelope the shader applies,
// which fades the light in over the first fifth of the pulse and out
// over the last.
func (v *view) pulseSheen(now time.Time) (float32, float32) {
	since := now.Sub(v.pulseStart)
	if since < 0 || since >= pulseLife {
		return 0, 0
	}
	t := float32(since) / float32(pulseLife)
	in := smoothstep(clampf(t*5, 0, 1))
	out := smoothstep(clampf((t-0.8)*5, 0, 1))
	return t, in * (1 - out)
}

// drawPulse paints the light over the selection, if it shows and the
// pulse runs; the grid starts at ox, oy, its cells are cw by ch.
func (v *view) drawPulse(p *ui.Painter, ox, oy, cw, ch float32) {
	now := p.Now()
	sheen, env := v.pulseSheen(now)
	if env <= 0 {
		return
	}
	rect, ok := v.selectionRect(ox, oy, cw, ch)
	if !ok || rect.W <= 0 || rect.H <= 0 {
		return
	}
	// The band's width follows the selection's diagonal, as the shader's
	// span_scale does, or its line height when that is larger.
	span := rect.W + rect.H*pulseSlant
	lobe := max(ch*pulseCellScale, span*pulseSpanScale)
	// The light runs from before the selection's first corner to past
	// its last, as the shader mixes -1.5 lobes to the span plus 1.5.
	center := -1.5*lobe + sheen*(span+3*lobe)
	// The shader screens its light over the selection's color, which is
	// white at the light's strength over any theme.
	light := ui.RGBA(255, 255, 255, 1)
	v.fillBand(p, rect, center, lobe, light, env*pulseBloom, 1.9)
	v.fillBand(p, rect, center, lobe, light, env*pulseCore, 0.55)
	// The rim lights the selection's edge as the band passes it.
	v.fillRim(p, rect, center, lobe, light, env*pulseRim)
}

// fillBand fills r with a band of light centered at center along the
// slanted line, sigma wide, in c at its peak.
func (v *view) fillBand(p *ui.Painter, r ui.Rect, center, sigma float32, c ui.Color, peak, width float32) {
	g := pulseBand(r, center, sigma*width, c, peak)
	if g != nil {
		p.FillGradient(r, *g, 0)
	}
}

// fillRim lights the border of r where the band crosses it.
func (v *view) fillRim(p *ui.Painter, r ui.Rect, center, sigma float32, c ui.Color, peak float32) {
	g := pulseBand(r, center, sigma*2.2, c, peak)
	if g == nil {
		return
	}
	edge := max(1, v.scale)
	for _, s := range [...]ui.Rect{
		{X: r.X, Y: r.Y, W: r.W, H: edge},
		{X: r.X, Y: r.Y + r.H - edge, W: r.W, H: edge},
		{X: r.X, Y: r.Y, W: edge, H: r.H},
		{X: r.X + r.W - edge, Y: r.Y, W: edge, H: r.H},
	} {
		p.FillGradient(s, *g, 0)
	}
}

// pulseBand returns the gradient that draws a band of c across r,
// centered at center along the slanted line, half wide. The gradient
// spans r along the line, so its stops sit where the band does.
func pulseBand(r ui.Rect, center, half float32, c ui.Color, peak float32) *ui.LinearGradient {
	if peak <= 0 || half <= 0 {
		return nil
	}
	// The line runs along (1, pulseSlant); a point's place on it is
	// x + pulseSlant*y, over the corners of r.
	n := float32(math.Hypot(1, pulseSlant))
	dx, dy := 1/n, pulseSlant/n
	lo, hi := linePlace(r, dx, dy)
	if span := hi - lo; span <= 0 {
		return nil
	} else {
		// The stops are fractions of r's extent along the line.
		f := (center - lo) / span
		step := half / span
		stops := make([]ui.GradientStop, 0, len(pulseGauss)+2)
		stops = append(stops, ui.GradientStop{At: 0, Color: c.Alpha(0)})
		for i, g := range pulseGauss {
			at := f - step + 2*step*float32(i)/float32(len(pulseGauss)-1)
			stops = append(stops, ui.GradientStop{At: clampf(at, 0, 1), Color: c.Alpha(peak * g)})
		}
		stops = append(stops, ui.GradientStop{At: 1, Color: c.Alpha(0)})
		// CSS angles run clockwise from upwards.
		angle := float32(math.Atan2(float64(dx), float64(-dy)) * 180 / math.Pi)
		return &ui.LinearGradient{Angle: angle, Stops: stops}
	}
}

// linePlace returns where r's corners sit along the direction dx, dy,
// the least and the most.
func linePlace(r ui.Rect, dx, dy float32) (float32, float32) {
	lo, hi := float32(math.Inf(1)), float32(math.Inf(-1))
	for _, p := range [...][2]float32{{r.X, r.Y}, {r.X + r.W, r.Y}, {r.X, r.Y + r.H}, {r.X + r.W, r.Y + r.H}} {
		u := p[0]*dx + p[1]*dy
		lo, hi = min(lo, u), max(hi, u)
	}
	return lo, hi
}

// pulseShows reports whether the light may show, so that a frame that
// has none does not work out where it would be.
func (v *view) pulseShows() bool { return !v.pulseStart.IsZero() }

// hasSelection reports whether a selection shows.
func (v *view) hasSelection() bool {
	_, ok := v.selectionRect(0, 0, 1, 1)
	return ok
}

// selectionRect returns the box the selection covers on the grid, in the
// pixels of a grid that starts at ox, oy with cells cw by ch.
func (v *view) selectionRect(ox, oy, cw, ch float32) (ui.Rect, bool) {
	b, ok := v.selectionBounds()
	if !ok {
		return ui.Rect{}, false
	}
	return ui.Rect{
		X: ox + float32(b[0])*cw,
		Y: oy + float32(b[1])*ch,
		W: float32(b[2]-b[0]+1) * cw,
		H: float32(b[3]-b[1]+1) * ch,
	}, true
}

func clampf(f, lo, hi float32) float32 {
	return min(max(f, lo), hi)
}

// smoothstep is 3t² - 2t³, the shader's easing for the pulse's fade.
func smoothstep(t float32) float32 {
	return t * t * (3 - 2*t)
}
