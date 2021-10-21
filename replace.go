package tools

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
)

func FindFunc(file *ast.File, name, receiver string) (*ast.FuncDecl, error) {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.String() == name {
				if (fn.Recv == nil && receiver == "") || (fn.Recv != nil && typStr(fn.Recv.List[0].Type) == receiver) {
					return fn, nil
				}
			}
		}
	}
	return nil, ErrDeclNotFound
}

type Replacer struct {
	*formatter
	filename string
	Err      error
}

func Replace(filename string, src []byte) (r *Replacer) {
	r = &Replacer{
		formatter: &formatter{
			src:    src,
			writer: bytes.NewBuffer(nil),
		},
		filename: filename,
	}

	r.file, r.Err = parser.ParseFile(token.NewFileSet(), filename, src, parser.ParseComments)
	return r
}

func (r *Replacer) Content() []byte {
	return (r.src)
}

func (r *Replacer) Func(name, receiver string, content []byte) *Replacer {
	if r.Err == nil {
		var decl *ast.FuncDecl
		decl, r.Err = FindFunc(r.file, name, receiver)
		r.Pos(decl.Pos(), decl.End(), content)
	}
	return r
}

func (r *Replacer) Pos(start, end token.Pos, content []byte) *Replacer {
	if r.Err == nil {
		r.writePos(1, start)
		r.write(content)
		r.writePos(end, -1)

		r.src, r.Err = format.Source(r.writer.Bytes())
		if r.Err == nil {
			r.writer.Reset()
			r.file, r.Err = parser.ParseFile(token.NewFileSet(), r.filename, r.src, parser.ParseComments)
		} else {
			r.src = r.writer.Bytes()
		}
	}
	return r
}
