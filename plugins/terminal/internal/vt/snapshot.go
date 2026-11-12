package vt

import (
	"errors"
	"runtime"
	"unsafe"
)

// Snapshot returns the terminal's complete state in libghostty-vt's binary
// snapshot format: the "GHOSTSNP" magic, the format's version and the
// records through FINISH, the scrollback included. Decode it with
// DecodeSnapshot, into a terminal of this library's version: the format is
// not yet compatible across versions.
func (t *Terminal) Snapshot() ([]byte, error) {
	var n uintptr
	switch r := Result(int32(call(fnSnapshotEncodeBuf, t.h, 0, 0, uintptr(unsafe.Pointer(&n))))); r {
	case success:
		return nil, nil
	case OutOfSpace:
	default:
		return nil, r
	}
	b := make([]byte, n)
	if err := result(call(fnSnapshotEncodeBuf, t.h, uintptr(unsafe.Pointer(&b[0])), n, uintptr(unsafe.Pointer(&n)))); err != nil {
		return nil, err
	}
	return b[:n], nil
}

// Snapshot decoder options (GhosttySnapshotDecoderOption).
const (
	snapshotDecoderMaxContinuation = 1
	snapshotDecoderRetain          = 2
)

// DecodeSnapshot makes a terminal of the state a snapshot holds, with the
// effects given: a terminal of the size the snapshot was taken at, its
// screen, scrollback, cursor and modes. The snapshot must be one this
// library's version wrote (Terminal.Snapshot).
func DecodeSnapshot(b []byte, effects Effects) (*Terminal, error) {
	if !Loaded() {
		return nil, errors.New("libghostty-vt is not loaded")
	}
	if len(b) == 0 {
		return nil, errors.New("libghostty-vt: no snapshot")
	}
	var d, h uintptr
	if err := result(call(fnSnapshotDecoderNewBuf, 0, uintptr(unsafe.Pointer(&d)), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))); err != nil {
		return nil, err
	}
	defer call(fnSnapshotDecoderFree, d)
	// The restored terminal keeps tracking the unfinished input the
	// snapshot ended in, so that it can be snapshotted again at once.
	max := uintptr(snapshotContinuationMax)
	if err := result(call(fnSnapshotDecoderSet, d, snapshotDecoderMaxContinuation, uintptr(unsafe.Pointer(&max)))); err != nil {
		return nil, err
	}
	retain := true
	if err := result(call(fnSnapshotDecoderSet, d, snapshotDecoderRetain, uintptr(unsafe.Pointer(&retain)))); err != nil {
		return nil, err
	}
	if err := result(call(fnSnapshotDecoderDecode, d, uintptr(unsafe.Pointer(&h)))); err != nil {
		return nil, err
	}
	runtime.KeepAlive(b)
	t := &Terminal{h: h, effects: effects}
	t.init()
	return t, nil
}
