// Package leb128 provides functionality to encode/decode signed and unsigned LEB128 data to and from 8 byte primitives.
//
// It deals only with 8 byte primitives; encodings whose value would not fit in a 64-bit integer return ErrOverflow.
//
// This package operates on a basic io.Reader rather than an io.ByteReader as the standard library does (i.e. the various Varint functions in https://pkg.go.dev/encoding/binary).
//
// The decoders return io.EOF when called on an empty stream, allowing callers reading a sequence of LEB128 values to detect end-of-input. A continuation bit set on the final byte before EOF returns io.ErrUnexpectedEOF.
//
// See https://en.wikipedia.org/wiki/LEB128 for more details.
package leb128

import (
	"errors"
	"io"
)

// maxWidth is the maximum number of bytes a canonical LEB128 encoding of a
// 64-bit integer occupies: ceil(64/7) = 10.
const maxWidth = 10

var ErrOverflow = errors.New("leb128: integer overflow (encoding exceeds 64 bits)")

// DecodeU64 reads a uleb128-encoded uint64 from r.
//
// It returns io.EOF if r is empty, io.ErrUnexpectedEOF if the encoding ends
// mid-integer (continuation bit set on the final byte read), and ErrOverflow
// if the encoding does not fit in 64 bits.
//
// If r implements io.ByteReader, DecodeU64 takes a fast path that avoids
// allocating per call and dispatches a single interface call per byte.
// *bytes.Buffer, *bytes.Reader, *bufio.Reader, and *strings.Reader all
// satisfy this interface.
func DecodeU64(r io.Reader) (uint64, error) {
	if br, ok := r.(io.ByteReader); ok {
		return decodeU64ByteReader(br)
	}
	return decodeU64Reader(r)
}

func decodeU64ByteReader(r io.ByteReader) (uint64, error) {
	var res uint64
	for i := range maxWidth {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF && i == 0 {
				return 0, io.EOF
			}
			if err == io.EOF {
				return 0, io.ErrUnexpectedEOF
			}
			return 0, err
		}
		if i == maxWidth-1 && b > 1 {
			return 0, ErrOverflow
		}
		res |= uint64(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return res, nil
		}
	}
	panic("unreachable: byte-10 overflow check guarantees loop exits")
}

func decodeU64Reader(r io.Reader) (uint64, error) {
	var (
		res uint64
		buf [1]byte
	)
	for i := range maxWidth {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			if err == io.EOF && i == 0 {
				return 0, io.EOF
			}
			if err == io.EOF {
				return 0, io.ErrUnexpectedEOF
			}
			return 0, err
		}
		b := buf[0]
		if i == maxWidth-1 && b > 1 {
			return 0, ErrOverflow
		}
		res |= uint64(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return res, nil
		}
	}
	panic("unreachable: byte-10 overflow check guarantees loop exits")
}

// DecodeS64 reads a sleb128-encoded int64 from r.
//
// It returns io.EOF if r is empty, io.ErrUnexpectedEOF if the encoding ends
// mid-integer (continuation bit set on the final byte read), and ErrOverflow
// if the encoding does not fit in 64 bits.
//
// If r implements io.ByteReader, DecodeS64 takes a fast path that avoids
// allocating per call and dispatches a single interface call per byte.
func DecodeS64(r io.Reader) (int64, error) {
	if br, ok := r.(io.ByteReader); ok {
		return decodeS64ByteReader(br)
	}
	return decodeS64Reader(r)
}

func decodeS64ByteReader(r io.ByteReader) (int64, error) {
	var (
		res   int64
		shift uint
	)
	for i := range maxWidth {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF && i == 0 {
				return 0, io.EOF
			}
			if err == io.EOF {
				return 0, io.ErrUnexpectedEOF
			}
			return 0, err
		}
		if i == maxWidth-1 && b != 0x00 && b != 0x7f {
			return 0, ErrOverflow
		}
		res |= int64(b&0x7f) << shift
		shift += 7
		if b&0x80 == 0 {
			if shift < 64 && b&0x40 != 0 {
				res |= ^int64(0) << shift
			}
			return res, nil
		}
	}
	panic("unreachable: byte-10 overflow check guarantees loop exits")
}

func decodeS64Reader(r io.Reader) (int64, error) {
	var (
		res   int64
		buf   [1]byte
		shift uint
	)
	for i := range maxWidth {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			if err == io.EOF && i == 0 {
				return 0, io.EOF
			}
			if err == io.EOF {
				return 0, io.ErrUnexpectedEOF
			}
			return 0, err
		}
		b := buf[0]
		if i == maxWidth-1 && b != 0x00 && b != 0x7f {
			return 0, ErrOverflow
		}
		res |= int64(b&0x7f) << shift
		shift += 7
		if b&0x80 == 0 {
			if shift < 64 && b&0x40 != 0 {
				res |= ^int64(0) << shift
			}
			return res, nil
		}
	}
	panic("unreachable: byte-10 overflow check guarantees loop exits")
}

// EncodeU64 converts num to a uleb128-encoded slice of bytes.
func EncodeU64(num uint64) []byte {
	// Most values in real workloads fit in one byte; special-case to skip
	// the loop entirely.
	if num < 0x80 {
		return []byte{byte(num)}
	}
	buf := make([]byte, 0, maxWidth)
	for {
		b := byte(num & 0x7f)
		num >>= 7
		if num == 0 {
			return append(buf, b)
		}
		buf = append(buf, b|0x80)
	}
}

// EncodeS64 converts num to a sleb128-encoded slice of bytes.
func EncodeS64(num int64) []byte {
	// Single-byte fast path: values in [-0x40, 0x40) encode in one byte.
	if num >= -0x40 && num < 0x40 {
		return []byte{byte(num) & 0x7f}
	}
	// From https://go.dev/ref/spec#Arithmetic_operators: "The shift
	// operators implement arithmetic shifts if the left operand is a signed
	// integer and logical shifts if it is an unsigned integer." We rely on
	// the arithmetic shift to drive num toward 0 (positive) or -1
	// (negative) as the loop terminator.
	buf := make([]byte, 0, maxWidth)
	for {
		b := byte(num & 0x7f)
		num >>= 7
		signBit := b & 0x40
		if (num == 0 && signBit == 0) || (num == -1 && signBit != 0) {
			return append(buf, b)
		}
		buf = append(buf, b|0x80)
	}
}
