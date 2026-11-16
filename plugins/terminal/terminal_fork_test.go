package terminal

import (
	"slices"
	"strings"
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

// TestCommandClickDirs: a file path printed as text is a link only where
// Options.LinkPath takes it, so that prose (w/, and/or) is none; without
// it, every path but a directory, one that ends in a slash, :line or not.
// A URL that ends in a slash is the URL, clicked in its path.
func TestCommandClickDirs(t *testing.T) {
	loadLib(t)
	row := "ls docs/ w/ ~/ ../ src/app/ docs/:12 w/:3 and/or https://x.org/a/ ok"
	words := []string{"docs/", "w/", "~/", "../", "src/app/", "docs/:12", "w/:3", "and/or", "x.org/a/"}
	for _, c := range []struct {
		name  string
		paths []string // what LinkPath takes; nil: no LinkPath
		want  []string
	}{
		{"LinkPath", []string{"docs/", "~/", "../", "src/app/", "docs/:12"}, []string{"docs/", "~/", "../", "src/app/", "docs/:12", "https://x.org/a/"}},
		{"no LinkPath", nil, []string{"and/or", "https://x.org/a/"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var opened, asked []string
			opts := Options{Conn: newPipe(), OpenLink: func(link string) bool {
				opened = append(opened, link)
				return true
			}}
			if c.paths != nil {
				opts.LinkPath = func(path string) bool {
					asked = append(asked, path)
					return slices.Contains(c.paths, path)
				}
			}
			term, err := New(opts)
			if err != nil {
				t.Fatal(err)
			}
			defer term.Close()
			term.Feed([]byte(row))
			tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill().AutoFocus() }, 1000, 200)
			tt.Frame()
			for _, word := range words {
				col := strings.Index(row, " "+word+" ") + 2
				if word == "x.org/a/" {
					col = strings.Index(row, word) + 1
				}
				x, y := cellCenter(term, col, 0)
				tt.ClickAtWith(ui.Cmd, x, y)
			}
			if !slices.Equal(opened, c.want) {
				t.Errorf("opened %q, want %q", opened, c.want)
			}
			if c.paths != nil && !slices.Contains(asked, "and/or") {
				t.Errorf("LinkPath was not asked of and/or: %q", asked)
			}
			// Hovered with Command, w/ is no link: nothing underlined.
			x, y := cellCenter(term, strings.Index(row, " w/ ")+1, 0)
			tt.Move(x, y)
			tt.HoldModifiers(ui.Cmd)
			if h := term.v.hover; h.on() {
				t.Errorf("w/ hovered as a link: %+v", h)
			}
			tt.HoldModifiers(0)
		})
	}
}
