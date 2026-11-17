//go:build linux && (amd64 || arm64)

package linux

import "github.com/egoist/mygo/internal/platform"

// On X11, an undecorated window with a CornerRadius has rounded corners,
// in one of two ways.
//
// With a compositor, a window of native UI made then has an alpha
// channel (blendsCorners): the content cuts its frames to the corners,
// with antialiased edges, transparent outside them (WindowCorners,
// platform.CornerSurface), and the compositor blends them with what is
// behind the window.
//
// Without one, the window's X window has a bounding shape, which X11's
// SHAPE extension sets, of a rectangle without the corners: the X server
// shows nothing of the window outside it, what is beneath showing there.
// The edges are hard, a pixel in or out. The shape is the bounding one,
// apart from the input shape (applyInputShape), which the X server
// intersects with it: the corners take no input either. A window showing
// a page is always cut so, and so is one of native UI made without a
// compositor, which keeps the visual it was made with, without an alpha
// channel, should a compositor start later. A window with an alpha
// channel whose compositor goes away is cut so until one comes back:
// what it leaves transparent would show black.
//
// Maximized, in full screen or tiled, the window fills its edges, with
// square corners either way.
//
// Wayland's windows have no such shape: GTK's decorations (useCSD) round
// those it decorates itself. A transparent window shows its own shape,
// and flyouts and panels have a look of their own (Flyout.Shadow).

// roundsCorners tells that the window has rounded corners.
func (w *window) roundsCorners() bool {
	o := w.opts
	return w.b.onX11 && o.CornerRadius > 0 && w.undecorated() && !w.csd &&
		!o.Transparent && o.Flyout == nil && !o.Panel
}

// gdkScreenIsComposited is gdk_screen_is_composited, once bound.
var gdkScreenIsComposited func(screen ptr) bool

// composited reports whether a compositor blends the windows of the
// window's screen.
func (w *window) composited() bool {
	if gdkScreenIsComposited == nil && !bind(libGDK, &gdkScreenIsComposited, "gdk_screen_is_composited") {
		return false
	}
	return gdkScreenIsComposited(gtkWidgetGetScreen(w.win))
}

// blendCorners gives a window of native UI about to round its corners
// an alpha channel when a compositor runs, for the content to cut its
// frames to them: before the window is realized, which fixes its visual.
// GTK paints nothing under the content then (app-paintable), which would
// fill the corners. The window follows the compositor as it goes and
// comes back (cbCompositedChanged).
func (w *window) blendCorners() {
	if !w.roundsCorners() || !w.opts.Surface || !w.composited() {
		return
	}
	screen := gtkWidgetGetScreen(w.win)
	visual := gdkScreenGetRGBAVisual(screen)
	if visual == 0 {
		return
	}
	gtkWidgetSetVisual(w.win, visual)
	gtkWidgetSetAppPaintable(w.win, true)
	w.alphaCorners = true
	if !w.b.compositeWatched {
		w.b.compositeWatched = true
		connect(screen, "composited-changed", cbCompositedChanged, 0)
	}
}

// blendsCorners tells that the content cuts the window's frames to its
// corners now: the window has an alpha channel, a compositor blends it,
// and it does not fill its edges.
func (w *window) blendsCorners() bool {
	return w.alphaCorners && w.state&(stateMaximized|stateFullscreen|stateTiled) == 0 && w.composited()
}

// WindowCorners returns the radius the content cuts its frames to.
func (s *surface) WindowCorners() float64 {
	if !s.w.blendsCorners() {
		return 0
	}
	return float64(s.w.opts.CornerRadius)
}

// cornersChanged follows a change of the window's corners, as it fills its
// edges or no longer does, or as the compositor goes or comes back: the
// shape, and the content's frames, which it paints whole again.
func (w *window) cornersChanged() {
	if !w.roundsCorners() {
		return
	}
	w.applyCornerShape()
	if w.alphaCorners && w.surface != nil {
		w.surface.send(platform.SurfaceEvent{Kind: platform.SurfaceResize})
	}
}

// applyCornerShape cuts the window's X window to its corners, at its size,
// or takes the shape away while the window fills its edges or its content
// cuts its frames to them. A window not realized yet gets it as it
// realizes, and again as its size changes (cbWindowAllocated) and as its
// corners change (cornersChanged).
func (w *window) applyCornerShape() {
	if w.closed || !w.roundsCorners() {
		return
	}
	win := gtkWidgetGetWindow(w.win)
	if win == 0 {
		return
	}
	loadWebViews() // binds the GDK and cairo region functions
	if w.state&(stateMaximized|stateFullscreen|stateTiled) != 0 || w.alphaCorners && w.composited() {
		gdkWindowShapeCombineRegion(win, 0, 0, 0)
		return
	}
	// GDK takes the region in the window's DIPs and scales it to the X
	// window's pixels itself (GDK_SCALE).
	var a gdkRectangle
	gtkWidgetGetAllocation(w.win, &a)
	region := cairoRegionCreateRectangle(&gdkRectangle{Width: a.Width, Height: a.Height})
	for _, r := range cornerCuts(a.Width, a.Height, float64(w.opts.CornerRadius)) {
		cairoRegionSubtractRectangle(region, &r)
	}
	gdkWindowShapeCombineRegion(win, region, 0, 0)
	cairoRegionDestroy(region)
}

// cornerCuts returns the rows of pixels that corners of radius cut off a
// window width by height, a row of each corner a rectangle one pixel high:
// those outside the corner's quarter of a circle, the one it crosses too.
// The radius is at most half the window's width and height.
func cornerCuts(width, height int32, radius float64) []gdkRectangle {
	r := min(radius, float64(width)/2, float64(height)/2)
	if r <= 0 {
		return nil
	}
	radii := [4]float64{r, r, r, r}
	return cornerPixels(platform.RectF{W: float64(width), H: float64(height)}, radii, radii)
}
