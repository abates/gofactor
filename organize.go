package gofactor

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"sort"
)

func Organize(filename string, input []byte) (output []byte, err error) {
	input, err = SeparateValues(filename, input)
	if err != nil {
		return input, err
	}

	o := &organizer{
		formatter: newFormatter(),
		typIndex:  make(map[string]*typSrc),
	}
	o.formatter.parse(filename, input)

	err = o.organize()
	if err == nil {
		output, err = format.Source(o.writer.Bytes())
		if err != nil {
			output = o.writer.Bytes()
			err = fmt.Errorf("Unexpected formatting error: %w", err)
		}
	}
	return output, err
}

type namedSrc interface {
	name() string
	write(*formatter)
}

type namedList []namedSrc

func (o namedList) Len() int           { return len(o) }
func (o namedList) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }
func (o namedList) Less(i, j int) bool { return o[i].name() < o[j].name() }
func (o namedList) write(f *formatter) {
	sort.Sort(o)
	for _, s := range o {
		s.write(f)
	}
}

type nodeName string

func (nn nodeName) name() string { return string(nn) }

type nodeSrc struct {
	nodeName
	start token.Pos
	end   token.Pos
}

func (n *nodeSrc) write(f *formatter) {
	f.writePos(n.start, n.end)
	f.write("\n\n")
}

type typSrc struct {
	nodeName
	start   token.Pos
	end     token.Pos
	values  namedList
	funcs   namedList
	methods namedList
}

func (t *typSrc) write(f *formatter) {
	f.writePos(t.start, t.end)
	f.write("\n\n")
	t.values.write(f)
	f.write("\n\n")
	t.funcs.write(f)
	f.write("\n\n")
	t.methods.write(f)
	f.write("\n\n")
}

type organizer struct {
	*formatter
	pkg      token.Pos
	imports  namedList
	values   namedList
	funcs    namedList
	typs     namedList
	typIndex map[string]*typSrc
	pos      token.Pos
}

func (o *organizer) analyzeValue(v *ast.GenDecl, start, end token.Pos) {
	vs := v.Specs[0].(*ast.ValueSpec)

	// only organize parenthesized blocks and
	// single named values
	if v.Lparen.IsValid() && len(vs.Names) == 1 {
		typName := ""
		if vs.Type == nil {
			typName = o.typStr(vs.Values[0])
		} else {
			typName = typStr(vs.Type)
		}

		typ, found := o.typIndex[typName]
		if found {
			typ.values = append(typ.values, &nodeSrc{nodeName: nodeName(v.Tok.String()), start: start, end: end})
			o.pos = end
			return
		}
	}
	o.values = append(o.values, &nodeSrc{nodeName: nodeName(v.Tok.String()), start: start, end: end})
}

func (o *organizer) analyzeType(v *ast.GenDecl) {
	start := v.Pos()
	if v.Doc != nil {
		start = v.Doc.Pos()
	}
	end := v.End()

	// only organize non-parenthesized types
	if v.Lparen.IsValid() {
		return
	}

	ts := v.Specs[0].(*ast.TypeSpec)
	o.typIndex[ts.Name.Name] = &typSrc{
		nodeName: nodeName(ts.Name.Name),
		start:    start,
		end:      end,
	}
	o.typs = append(o.typs, o.typIndex[ts.Name.Name])
}

func (o *organizer) analyzeFunc(v *ast.FuncDecl) {
	start := v.Pos()
	if v.Doc != nil {
		start = v.Doc.Pos()
	}
	end := v.End()
	o.pos = end

	if v.Recv == nil { // regular function
		if v.Type.Results != nil {
			for _, result := range v.Type.Results.List {
				if typ, found := o.typIndex[typStr(result.Type)]; found {
					typ.funcs = append(typ.funcs, &nodeSrc{nodeName: nodeName(v.Name.Name), start: start, end: end})
					return
				}
			}
		}
	} else { // method
		if typ, found := o.typIndex[typStr(v.Recv.List[0].Type)]; found {
			typ.methods = append(typ.methods, &nodeSrc{nodeName: nodeName(v.Name.Name), start: start, end: end})
			return
		}
	}

	o.funcs = append(o.funcs, &nodeSrc{nodeName: nodeName(v.Name.Name), start: start, end: end})
}

func (o *organizer) analyzeTypes() {
	for _, decl := range o.files[o.currentFile].Decls {
		if d, ok := decl.(*ast.GenDecl); ok {
			if d.Tok == token.TYPE {
				o.analyzeType(d)
			}
		}
	}
}

func (o *organizer) organize() error {
	o.analyzeTypes()
	for _, decl := range o.files[o.currentFile].Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			o.analyzeFunc(d)
		case *ast.GenDecl:
			start := d.Pos()
			if d.Doc != nil {
				start = d.Doc.Pos()
			}
			end := d.End()

			if d.Tok == token.CONST || d.Tok == token.VAR {
				o.analyzeValue(d, start, end)
			} else if d.Tok == token.IMPORT {
				o.imports = append(o.imports, &nodeSrc{nodeName: "import", start: start, end: end})
			} else if d.Tok != token.TYPE {
				return fmt.Errorf("Unknown declaration node: %s", d.Tok)
			}
		case *ast.BadDecl:
			return fmt.Errorf("Syntax error at %d", decl.Pos())
		default:
			return fmt.Errorf("Unknown declaration node %T", decl)
		}
	}

	pkg := o.files[o.currentFile].Package
	o.writePos(1, pkg)
	o.write(o.readline(o.currentFile, pkg))
	o.imports.write(o.formatter)
	o.values.write(o.formatter)
	o.funcs.write(o.formatter)
	o.typs.write(o.formatter)

	return nil
}
