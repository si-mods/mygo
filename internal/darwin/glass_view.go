//go:build darwin

package darwin

import "github.com/egoist/mygo/internal/platform"

// The system's Liquid Glass in a window of native UI (ui.SystemGlass,
// platform.SystemGlassSurface): an NSGlassEffectView in a clip view, a
// subview of the window's content view under the surface and its web
// views, over the window's material. The content paints a hole where it
// shows it, so what it paints after the glass shows over it. A frame's
// glass reuses the views of the last one in paint order, the first at the
// bottom; those it leaves hide. The panes Glass{Native: true} places
// (surface.glass) sit above web views and are not these.

// systemGlassView is one pane of ui.SystemGlass: the NSGlassEffectView and
// the clip view around it, which shows the part of it the content shows.
type systemGlassView struct {
	clip, glass id
	shown       platform.SystemGlassPlacement
	hidden      bool
}

// SupportsGlass reports whether AppKit has NSGlassEffectView: from macOS 26.
func (s *surface) SupportsGlass() bool { return hasClass("NSGlassEffectView") }

// PlaceSystemGlass shows ui.SystemGlass where the content's last frame
// painted it, and hides the panes it no longer paints.
func (s *surface) PlaceSystemGlass(list []platform.SystemGlassPlacement) {
	host := s.w
	if host.closed || !s.SupportsGlass() {
		return
	}
	height := msgRect(host.view, sel("bounds")).Size.Height
	withPool(func() {
		for i, p := range list {
			if i == len(s.systemGlass) {
				s.systemGlass = append(s.systemGlass, s.newSystemGlassView())
			}
			s.systemGlass[i].place(p, height)
		}
		for i := len(list); i < len(s.systemGlass); i++ {
			if v := &s.systemGlass[i]; !v.hidden {
				send(v.clip, "setHidden:", 1)
				v.hidden = true
			}
		}
	})
}

// newSystemGlassView makes a pane of glass, hidden, over those made before it.
func (s *surface) newSystemGlassView() systemGlassView {
	v := systemGlassView{hidden: true}
	v.clip = msgInitRect(send(class("MyGoClipView"), "alloc"), sel("initWithFrame:"), NSRect{})
	send(v.clip, "setWantsLayer:", 1)
	send(v.clip, "setAutoresizingMask:", nsViewMinYMargin)
	send(v.clip, "setHidden:", 1)
	v.glass = msgInitRect(send(class("NSGlassEffectView"), "alloc"), sel("initWithFrame:"), NSRect{})
	send(v.clip, "addSubview:", uintptr(v.glass))
	view := s.w.view
	switch {
	case len(s.systemGlass) > 0:
		send(view, "addSubview:positioned:relativeTo:", uintptr(v.clip), nsWindowAbove, uintptr(s.systemGlass[len(s.systemGlass)-1].clip))
	case s.w.effect != 0:
		send(view, "addSubview:positioned:relativeTo:", uintptr(v.clip), nsWindowAbove, uintptr(s.w.effect))
	default:
		send(view, "addSubview:positioned:relativeTo:", uintptr(v.clip), nsWindowBelow, 0)
	}
	return v
}

// nsWindowAbove is NSWindowAbove, for addSubview:positioned:relativeTo:.
const nsWindowAbove = 1

// place shows the pane as p tells, in a content view height points tall,
// changing what changed since it was last placed.
func (v *systemGlassView) place(p platform.SystemGlassPlacement, height float64) {
	old := v.shown
	if v.hidden || p.Clip != old.Clip {
		c := p.Clip
		msgSetRect(v.clip, sel("setFrame:"), NSRect{Origin: NSPoint{c.X, height - c.Y - c.H}, Size: NSSize{c.W, c.H}})
	}
	if v.hidden || p.Clip != old.Clip || p.Frame != old.Frame {
		// The clip view is flipped. It cuts the glass only where the
		// content clips it, which keeps the glass's own edges otherwise.
		c, f := p.Clip, p.Frame
		msgSetRect(v.glass, sel("setFrame:"), NSRect{Origin: NSPoint{f.X - c.X, f.Y - c.Y}, Size: NSSize{f.W, f.H}})
		send(send(v.clip, "layer"), "setMasksToBounds:", boolArg(c != f))
	}
	if v.hidden || p.Radius != old.Radius {
		msgSetFloat(v.glass, sel("setCornerRadius:"), p.Radius)
	}
	if v.hidden || p.Clear != old.Clear {
		style := uintptr(0) // NSGlassEffectViewStyleRegular
		if p.Clear {
			style = 1 // NSGlassEffectViewStyleClear
		}
		send(v.glass, "setStyle:", style)
	}
	if v.hidden || p.Tint != old.Tint {
		var tint id
		if p.Tint.A > 0 {
			tint = nsColor(p.Tint)
		}
		send(v.glass, "setTintColor:", uintptr(tint))
	}
	if v.hidden || p.Opacity != old.Opacity {
		msgSetFloat(v.clip, sel("setAlphaValue:"), p.Opacity)
	}
	if v.hidden {
		send(v.clip, "setHidden:", 0)
		v.hidden = false
	}
	v.shown = p
}

// destroySystemGlass takes the panes of ui.SystemGlass out of the window.
func (s *surface) destroySystemGlass() {
	for _, v := range s.systemGlass {
		send(v.clip, "removeFromSuperview")
		release(v.glass)
		release(v.clip)
	}
	s.systemGlass = nil
}
