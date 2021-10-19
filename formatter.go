package tools

import (
	"bytes"
	"go/ast"
	"go/token"
	"strings"
)

type formatter struct {
	*Tools

	filename string
	file     *ast.File
	src      []byte
	writer   *bytes.Buffer
}

func (f *formatter) readline(start token.Pos) string {
	i := strings.Index(string(f.src[start-1:]), "\n")
	if i >= 0 {
		return string(f.src[start-1 : int(start)+i])
	}
	return ""
}

func (f *formatter) writeStr(str string) {
	f.write([]byte(str))
}

func (f *formatter) write(content []byte) {
	//println("Writing:", str)
	f.writer.Write(content)
}

func (f *formatter) writePos(start, end token.Pos) {
	//println("Writing:", string(f.src[start-1:end-1]))
	if end == -1 {
		end = f.file.End()
	}
	f.writer.Write(f.src[start-1 : end-1])
}

func (f *formatter) reload() error {
	_, err := f.setSrc(f.filename, f.writer.Bytes())
	return err
}
