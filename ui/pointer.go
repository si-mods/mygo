package ui

import "time"

// pointerState is where the pointer is in the window's content, in DIPs,
// whether it is over the window, and whether its primary button is down.
type pointerState struct {
	x, y          float32
	over, primary bool
}

func (rt *engine) pointer() pointerState {
	return pointerState{rt.pointerX, rt.pointerY, rt.pointerIn, rt.buttons&1 != 0}
}

// Pointer returns where the pointer is in the window, in DIPs of the
// window's content, whether it is over the window, and whether its
// primary button is down, wherever it was pressed. x and y are the
// window's, as the root element's box is, not the painter's own: a
// drawing inside a scrolled or transformed element subtracts that
// element's position itself. Where the pointer left the window, x and y
// stay where it was last. Reading it asks for no frame: a drawing that
// follows the pointer asks for its moves with WatchPointer. A finger on a
// touch screen moves none of it.
func (p *Painter) Pointer() (x, y float32, over, primary bool) {
	s := p.rt.pointer()
	return s.x, s.y, s.over, s.primary
}

// WatchPointer asks for the window to be painted again when the pointer
// moves in it, leaves it, or presses or lets go of its primary button, at
// most fps frames a second, for a drawing that follows the pointer
// (Pointer), as a light under it: unless something else changed
// meanwhile, the frame paints the elements of this one again without
// building the view, as AnimationFrame's. Call it in every frame while
// the drawing follows the pointer; a frame that does not call it stops
// the watch. fps counts from 1 to 240. It makes no element look at the
// pointer: what the pointer hovers, presses and clicks stays as without
// it, and moves over an element that takes them (Element.HandleInput), as
// a terminal, count. On macOS, moves over a native web view (WKWebView)
// in the window come to the web view, not to the window: they are not
// seen, as they hover nothing.
func (p *Painter) WatchPointer(fps float32) {
	if !(fps > 0) {
		return
	}
	p.rt.watchRate = max(p.rt.watchRate, min(max(fps, 1), 240))
	p.asks()
}

// pointerWatched asks for the frame that WatchPointer asked for after the
// pointer changed. A move, or the pointer leaving (move), asks for one
// painting the elements again, at once when the last frame began at least
// 1/watchRate s ago, else then; another event, as a press, keeps the
// frame built anew it asked for, and makes sure one comes. A frame built
// anew asked for meanwhile paints it anyway.
func (rt *engine) pointerWatched(move bool) {
	if rt.inFrame || rt.buildAsked {
		return
	}
	if !move {
		rt.requestFrame()
		return
	}
	rt.redraw = true
	if rt.paceRate != 0 && rt.watchRate > rt.paceRate {
		// A drawing that moves slowly (AnimationFrameRate) paced the
		// frames that only paint below the watch's rate.
		rt.paceRate, rt.paceEvery = 0, 1
		rt.host.setFrameRate(0)
	}
	due := rt.watchFrom.Add(time.Duration(float64(time.Second) / float64(rt.watchRate)))
	if !rt.now().Before(due) {
		rt.repaintDue = time.Time{}
		rt.host.requestFrame()
		return
	}
	if rt.repaintDue.IsZero() || due.Before(rt.repaintDue) {
		rt.repaintDue = due
		rt.host.repaintTimer(due)
	}
}
