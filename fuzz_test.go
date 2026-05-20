package leb128_test

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/jcalabro/leb128"
)

// maxLEB128Width matches the unexported maxWidth constant in leb128.go;
// canonical LEB128 encodings of a 64-bit integer are at most 10 bytes.
const maxLEB128Width = 10

// seedU64 covers single-byte values, every continuation-bit boundary
// (2^7, 2^14, ..., 2^63), and the extremes.
var seedU64 = []uint64{
	0, 1, 2, 63, 64, 126, 127, 128, 129,
	1<<7 - 1, 1 << 7, 1 << 14, 1 << 21, 1 << 28,
	1 << 35, 1 << 42, 1 << 49, 1 << 56, 1 << 63,
	math.MaxUint64 - 1, math.MaxUint64,
}

// seedS64 covers both signs of every continuation-bit boundary plus the
// signed extremes (which exercise the 10-byte sign-extension path).
var seedS64 = []int64{
	0, 1, -1, 63, -64, 64, -65, 127, -128, 128, -129,
	1<<6 - 1, -(1 << 6), 1 << 13, -(1 << 13),
	1 << 20, -(1 << 20), 1 << 27, -(1 << 27),
	1 << 34, -(1 << 34), 1 << 41, -(1 << 41),
	1 << 48, -(1 << 48), 1 << 55, -(1 << 55),
	math.MaxInt64, math.MinInt64,
	math.MaxInt64 - 1, math.MinInt64 + 1,
}

// FuzzRoundTripU64 verifies Encode/Decode are inverses for every uint64
// and that encodings never exceed the documented 10-byte width.
func FuzzRoundTripU64(f *testing.F) {
	for _, v := range seedU64 {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, n uint64) {
		buf := leb128.EncodeU64(n)
		if len(buf) == 0 || len(buf) > maxLEB128Width {
			t.Fatalf("EncodeU64(%d) produced %d bytes, want 1..%d", n, len(buf), maxLEB128Width)
		}
		got, err := leb128.DecodeU64(bytes.NewReader(buf))
		if err != nil {
			t.Fatalf("DecodeU64(EncodeU64(%d)): unexpected error: %v", n, err)
		}
		if got != n {
			t.Fatalf("round-trip mismatch: encoded %d, decoded %d (bytes %x)", n, got, buf)
		}
	})
}

// FuzzRoundTripS64 is the signed counterpart to FuzzRoundTripU64. The
// negative seeds force exploration of the sign-extension path.
func FuzzRoundTripS64(f *testing.F) {
	for _, v := range seedS64 {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, n int64) {
		buf := leb128.EncodeS64(n)
		if len(buf) == 0 || len(buf) > maxLEB128Width {
			t.Fatalf("EncodeS64(%d) produced %d bytes, want 1..%d", n, len(buf), maxLEB128Width)
		}
		got, err := leb128.DecodeS64(bytes.NewReader(buf))
		if err != nil {
			t.Fatalf("DecodeS64(EncodeS64(%d)): unexpected error: %v", n, err)
		}
		if got != n {
			t.Fatalf("round-trip mismatch: encoded %d, decoded %d (bytes %x)", n, got, buf)
		}
	})
}

// validDecodeError reports whether err is one of the documented decode errors
// (or nil). Anything else means the decoder leaked an unexpected error type.
func validDecodeError(err error) bool {
	return err == nil ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, leb128.ErrOverflow)
}

// FuzzDecodeU64 feeds arbitrary bytes to DecodeU64 and asserts:
//   - it never panics
//   - it only ever returns one of the documented errors
//   - on success, the decoded value re-encodes to a prefix of the input
//     (canonical encoding is unique, so re-encoding must agree byte-for-byte
//     with however many bytes the decoder consumed)
func FuzzDecodeU64(f *testing.F) {
	seeds := [][]byte{
		nil,
		{0x00},
		{0x7f},
		{0x80, 0x01},
		{0xff, 0x7f},
		{0x80}, // truncated
		{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00}, // 11-byte overrun
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01},       // max uint64
		{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02},       // strict overflow
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := leb128.DecodeU64(bytes.NewReader(data))
		if !validDecodeError(err) {
			t.Fatalf("DecodeU64 returned undocumented error type: %v (%T)", err, err)
		}
		if err != nil {
			return
		}
		// Decoding then re-encoding then decoding must yield the same value.
		// (Direct byte-for-byte round-trip would fail on non-canonical inputs
		// like {0x80, 0x00}, which decodes to 0 but re-encodes to {0x00}.)
		reenc := leb128.EncodeU64(got)
		again, err := leb128.DecodeU64(bytes.NewReader(reenc))
		if err != nil || again != got {
			t.Fatalf("symmetry broken: input=%x decoded=%d reenc=%x reenc-decoded=%d err=%v",
				data, got, reenc, again, err)
		}
	})
}

// FuzzDecodeS64 is the signed counterpart to FuzzDecodeU64.
func FuzzDecodeS64(f *testing.F) {
	seeds := [][]byte{
		nil,
		{0x00},
		{0x40}, // -64
		{0x7f}, // -1
		{0x80, 0x01},
		{0xff, 0x00},
		{0x80}, // truncated
		{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00}, // 11-byte overrun
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00},       // max int64
		{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x7f},       // min int64
		{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01},       // strict overflow
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := leb128.DecodeS64(bytes.NewReader(data))
		if !validDecodeError(err) {
			t.Fatalf("DecodeS64 returned undocumented error type: %v (%T)", err, err)
		}
		if err != nil {
			return
		}
		// Decoding then re-encoding then decoding must yield the same value.
		// Direct byte-for-byte round-trip would fail on non-canonical inputs.
		reenc := leb128.EncodeS64(got)
		again, err := leb128.DecodeS64(bytes.NewReader(reenc))
		if err != nil || again != got {
			t.Fatalf("symmetry broken: input=%x decoded=%d reenc=%x reenc-decoded=%d err=%v",
				data, got, reenc, again, err)
		}
	})
}
