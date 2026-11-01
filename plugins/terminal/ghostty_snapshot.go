package terminal

import (
	"errors"

	"github.com/egoist/mygo/plugins/terminal/internal/vt"
)

// GhosttySnapshot returns the terminal's complete state in libghostty-vt's
// binary snapshot format: the "GHOSTSNP" magic, the format's version and
// the records through FINISH, the scrollback included: its screen and
// scrollback with their styles, its cursor and its modes. It is smaller
// and faster than Snapshot, whose escape sequences replay the output, and
// it is what a server keeping a session stores. Restore it with
// RestoreGhosttySnapshot; the format carries no compatibility guarantee
// across versions of libghostty-vt, so a terminal of this package's
// version must read it.
func (t *Terminal) GhosttySnapshot() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.term == nil {
		return nil, errors.New("terminal: closed")
	}
	return t.term.Snapshot()
}

// RestoreGhosttySnapshot replaces the terminal's screen, scrollback, cursor
// and modes with those GhosttySnapshot took, as they were at the size the
// snapshot was taken at: the terminal has that size thereafter, which
// Resize changes, telling the program. It is for a terminal no view shows,
// as a server's that resumes a session.
func (t *Terminal) RestoreGhosttySnapshot(b []byte) error {
	term, err := vt.DecodeSnapshot(b, t.effects())
	if err != nil {
		return err
	}
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		term.Free()
		return errors.New("terminal: closed")
	}
	t.rmu.Lock()
	cols, rows := term.Size()
	// A synchronized update held by the screen restored ends with it, and
	// a search's matches, which are cells of it, do too.
	t.held = false
	t.endSearch()
	t.term.Free()
	t.term = term
	t.size.cols, t.size.rows = cols, rows
	term.Resize(cols, rows, t.size.cellW, t.size.cellH)
	t.configure()
	t.rmu.Unlock()
	t.mu.Unlock()
	t.redraw()
	return nil
}
