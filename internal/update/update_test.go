package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCompare(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.3", "1.2.10", -1},
		{"1.10.0", "1.9.9", 1},
		{"v2.0.0", "1.99.0", 1},
		{"1.2", "1.2.0", -1},
		{"1.0.0-beta.1", "1.0.0", -1},
		{"1.0.0-beta.2", "1.0.0-beta.10", -1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0+build.5", "1.0.0", 0},
	} {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
		if got := Compare(c.b, c.a); got != -c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.b, c.a, got, -c.want)
		}
	}
}

func TestSignatures(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	pk, err := ParsePublicKey(base64.StdEncoding.EncodeToString(pub))
	if err != nil {
		t.Fatal(err)
	}
	sk, err := ParsePrivateKey(base64.StdEncoding.EncodeToString(priv))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("archive"))
	sig := Sign(sk, sum[:])
	if err := Verify(pk, sum[:], sig); err != nil {
		t.Fatal(err)
	}
	other := sha256.Sum256([]byte("tampered"))
	if Verify(pk, other[:], sig) == nil {
		t.Error("a tampered archive verified")
	}
	otherPub, _, _ := ed25519.GenerateKey(nil)
	if Verify(otherPub, sum[:], sig) == nil {
		t.Error("another key verified the signature")
	}
	if _, err := ParsePublicKey("not a key"); err == nil {
		t.Error("ParsePublicKey accepted garbage")
	}
}

func TestArchive(t *testing.T) {
	src := t.TempDir()
	write := func(name, content string, mode os.FileMode) {
		p := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	write("App.app/Contents/MacOS/App", "binary", 0o755)
	write("App.app/Contents/Resources/data/a.txt", "a", 0o644)
	if runtime.GOOS != "windows" {
		if err := os.Symlink("data/a.txt", filepath.Join(src, "App.app/Contents/Resources/link")); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := WriteArchive(&buf, src, []string{"App.app"}); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := ExtractArchive(bytes.NewReader(buf.Bytes()), dst); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dst, "App.app/Contents/Resources/data/a.txt")); err != nil || string(b) != "a" {
		t.Errorf("a.txt = %q, %v", b, err)
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(filepath.Join(dst, "App.app/Contents/MacOS/App")); err != nil || info.Mode().Perm()&0o100 == 0 {
			t.Errorf("the executable lost its mode: %v", err)
		}
		if link, err := os.Readlink(filepath.Join(dst, "App.app/Contents/Resources/link")); err != nil || link != "data/a.txt" {
			t.Errorf("link = %q, %v", link, err)
		}
	}
}

// TestArchiveExecBits checks that programs keep exec bits that files made
// on Windows lack: archives get them, and archives made without them
// install with them.
func TestArchiveExecBits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no exec bits")
	}
	const elf, text = "\x7fELF program", "text"
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "app"), []byte(elf), 0o644)
	os.WriteFile(filepath.Join(src, "notes.txt"), []byte(text), 0o644)
	var buf bytes.Buffer
	if err := WriteArchive(&buf, src, []string{"app", "notes.txt"}); err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	modes := map[string]int64{}
	for tr := tar.NewReader(gz); ; {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		modes[hdr.Name] = hdr.Mode
	}
	if modes["app"] != 0o755 || modes["notes.txt"] != 0o644 {
		t.Errorf("archive modes = %v", modes)
	}

	// An archive made on Windows, with every file 0666.
	buf.Reset()
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, content := range map[string]string{"app": elf, "notes.txt": text} {
		tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: 0o666, Size: int64(len(content))})
		tw.Write([]byte(content))
	}
	tw.Close()
	gw.Close()
	dst := t.TempDir()
	if err := ExtractArchive(&buf, dst); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(dst, "app")); err != nil || info.Mode().Perm() != 0o755 {
		t.Errorf("app installed as %v, %v", info.Mode(), err)
	}
	if info, err := os.Stat(filepath.Join(dst, "notes.txt")); err != nil || info.Mode().Perm()&0o111 != 0 {
		t.Errorf("notes.txt installed as %v, %v", info.Mode(), err)
	}

	// Deltas get them too.
	var delta bytes.Buffer
	if err := WriteDelta(&delta, "1", "2", t.TempDir(), src, nil); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "app")
	if err := ApplyDelta(bytes.NewReader(delta.Bytes()), int64(delta.Len()), "1", "2", t.TempDir(), out); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(out, "app")); err != nil || info.Mode().Perm() != 0o755 {
		t.Errorf("the delta made app %v, %v", info.Mode(), err)
	}

	// A delta made on Windows.
	old, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	data := deflate([]byte(elf))
	sum := sha256.Sum256([]byte(elf))
	e := deltaEntry{Path: "app", Mode: 0o666, Size: int64(len(elf)), SHA256: hex.EncodeToString(sum[:]), Data: int64(len(data))}
	if err := applyFile(e, io.NewSectionReader(bytes.NewReader(data), 0, int64(len(data))), old, root); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(dir, "app")); err != nil || info.Mode().Perm() != fs.FileMode(0o755) {
		t.Errorf("the delta made app %v, %v", info.Mode(), err)
	}
}

func TestReleaseNotes(t *testing.T) {
	changelog := "# Changelog\n\nIntro.\n\n## 1.2.0 - 2026-09-19\n\nFaster.\n\n### Fixed\n- Drafts\n\n## 1.1.0\n- Tray\n"
	if got := ReleaseNotes(changelog, "1.2.0"); got != "Faster.\n\n### Fixed\n- Drafts" {
		t.Errorf("notes = %q", got)
	}
	if got := ReleaseNotes(changelog, "v1.1.0"); got != "- Tray" {
		t.Errorf("notes = %q", got)
	}
	if got := ReleaseNotes(changelog, "1.3.0"); got != "" {
		t.Errorf("notes = %q", got)
	}

	// Keep a Changelog, with a Windows checkout's line endings.
	keep := "# Changelog\r\n\r\n## [Unreleased]\r\n\r\n- Soon\r\n\r\n## [1.2.0] - 2026-09-19\r\n\r\n- Faster\r\n\r\n## [v1.1.0](https://example.com/compare/v1.0.0...v1.1.0)\r\n- Tray\r\n"
	for version, want := range map[string]string{"1.2.0": "- Faster", "1.1.0": "- Tray", "Unreleased": "- Soon", "1.0.0": ""} {
		if got := ReleaseNotes(keep, version); got != want {
			t.Errorf("notes of %s = %q, want %q", version, got, want)
		}
	}
}
