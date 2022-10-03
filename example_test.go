package leb128_test

import (
	"bytes"
	"fmt"

	"github.com/jcalabro/leb128"
)

func ExampleDecodeU64() {
	buf := bytes.NewBuffer([]byte{128, 2})
	num, err := leb128.DecodeU64(buf)
	if err != nil {
		panic(err)
	}
	fmt.Println(num)
	// Output:
	// 256
}

func ExampleEncodeU64() {
	buf := leb128.EncodeU64(256)
	fmt.Println(buf)
	// Output:
	// [128 2]
}

func ExampleDecodeS64() {
	buf := bytes.NewBuffer([]byte{128, 126})
	num, err := leb128.DecodeS64(buf)
	if err != nil {
		panic(err)
	}
	fmt.Println(num)
	// Output:
	// -256
}

func ExampleEncodeS64() {
	buf := leb128.EncodeS64(-256)
	fmt.Println(buf)
	// Output:
	// [128 126]
}
