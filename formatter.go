package gofactor

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
	"strings"

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

type fileFormatter struct {
	*Formatter

	file   *ast.File
	src    []byte
	writer *bytes.Buffer
}

func (f *fileFormatter) readline(start token.Pos) string {
	i := strings.Index(string(f.src[start-1:]), "\n")
	if i >= 0 {
		return string(f.src[start-1 : int(start)+i])
	}
	return ""
}

func (f *fileFormatter) write(str string) {
	//println("Writing:", str)
	f.writer.Write([]byte(str))
}

func (f *fileFormatter) writePos(start, end token.Pos) {
	//println("Writing:", string(f.src[start-1:end-1]))
	f.writer.Write(f.src[start-1 : end-1])
}

type Formatter struct {
	path    string
	files   map[string]*ast.File
	sources map[string][]byte
	info    *types.Info
	pkg     *types.Package
	pkgname string
}

func NewFormatter() *Formatter {
	f := &Formatter{
		files:   make(map[string]*ast.File),
		sources: make(map[string][]byte),
	}
	return f
}

func (f *Formatter) AddDir(path string) error {
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

func (f *Formatter) AddFile(filename string, src []byte) error {
	if _, found := f.files[filename]; found {
		return fs.ErrExist
	}
	return f.addFile(filename, src)
}

func (f *Formatter) addFile(filename string, src []byte) error {
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

func (f *Formatter) Load() error {
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

func (f *Formatter) Organize(filename string) (output []byte, err error) {
	output, err = f.SeparateValues(filename)
	if err != nil {
		return
	}

	o := &organizer{
		fileFormatter: &fileFormatter{
			Formatter: f,
			file:      f.files[filename],
			src:       f.sources[filename],
			writer:    bytes.NewBuffer(nil),
		},
		typIndex: make(map[string]*typSrc),
	}

	err = o.organize()
	if err == nil {
		output, err = f.setSrc(filename, o.writer.Bytes())
	}
	return
}

func (f *Formatter) parse(filename string, src []byte) error {
	astFile, err := parser.ParseFile(token.NewFileSet(), filename, src, parser.ParseComments)
	if err == nil {
		f.files[filename] = astFile
		f.sources[filename] = src
	}
	return err
}

func (f *Formatter) SeparateValues(filename string) ([]byte, error) {
	file, found := f.files[filename]
	if !found {
		return nil, fs.ErrNotExist
	}

	vf := &valueCleaner{
		fileFormatter: &fileFormatter{
			Formatter: f,
			file:      file,
			src:       f.sources[filename],
			writer:    bytes.NewBuffer(nil),
		},
	}

	vf.separateValDecls()
	return f.setSrc(filename, vf.writer.Bytes())
}

func (f *Formatter) setSrc(filename string, src []byte) (output []byte, err error) {
	if output, err = format.Source(src); err == nil {
		if err = f.addFile(filename, output); err == nil {
			err = f.Load()
		}
	} else {
		output = src
	}
	return output, err
}

func (f *Formatter) typStr(expr ast.Expr) (str string) {
	if expr == nil {
		return ""
	}

	name := types.TypeString(f.info.TypeOf(expr), types.RelativeTo(f.pkg))
	return name
}
