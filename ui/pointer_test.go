package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// TestPointer: Painter.Pointer and Context.Pointer read where the pointer
// is in the window, whether it is over it, and whether its primary button
// is down.
func TestPointer(t *testing.T) {
	var got, built pointerState
	tt := NewTester(func(c *Context) {
		built.x, built.y, built.over, built.primary = c.Pointer()
		Box(c).Fill().Draw(func(p *Painter, r Rect) {
			got.x, got.y, got.over, got.primary = p.Pointer()
		})
	}, 100, 80)
	tt.Move(30, 20)
	if want := (pointerState{30, 20, true, false}); got != want || built != want {
		t.Errorf("after a move: painted %+v, built %+v, want %+v", got, built, want)
	}
	tt.Press(40, 25)
	if want := (pointerState{40, 25, true, true}); got != want {
		t.Errorf("pressed: %+v, want %+v", got, want)
	}
	tt.Release(40, 25)
	if got.primary {
		t.Errorf("released: %+v", got)
	}
	tt.PressButton(1, 40, 25)
	if got.primary {
		t.Errorf("the secondary button pressed reads as the primary: %+v", got)
	}
	tt.ReleaseButton(1, 40, 25)
	tt.send(platform.SurfaceEvent{Kind: platform.PointerLeave})
	tt.Frame()
	if want := (pointerState{40, 25, false, false}); got != want {
		t.Errorf("left: %+v, want %+v", got, want)
	}
}

// watchView counts the frames built and painted, and watches the pointer
// at fps when it is above 0.
func watchView(fps *float32, builds, paints *int) func(*Context) {
	return func(c *Context) {
		*builds++
		Box(c).Fill().Draw(func(p *Painter, r Rect) {
			*paints++
			p.WatchPointer(*fps)
		})
	}
}

// move sends a pointer move as a window does, without the Tester's frame
// after each event.
func move(tt *Tester, x, y float32) {
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: float64(x), Y: float64(y)})
}

// TestWatchPointer: a frame that watches the pointer has its moves paint
// the window again, without building it, at most at the rate it asks for;
// a frame that does not asks for none.
func TestWatchPointer(t *testing.T) {
	var fps float32
	var builds, paints int
	tt := NewTester(watchView(&fps, &builds, &paints), 100, 80)

	// Not watched: a move asks for nothing.
	move(tt, 10, 10)
	if tt.FramePending() || !tt.h.repaintAt.IsZero() {
		t.Fatalf("an unwatched move asks for a frame (pending %v, timer %v)", tt.FramePending(), tt.h.repaintAt)
	}

	fps = 30
	tt.Frame()
	start := tt.Now()
	b, p := builds, paints
	// Within 1/30 s of the frame: one frame, when it is due.
	move(tt, 20, 10)
	move(tt, 30, 10)
	move(tt, 40, 10)
	if tt.FramePending() {
		t.Fatal("a move asks for a frame at once, within 1/30 s of the last one")
	}
	if want := start.Add(time.Second / 30); !tt.h.repaintAt.Equal(want) {
		t.Fatalf("the timer is at %v, want %v", tt.h.repaintAt.Sub(start), want.Sub(start))
	}
	tt.Advance(50 * time.Millisecond)
	if builds != b || paints != p+1 {
		t.Errorf("three moves within 1/30 s: %d builds and %d paints, want 0 and 1", builds-b, paints-p)
	}
	// Past 1/30 s since the last frame, a move paints at once.
	b, p = builds, paints
	tt.Advance(100 * time.Millisecond)
	move(tt, 50, 10)
	if !tt.FramePending() {
		t.Fatal("a move 1/10 s after the last frame asks for none at once")
	}
	tt.Advance(20 * time.Millisecond)
	if builds != b || paints != p+1 {
		t.Errorf("a late move: %d builds and %d paints, want 0 and 1", builds-b, paints-p)
	}
	// A move to where the pointer is changes nothing.
	tt.Advance(100 * time.Millisecond)
	move(tt, 50, 10)
	if tt.FramePending() || !tt.h.repaintAt.IsZero() {
		t.Error("a move to the same place asks for a frame")
	}
	// Leaving the window and pressing count too.
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerLeave})
	if !tt.FramePending() {
		t.Error("the pointer leaving asks for no frame")
	}

	// A frame that stops watching ends the watch.
	fps = 0
	tt.Frame()
	tt.Advance(100 * time.Millisecond)
	move(tt, 60, 10)
	if tt.FramePending() || !tt.h.repaintAt.IsZero() {
		t.Error("a move after the watch stopped asks for a frame")
	}
}

// TestWatchPointerTargets: watching the pointer changes no press target,
// and moves over an element that takes them still paint.
func TestWatchPointerTargets(t *testing.T) {
	clicks, took := 0, 0
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Draw(func(p *Painter, r Rect) { p.WatchPointer(30) }).Children(func() {
			if Box(c).Size(40, 20).Clicked() {
				clicks++
			}
			Box(c).Size(100, 40).HandleInput(func(ev InputEvent) bool {
				if ev.Kind == InputPointerMove {
					took++
				}
				return true
			})
		})
	}, 100, 80)
	tt.Press(80, 70)
	if tt.rt.pressed != nil {
		t.Errorf("a press on nothing presses %v", tt.rt.pressed.id)
	}
	tt.Release(80, 70)
	tt.ClickAt(10, 10)
	if clicks != 1 {
		t.Errorf("the box got %d clicks, want 1", clicks)
	}
	tt.Advance(100 * time.Millisecond)
	move(tt, 50, 40)
	if took == 0 || !tt.FramePending() {
		t.Errorf("a move over a handler: taken %d times, frame pending %v", took, tt.FramePending())
	}
}

// TestWatchPointerAnimation: a move while a drawing animates at the
// display's rate adds no frame: the animation's next one paints it, and
// stops the watch's timer.
func TestWatchPointerAnimation(t *testing.T) {
	paints := 0
	tt := NewTester(func(c *Context) {
		Box(c).Fill().Draw(func(p *Painter, r Rect) {
			paints++
			p.AnimationFrame()
			p.WatchPointer(30)
		})
	}, 100, 80)
	p := paints
	move(tt, 20, 10)
	tt.Advance(34 * time.Millisecond)
	if got := paints - p; got != 2 {
		t.Errorf("an animation at 60 a second and a move paint %d frames in 34 ms, want 2", got)
	}
}

// TestWatchPointerEnds: the frame that stops watching stops the timer a
// move armed.
func TestWatchPointerEnds(t *testing.T) {
	var fps float32 = 30
	var builds, paints int
	tt := NewTester(watchView(&fps, &builds, &paints), 100, 80)
	move(tt, 20, 10)
	if tt.h.repaintAt.IsZero() {
		t.Fatal("a move within 1/30 s armed no timer")
	}
	fps = 0
	tt.Frame()
	if !tt.h.repaintAt.IsZero() {
		t.Errorf("the timer stays armed at %v after the watch ended", tt.h.repaintAt.Sub(tt.Now()))
	}
	p := paints
	tt.Advance(100 * time.Millisecond)
	if paints != p {
		t.Errorf("%d frames painted after the watch ended", paints-p)
	}
}

// TestWatchPointerPress: a press while watched builds the view anew, as
// without the watch.
func TestWatchPointerPress(t *testing.T) {
	var fps float32 = 30
	var builds, paints int
	tt := NewTester(watchView(&fps, &builds, &paints), 100, 80)
	tt.Advance(100 * time.Millisecond)
	b := builds
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerDown, X: 20, Y: 10, Clicks: 1})
	if !tt.FramePending() {
		t.Fatal("a press asks for no frame")
	}
	tt.Advance(20 * time.Millisecond)
	if builds != b+1 {
		t.Errorf("a press built %d frames, want 1", builds-b)
	}
}
