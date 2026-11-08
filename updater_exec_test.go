//go:build !js

package mygo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMakeExecutable checks that the app's executable, which archives built
// on Windows gave no exec bits, gets those of its read bits.
func TestMakeExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no exec bits")
	}
	dir := t.TempDir()
	for _, c := range []struct{ perm, want os.FileMode }{{0o644, 0o755}, {0o600, 0o700}, {0o750, 0o750}} {
		exe := filepath.Join(dir, "app")
		if err := os.WriteFile(exe, []byte("app"), c.perm); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(exe, c.perm); err != nil {
			t.Fatal(err)
		}
		// Through a link, as a command in ~/.local/bin runs it.
		link := filepath.Join(dir, "link")
		if err := os.Symlink(exe, link); err != nil {
			t.Fatal(err)
		}
		if err := makeExecutable(link); err != nil {
			t.Fatal(err)
		}
		if info, err := os.Stat(exe); err != nil || info.Mode().Perm() != c.want {
			t.Errorf("%o became %v, %v, want %o", c.perm, info.Mode(), err, c.want)
		}
		os.Remove(exe)
		os.Remove(link)
	}
}
