package terminal

import "testing"

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
		"docs": false, "a/b": false, "main.go:12": false, "a.go:1:2": false,
	} {
		if got := isDir(path); got != want {
			t.Errorf("isDir(%q) = %v, want %v", path, got, want)
		}
	}
}
