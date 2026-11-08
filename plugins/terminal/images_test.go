package terminal

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	imagecolor "image/color"
	"image/draw"
	"image/png"
	"slices"
	"testing"

	"github.com/egoist/mygo/plugins/terminal/internal/vt"
	"github.com/egoist/mygo/ui"
)

// kittyImage returns the Kitty graphics sequence that transmits and
// places a w×h image of one color, as RGBA, with more keys.
func kittyImage(id, w, h int, c ui.Color, keys string) string {
	px := bytes.Repeat([]byte{c.R, c.G, c.B, 0xff}, w*h)
	return fmt.Sprintf("\x1b_Ga=T,f=32,s=%d,v=%d,i=%d,q=2%s;%s\x1b\\", w, h, id, keys, base64.StdEncoding.EncodeToString(px))
}

// Images programs place (Kitty graphics) show where they placed them,
// over the text unless placed under it, and go when deleted: grok shows a
// pasted image's preview so.
func TestImages(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), Theme: LightTheme()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	tt.Frame()
	at := func(col, row int) ui.Color {
		x, y := cellCenter(term, col, row)
		c := tt.Image().RGBAAt(int(x*term.v.scale), int(y*term.v.scale))
		return ui.RGB(c.R, c.G, c.B)
	}
	red, blue := ui.RGB(255, 0, 0), ui.RGB(0, 0, 255)
	bg := LightTheme().Background
	// Two cells by two of red at row 1, column 2; the cursor stays.
	term.Feed([]byte("\x1b[2;3H" + kittyImage(7, 2*term.v.cellW, 2*term.v.cellH, red, ",C=1")))
	tt.Frame()
	for _, c := range []struct {
		col, row int
		want     ui.Color
	}{{2, 1, red}, {3, 2, red}, {4, 1, bg}, {2, 3, bg}} {
		if got := at(c.col, c.row); got != c.want {
			t.Errorf("cell %d,%d is %v, want %v", c.col, c.row, got, c.want)
		}
	}
	// Under the cells' backgrounds, a cell with its own hides it.
	term.Feed([]byte("\x1b[5;1H" + kittyImage(8, term.v.cellW, term.v.cellH, blue, ",z=-1073741825,C=1") + "\x1b[42m \x1b[0m"))
	tt.Frame()
	if got := at(0, 4); got == blue {
		t.Errorf("an image under the backgrounds shows over a cell's own")
	}
	// Deleted, it goes.
	term.Feed([]byte("\x1b_Ga=d,d=I,i=7,q=2\x1b\\"))
	tt.Frame()
	if got := at(2, 1); got != bg {
		t.Errorf("a deleted image's cell is %v, want %v", got, bg)
	}
}

// Images sent as PNG (f=100), as grok sends a pasted image's preview,
// decode and show.
func TestImagePNG(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), Theme: LightTheme()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	tt := ui.NewTester(func(c *ui.Context) { View(c, term).Fill() }, 400, 200)
	tt.Frame()
	src := image.NewNRGBA(image.Rect(0, 0, 3*term.v.cellW, 2*term.v.cellH))
	draw.Draw(src, src.Bounds(), image.NewUniform(imagecolor.NRGBA{G: 255, A: 255}), image.Point{}, draw.Src)
	var b bytes.Buffer
	png.Encode(&b, src)
	data := base64.StdEncoding.EncodeToString(b.Bytes())
	// In chunks, as programs send big ones.
	term.Feed([]byte("\x1b[2;2H\x1b_Ga=T,f=100,q=2,m=1;" + data[:40] + "\x1b\\\x1b_Gm=0;" + data[40:] + "\x1b\\"))
	tt.Frame()
	x, y := cellCenter(term, 2, 2)
	c := tt.Image().RGBAAt(int(x*term.v.scale), int(y*term.v.scale))
	if got := ui.RGB(c.R, c.G, c.B); got != ui.RGB(0, 255, 0) {
		t.Errorf("the image's cell is %v, want green", got)
	}
}

// placements returns the placements of the screen that shows that a
// program can place again: the virtual ones and the direct ones whose top
// is on the screen.
func placements(term *Terminal) []vt.Placement {
	term.mu.Lock()
	defer term.mu.Unlock()
	var out []vt.Placement
	term.term.Placements(func(pl vt.Placement) {
		if pl.Virtual || pl.Row >= 0 {
			out = append(out, pl)
		}
	})
	slices.SortFunc(out, func(a, b vt.Placement) int { return int(a.ImageID) - int(b.ImageID) })
	return out
}

// Images brings the images a terminal shows now, not those it showed, to
// a terminal its snapshot made: each image's pixels once, its placements
// where they are with the program's ids, the cursor left as it was.
func TestImagesExport(t *testing.T) {
	loadLib(t)
	red, blue := ui.RGB(255, 0, 0), ui.RGB(0, 0, 255)
	// No view tells the size of a cell in pixels, as on a server: an
	// image's cells are unknown.
	a := headless(t)
	a.Resize(80, 24)
	// Sent again, as a program redrawing at a new size does: the last one
	// counts.
	for range 5 {
		a.Feed([]byte("\x1b[2;3H" + kittyImage(7, 4, 4, red, ",p=2,c=2,r=1,C=1")))
	}
	a.Feed([]byte("\x1b[4;5H" + kittyImage(8, 2, 2, blue, ",C=1")))
	// Deleted.
	a.Feed([]byte("\x1b[5;1H" + kittyImage(9, 2, 2, blue, ",C=1") + "\x1b_Ga=d,d=I,i=9,q=2\x1b\\"))
	// Virtual, for Unicode placeholders.
	a.Feed([]byte(kittyImage(10, 2, 2, red, ",U=1,p=3,c=2,r=1")))
	// Scrolled off, it can be placed again no more.
	a.Feed([]byte("\x1b[1;1H" + kittyImage(11, 2, 2, red, ",C=1")))
	a.Feed([]byte("\x1b[24;1H\r\n"))
	// At the top, which the cells it takes, unknown, do not tell visible.
	a.Feed([]byte("\x1b[1;1H" + kittyImage(12, 2, 2, blue, ",C=1")))
	a.Feed([]byte("\x1b[3;7Hhi"))

	images := a.Images()
	if n := bytes.Count(images, []byte("a=t,")); n != 4 {
		t.Errorf("%d images sent, want 4 (7, 8, 10, 12):\n%q", n, images)
	}
	for _, want := range []string{"i=7,p=2,", "a=p,U=1,i=10,p=3,c=2,r=1,"} {
		if !bytes.Contains(images, []byte(want)) {
			t.Errorf("images lack %q:\n%q", want, images)
		}
	}
	for _, gone := range []string{"i=9,", "i=11,"} {
		if bytes.Contains(images, []byte(gone)) {
			t.Errorf("images have %q:\n%q", gone, images)
		}
	}

	b := headless(t)
	b.Resize(80, 24)
	b.Feed(a.Snapshot())
	b.Feed(images)
	want, got := placements(a), placements(b)
	if len(want) != 4 || !slices.Equal(got, want) {
		t.Errorf("placements after the snapshot and images\n%+v, want\n%+v", got, want)
	}
	if x, y := cursor(b); x != 8 || y != 2 {
		t.Errorf("cursor after the images = %d,%d, want 8,2", x, y)
	}
	a.mu.Lock()
	img, _ := a.term.Image(7)
	a.mu.Unlock()
	b.mu.Lock()
	got7, ok := b.term.Image(7)
	b.mu.Unlock()
	if !ok || !bytes.Equal(decoded(got7).Pix, decoded(img).Pix) {
		t.Errorf("image 7's pixels did not come")
	}
	// What the program sends next acts on them.
	b.Feed([]byte("\x1b_Ga=d,d=i,i=7,p=2,q=2\x1b\\"))
	if pl := placements(b); len(pl) != 3 {
		t.Errorf("after deleting placement 2 of image 7: %+v", pl)
	}
}
