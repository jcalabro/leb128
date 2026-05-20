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
func DecodeU64(r io.Reader) (uint64, error) {
	var (
		res uint64
		buf [1]byte
	)

	for i := 0; i < maxWidth; i++ {
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

		// On the 10th (final) byte, only bit 0 of the payload fits in
		// uint64; any other bit set, including the continuation bit
		// (which would require an 11th byte), means overflow.
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
func DecodeS64(r io.Reader) (int64, error) {
	var (
		res   int64
		buf   [1]byte
		shift uint
	)

	for i := 0; i < maxWidth; i++ {
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

		// On the 10th byte, bit 0 of the payload becomes the int64 sign
		// bit and bits 1-6 must agree with it (sign extension). The only
		// payloads satisfying that are 0x00 (positive) and 0x7f
		// (negative); anything else (including a set continuation bit)
		// means the encoded value overflows.
		if i == maxWidth-1 && b != 0x00 && b != 0x7f {
			return 0, ErrOverflow
		}

		res |= int64(b&0x7f) << shift
		shift += 7

		if b&0x80 == 0 {
			// sign-extend negative values; the shift < 64 guard avoids
			// relying on Go's well-defined-but-confusing behavior for
			// shifts that exceed the operand width.
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
	buf := make([]byte, 0, maxWidth)
	for {
		// From https://go.dev/ref/spec#Arithmetic_operators: "The shift
		// operators implement arithmetic shifts if the left operand is a
		// signed integer and logical shifts if it is an unsigned
		// integer." We rely on the arithmetic shift to drive num toward
		// 0 (positive) or -1 (negative) as the loop terminator.
		b := byte(num & 0x7f)
		num >>= 7
		signBit := b & 0x40
		if (num == 0 && signBit == 0) || (num == -1 && signBit != 0) {
			return append(buf, b)
		}
		buf = append(buf, b|0x80)
	}
}
