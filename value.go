package gofactor

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
)

func SeparateValues(filename string, input []byte) ([]byte, error) {
	cc := &valueCleaner{formatter: newFormatter()}
	err := cc.formatter.parse(filename, input)
	if err != nil {
		return nil, err
	}

	cc.separateValDecls()
	output, err := format.Source(cc.writer.Bytes())
	if err != nil {
		output = cc.writer.Bytes()
	}
	return output, err
}

type valueCleaner struct {
	*formatter
}

func (cc *valueCleaner) separateValDecl(decl *ast.GenDecl) {
	pos := decl.Pos()
	// only refactor parenthesized decalarations
	if decl.Lparen.IsValid() {
		lastType := ""
		var prev *ast.ValueSpec
		for _, spec := range decl.Specs {
			vs := spec.(*ast.ValueSpec)
			if len(vs.Names) < 2 {
				if lastType == "" {
					lastType = typStr(vs.Type)
				}

				// check if the next spec type is different than
				// the previous, and if so, close the block
				// and start a new one
				if vs.Type != nil && lastType != typStr(vs.Type) {
					// make sure to capture the comment
					end := prev.End()
					if prev.Comment != nil {
						end = prev.Comment.End()
					}

					// write out the previous spec
					cc.writePos(pos, end)

					start := vs.Pos()
					if vs.Doc != nil {
						start = vs.Doc.Pos()
					}

					// end the const block and start a new one
					// with the next spec
					cc.write(fmt.Sprintf("\n)\n\n%s (\n", decl.Tok))
					lastType = typStr(vs.Type)
					pos = start
				}
			}
			prev = vs
		}

		if pos != decl.Pos() {
			cc.writePos(pos, decl.End())
		}
	}

	if pos == decl.Pos() {
		cc.writePos(decl.Pos(), decl.End())
	}
}

func (cc *valueCleaner) separateValDecls() {
	pos := token.Pos(1)
	for _, decl := range cc.files[cc.currentFile].Decls {
		if d, ok := decl.(*ast.GenDecl); ok {
			cc.writePos(pos, d.Pos())
			if d.Tok == token.CONST || d.Tok == token.VAR {
				cc.separateValDecl(d)
				pos = decl.End()
			} else {
				pos = decl.Pos()
			}
		} else {
			cc.writePos(pos, decl.End())
			pos = decl.End()
		}
	}

	if pos != cc.files[cc.currentFile].End() {
		cc.writePos(pos, cc.files[cc.currentFile].End())
	}
}
