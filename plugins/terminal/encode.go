package terminal

import (
	"slices"

	"github.com/egoist/mygo/plugins/terminal/internal/vt"
)

// KeyAction is what happened to a key.
type KeyAction int

const (
	// KeyPress is a key pressed: the zero value, the common case.
	KeyPress KeyAction = iota
	KeyRelease
	KeyRepeat
)

// Modifiers are the modifier keys held with a key or the pointer: the same
// bits as ui.Modifiers for Shift, Ctrl, Alt and Super, and Ghostty's for
// Caps Lock, Num Lock and the right-hand side of each modifier, which is
// only meaningful beside its modifier (ModShiftRight beside ModShift).
type Modifiers uint16

const (
	ModShift Modifiers = 1 << iota
	ModCtrl
	ModAlt
	ModSuper
	ModCapsLock
	ModNumLock
	ModShiftRight
	ModCtrlRight
	ModAltRight
	ModSuperRight
)

// KeyEvent is a key to encode for the program, as a W3C KeyboardEvent
// carries one: the physical key (Key, from its code), the action, the
// modifiers held and those that made its text, the text typed, the
// character the key types without modifiers, and whether an input method
// is composing.
type KeyEvent struct {
	// Key is the key by its place on the keyboard, as ParseKey reads a
	// KeyboardEvent.code.
	Key Key
	// Action is what the key did: pressed, released or repeated.
	Action KeyAction
	// Mods are the modifiers held.
	Mods Modifiers
	// Consumed are the modifiers that made Text, as Shift an "A".
	Consumed Modifiers
	// Text is what the key types, as UTF-8, without what Ctrl or Alt
	// change; empty for keys that type nothing.
	Text string
	// Unshifted is the character the key types without modifiers.
	Unshifted rune
	// Composing tells that an input method is composing.
	Composing bool
}

// vtKeyEvent converts e to what the encoder takes.
func (e KeyEvent) vtKeyEvent() vt.KeyEvent {
	action := vt.KeyPress
	switch e.Action {
	case KeyRelease:
		action = vt.KeyRelease
	case KeyRepeat:
		action = vt.KeyRepeat
	}
	return vt.KeyEvent{
		Action: action, Key: e.Key, Mods: vt.Mods(e.Mods), Consumed: vt.Mods(e.Consumed),
		Text: e.Text, Unshifted: e.Unshifted, Composing: e.Composing,
	}
}

// EncodeKey returns the bytes the program reads for ev, encoded as the
// terminal's modes ask: legacy sequences, xterm's modifyOtherKeys, or the
// Kitty keyboard protocol, with cursor key application and Option as Alt
// (Options.OptionAsAlt). It is nil for a key that sends nothing.
func (t *Terminal) EncodeKey(ev KeyEvent) []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.term == nil {
		return nil
	}
	if t.keys == nil {
		e, err := vt.NewKeyEncoder()
		if err != nil {
			return nil
		}
		t.keys = e
	}
	t.keys.Sync(t.term, t.opts.OptionAsAlt)
	return slices.Clone(t.keys.Encode(ev.vtKeyEvent()))
}

// MouseAction is what the pointer did.
type MouseAction int

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseMotion
)

// MouseButton is the button of a pointer event: 1 to 3 as a browser names
// them, then the wheel's.
type MouseButton int

const (
	MouseNoButton MouseButton = iota
	MouseLeft
	MouseRight
	MouseMiddle
	MouseWheelUp
	MouseWheelDown
	MouseWheelLeft
	MouseWheelRight
)

// MouseEvent is a pointer event in cells: Col and Row count from zero from
// the top left of the grid, as a client computing them from a click in a
// view of it would.
type MouseEvent struct {
	// Action is what the pointer did.
	Action MouseAction
	// Button is the button pressed, released or held; MouseNoButton for
	// motion without one.
	Button MouseButton
	// Mods are the modifiers held.
	Mods Modifiers
	// Col and Row are the cell of the pointer.
	Col, Row int
}

// vtMouseAction and vtMouseButton convert what the encoder takes.
func (a MouseAction) vtMouseAction() vt.MouseAction {
	switch a {
	case MouseRelease:
		return vt.MouseRelease
	case MouseMotion:
		return vt.MouseMotion
	}
	return vt.MousePress
}

func (b MouseButton) vtMouseButton() int {
	switch b {
	case MouseLeft:
		return vt.ButtonLeft
	case MouseRight:
		return vt.ButtonRight
	case MouseMiddle:
		return vt.ButtonMiddle
	case MouseWheelUp:
		return vt.WheelUp
	case MouseWheelDown:
		return vt.WheelDown
	case MouseWheelLeft:
		return vt.WheelLeft
	case MouseWheelRight:
		return vt.WheelRight
	}
	return vt.ButtonNone
}

// EncodeMouse returns the bytes the program reads for ev, in the format
// and for the events it asked for (the mouse modes), within reason: nil
// when it asked for none, and for the motion it did not ask for. The
// cell's size is the terminal's (ResizePixels), so a report in pixels
// (mode 1016) counts the cell's top left corner.
func (t *Terminal) EncodeMouse(ev MouseEvent) []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.term == nil {
		return nil
	}
	if t.mouse == nil {
		m, err := vt.NewMouseEncoder()
		if err != nil {
			return nil
		}
		t.mouse = m
	}
	switch ev.Action {
	case MousePress:
		t.mousePressed = true
	case MouseRelease:
		t.mousePressed = false
	}
	cols, rows := t.term.Size()
	cellW, cellH := t.size.cellW, t.size.cellH
	t.mouse.Sync(t.term, cols*cellW, rows*cellH, cellW, cellH, 0, 0, t.mousePressed)
	col, row := max(ev.Col, 0), max(ev.Row, 0)
	return slices.Clone(t.mouse.Encode(ev.Action.vtMouseAction(), ev.Button.vtMouseButton(), vt.Mods(ev.Mods), float32(col*cellW), float32(row*cellH)))
}

// EncodeFocus returns the report of the terminal's window gaining or
// losing the focus (mode 1004), or nil when the program did not ask for
// focus reports.
func (t *Terminal) EncodeFocus(in bool) []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.term == nil || !t.term.Mode(vt.ModeFocusEvent) {
		return nil
	}
	return vt.EncodeFocus(in)
}

// EncodePaste returns text as the program reads a paste: bracketed when it
// asked for bracketed pastes (mode 2004), and with newlines as carriage
// returns and the control characters that could run commands dropped
// otherwise. It is nil for empty text.
func (t *Terminal) EncodePaste(text string) []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.term == nil {
		return nil
	}
	return vt.EncodePaste(text, t.term.Mode(vt.ModeBracketedPaste))
}
