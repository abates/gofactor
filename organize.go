package tools

import (
	"go/token"
	"sort"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

func Organize(filename string, input []byte) (output []byte, err error) {
	tools := New()
	err = tools.Add(filename, input)

	if err == nil {
		output, err = tools.Organize(filename)
	}
	return
}

type organizer struct {
	file  *dst.File
	types map[string]sortableSource
}

func (o *organizer) analyzeTypes() (names []string) {
	o.types = make(map[string]sortableSource)
	walk := func(cursor *dstutil.Cursor) bool {
		cont := false
		switch n := cursor.Node().(type) {
		case *dst.GenDecl:
			if n.Tok == token.TYPE {
				// only organize non-parenthesized types
				if !n.Lparen {
					ts := n.Specs[0].(*dst.TypeSpec)
					o.types[ts.Name.Name] = sortableSource{n}
					names = append(names, ts.Name.Name)
					cursor.Delete()
				}
			}
		case *dst.File:
			cont = true
		}

		return cont
	}

	o.file = dstutil.Apply(o.file, walk, nil).(*dst.File)
	return names
}

func (o *organizer) analyzeFunc(cursor *dstutil.Cursor) {
	fn := cursor.Node().(*dst.FuncDecl)
	typName := ""
	if fn.Recv == nil {
		if fn.Type.Results != nil {
			for _, result := range fn.Type.Results.List {
				typName = typStr(result.Type)
				if _, found := o.types[typName]; found {
					o.types[typName] = append(o.types[typName], fn)
					cursor.Delete()
					break
				}
			}
		}
	} else {
		typName := typStr(fn.Recv.List[0].Type)
		if _, found := o.types[typName]; found {
			o.types[typName] = append(o.types[typName], fn)
			cursor.Delete()
		}
	}
}

func (o *organizer) analyzeValue(cursor *dstutil.Cursor) {
	decl := cursor.Node().(*dst.GenDecl)
	vs := decl.Specs[0].(*dst.ValueSpec)
	typName := ""
	if decl.Lparen && len(vs.Names) == 1 {
		if vs.Type == nil {
			typName = typStr(vs.Values[0])
		} else {
			typName = typStr(vs.Type)
		}
	}

	if _, found := o.types[typName]; found {
		o.types[typName] = append(o.types[typName], decl)
		cursor.Delete()
	}
}

func (o *organizer) organize() *dst.File {
	names := o.analyzeTypes()

	walk := func(cursor *dstutil.Cursor) bool {
		cont := false
		switch n := cursor.Node().(type) {
		case *dst.FuncDecl:
			o.analyzeFunc(cursor)
		case *dst.GenDecl:
			if n.Tok == token.CONST || n.Tok == token.VAR {
				o.analyzeValue(cursor)
			}
		case *dst.File:
			cont = true
		}

		return cont
	}

	result := dstutil.Apply(o.file, walk, nil).(*dst.File)
	sort.Sort(sortableSource(result.Decls))
	sort.Strings(names)
	for _, name := range names {
		sort.Sort(o.types[name])
		result.Decls = append(result.Decls, o.types[name]...)
	}

	return result
}

type sortableSource []dst.Decl

func (ss sortableSource) prec(i int) int {
	if decl, ok := ss[i].(*dst.GenDecl); ok {
		switch decl.Tok {
		case token.IMPORT:
			return 0
		case token.TYPE:
			return 1
		case token.CONST:
			return 2
		case token.VAR:
			return 3
		}
	} else if decl, ok := ss[i].(*dst.FuncDecl); ok {
		if decl.Recv == nil {
			return 4
		}
	}
	return 5
}

func (ss sortableSource) name(i int) (name string) {
	switch n := ss[i].(type) {
	case *dst.FuncDecl:
		name = n.Name.Name
	case *dst.GenDecl:
		name = n.Tok.String()
	}
	return
}

func (ss sortableSource) Less(i, j int) bool {
	if ss.prec(i) != ss.prec(j) {
		return ss.prec(i) < ss.prec(j)
	}

	return ss.name(i) < ss.name(j)
}

func (ss sortableSource) Len() int {
	return len(ss)
}

func (ss sortableSource) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}
