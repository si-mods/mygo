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
		{`PATH=C:\bin;D:\bin`, `C:`, `C:\bin`, `C:\bin`},
		{`PATH=C:\bin;D:\bin`, `D:`, `D:\bin`, `D:\bin`},
		{`Option C:/D`, `C:`, `C:/D`, `C:/D`}, // accepted: a drive path as far as it looks
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

// mkRow is a viewport row for pathAtRows: a wide rune takes two columns.
func mkRow(s string, wrap, full bool) linkRow {
	r := linkRow{wrap: wrap, full: full}
	x := 0
	for _, c := range s {
		r.text, r.cols = append(r.text, c), append(r.cols, x)
		x++
		if c > 0x2e80 || c == '●' {
			x++
		}
	}
	r.width = x
	return r
}

func TestPathAtRows(t *testing.T) {
	const win = `C:\Users\me\AppData\Local\Temp\claude\b2137de8ae8437d6\scratchpad\ego-e2e\table-4.png`
	any := func(string) bool { return true }
	for _, c := range []struct {
		name    string
		rows    []linkRow
		at, col int
		isLink  func(string) bool
		path    string
		x0, x1  int
		asked   int // how many times isLink is called; -1: not checked
	}{
		{"program wrap, first row", []linkRow{
			mkRow(`● 修复前：截图 C:\Users\me\AppData\Local\Temp\claude\b2137de8`, false, true),
			mkRow(`  ae8437d6\scratchpad\ego-e2e\table-4.png。和你看到`, false, false),
		}, 0, 30, any, win, 16, 62, -1},
		{"program wrap, second row", []linkRow{
			mkRow(`● 修复前：截图 C:\Users\me\AppData\Local\Temp\claude\b2137de8`, false, true),
			mkRow(`  ae8437d6\scratchpad\ego-e2e\table-4.png。和你看到`, false, false),
		}, 1, 20, any, win, 2, 41, -1},
		{"program wrap, hover in the indent", []linkRow{
			mkRow(`see C:\Users\me\AppData\Local\Temp\claude\b2137de8`, false, true),
			mkRow(`  ae8437d6\x.png`, false, false),
		}, 1, 0, func(p string) bool { return strings.HasSuffix(p, "x.png") }, "", 0, 0, -1},
		{"three rows", []linkRow{
			mkRow("open /aaaa/bbbb/cccc", false, true),
			mkRow("  dddd/eeee/ffff", false, true),
			mkRow("  gggg/hhhh.txt ok", false, false),
		}, 1, 5, any, "/aaaa/bbbb/cccc" + "dddd/eeee/ffff" + "gggg/hhhh.txt", 2, 16, -1},
		{"soft wrap keeps the next row's spaces", []linkRow{
			mkRow("see src/aaaa/bbbb", true, true),
			mkRow("  cccc.go", false, false),
		}, 0, 6, any, "src/aaaa/bbbb", 4, 17, -1},
		{"soft wrap continues exactly", []linkRow{
			mkRow("see src/aaaa/bb", true, false),
			mkRow("bb/cccc.go ok", false, false),
		}, 1, 3, any, "src/aaaa/bbbb/cccc.go", 0, 10, -1},
		{"over-joined candidate rejected by isLink", []linkRow{
			mkRow("see src/aaaa/main.go", false, true),
			mkRow("  for details", false, false),
		}, 0, 10, func(p string) bool { return p == "src/aaaa/main.go" }, "src/aaaa/main.go", 4, 20, 2},
		{"short words at the edge are no wrap", []linkRow{
			mkRow("... see src/main.go", false, true),
			mkRow("  for details", false, false),
		}, 0, 10, any, "src/main.go", 8, 19, 1},
		{"rule row", []linkRow{
			mkRow(strings.Repeat("─", 20), false, true),
			mkRow("foo.txt is here", false, false),
		}, 1, 2, any, "foo.txt", 0, 7, 1},
		{"prose ending at the edge", []linkRow{
			mkRow("some prose see the", false, true),
			mkRow("  config.yaml is here", false, false),
		}, 1, 5, any, "config.yaml", 2, 13, 1},
		{"full-width punctuation at the edge", []linkRow{
			mkRow("见文件 src/aaaa/bbbb。", false, true),
			mkRow("  cccc/dddd.go", false, false),
		}, 1, 5, any, "cccc/dddd.go", 2, 14, 1},
		{"a row that does not go on is not joined", []linkRow{
			mkRow("see src/aaaa", false, false),
			mkRow("bbbb/cccc.go", false, false),
		}, 1, 3, any, "bbbb/cccc.go", 0, 12, -1},
		{"soft-wrapped path is one candidate", []linkRow{
			mkRow("see /aaaa/bbbb/cccc", true, true),
			mkRow("dddd/eeee/ffff", true, true),
			mkRow("gggg/hhhh.txt ok", false, false),
		}, 1, 5, func(string) bool { return false }, "", 0, 0, 1},
		{"program-wrapped path, only the two rows taken", []linkRow{
			mkRow("open /aaaa/bbbb/cccc", false, true),
			mkRow("  dddd/eeee/ffff", false, true),
			mkRow("  gggg/hhhh.txt ok", false, false),
		}, 1, 5, func(p string) bool { return p == "/aaaa/bbbb/ccccdddd/eeee/ffff" }, "/aaaa/bbbb/ccccdddd/eeee/ffff", 2, 16, 2},
		{"wide rune at the end of a row", []linkRow{
			mkRow("see /aaaa/bbbb/ccc文", false, true),
			mkRow("  件.txt more", false, false),
		}, 1, 3, any, "/aaaa/bbbb/ccc文件.txt", 2, 8, -1},
		{"a word away from the edges costs one candidate", []linkRow{
			mkRow("aaaa aaaa aaaa aaaa", true, true),
			mkRow("bb src/a.go bb bb bb", true, true),
			mkRow("cccc cccc cccc cccc", true, true),
		}, 1, 5, any, "src/a.go", 3, 11, 1},
		{"prose in a soft-wrapped block", []linkRow{
			mkRow("aaaa aaaa aaaa aaaa", true, true),
			mkRow("bbbb bbbb bbbb bbbb", true, true),
			mkRow("cccc cccc cccc cccc", true, true),
		}, 1, 6, any, "", 0, 0, 0},
		{"prose before the path", []linkRow{
			mkRow(`some words C:\a\b`, false, true),
			mkRow(`  c\d.png`, false, false),
		}, 0, 2, any, "", 0, 0, -1},
		{"prose after the path", []linkRow{
			mkRow(`some words C:\a\b`, false, true),
			mkRow(`  c\d.png and words`, false, false),
		}, 1, 15, any, "", 0, 0, -1},
	} {
		t.Run(c.name, func(t *testing.T) {
			asked := 0
			isLink := func(p string) bool { asked++; return c.isLink(p) }
			p, x0, x1 := pathAtRows(c.rows, c.at, c.col, isLink)
			if p != c.path || x0 != c.x0 || x1 != c.x1 {
				t.Errorf("pathAtRows = %q %d..%d, want %q %d..%d", p, x0, x1, c.path, c.x0, c.x1)
			}
			if c.asked >= 0 && asked != c.asked {
				t.Errorf("isLink asked %d times, want %d", asked, c.asked)
			}
		})
	}
}

// TestPathAtRowsSingle: a row alone gives what pathAt gives.
func TestPathAtRowsSingle(t *testing.T) {
	for _, s := range []string{`write(notes.md)`, `"a b/c.go":12 x`, `见src/a.go，然后`, `C:\a\b.txt:3:4 ok`} {
		r := mkRow(s, false, false)
		for col := range r.cols {
			p, x0, x1 := pathAt(r.text, r.cols, r.cols[col])
			q, y0, y1 := pathAtRows([]linkRow{r}, 0, r.cols[col], func(string) bool { return true })
			if p != q || x0 != y0 || x1 != y1 {
				t.Errorf("%q col %d: pathAtRows = %q %d..%d, pathAt = %q %d..%d", s, col, q, y0, y1, p, x0, x1)
			}
		}
	}
}

// TestPathAtRowsRules: a rule made of one ASCII rune that fills a row is
// neither a path nor joined to the file name that starts the next row.
func TestPathAtRowsRules(t *testing.T) {
	for _, r := range "-=_.+#~" {
		rows := []linkRow{
			mkRow(strings.Repeat(string(r), 48), false, true),
			mkRow("foo.txt is here", false, false),
		}
		isLink := func(string) bool { return true }
		if p, x0, x1 := pathAtRows(rows, 1, 2, isLink); p != "foo.txt" || x0 != 0 || x1 != 7 {
			t.Errorf("%c: foo.txt = %q %d..%d, want foo.txt 0..7", r, p, x0, x1)
		}
		if p, _, _ := pathAtRows(rows, 0, 5, isLink); p != "" {
			t.Errorf("%c: the rule is the link %q", r, p)
		}
	}
}
