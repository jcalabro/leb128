// @TODO: package doc comment
// https://en.wikipedia.org/wiki/LEB128#C-like_pseudocode
package leb128

import (
	"io"
)

// DecodeU64 converts a uleb128 byte stream to a uint64. Be careful
// to ensure that your stream does not overflow - this function is only
// approproat ewhen you know the number in question can be contained
// in a uint64.
func DecodeU64(r io.Reader) (uint64, error) {
	var res uint64

	bit := int8(0)
	buf := make([]byte, 1)
	for {
		_, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		b := buf[0]

		res |= uint64(b&0x7f) << (7 * bit)
		bit++
	}

	return res, nil
}

// DecodeI64 converts a sleb128 byte stream to a int64. Be careful
// to ensure that your stream does not overflow - this function is only
// approproate when you know the number in question can be contained
// in an int64.
func DecodeS64(r io.Reader) (int64, error) {
	var res int64

	shift := 0
	buf := make([]byte, 1)
	for {
		_, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		b := buf[0]

		res |= int64(b&0x7f) << shift
		shift += 7

		if b&0x80 == 0 {
			if b&0x40 != 0 {
				// signed
				res |= ^0 << shift
			}
			break
		}
	}

	return res, nil
}

// EncodeU64 converts num to a uleb128 encoded array of bytes
func EncodeU64(num uint64) []byte {
	buf := make([]byte, 0)

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
	buf := make([]byte, 0)

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
