package gofactor

import (
	"fmt"
	"go/ast"
	"go/token"
)

func SeparateValues(filename string, input []byte) (output []byte, err error) {
	formatter := NewFormatter()
	err = formatter.AddFile(filename, input)
	if err == nil {
		err = formatter.Load()
	}

	if err == nil {
		output, err = formatter.SeparateValues(filename)
	}
	return
}

type valueCleaner struct {
	*fileFormatter
}

func (vc *valueCleaner) separateValDecl(decl *ast.GenDecl) {
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
					vc.writePos(pos, end)

					start := vs.Pos()
					if vs.Doc != nil {
						start = vs.Doc.Pos()
					}

					// end the const block and start a new one
					// with the next spec
					vc.write(fmt.Sprintf("\n)\n\n%s (\n", decl.Tok))
					lastType = typStr(vs.Type)
					pos = start
				}
			}
			prev = vs
		}

		if pos != decl.Pos() {
			vc.writePos(pos, decl.End())
		}
	}

	if pos == decl.Pos() {
		vc.writePos(decl.Pos(), decl.End())
	}
}

func (vc *valueCleaner) separateValDecls() {
	pos := token.Pos(1)
	for _, decl := range vc.file.Decls {
		if d, ok := decl.(*ast.GenDecl); ok {
			vc.writePos(pos, d.Pos())
			if d.Tok == token.CONST || d.Tok == token.VAR {
				vc.separateValDecl(d)
				pos = decl.End()
			} else {
				pos = decl.Pos()
			}
		} else {
			vc.writePos(pos, decl.End())
			pos = decl.End()
		}
	}

	if pos != vc.file.End() {
		vc.writePos(pos, vc.file.End())
	}
}
