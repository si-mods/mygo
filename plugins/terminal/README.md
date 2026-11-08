# terminal

A terminal for [MyGo](https://github.com/egoist/mygo) apps of native UI: a
view that runs the user's shell, or any program, in a pseudo-terminal, with
[Ghostty](https://ghostty.org)'s terminal emulator, libghostty-vt, loaded
at run time with no cgo.

```go
import "github.com/egoist/mygo/plugins/terminal"

term, err := terminal.New(terminal.Options{})
// ...
mygo.NewWindow(mygo.WindowOptions{Title: "Terminal", Content: ui.View(func(c *ui.Context) {
	terminal.View(c, term).Fill().AutoFocus()
})})
```

See [the documentation](../../docs/plugins/terminal.md), and
`examples/terminal` for an app.

## Headless terminals

A terminal no view shows keeps a session for a server: `Feed` shows what
the program writes, `Send` gives it what a client typed, and `Resize` sets
the size. `GhosttySnapshot` takes the state to store — libghostty-vt's
binary snapshot, the "GHOSTSNP" format: the screen, the scrollback, the
cursor and the modes — and `RestoreGhosttySnapshot` puts it back, into a
terminal of the size it was taken at. `Snapshot` is the other form: escape
sequences that replay the output into a terminal of the same size.
`Images` follows either with the images it shows now (Kitty graphics), as
escape sequences, which a terminal that reads no snapshot of them needs.

The events a client sends are encoded for the program with the terminal's
modes, so a client need only forward what its platform gives it:

```go
snap, err := term.GhosttySnapshot()
if err != nil { ... }
if err := term.RestoreGhosttySnapshot(snap); err != nil { ... }

term.Send(term.EncodeKey(terminal.KeyEvent{
	Key:       terminal.ParseKey("KeyC"), // the W3C KeyboardEvent.code
	Action:    terminal.KeyPress,
	Mods:      terminal.ModCtrl,
	Unshifted: 'c',
}))
term.Send(term.EncodeMouse(terminal.MouseEvent{
	Action: terminal.MousePress, Button: terminal.MouseLeft, Col: 2, Row: 3,
}))
term.Send(term.EncodeFocus(true))
term.Send(term.EncodePaste("foo\r\nbar"))
```

`EncodeKey` follows the keyboard protocol the program asked for, legacy
sequences, xterm's modifyOtherKeys or the Kitty keyboard protocol;
`EncodeMouse` the mouse tracking it asked for; `EncodePaste` bracketed
pastes; `EncodeFocus` reports the focus only when the program asked for
them (mode 1004). `ParseKey` reads W3C's `KeyboardEvent.code` names
("KeyA", "ArrowUp", "Numpad1", "F12"), and an unknown one is
`KeyUnidentified`.

`go generate` here builds libghostty-vt for every platform with Zig, from
the commit of Ghostty in `internal/libbuild`, into `build/`, and writes
`mygo-plugin.json`, which names the files and their SHA-256 for the CLI and
this package: publish them as the assets of the release that file points
at. It writes `ghostty_themes.go` too, the themes that Ghostty ships, which
`go run ./internal/libbuild -themes` writes alone, without Zig.
