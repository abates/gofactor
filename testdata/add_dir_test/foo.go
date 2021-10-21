package main

import (
	"fmt"
	"io"
)

func foo(writer io.Writer) {
	fmt.Fprintf(writer, "foo foo\n")
}
