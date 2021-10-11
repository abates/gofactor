package gofactor

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
)

func parse(src []byte) (f *formatter, err error) {
	f = &formatter{
		src:    src,
		writer: &bytes.Buffer{},
	}
	fset := token.NewFileSet()
	f.f, err = parser.ParseFile(fset, "", src, parser.ParseComments)
	return
}

type formatter struct {
	f      *ast.File
	src    []byte
	writer *bytes.Buffer
}

func (f *formatter) write(str string) {
	f.writer.Write([]byte(str))
}

func (f *formatter) writePos(start, end token.Pos) {
	f.writer.Write(f.src[start-1 : end-1])
}

func (f *formatter) typStr(vs *ast.ValueSpec) string {
	if vs == nil {
		return ""
	}
	return string(f.src[vs.Type.Pos():vs.Type.End()])
}
