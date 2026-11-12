package terminal

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// urlPattern finds the URLs programs print as text, which a Command+click
// opens as it does hyperlinks. A URL sends what is not ASCII encoded, so it
// ends at the first such character: the CJK text or full-width punctuation
// that often follows one without a space.
var urlPattern = regexp.MustCompile(`(?:(?:https?|ftp|file)://|mailto:)[^\s"'<>` + "`" + `\x00-\x1f\x7f-\x{10ffff}]+`)

// proseDelimiters end a path printed in Chinese or Japanese prose, which
// puts no space around it; a quoted path may hold them.
const proseDelimiters = "，。；：！？、（）【】《》「」『』"

// pathToken finds what may be a file path programs print as text, unquoted:
// a run of anything but spaces, quotes and proseDelimiters.
var pathToken = regexp.MustCompile(`[^\s"'<>` + "`" + proseDelimiters + `]+`)

// pathInToken finds a path inside a token that is not one as a whole, as
// main.go:12 in grep's main.go:12:code, build/app.js in --out=build/app.js
// or /usr/bin in PATH=/usr/bin:/bin. A drive path, as C:\src\a.go, starts at
// a word boundary: abc:/foo holds no drive.
var pathInToken = regexp.MustCompile(`(?:\b[A-Za-z]:[\\/][^\s:;]+|(?:~|\.{1,2})?/[^\s:]+|[\p{L}\p{N}_.@+-]+(?:/[\p{L}\p{N}_.@+-]+)+/?)(?::\d+(?::\d+)?)?|[\p{L}\p{N}_@+-][\p{L}\p{N}_.@+-]*\.[\p{L}\p{N}_]+(?::\d+(?::\d+)?)?`)

// drivePath finds the drive a Windows path starts with, as C:\ or D:/. Only
// a path with one takes \ as a separator.
var drivePath = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

// quotedPlace finds the place after a quoted path, as "main.py":12.
var quotedPlace = regexp.MustCompile(`^:\d+(?::\d+)?`)

// pathPlace finds the place in the file a path ends with: :line or
// :line:col (compilers, grep), (line,col) (TypeScript, MSBuild), or #Lline
// or #LlineCcol (GitHub).
var pathPlace = regexp.MustCompile(`:(\d+)(?::(\d+))?:?$|\((\d+)(?:,(\d+))?\)$|#L(\d+)(?:C(\d+))?$`)

// pathAt returns the file path of text at column col, as urlAt, or "":
// absolute, from ~, ./ or ../, with a slash, or a file name with an
// extension or a line; the place in the file is written path:line:col
// however it was printed, and a quoted path loses its quotes.
func pathAt(text []rune, cols []int, col int) (path string, x0, x1 int) {
	s := string(text)
	span := func(a, b int) (int, int, bool) {
		start := utf8.RuneCountInString(s[:a])
		end := start + utf8.RuneCountInString(s[a:b])
		if end > start && end <= len(cols) && cols[start] <= col && col <= cols[end-1] {
			return cols[start], cols[end-1] + 1, true
		}
		return 0, 0, false
	}
	// A quoted path first, as it may hold spaces and what the tokens below
	// end at.
	for i := 0; i < len(s); i++ {
		a, b := quotedAt(s, i)
		if b < 0 {
			continue
		}
		p, pa, pb := pathOf(s[a:b])
		if p == "" {
			continue
		}
		if x0, x1, ok := span(a+pa, a+pb); ok {
			return p, x0, x1
		}
		i = b - 1
	}
	for _, m := range pathToken.FindAllStringIndex(s, -1) {
		tok := s[m[0]:m[1]]
		if p, a, b := pathOf(tok); p != "" {
			if x0, x1, ok := span(m[0]+a, m[0]+b); ok {
				return p, x0, x1
			}
			continue
		}
		for _, n := range pathInToken.FindAllStringIndex(tok, -1) {
			if p, a, b := pathOf(tok[n[0]:n[1]]); p != "" {
				if x0, x1, ok := span(m[0]+n[0]+a, m[0]+n[0]+b); ok {
					return p, x0, x1
				}
			}
		}
	}
	return "", 0, 0
}

// maxLinkRows is how many rows above and below the hovered one a path may
// wrap over.
const maxLinkRows = 8

// linkRow is a row of the viewport as pathAtRows reads it: its text and the
// column of each rune, whether it soft-wraps into the next row, whether its
// text reaches the last column, and the width of the terminal.
type linkRow struct {
	text       []rune
	cols       []int
	wrap, full bool
	width      int
}

// goesOn reports whether the row's text may go on in the next: a terminal
// wraps it, or a program wrapped it at the right edge.
func (r linkRow) goesOn() bool { return r.wrap || r.full }

// pathAtRows is pathAt over consecutive rows of the viewport, for a path
// that wraps; at is the hovered row in rows and col its column. It returns
// the columns the path covers on that row only. Only the word under the
// pointer decides which rows are tried (wordRows), so prose costs one pathAt.
// The longest run of rows is tried first, then shorter ones down to the
// hovered row alone, until isLink takes the path: a program's wrap is a
// guess, and text that ends at the last column may be followed by an
// unrelated row. A terminal's soft wrap is certain, so a run never cuts one.
func pathAtRows(rows []linkRow, at, col int, isLink func(string) bool) (path string, x0, x1 int) {
	lo, hi := wordRows(rows, at, col)
	tried := map[string]bool{}
	for n := hi - lo; n >= 0; n-- {
		for a := lo; a <= at && a+n <= hi; a++ {
			b := a + n
			if b < at || a > lo && rows[a-1].wrap || b < hi && rows[b].wrap {
				continue
			}
			var p string
			if n == 0 {
				p, x0, x1 = pathAt(rows[at].text, rows[at].cols, col)
			} else {
				p, x0, x1 = pathOverRows(rows[a:a+n+1], at-a, col)
			}
			if p != "" && !tried[p] {
				if isLink(p) {
					return p, x0, x1
				}
				tried[p] = true
			}
		}
	}
	return "", 0, 0
}

// runeAt returns the index of the rune of r at column col, or -1; a wide
// rune also holds the column after it.
func runeAt(r linkRow, col int) int {
	for j, c := range r.cols {
		if c <= col && (j+1 == len(r.cols) || r.cols[j+1] > col) && col-c <= 1 {
			return j
		}
	}
	return -1
}

// startOf returns the index of the first rune of rows[i] that is text: the
// row's first, or after its indent when a program wrapped the row above.
func startOf(rows []linkRow, i int) int {
	if i == 0 || rows[i-1].wrap {
		return 0
	}
	n := 0
	for n < len(rows[i].text) && rows[i].text[n] == ' ' {
		n++
	}
	return n
}

// oneWord reports whether the text of rows[i] is a single word.
func oneWord(rows []linkRow, i int) bool {
	t := rows[i].text
	st := startOf(rows, i)
	return st < len(t) && !slices.Contains(t[st:], ' ')
}

// wordRows returns the first and last row the word at col of row at may
// continue over: up while it starts the text of a row the row above goes on
// into, down while it ends a row that goes on, over rows that are one word.
// Anywhere else, the hovered row alone.
func wordRows(rows []linkRow, at, col int) (lo, hi int) {
	lo, hi = at, at
	r := rows[at]
	j := runeAt(r, col)
	if j < 0 || r.text[j] == ' ' {
		return
	}
	s, e := j, j
	for s > 0 && r.text[s-1] != ' ' {
		s--
	}
	for e+1 < len(r.text) && r.text[e+1] != ' ' {
		e++
	}
	if s == startOf(rows, at) {
		for lo > 0 && at-lo < maxLinkRows && rows[lo-1].goesOn() {
			if lo--; !oneWord(rows, lo) {
				break
			}
		}
	}
	if e == len(r.text)-1 {
		for hi < len(rows)-1 && hi-at < maxLinkRows && rows[hi].goesOn() {
			if hi++; !oneWord(rows, hi) {
				break
			}
		}
	}
	return
}

// wrapped reports whether prev, a row that reaches the last column, goes
// on in a row whose text after its indent is next: a program breaks only a
// word too long for a row, so a rule or prose that ends at the edge is no
// wrap. The word is the run of non-spaces on both sides of the break; a
// run of one rune, as ---, is a rule.
func wrapped(prev linkRow, next []rune, cols []int) bool {
	isText := func(r rune) bool {
		return r > ' ' && r < 0x80 || unicode.IsLetter(r) || unicode.IsDigit(r)
	}
	n := len(prev.text)
	if n == 0 || len(next) == 0 || !isText(prev.text[n-1]) || !isText(next[0]) {
		return false
	}
	t := n - 1
	for t > 0 && prev.text[t-1] != ' ' {
		t--
	}
	h := 0
	for h+1 < len(next) && next[h+1] != ' ' {
		h++
	}
	if rule(prev.text[t:]) || rule(next[:h+1]) {
		return false
	}
	w1, indent := prev.width-prev.cols[t], cols[0]
	w2 := cols[h] + 1 - indent
	return w1+w2 > prev.width-indent
}

// rule reports whether word is two or more of one rune.
func rule(word []rune) bool {
	return len(word) > 1 && !slices.ContainsFunc(word, func(r rune) bool { return r != word[0] })
}

// pathOverRows runs pathAt on the rows joined into one line, and returns
// the part of the path on row at.
func pathOverRows(rows []linkRow, at, col int) (path string, x0, x1 int) {
	var line []rune
	var row, colOf []int // the row and the column of each rune of line
	hovered := -1
	for i, r := range rows {
		text, cols := r.text, r.cols
		skip := 0
		if i > 0 && !rows[i-1].wrap {
			skip = startOf(rows, i)
			text, cols = text[skip:], cols[skip:]
			if !wrapped(rows[i-1], text, cols) {
				return "", 0, 0
			}
		}
		if i == at {
			if hovered = runeAt(r, col); hovered < skip {
				return "", 0, 0 // in the indent, or no rune
			}
			hovered = len(line) + hovered - skip
		}
		for j, c := range text {
			line, row, colOf = append(line, c), append(row, i), append(colOf, cols[j])
		}
	}
	ident := make([]int, len(line))
	for i := range ident {
		ident[i] = i
	}
	path, a, b := pathAt(line, ident, hovered)
	if path == "" {
		return "", 0, 0
	}
	x0, x1 = -1, 0
	for i := a; i < b; i++ {
		if row[i] == at {
			if x0 < 0 {
				x0 = colOf[i]
			}
			x1 = colOf[i] + 1
		}
	}
	if x0 < 0 {
		return "", 0, 0
	}
	return path, x0, x1
}

// quotedAt returns the bytes of s, a up to b, of a quoted run that opens at
// byte i and the place after it, or b < 0. A quote between letters or digits,
// as in don't, is an apostrophe and opens or closes nothing, and the inside
// of a quoted path neither starts nor ends with a space.
func quotedAt(s string, i int) (a, b int) {
	q := s[i]
	if q != '"' && q != '\'' || i > 0 && isAlnum(s[i-1]) {
		return 0, -1
	}
	j := strings.IndexByte(s[i+1:], q)
	if j < 0 {
		return 0, -1
	}
	j += i + 1
	inner := s[i+1 : j]
	if inner == "" || strings.ContainsAny(inner, "\r\n") || strings.TrimSpace(inner) != inner {
		return 0, -1
	}
	b = j + 1
	b += len(quotedPlace.FindString(s[b:]))
	if b < len(s) && isAlnum(s[b]) {
		return 0, -1
	}
	return i, b
}

// isAlnum reports whether c is an ASCII letter or digit.
func isAlnum(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}

// pathOf returns the path a token of pathToken, or a quoted run, prints,
// and the bytes of the token it shows in, a up to b, or "".
func pathOf(tok string) (path string, a, b int) {
	a, b = 0, len(tok)
	quoted := tok[0] == '"' || tok[0] == '\''
	if !quoted {
		// Brackets and emphasis around a path are not part of it, nor is
		// the call a path is the argument of, as write(notes.md); a bracket
		// the path opens is, as icon_(copy).png.
		a = len(tok) - len(strings.TrimLeft(tok, "([{*"))
		// The name of an option or a variable the path is the value of,
		// as --out=build/app.js or PATH=/usr/bin, is not part of it.
		if i := strings.IndexByte(tok[a:], '='); i >= 0 && !strings.Contains(tok[a:a+i], "/") {
			a += i + 1
		}
		b = a + len(balanced(tok[a:]))
		if strings.HasSuffix(tok[a:b], ")") && !pathPlace.MatchString(tok[a:b]) {
			if i := opening(tok[a:b]); i > 0 {
				a, b = a+i+1, b-1
			}
		}
		for b > a && strings.IndexByte(",;.!?]", tok[b-1]) >= 0 {
			b--
		}
		if !pathPlace.MatchString(tok[a:b]) {
			b = a + len(trimURL(tok[a:b]))
		}
		if b <= a {
			return "", 0, 0
		}
	}
	name, line, col := tok[a:b], 0, 0
	if loc := pathPlace.FindStringSubmatchIndex(name); loc != nil {
		place := name
		name = place[:loc[0]]
		for i := 2; i < len(loc); i += 4 {
			if loc[i] >= 0 {
				line, _ = strconv.Atoi(place[loc[i]:loc[i+1]])
				if loc[i+2] >= 0 {
					col, _ = strconv.Atoi(place[loc[i+2]:loc[i+3]])
				}
				break
			}
		}
		if strings.HasSuffix(place, ":") {
			b-- // the colon after main.go:12:3: ends a compiler's location
		}
	}
	if quoted {
		name = name[1 : len(name)-1]
	}
	// ~/ ./ and ../ alone are directories, as docs/ is. They hold no letter.
	dirOnly := name == "~/" || name == "./" || name == "../"
	drive := drivePath.MatchString(name)
	rest := name
	if drive {
		rest = name[2:] // the colon after the drive letter is part of it
	}
	if !dirOnly && (name == "" || strings.ContainsAny(rest, ":\x00\r\n") ||
		strings.IndexFunc(name, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) < 0) {
		return "", 0, 0
	}
	// Words are not paths: a name without a slash takes a letter, and an
	// extension or a line, so that version 1.2.3 and 12:30 are neither.
	if !drive && !strings.Contains(name, "/") {
		dot := strings.LastIndexByte(name, '.')
		if strings.IndexFunc(name, unicode.IsLetter) < 0 || line == 0 && (dot < 0 || dot == len(name)-1) {
			return "", 0, 0
		}
	}
	path = name
	if line > 0 {
		path += ":" + strconv.Itoa(line)
		if col > 0 {
			path += ":" + strconv.Itoa(col)
		}
	}
	return path, a, b
}

// balanced returns s up to a closing bracket it did not open, as the ) of
// "(image.png)已保存".
func balanced(s string) string {
	var depth [3]int
	for i, r := range s {
		if k := strings.IndexRune("([{", r); k >= 0 {
			depth[k]++
		} else if k := strings.IndexRune(")]}", r); k >= 0 {
			if depth[k] == 0 {
				return s[:i]
			}
			depth[k]--
		}
	}
	return s
}

// opening returns the index of the ( that opens the ) s ends with, or -1.
func opening(s string) int {
	depth := 0
	for i := len(s) - 1; i >= 0; i-- {
		switch s[i] {
		case ')':
			depth++
		case '(':
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return -1
}

// urlAt returns the URL of text, a row whose runes are in columns cols, at
// column col, and the columns it spans, x0 up to x1, or "".
func urlAt(text []rune, cols []int, col int) (url string, x0, x1 int) {
	return matchAt(urlPattern, text, cols, col)
}

func matchAt(re *regexp.Regexp, text []rune, cols []int, col int) (string, int, int) {
	s := string(text)
	for _, m := range re.FindAllStringIndex(s, -1) {
		u := trimURL(s[m[0]:m[1]])
		if u == "" {
			continue
		}
		start := len([]rune(s[:m[0]]))
		end := start + len([]rune(u))
		if start < len(cols) && cols[start] <= col && col <= cols[end-1] {
			return u, cols[start], cols[end-1] + 1
		}
	}
	return "", 0, 0
}

// trimURL trims what likely ends the sentence around a URL or a path rather
// than it: punctuation, as the 。 or … of prose, and closing brackets that it
// did not open.
func trimURL(u string) string {
	for len(u) > 0 {
		if r, n := utf8.DecodeLastRuneInString(u); r >= utf8.RuneSelf && unicode.IsPunct(r) {
			u = u[:len(u)-n]
			continue
		}
		switch last := u[len(u)-1]; last {
		case '.', ',', ';', ':', '!', '?':
			u = u[:len(u)-1]
			continue
		case ')', ']', '}':
			open := map[byte]string{')': "(", ']': "[", '}': "{"}[last]
			if strings.Count(u, open) < strings.Count(u, string(last)) {
				u = u[:len(u)-1]
				continue
			}
		}
		return u
	}
	return u
}

// isDir reports whether a path printed as text names a directory, by the
// slash at its end, as docs/, ~/ or docs/:12: one a word before a slash in
// prose (w/, I/, yes/no/) prints too. pathAt writes every place as
// path:line:col, so the line is stripped before the slash is read.
func isDir(path string) bool {
	path = lineSuffix.ReplaceAllString(path, "")
	return strings.HasSuffix(path, "/") || drivePath.MatchString(path) && strings.HasSuffix(path, `\`)
}

// lineSuffix is the :line or :line:col at the end of a path.
var lineSuffix = regexp.MustCompile(`:\d+(?::\d+)?$`)

// isLink reports whether a file path printed as text is a link, as
// Options.LinkPath decides; t.mu is held. Without LinkPath a directory is
// not a link.
func (t *Terminal) isLink(path string) bool {
	if t.opts.LinkPath != nil {
		return t.opts.LinkPath(path)
	}
	return !isDir(path)
}
