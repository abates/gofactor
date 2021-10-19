package tools

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"io/ioutil"
	"path/filepath"

	"golang.org/x/tools/go/loader"
)

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

type Tools struct {
	path    string
	files   map[string]*ast.File
	sources map[string][]byte
	info    *types.Info
	pkg     *types.Package
	pkgname string
}

func New() *Tools {
	f := &Tools{
		files:   make(map[string]*ast.File),
		sources: make(map[string][]byte),
	}
	return f
}

func (f *Tools) AddDir(path string) error {
	filenames, err := filepath.Glob(filepath.Join(path, "*.go"))
	if err != nil {
		return err
	}

	for _, filename := range filenames {
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		err = f.AddFile(filename, src)
		if err != nil {
			return err
		}
	}
	return f.Load()
}

func (f *Tools) AddFile(filename string, src []byte) error {
	if _, found := f.files[filename]; found {
		return fs.ErrExist
	}
	return f.addFile(filename, src)
}

func (f *Tools) addFile(filename string, src []byte) error {
	err := f.parse(filename, src)
	if err == nil {
		astFile := f.files[filename]
		pkgname := astFile.Name.Name
		if f.pkgname == "" {
			f.pkgname = pkgname
		} else if f.pkgname != pkgname {
			return fmt.Errorf("Found package %s and %s", f.pkgname, pkgname)
		}
	}
	return err
}

func (f *Tools) Load() error {
	fset := token.NewFileSet()
	files := []*ast.File{}
	for filename, file := range f.files {
		files = append(files, file)
		fset.AddFile(filename, -1, len(f.sources[filename]))
	}

	config := loader.Config{Fset: fset}
	config.CreateFromFiles(f.pkgname, files...)
	program, err := config.Load()
	if err == nil {
		pkg := program.Package(f.pkgname)
		f.info = &pkg.Info
		f.pkg = pkg.Pkg
	}
	return err
}

func (f *Tools) Organize(filename string) (output []byte, err error) {
	output, err = f.SeparateValues(filename)
	if err != nil {
		return
	}

	o := &organizer{
		formatter: &formatter{
			Tools:    f,
			filename: filename,
			file:     f.files[filename],
			src:      f.sources[filename],
			writer:   bytes.NewBuffer(nil),
		},
		typIndex: make(map[string]*typSrc),
	}

	err = o.organize()
	if err == nil {
		output, err = f.setSrc(filename, o.writer.Bytes())
	}
	return
}

func (f *Tools) parse(filename string, src []byte) error {
	astFile, err := parser.ParseFile(token.NewFileSet(), filename, src, parser.ParseComments)
	if err == nil {
		f.files[filename] = astFile
		f.sources[filename] = src
	}
	return err
}

func (f *Tools) SeparateValues(filename string) ([]byte, error) {
	file, found := f.files[filename]
	if !found {
		return nil, fs.ErrNotExist
	}

	vf := &valueCleaner{
		formatter: &formatter{
			Tools:    f,
			filename: filename,
			file:     file,
			src:      f.sources[filename],
			writer:   bytes.NewBuffer(nil),
		},
	}

	vf.separateValDecls()
	return f.setSrc(filename, vf.writer.Bytes())
}

func (f *Tools) setSrc(filename string, src []byte) (output []byte, err error) {
	if output, err = format.Source(src); err == nil {
		if err = f.addFile(filename, output); err == nil {
			err = f.Load()
		}
	} else {
		output = src
	}
	return output, err
}

func (f *Tools) typStr(expr ast.Expr) (str string) {
	if expr == nil {
		return ""
	}

	name := types.TypeString(f.info.TypeOf(expr), types.RelativeTo(f.pkg))
	return name
}
