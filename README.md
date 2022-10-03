# leb128

[![GoDoc](https://pkg.go.dev/badge/github.com/jcalabro/leb128?status.svg)](https://pkg.go.dev/github.com/jcalabro/leb128) [![Tests](https://github.com/jcalabro/leb128/actions/workflows/ci.yaml/badge.svg)](https://github.com/jcalabro/leb128/actions/workflows/ci.yaml) [![codecov](https://codecov.io/github/jcalabro/leb128/branch/main/graph/badge.svg?token=ILKTKORT5D)](https://codecov.io/github/jcalabro/leb128)

Go implementation of signed/unsigned LEB128. Encodes/decodes 8 byte integers.

## Usage

Full documentation is available at [gopkg.dev](https://pkg.go.dev/github.com/jcalabro/leb128).

#### Decoding

Read from an `io.Reader` in to an `int64` or `uint64`:

```go
package main

import (
	"bytes"
	"fmt"

	"github.com/jcalabro/leb128"
)

func unsigned() {
	// using a buffer of length >10 in either the signed/unsigned case
	// will return the error leb128.ErrOverflow

	buf := bytes.NewBuffer([]byte{128, 2})
	num, err := leb128.DecodeU64(buf)
	if err != nil {
		panic(err)
	}
	fmt.Println(num) // 256
}

func signed() {
	buf := bytes.NewBuffer([]byte{128, 126})
	num, err := leb128.DecodeS64(buf)
	if err != nil {
		panic(err)
	}
	fmt.Println(num) // -256
}

func main() {
	unsigned()
	signed()
}
```

#### Encoding

Read convert an `int64` or `uint64` to a `[]byte`:

```go
package main

import (
	"bytes"
	"fmt"

	"github.com/jcalabro/leb128"
)

func unsigned() {
	buf := leb128.EncodeU64(256)
	fmt.Println(buf) // [128 2]
}

func signed() {
	buf := leb128.EncodeS64(-256)
	fmt.Println(buf) // [128 126]
}

func main() {
	unsigned()
	signed()
}
```
