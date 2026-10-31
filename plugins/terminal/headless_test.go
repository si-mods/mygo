package terminal

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/egoist/mygo/plugins/terminal/internal/vt"
)

// headless makes a terminal with no view and no program: Conn never
// answers, and Feed shows what a program would write.
func headless(t *testing.T) *Terminal {
	t.Helper()
	term, err := New(Options{Conn: newPipe()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { term.Close() })
	return term
}

// cursor returns the cursor's column and row of a terminal's screen.
func cursor(t *Terminal) (x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.term.Cursor()
}

func TestGhosttySnapshot(t *testing.T) {
	loadLib(t)
	term := headless(t)
	term.Resize(24, 6)
	term.Feed([]byte("\x1b[1;31mred\x1b[0m \x1b[4munder\x1b[0m\r\n"))
	for i := range 20 {
		term.Feed([]byte("line " + itoa(i) + "\r\n"))
	}
	term.Feed([]byte("last"))
	if cols, rows := term.Size(); cols != 24 || rows != 6 {
		t.Fatalf("size = %d×%d", cols, rows)
	}
	wantX, wantY := cursor(term)
	want := term.Text()
	if !strings.Contains(want, "line 000") || !strings.Contains(want, "line 014") {
		t.Fatalf("text = %q, want the scrollback in it", want)
	}

	snap, err := term.GhosttySnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(snap, []byte("GHOSTSNP")) {
		t.Fatalf("snapshot starts %q", snap[:8])
	}
	if v := binary.LittleEndian.Uint16(snap[8:10]); v != 1 {
		t.Fatalf("snapshot version = %d, want 1", v)
	}

	other := headless(t)
	other.Resize(24, 6)
	if err := other.RestoreGhosttySnapshot(snap); err != nil {
		t.Fatal(err)
	}
	if got := other.Text(); got != want {
		t.Errorf("text after restore = %q, want %q", got, want)
	}
	if x, y := cursor(other); x != wantX || y != wantY {
		t.Errorf("cursor after restore = %d,%d, want %d,%d", x, y, wantX, wantY)
	}
	if cols, rows := other.Size(); cols != 24 || rows != 6 {
		t.Errorf("size after restore = %d×%d, want 24×6", cols, rows)
	}
	// The restored terminal is a terminal of its own: output goes on it.
	other.Feed([]byte("+"))
	if got := other.Text(); got != want+"+" && got != strings.TrimRight(want, " ")+"+" {
		t.Errorf("text after output = %q", got)
	}
}

func TestEncodeKey(t *testing.T) {
	loadLib(t)
	term := headless(t)
	up := KeyEvent{Key: ParseKey("ArrowUp"), Action: KeyPress}
	if got := term.EncodeKey(up); string(got) != "\x1b[A" {
		t.Errorf("ArrowUp = %q, want %q", got, "\x1b[A")
	}
	term.Feed([]byte("\x1b[?1h"))
	if got := term.EncodeKey(up); string(got) != "\x1bOA" {
		t.Errorf("ArrowUp in application mode = %q, want %q", got, "\x1bOA")
	}
	ctrlC := KeyEvent{Key: ParseKey("KeyC"), Action: KeyPress, Mods: ModCtrl, Unshifted: 'c'}
	if got := term.EncodeKey(ctrlC); !bytes.Equal(got, []byte{0x03}) {
		t.Errorf("Ctrl+C = %q, want 0x03", got)
	}
	if got := term.EncodeKey(KeyEvent{Key: ParseKey("Enter"), Action: KeyPress}); string(got) != "\r" {
		t.Errorf("Enter = %q, want %q", got, "\r")
	}
	term.Feed([]byte("\x1b[>1u"))
	if got := term.EncodeKey(KeyEvent{Key: ParseKey("Escape"), Action: KeyPress}); string(got) != "\x1b[27u" {
		t.Errorf("Escape with the Kitty protocol = %q, want %q", got, "\x1b[27u")
	}
}

func TestParseKey(t *testing.T) {
	for code, want := range map[string]Key{
		"KeyA":      vt.KeyA,
		"ArrowUp":   vt.KeyArrowUp,
		"Numpad1":   vt.KeyNumpad1,
		"F12":       vt.KeyF12,
		"Escape":    vt.KeyEscape,
		"ShiftLeft": vt.KeyShiftLeft,
	} {
		if got := ParseKey(code); got != want {
			t.Errorf("ParseKey(%q) = %d, want %d", code, got, want)
		}
	}
	if got := ParseKey("NoSuchKey"); got != KeyUnidentified {
		t.Errorf("ParseKey of an unknown code = %d, want unidentified", got)
	}
}

func TestEncodePaste(t *testing.T) {
	loadLib(t)
	term := headless(t)
	if got := term.EncodePaste("hi"); string(got) != "hi" {
		t.Errorf("paste without bracketing = %q, want %q", got, "hi")
	}
	term.Feed([]byte("\x1b[?2004h"))
	if got := term.EncodePaste("hi"); string(got) != "\x1b[200~hi\x1b[201~" {
		t.Errorf("bracketed paste = %q, want %q", got, "\x1b[200~hi\x1b[201~")
	}
}

func TestEncodeFocus(t *testing.T) {
	loadLib(t)
	term := headless(t)
	if got := term.EncodeFocus(true); got != nil {
		t.Errorf("focus report without mode 1004 = %q, want nil", got)
	}
	term.Feed([]byte("\x1b[?1004h"))
	if got := term.EncodeFocus(true); string(got) != "\x1b[I" {
		t.Errorf("focus gained = %q, want %q", got, "\x1b[I")
	}
	if got := term.EncodeFocus(false); string(got) != "\x1b[O" {
		t.Errorf("focus lost = %q, want %q", got, "\x1b[O")
	}
}

func TestEncodeMouse(t *testing.T) {
	loadLib(t)
	term := headless(t)
	press := MouseEvent{Action: MousePress, Button: MouseLeft, Col: 2, Row: 3}
	if got := term.EncodeMouse(press); got != nil {
		t.Errorf("mouse report without tracking = %q, want nil", got)
	}
	term.Feed([]byte("\x1b[?1000h\x1b[?1006h"))
	if got := term.EncodeMouse(press); string(got) != "\x1b[<0;3;4M" {
		t.Errorf("left press = %q, want %q", got, "\x1b[<0;3;4M")
	}
	if got := term.EncodeMouse(MouseEvent{Action: MouseRelease, Button: MouseLeft, Col: 2, Row: 3}); string(got) != "\x1b[<0;3;4m" {
		t.Errorf("left release = %q, want %q", got, "\x1b[<0;3;4m")
	}
}
