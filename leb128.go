// Package leb128 provides functionality to encode/decode signed and unsigned LEB128 data to and from 8 byte primitives.
//
// It deals only with 8 byte primitives; attempting to decode integers larger than that will cause an ErrOverflow.
//
// This package operates on a basic io.Reader rather than an io.ByteReader as the standard library does (i.e. the various Varint functions in https://pkg.go.dev/encoding/binary).
//
// See https://en.wikipedia.org/wiki/LEB128 for more details.
package leb128

import (
	"errors"
	"io"
)

var (
	ErrOverflow   = errors.New("LEB128 integer overflow")
	ErrNonMinimal = errors.New("LEB128 integer encoding was not minimal")
)

// DecodeU32 converts a uleb128 byte stream to a uint32. Be careful
// to ensure that your data can fit in 4 bytes.
func DecodeU32(r io.Reader) (uint32, error) {
	var res uint32 = 0
	var shift uint = 0

	var buf [1]byte

	for i := 0; ; i++ {
		_, err := io.ReadFull(r, buf[:])
		if err == io.EOF {
			if i == 0 {
				return 0, io.EOF
			} else {
				return 0, ErrNonMinimal
			}
		}
		if err != nil {
			return 0, err
		}

		b := buf[0]
		res |= uint32(b&0x7F) << shift
		shift += 7

		if (b & 0x80) == 0 {
			if shift > 32 && b > 0b1111 {
				return 0, ErrOverflow
			} else if shift > 7 && b == 0 {
				return 0, ErrNonMinimal
			}
			return res, nil
		} else if shift > 32 {
			return 0, ErrOverflow
		}
	}
}

// DecodeU64 converts a uleb128 byte stream to a uint64. Be careful
// to ensure that your data can fit in 8 bytes.
func DecodeU64(r io.Reader) (uint64, error) {
	var res uint64 = 0
	var shift uint = 0

	var buf [1]byte

	for i := 0; ; i++ {
		_, err := io.ReadFull(r, buf[:])
		if err == io.EOF {
			if i == 0 {
				return 0, io.EOF
			} else {
				return 0, ErrNonMinimal
			}
		}
		if err != nil {
			return 0, err
		}

		b := buf[0]
		res |= uint64(b&0x7F) << shift
		shift += 7

		if (b & 0x80) == 0 {
			if shift > 64 && b > 1 {
				return 0, ErrOverflow
			} else if shift > 7 && b == 0 {
				return 0, ErrNonMinimal
			}
			return res, nil
		} else if shift > 64 {
			return 0, ErrOverflow
		}
	}
}

// DecodeS64 converts a sleb128 byte stream to a int64. Be careful
// to ensure that your data can fit in 8 bytes.
func DecodeS64(r io.Reader) (int64, error) {
	var res int64 = 0
	var shift uint = 0
	var prev byte = 0

	var buf [1]byte

	for i := 0; ; i++ {
		_, err := io.ReadFull(r, buf[:])
		if err == io.EOF {
			if i == 0 {
				return 0, io.EOF
			} else {
				return 0, ErrNonMinimal
			}
		}
		if err != nil {
			return 0, err
		}

		b := buf[0]
		res |= int64(b&0x7F) << shift
		shift += 7

		if (b & 0x80) == 0 {
			if shift > 64 && b != 0 && b != 0x7f {
				// the 10th byte (if present) must contain only the sign-extended sign bit
				return 0, ErrOverflow
			} else if shift > 7 &&
				((b == 0 && prev&0x40 == 0) || (b == 0x7f && prev&0x40 > 0)) {
				// overlong if the sign bit of penultimate byte has been extended
				return 0, ErrNonMinimal
			} else if shift < 64 && b&0x40 > 0 {
				// sign extend negative numbers
				res |= -1 << shift
			}
			return res, nil
		} else if shift > 64 {
			return 0, ErrOverflow
		}
		prev = b
	}
}

// EncodeU32 converts num to a uleb128 encoded array of bytes
func EncodeU32(num uint32) []byte {
	buf := make([]byte, 0, 4)

	done := false
	for !done {
		b := byte(num & 0x7F)

		num = num >> 7
		if num == 0 {
			done = true
		} else {
			b |= 0x80
		}

		buf = append(buf, b)
	}

	return buf
}

// EncodeU64 converts num to a uleb128 encoded array of bytes
func EncodeU64(num uint64) []byte {
	buf := make([]byte, 0, 8)

	done := false
	for !done {
		b := byte(num & 0x7F)

		num = num >> 7
		if num == 0 {
			done = true
		} else {
			b |= 0x80
		}

		buf = append(buf, b)
	}

	return buf
}

// EncodeS64 converts num to a sleb128 encoded array of bytes
func EncodeS64(num int64) []byte {
	buf := make([]byte, 0, 8)

	done := false
	for !done {
		//
		// From https://go.dev/ref/spec#Arithmetic_operators:
		//
		// "The shift operators implement arithmetic shifts
		// if the left operand is a signed integer and
		// logical shifts if it is an unsigned integer"
		//

		b := byte(num & 0x7F)
		num >>= 7 // arithmetic shift
		signBit := b & 0x40
		if (num == 0 && signBit == 0) ||
			(num == -1 && signBit != 0) {
			done = true
		} else {
			b |= 0x80
		}

		buf = append(buf, b)
	}

	return buf
}
