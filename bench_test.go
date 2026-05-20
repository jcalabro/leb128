package leb128_test

import (
	"bytes"
	"io"
	"math"
	"testing"

	"github.com/jcalabro/leb128"
)

// Inputs span the realistic distribution: most LEB128 values in DWARF and
// WebAssembly fit in 1-3 bytes, but worst-case full-width encodings exercise
// the slow path. Naming is "<scheme><value-name>" to keep output readable.
var (
	uSmall  = uint64(0x3f)               // 1 byte
	uMedium = uint64(0x12345)            // 3 bytes
	uLarge  = uint64(math.MaxUint64)     // 10 bytes
	sSmall  = int64(-0x3f)               // 1 byte (sign-extension path)
	sMedium = int64(-0x12345)            // 3 bytes
	sLarge  = int64(math.MinInt64)       // 10 bytes (worst-case)
)

// Sinks prevent the compiler from dead-storing benchmark outputs.
var (
	sinkBytes []byte
	sinkU64   uint64
	sinkS64   int64
)

func BenchmarkEncodeU64Small(b *testing.B)  { benchEncodeU64(b, uSmall) }
func BenchmarkEncodeU64Medium(b *testing.B) { benchEncodeU64(b, uMedium) }
func BenchmarkEncodeU64Large(b *testing.B)  { benchEncodeU64(b, uLarge) }

func BenchmarkEncodeS64Small(b *testing.B)  { benchEncodeS64(b, sSmall) }
func BenchmarkEncodeS64Medium(b *testing.B) { benchEncodeS64(b, sMedium) }
func BenchmarkEncodeS64Large(b *testing.B)  { benchEncodeS64(b, sLarge) }

func BenchmarkDecodeU64Small(b *testing.B)  { benchDecodeU64(b, uSmall) }
func BenchmarkDecodeU64Medium(b *testing.B) { benchDecodeU64(b, uMedium) }
func BenchmarkDecodeU64Large(b *testing.B)  { benchDecodeU64(b, uLarge) }

func BenchmarkDecodeS64Small(b *testing.B)  { benchDecodeS64(b, sSmall) }
func BenchmarkDecodeS64Medium(b *testing.B) { benchDecodeS64(b, sMedium) }
func BenchmarkDecodeS64Large(b *testing.B)  { benchDecodeS64(b, sLarge) }

func benchEncodeU64(b *testing.B, v uint64) {
	expected := leb128.EncodeU64(v)
	b.SetBytes(int64(len(expected)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkBytes = leb128.EncodeU64(v)
	}
}

func benchEncodeS64(b *testing.B, v int64) {
	expected := leb128.EncodeS64(v)
	b.SetBytes(int64(len(expected)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkBytes = leb128.EncodeS64(v)
	}
}

// nonByteReader wraps an io.Reader to hide any io.ByteReader implementation,
// forcing decoders onto the slow path. Used to benchmark the io.Reader case
// separately from the io.ByteReader fast path.
type nonByteReader struct{ r io.Reader }

func (n *nonByteReader) Read(p []byte) (int, error) { return n.r.Read(p) }

// benchDecodeU64 reuses a single bytes.Reader across iterations via Reset to
// keep the measurement focused on DecodeU64 itself, not reader allocation.
// bytes.Reader implements io.ByteReader, so this exercises the fast path.
func benchDecodeU64(b *testing.B, v uint64) {
	encoded := leb128.EncodeU64(v)
	r := bytes.NewReader(encoded)
	b.SetBytes(int64(len(encoded)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reset(encoded)
		got, err := leb128.DecodeU64(r)
		if err != nil {
			b.Fatal(err)
		}
		sinkU64 = got
	}
}

func benchDecodeS64(b *testing.B, v int64) {
	encoded := leb128.EncodeS64(v)
	r := bytes.NewReader(encoded)
	b.SetBytes(int64(len(encoded)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reset(encoded)
		got, err := leb128.DecodeS64(r)
		if err != nil {
			b.Fatal(err)
		}
		sinkS64 = got
	}
}

// Slow-path variants: same inputs, but the reader does not implement
// io.ByteReader, so DecodeU64/DecodeS64 fall back to io.ReadFull.
func BenchmarkDecodeU64SlowSmall(b *testing.B)  { benchDecodeU64Slow(b, uSmall) }
func BenchmarkDecodeU64SlowMedium(b *testing.B) { benchDecodeU64Slow(b, uMedium) }
func BenchmarkDecodeU64SlowLarge(b *testing.B)  { benchDecodeU64Slow(b, uLarge) }

func BenchmarkDecodeS64SlowSmall(b *testing.B)  { benchDecodeS64Slow(b, sSmall) }
func BenchmarkDecodeS64SlowMedium(b *testing.B) { benchDecodeS64Slow(b, sMedium) }
func BenchmarkDecodeS64SlowLarge(b *testing.B)  { benchDecodeS64Slow(b, sLarge) }

func benchDecodeU64Slow(b *testing.B, v uint64) {
	encoded := leb128.EncodeU64(v)
	inner := bytes.NewReader(encoded)
	r := &nonByteReader{r: inner}
	b.SetBytes(int64(len(encoded)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inner.Reset(encoded)
		got, err := leb128.DecodeU64(r)
		if err != nil {
			b.Fatal(err)
		}
		sinkU64 = got
	}
}

func benchDecodeS64Slow(b *testing.B, v int64) {
	encoded := leb128.EncodeS64(v)
	inner := bytes.NewReader(encoded)
	r := &nonByteReader{r: inner}
	b.SetBytes(int64(len(encoded)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inner.Reset(encoded)
		got, err := leb128.DecodeS64(r)
		if err != nil {
			b.Fatal(err)
		}
		sinkS64 = got
	}
}
