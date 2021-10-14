package gofactor

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/loader"
)

type formatter struct {
	files       map[string]*ast.File
	fset        *token.FileSet
	src         map[string][]byte
	info        *types.Info
	pkg         *types.Package
	pkgName     string
	writer      *bytes.Buffer
	currentFile string

	config loader.Config
}

func newFormatter() *formatter {
	f := &formatter{
		fset: token.NewFileSet(),

		files:  make(map[string]*ast.File),
		src:    make(map[string][]byte),
		writer: &bytes.Buffer{},
	}
	f.config.Fset = f.fset
	return f
}

func (f *formatter) typeCheck(file *ast.File) error {
	f.config.CreateFromFiles(file.Name.Name, file)
	program, err := f.config.Load()
	if err == nil {
		pkg := program.Package(file.Name.Name)
		f.info = &pkg.Info
		f.pkg = pkg.Pkg
	}
	return err
}

func (f *formatter) typeCheck1(filename string) (err error) {
	f.info = &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	var conf types.Config
	conf.Importer = importer.Default()

	f.pkg, err = conf.Check(filename, f.fset, []*ast.File{f.files[filename]}, f.info)
	return err
}

func (f *formatter) parse(filename string, src []byte) error {
	astFile, err := parser.ParseFile(f.fset, "", src, parser.ParseComments)
	if err == nil {
		f.files[filename] = astFile
		f.src[filename] = src
		f.currentFile = filename
		err = f.typeCheck(astFile)
	}
	return err
}

func (f *formatter) readline(file string, start token.Pos) string {
	i := strings.Index(string(f.src[file][start-1:]), "\n")
	if i >= 0 {
		return string(f.src[file][start-1 : i+1])
	}
	return ""
}

func (f *formatter) write(str string) {
	//println("Writing:", str)
	f.writer.Write([]byte(str))
}

func (f *formatter) writePos(start, end token.Pos) {
	//println("Writing:", string(f.src[start-1:end-1]))
	f.writer.Write(f.src[f.currentFile][start-1 : end-1])
}

func (f *formatter) typStr(expr ast.Expr) (str string) {
	if expr == nil {
		return ""
	}

	name := types.TypeString(f.info.TypeOf(expr), types.RelativeTo(f.pkg))
	return name
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
