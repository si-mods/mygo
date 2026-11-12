package terminal

import (
	"strings"
	"testing"
)

func TestURLAt(t *testing.T) {
	row := "see https://example.com/a_(b)?q=1. and (http://x.org/y), mailto:me@x.org!"
	text := []rune(row)
	cols := make([]int, len(text))
	for i := range cols {
		cols[i] = i
	}
	for col, want := range map[int]string{
		0:  "",
		4:  "https://example.com/a_(b)?q=1",
		20: "https://example.com/a_(b)?q=1",
		33: "", // the period after it
		40: "http://x.org/y",
		58: "mailto:me@x.org",
	} {
		if got, _, _ := urlAt(text, cols, col); got != want {
			t.Errorf("urlAt(%d) = %q, want %q", col, got, want)
		}
	}
}

// TestURLAtNotASCII: a URL ends where text that is not ASCII starts, as
// CJK or full-width punctuation after it with no space.
func TestURLAtNotASCII(t *testing.T) {
	for row, want := range map[string]string{
		"https://example.com/a的说明":      "https://example.com/a",
		"https://example.com/a，然后":      "https://example.com/a",
		"见https://example.com/a。":       "https://example.com/a",
		"（https://example.com/a）":       "https://example.com/a",
		"https://example.com/%E4%BD%A0": "https://example.com/%E4%BD%A0",
		"mailto:me@x.org，谢谢":            "mailto:me@x.org",
	} {
		text := []rune(row)
		cols := make([]int, len(text))
		for i := range cols {
			cols[i] = i
		}
		col := len([]rune(row[:strings.Index(row, ":")]))
		if got, _, _ := urlAt(text, cols, col); got != want {
			t.Errorf("urlAt(%q) = %q, want %q", row, got, want)
		}
	}
}

func TestPathAt(t *testing.T) {
	for _, c := range []struct {
		row  string
		col  int
		want string
	}{
		{"Updated internal/app/embed.go:51 and more", 10, "internal/app/embed.go:51"},
		{"  ⎿  Read /Users/me/repo/main.go (12 lines)", 12, "/Users/me/repo/main.go"},
		{"see ./notes/todo.txt.", 8, "./notes/todo.txt"},
		{"open ~/Downloads/clip.mp4", 10, "~/Downloads/clip.mp4"},
		{"main.go:250:3: undefined: x", 2, "main.go:250:3"},
		{"(src/a.go)", 3, "src/a.go"},
		{"write(notes.md)", 8, "notes.md"},
		{"notes.md (new file +2)", 2, "notes.md"},
		{"See notes.md.", 7, "notes.md"},
		{"notes.md:2:3", 2, "notes.md:2:3"},
		{"just words here", 5, ""},
		{"version 1.2.3 ok", 9, ""},
		{"docs/", 2, "docs/"},
		{"ls dist/ now", 4, "dist/"},
		{"see a/b/ and", 5, "a/b/"},
		{"README.md", 3, "README.md"},
		{"docs/a", 5, "docs/a"},
		{"cd docs/.", 5, "docs/"},
		{"in (docs/), then", 5, "docs/"},
		{"the foo/ dir", 5, "foo/"},
		{"the foo/ dir", 1, ""},
		{"cd ~/ now", 4, "~/"},
		{"cd ./ now", 4, "./"},
		{"cd ../ now", 4, "../"},
		{"and/or", 2, "and/or"},
		{"trailing / alone", 9, ""},
		{"/tmp/scratchpad/shot2_zoom.png。", 8, "/tmp/scratchpad/shot2_zoom.png"},
		{"/a/b.png，下一句", 3, "/a/b.png"},
		{"看 ./x.go:12。", 4, "./x.go:12"},
		{"（/tmp/a.txt）", 3, "/tmp/a.txt"},
		{"docs/中文.png", 6, "docs/中文.png"},
		{"写入 /tmp/报告.md。", 8, "/tmp/报告.md"},
		{"见 /tmp/a.png…", 4, "/tmp/a.png"},
		{"《/tmp/a.png〉", 3, "/tmp/a.png"},
		{"a/b.png）。", 2, "a/b.png"},
		{"./x.go:12”。", 2, "./x.go:12"},
	} {
		text := []rune(c.row)
		cols := make([]int, len(text))
		for i := range cols {
			cols[i] = i
		}
		if got, _, _ := pathAt(text, cols, c.col); got != c.want {
			t.Errorf("pathAt(%q, %d) = %q, want %q", c.row, c.col, got, c.want)
		}
	}
}

func TestPathAtSpanCJKPunct(t *testing.T) {
	text := []rune("看 /tmp/a.png”。")
	cols := make([]int, len(text))
	for i := range cols {
		cols[i] = i
	}
	if path, x0, x1 := pathAt(text, cols, 4); path != "/tmp/a.png" || x0 != 2 || x1 != 12 {
		t.Errorf("pathAt = %q %d..%d, want /tmp/a.png 2..12", path, x0, x1)
	}
}

func TestPathAtSpan(t *testing.T) {
	text := []rune("write(notes.md)")
	cols := make([]int, len(text))
	for i := range cols {
		cols[i] = i
	}
	if path, x0, x1 := pathAt(text, cols, 8); path != "notes.md" || x0 != 6 || x1 != 14 {
		t.Errorf("pathAt = %q %d..%d, want notes.md 6..14", path, x0, x1)
	}
}

func TestURLAtSpan(t *testing.T) {
	text := []rune("go to https://x.org/a. now")
	cols := make([]int, len(text))
	for i := range cols {
		cols[i] = i
	}
	if url, x0, x1 := urlAt(text, cols, 10); url != "https://x.org/a" || x0 != 6 || x1 != 21 {
		t.Errorf("urlAt = %q %d..%d, want https://x.org/a 6..21", url, x0, x1)
	}
}

func TestIsDir(t *testing.T) {
	for path, want := range map[string]bool{
		"docs/": true, "~/": true, "docs/:12": true, "w/:3:4": true, "/a/b/": true,
		`C:\Users\`: true, `C:\Users\:3`: true, `C:/x/`: true, `C:\a.txt`: false, `a\b\`: false,
		"docs": false, "a/b": false, "main.go:12": false, "a.go:1:2": false,
	} {
		if got := isDir(path); got != want {
			t.Errorf("isDir(%q) = %v, want %v", path, got, want)
		}
	}
}

// TestPathAtDrive: a path with a drive prefix takes \ as a separator and
// keeps the separators as printed; no other path does.
func TestPathAtDrive(t *testing.T) {
	for _, c := range []struct {
		row  string
		at   string // the text the click lands on, its first rune
		want string
		span string // the text the link covers, or "" for none
	}{
		{`修复前：截图 C:\Users\me\AppData\Local\Temp\ego-e2e\table-4.png。和你看到`, `Users`, `C:\Users\me\AppData\Local\Temp\ego-e2e\table-4.png`, `C:\Users\me\AppData\Local\Temp\ego-e2e\table-4.png`},
		{`截图 C:/Users/me/AppData/Local/Temp/table-4.png。和你`, `Users`, `C:/Users/me/AppData/Local/Temp/table-4.png`, `C:/Users/me/AppData/Local/Temp/table-4.png`},
		{`C:\a\b.png`, `a`, `C:\a\b.png`, `C:\a\b.png`},
		{`open D:/x/y.go now`, `x/`, `D:/x/y.go`, `D:/x/y.go`},
		{`C:\src\main.go:12:3`, `src`, `C:\src\main.go:12:3`, `C:\src\main.go:12:3`},
		{`C:\src\main.go(12,3)`, `src`, `C:\src\main.go:12:3`, `C:\src\main.go(12,3)`},
		{`C:\src\main.go:12:3: undefined`, `src`, `C:\src\main.go:12:3`, `C:\src\main.go:12:3`},
		{`C:\src\main.go:12:code`, `src`, `C:\src\main.go:12`, `C:\src\main.go:12`},
		{`--out=C:\build\app.js`, `build`, `C:\build\app.js`, `C:\build\app.js`},
		{`"C:\Program Files\app\a.txt"`, `Files`, `C:\Program Files\app\a.txt`, `"C:\Program Files\app\a.txt"`},
		{`"C:\Program Files\app\a.txt":12`, `Files`, `C:\Program Files\app\a.txt:12`, `"C:\Program Files\app\a.txt":12`},
		{`(C:\a\b.png)`, `a\b`, `C:\a\b.png`, `C:\a\b.png`},
		{`see C:\a\b.png.`, `a\b`, `C:\a\b.png`, `C:\a\b.png`},
		{`C:\a\b.png, then`, `a\b`, `C:\a\b.png`, `C:\a\b.png`},
		{`（C:\a\b.png）`, `a\b`, `C:\a\b.png`, `C:\a\b.png`},
		{`cd C:\Users\ now`, `Users`, `C:\Users\`, `C:\Users\`},
		// Not drive paths: the drive letter must start the path, and \ is no
		// separator elsewhere.
		{`abc:/foo`, `foo`, `/foo`, `/foo`},
		{`http:/x`, `x`, `/x`, `/x`},
		{`a\b\c`, `b`, ``, ``},
		{`\alpha and \n`, `alpha`, ``, ``},
		{`C:\`, `C`, `C:\`, `C:\`},
		{`C:foo.txt`, `foo`, `foo.txt`, `foo.txt`},
		{`12:30`, `12`, ``, ``},
	} {
		text := []rune(c.row)
		cols := make([]int, len(text))
		for i := range cols {
			cols[i] = i
		}
		col := len([]rune(c.row[:strings.Index(c.row, c.at)]))
		path, x0, x1 := pathAt(text, cols, col)
		if path != c.want || string(text[x0:x1]) != c.span {
			t.Errorf("pathAt(%q, %q) = %q %q, want %q %q", c.row, c.at, path, string(text[x0:x1]), c.want, c.span)
		}
	}
}
