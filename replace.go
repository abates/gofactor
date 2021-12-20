package tools

import (
	"bytes"
	"fmt"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
)

type Replacer struct {
	filename string
	file     *dst.File
	Err      error
}

func Replace(filename string, src []byte) (r *Replacer) {
	r = &Replacer{
		filename: filename,
	}

	r.file, r.Err = decorator.Parse(src)
	return r
}

func (r *Replacer) Content() []byte {
	writer := &bytes.Buffer{}
	decorator.Fprint(writer, r.file)
	return writer.Bytes()
}

func (r *Replacer) Func(name, receiver string, content string) *Replacer {
	if r.Err == nil {
		content = fmt.Sprintf("package %s\n\n%s", r.file.Name.Name, content)
		var tmpfile *dst.File
		tmpfile, r.Err = decorator.Parse(content)
		if r.Err != nil {
			return r
		}
		src := tmpfile.Decls[0]
		src.Decorations().After = dst.EmptyLine

		walk := func(cursor *dstutil.Cursor) bool {
			if fn, ok := cursor.Node().(*dst.FuncDecl); ok {
				if fn.Name.String() == name {
					if (fn.Recv == nil && receiver == "") || (fn.Recv != nil && typStr(fn.Recv.List[0].Type) == receiver) {
						cursor.Replace(src)
						return false
					}
				}
			}
			return true
		}

		r.file = dstutil.Apply(r.file, nil, walk).(*dst.File)
	}
	return r
}
