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
	//println("Writing:", str)
	f.writer.Write([]byte(str))
}

func (f *formatter) writePos(start, end token.Pos) {
	//println("Writing:", string(f.src[start-1:end-1]))
	f.writer.Write(f.src[start-1 : end-1])
}

func typStr(expr ast.Expr) (str string) {
	if expr == nil {
		return ""
	}

	switch t := expr.(type) {
	case *ast.Ident:
		str = t.Name
	case *ast.StarExpr:
		str = typStr(t.X)
	}
	return str
}
