package terminal

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestDrawnMenu checks that a window drawing its menus (SetDrawMenus)
// shows the terminal's context menu in the window, and that Paste reaches
// the program.
func TestDrawnMenu(t *testing.T) {
	loadLib(t)
	conn := newPipe()
	term, err := New(Options{Conn: conn})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Feed([]byte("some text"))
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetDrawMenus(true)
		View(c, term).Fill().AutoFocus()
	}, 400, 200)
	x, y := cellCenter(term, 1, 0)
	tt.RightClickAt(x, y)
	// Menu reports the menu drawn in the window too (SetMenuPresentation).
	if got := tt.Menu(); !slices.Contains(got, "Copy") || !slices.Contains(got, "Clear") {
		t.Errorf("drawn menu = %q", got)
	}
	if !tt.HasText("Copy") || !tt.HasText("Clear") {
		t.Fatalf("no drawn menu: %q", tt.Texts())
	}
	tt.SetClipboard("drawn")
	if err := tt.Click("Paste"); err != nil {
		t.Fatal(err)
	}
	if got := conn.take(term); got != "drawn" {
		t.Errorf("the drawn menu's Paste sent %q", got)
	}
	if tt.HasText("Clear") {
		t.Error("the menu stays after a choice")
	}
}

func TestProgress(t *testing.T) {
	loadLib(t)
	var got [][2]int
	term, err := New(Options{Conn: newPipe(), OnProgress: func(state, percent int) {
		got = append(got, [2]int{state, percent})
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	// One report a write: the last of several in one.
	term.Feed([]byte("\x1b]9;4;1;40\x1b\\"))
	term.Feed([]byte("\x1b]9;4;1;50\x07\x1b]9;4;2;70\x07"))
	term.Feed([]byte("\x1b]9;4;3\x07"))
	term.Feed([]byte("plain text"))
	term.Feed([]byte("\x1b]9;4;0\x07"))
	want := [][2]int{{1, 40}, {2, 70}, {3, -1}, {0, -1}}
	if !slices.Equal(got, want) {
		t.Errorf("progress = %v, want %v", got, want)
	}
}
