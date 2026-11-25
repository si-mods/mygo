package ui

import "github.com/egoist/mygo/internal/scene"

// Frames that only paint again. A drawing that moves with the time
// (Painter.AnimationFrame, AnimationFrameRate, After) while nothing else
// changes has the next frame paint the elements of this one again, without
// building the view; yet painting them all again, every background, text
// run and Draw callback of the window, costs as much as what the window
// shows. So the frame painting the window keeps, for each Draw or DrawOver
// callback, or material (Element.Material), that asked to be painted again, the operations of the scene it
// painted and the painter as it was: a frame that only paints again runs
// those callbacks alone, as they were, and puts what they paint in the
// place of what they painted, in the last frame's scene. Anything else
// asking for such a frame (a pose moving, a drawing in a canvas or a
// layer, content changing) has the frame paint the window whole, as do the
// inspector and a drag.

// paintPart is a Draw (or DrawOver, or the material of its background)
// callback of element e, called with box, that painted the operations of
// the window's scene from start to end, with the painter as it was then,
// its opaque backgrounds and holes apart.
type paintPart struct {
	e          *node
	kind       partKind
	box        Rect
	start, end int
	painter    Painter
	under      []opaqueBox
	holes      []Rect
	// from and to are where it painted the frame being painted, at the
	// end of the scene's operations.
	from, to int
}

// partKind is what of an element a part paints: its Draw callback, its
// DrawOver callback, or the material of its background (Element.Material).
type partKind uint8

const (
	partDraw partKind = iota
	partDrawOver
	partMaterial
)

// paintKind calls what kind names of e with box.
func (p *Painter) paintKind(e *node, box Rect, kind partKind) {
	switch kind {
	case partDrawOver:
		e.paintAfterFn(p, box)
	case partMaterial:
		e.material.PaintMaterial(p, box, e.radius)
	default:
		e.paintFn(p, box)
	}
}

// draw calls e's Draw callback (DrawOver's when after) with box, noting
// what it painted when it asks to be painted again (paintPart).
func (p *Painter) draw(e *node, box Rect, after bool) {
	kind := partDraw
	if after {
		kind = partDrawOver
	}
	p.part(e, box, kind)
}

// part paints what kind names of e with box, noting what it painted when
// it asks to be painted again (paintPart): a material that asks for
// frames (AnimationFrame, WatchPointer) is painted again alone, as a Draw
// callback is.
func (p *Painter) part(e *node, box Rect, kind partKind) {
	if p.drawing || p.tracking || p.s != &p.rt.scene {
		// Into a layer or a canvas, painted whole.
		p.paintKind(e, box, kind)
		return
	}
	rt := p.rt
	start, state, uses := len(p.s.Ops), *p, rt.text.TransientUses()
	// Layers of paths drawn on the CPU end where a part does, which is
	// painted again alone (pathSpill).
	sp := &rt.paths.spill
	sp.run++
	opened := sp.opened
	p.track(kind, true)
	p.paintKind(e, box, kind)
	p.track(kind, false)
	sp.run++
	if p.asked {
		rt.partTransient += rt.text.TransientUses() - uses
		sp.inParts += sp.opened - opened
		rt.drawn = append(rt.drawn, paintPart{e: e, kind: kind, box: box, start: start, end: len(p.s.Ops), painter: state,
			under: append([]opaqueBox(nil), state.under...), holes: append([]Rect(nil), state.holes...)})
	}
}

// track opens (on) or closes the part of kind, from which asks records
// that it asks to be painted again. A Draw or DrawOver callback paints as
// drawing; a material does not, as it paints as an element's background
// (the system's glass shows only outside Draw callbacks, systemGlassShows).
func (p *Painter) track(kind partKind, on bool) {
	if on {
		p.asked = false
	}
	if kind == partMaterial {
		p.tracking = on
	} else {
		p.drawing = on
	}
}

// asks notes that the drawing painting asks to be painted again: by
// itself, as its part, from a Draw or DrawOver callback or a material of
// the window's painting, else with the whole window.
func (p *Painter) asks() {
	if p.drawing || p.tracking {
		p.asked = true
	} else {
		p.rt.untracked = true
	}
}

// partsGrowth is how much more glyphs, stops, effects, colors and words a
// scene painted in parts holds than it did painted whole, past which the
// next frame paints it whole: what the parts painted before stays in them.
const partsGrowth = 4096

// sceneData returns how many glyphs, stops, effects, colors and words s
// holds.
func sceneData(s *scene.Scene) int {
	return len(s.Glyphs) + len(s.Stops) + len(s.Effects) + len(s.Wide) + len(s.Words)
}

// repaintParts paints the parts of the last frame painted again (see
// paintPart), and reports whether it could: false when the window must be
// painted whole, as when the glyph atlases, which every window shares,
// moved or forgot the glyphs the rest of the scene draws, as another
// window making room in them does, or when the rest of the scene draws
// masks drawn for its frame alone, in room of the atlas that the frames
// since drew over: painted whole again, they last.
func (rt *engine) repaintParts() bool {
	s := &rt.scene
	if rt.untracked || len(rt.drawn) == 0 || rt.insp.open || rt.drag != nil || rt.noParts ||
		sceneData(s) > 2*rt.partsBase+partsGrowth || atlasGenerations(s) != rt.atlases || rt.transient {
		return false
	}
	rt.painting = true
	defer func() { rt.painting = false }()
	rt.partPaints++
	rt.paths.spill.beginPaint()
	n := len(s.Ops)
	// The canvases the last frame drew are drawn: those the parts draw
	// anew are this frame's.
	clear(s.Canvases)
	s.Canvases = s.Canvases[:0]
	for i := range rt.drawn {
		pt := &rt.drawn[i]
		rt.painter = pt.painter
		p := &rt.painter
		p.under = append(rt.under[:0], pt.under...)
		p.holes = append(rt.holes[:0], pt.holes...)
		// A part that no longer asks to be painted again stays, as what
		// it painted last stays in the scene.
		pt.from = len(s.Ops)
		p.track(pt.kind, true)
		rt.paths.spill.run++
		p.paintKind(pt.e, pt.box, pt.kind)
		rt.paths.spill.run++
		p.track(pt.kind, false)
		pt.to = len(s.Ops)
		rt.under, rt.holes = p.under[:0], p.holes[:0]
	}
	ops, last := rt.partOps[:0], 0
	for i := range rt.drawn {
		pt := &rt.drawn[i]
		ops = append(ops, s.Ops[last:pt.start]...)
		start := len(ops)
		ops = append(ops, s.Ops[pt.from:pt.to]...)
		last, pt.start, pt.end = pt.end, start, len(ops)
	}
	ops = append(ops, s.Ops[last:n]...)
	clear(s.Ops)
	rt.partOps, s.Ops = s.Ops[:0], ops
	rt.finishSpills()
	return true
}
