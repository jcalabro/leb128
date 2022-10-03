package leb128_test

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"github.com/jcalabro/leb128"

	"github.com/stretchr/testify/require"
)

type errorReader struct{}

func (er *errorReader) Read(_ []byte) (int, error) {
	return 0, fmt.Errorf("test error")
}

func TestUnsigned(t *testing.T) {
	// simple low-range cases
	for ndx := uint64(0); ndx < 512; ndx++ {
		buf := leb128.EncodeU64(ndx)
		require.NotEmpty(t, buf)
		if ndx >= 384 { // [384,512)
			// i.e. 384 => [128,3]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx), buf[0])
			require.Equal(t, byte(3), buf[1])
		} else if ndx >= 256 { // [256,384)
			// i.e. 256 => [128,2]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx-128), buf[0])
			require.Equal(t, byte(2), buf[1])
		} else if ndx >= 128 { // [128,256)
			// i.e. 256 => [128,1]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx), buf[0])
			require.Equal(t, byte(1), buf[1])
		} else { // [0,128)
			require.Len(t, buf, 1)
			require.Equal(t, byte(ndx), buf[0])
		}

		// translate back
		res, err := leb128.DecodeU64(bytes.NewBuffer(buf))
		require.NoError(t, err)
		require.Equal(t, ndx, res)
	}

	{
		// max uint64
		expected := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}

		buf := leb128.EncodeU64(math.MaxUint64)
		require.Equal(t, expected, buf)

		// translate back
		res, err := leb128.DecodeU64(bytes.NewBuffer(buf))
		require.NoError(t, err)
		require.Equal(t, uint64(math.MaxUint64), res)
	}

	{
		// empty buffer
		res, err := leb128.DecodeU64(bytes.NewBuffer([]byte{}))
		require.NoError(t, err)
		require.Zero(t, res)
	}

	{
		// read error
		res, err := leb128.DecodeU64(&errorReader{})
		require.Error(t, err)
		require.Zero(t, res)
	}

	{
		// ensure that we stop at the correct time
		input := []byte{0x78, 0x10, 0xf, 0xa, 0xb, 0x90, 0x01, 0, 0xff, 0xff, 0xff}
		res, err := leb128.DecodeU64(bytes.NewBuffer(input))
		require.NoError(t, err)
		require.Equal(t, uint64(120), res)
	}

	{
		// restrict to 10 bytes (final bytes would overflow an 8 byte integer)
		input := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x0}

		res, err := leb128.DecodeU64(bytes.NewBuffer(input))
		require.ErrorIs(t, err, leb128.ErrOverflow)
		require.Equal(t, uint64(0), res)
	}
}

func TestEncodeS64(t *testing.T) {
	// simple low-range positive cases
	for ndx := int64(0); ndx < 512; ndx++ {
		buf := leb128.EncodeS64(ndx)
		require.NotEmpty(t, buf)
		if ndx >= 384 { // [384,512)
			// i.e. 384 => [128,3]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx), buf[0])
			require.Equal(t, byte(3), buf[1])
		} else if ndx >= 256 { // [256,384)
			// i.e. 256 => [128,2]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx+128), buf[0])
			require.Equal(t, byte(2), buf[1])
		} else if ndx >= 128 { // [128,256)
			// i.e. 256 => [128,1]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx), buf[0])
			require.Equal(t, byte(1), buf[1])
		} else if ndx >= 64 { // [0,64)
			// i.e. 64 => [192,1]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx+128), buf[0])
			require.Equal(t, byte(0), buf[1])
		} else { // [0,64)
			require.Len(t, buf, 1)
			require.Equal(t, byte(ndx), buf[0])
		}

		// translate back
		res, err := leb128.DecodeS64(bytes.NewBuffer(buf))
		require.NoError(t, err)
		require.Equal(t, ndx, res)
	}

	// simple low-range negative cases
	for ndx := int64(-512); ndx < 0; ndx++ {
		buf := leb128.EncodeS64(ndx)
		require.NotEmpty(t, buf)
		if ndx < -384 { // [-512,-384)
			// i.e. -512 => [128,124]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx+128), buf[0])
			require.Equal(t, byte(124), buf[1])
		} else if ndx < -256 { // [-384,-256)
			// i.e. -384 => [128,125]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx), buf[0])
			require.Equal(t, byte(125), buf[1])
		} else if ndx < -128 { // [-256,-128)
			// i.e. -256 => [128,126]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx+128), buf[0])
			require.Equal(t, byte(126), buf[1])
		} else if ndx < -64 { // [-128,-64)
			// i.e. -128 => [128,127]
			require.Len(t, buf, 2)
			require.Equal(t, byte(ndx), buf[0])
			require.Equal(t, byte(127), buf[1])
		} else {
			require.Len(t, buf, 1)
			require.Equal(t, byte(ndx+128), buf[0])
		}

		// translate back
		res, err := leb128.DecodeS64(bytes.NewBuffer(buf))
		require.NoError(t, err)
		require.Equal(t, ndx, res)
	}

	{
		// max int64
		expected := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0}

		buf := leb128.EncodeS64(math.MaxInt64)
		require.Equal(t, expected, buf)

		// translate back
		res, err := leb128.DecodeS64(bytes.NewBuffer(buf))
		require.NoError(t, err)
		require.Equal(t, int64(math.MaxInt64), res)
	}

	{
		// min int64
		expected := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x7f}

		buf := leb128.EncodeS64(math.MinInt64)
		require.Equal(t, expected, buf)

		// translate back
		res, err := leb128.DecodeS64(bytes.NewBuffer(buf))
		require.NoError(t, err)
		require.Equal(t, int64(math.MinInt64), res)
	}

	{
		// empty buffer
		res, err := leb128.DecodeS64(bytes.NewBuffer([]byte{}))
		require.NoError(t, err)
		require.Zero(t, res)
	}

	{
		// read error
		res, err := leb128.DecodeS64(&errorReader{})
		require.Error(t, err)
		require.Zero(t, res)
	}

	{
		// restrict to 10 bytes (final bytes overflow an 8 byte integer)
		input := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0xff}

		res, err := leb128.DecodeS64(bytes.NewBuffer(input))
		require.ErrorIs(t, err, leb128.ErrOverflow)
		require.Equal(t, int64(0), res)
	}
}
