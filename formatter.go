package tools

import (
	"bytes"
	"go/ast"
	"go/token"
	"strings"
)

type formatter struct {
	*Tools

	file   *ast.File
	src    []byte
	writer *bytes.Buffer
}

func (f *formatter) readline(start token.Pos) string {
	i := strings.Index(string(f.src[start-1:]), "\n")
	if i >= 0 {
		return string(f.src[start-1 : int(start)+i])
	}
	return ""
}

func (f *formatter) write(str string) {
	//println("Writing:", str)
	f.writer.Write([]byte(str))
}

func (f *formatter) writePos(start, end token.Pos) {
	//println("Writing:", string(f.src[start-1:end-1]))
	f.writer.Write(f.src[start-1 : end-1])
}
