package main

import (
	"strings"
	"testing"
)

func TestFoo(t *testing.T) {
	want := "foo foo\n"
	writer := &strings.Builder{}
	foo(writer)
	got := writer.String()
	if want != got {
		t.Errorf("Wanted %q got %q", want, got)
	}
}
