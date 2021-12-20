package tools

import (
	"go/token"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

func SeparateValues(filename string, input []byte) (output []byte, err error) {
	tools := New()
	err = tools.Add(filename, input)

	if err == nil {
		output, err = tools.SeparateValues(filename)
	}
	return
}

type valueCleaner struct {
	file *dst.File
}

func (vc *valueCleaner) separateValDecl(decl *dst.GenDecl) (results []dst.Node) {
	// only refactor parenthesized decalarations
	if decl.Lparen {
		lastType := ""
		newDecl := &dst.GenDecl{Tok: decl.Tok, Lparen: true, Rparen: true, Decs: decl.Decs}
		for _, spec := range decl.Specs {
			vs := spec.(*dst.ValueSpec)
			if len(vs.Names) < 2 {
				if lastType == "" {
					lastType = typStr(vs.Type)
				}

				// check if the next spec type is different than
				// the previous, and if so, close the block
				// and start a new one
				if vs.Type != nil && lastType != typStr(vs.Type) {
					// end the block and start a new one
					// with the next spec
					lastType = typStr(vs.Type)
					newDecl.Specs[len(newDecl.Specs)-1].Decorations().After = dst.None
					results = append(results, newDecl)
					newDecl = &dst.GenDecl{Tok: decl.Tok, Lparen: true, Rparen: true}
					spec.Decorations().Before = dst.NewLine
				}
				newDecl.Specs = append(newDecl.Specs, spec)
			}
		}
		results = append(results, newDecl)
	} else {
		results = []dst.Node{decl}
	}
	return
}

func (vc *valueCleaner) walk(cursor *dstutil.Cursor) bool {
	if d, ok := cursor.Node().(*dst.GenDecl); ok {
		if d.Tok == token.CONST || d.Tok == token.VAR {
			results := vc.separateValDecl(d)
			cursor.Replace(results[0])
			for _, n := range results[1:] {
				n.Decorations().Before = dst.EmptyLine
				cursor.InsertAfter(n)
			}
		}
	}

	_, ok := cursor.Node().(*dst.File)
	return ok
}

func (vc *valueCleaner) separateValDecls() *dst.File {
	return dstutil.Apply(vc.file, vc.walk, nil).(*dst.File)
}
