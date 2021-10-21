package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func bar(reader io.Reader) (err error) {
	content, err := ioutil.ReadAll(reader)
	if err == nil {
		_, err = os.Stdout.Write(content)
	}
	return
}

func foo(writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "foo foo\n")
	return err
}
