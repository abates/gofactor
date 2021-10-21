package tools

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path"
	"path/filepath"

	"golang.org/x/tools/go/loader"
)

var (
	ErrDeclNotFound    = errors.New("Declaration not found")
	ErrPackageMismatch = errors.New("Different package declarations found")
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

type Change struct {
	Filename string
	Orig     []byte
	Current  []byte
}

type Tools struct {
	changed map[string]Change
	files   map[string]*ast.File
	sources map[string][]byte
	info    *types.Info
	pkg     *types.Package
	pkgname string
}

func New() *Tools {
	f := &Tools{
		changed: make(map[string]Change),
		files:   make(map[string]*ast.File),
		sources: make(map[string][]byte),
	}
	return f
}

// AddDir will add all the *.go files in the given directory
// to the local file set.  Each file will be parsed.  If any
// parsing errors occur processing stops and the associated
// error is returned
func (f *Tools) AddDir(fsys fs.FS, dir string) error {
	files, err := fs.Glob(fsys, path.Join(filepath.ToSlash(dir), "*.go"))
	if err == nil {
		err = f.AddFiles(fsys, files...)
	}
	return err
}

// AddFiles will add all the files supplied
func (f *Tools) AddFiles(fsys fs.FS, files ...string) (err error) {
	for i := 0; i < len(files) && err == nil; i++ {
		err = f.AddFile(fsys, files[i])
	}
	err = f.Load()
	return
}

// AddFile will read the file and add it to the local fileset
func (f *Tools) AddFile(fsys fs.FS, filename string) error {
	src, err := fs.ReadFile(fsys, filename)
	if err == nil {
		err = f.Add(filename, src)
	}
	return err
}

// Add will attempt to add the given source to the file set
// and parse the content. If the filename already exists in the
// file set then fs.ErrExist is returned. Otherwise any parse
// errors are returned
func (f *Tools) Add(filename string, src []byte) error {
	if _, found := f.files[filename]; found {
		return fs.ErrExist
	}
	return f.addFile(filename, src)
}

func (f *Tools) addFile(filename string, src []byte) error {
	orig, found := f.sources[filename]
	if found && !bytes.Equal(orig, src) {
		if c, found := f.changed[filename]; found {
			c.Current = src
		} else {
			f.changed[filename] = Change{Filename: filename, Orig: orig, Current: src}
		}
	}

	err := f.parse(filename, src)
	if err == nil {
		astFile := f.files[filename]
		pkgname := astFile.Name.Name
		if f.pkgname == "" {
			f.pkgname = pkgname
		} else if f.pkgname != pkgname {
			return fmt.Errorf("%w: %s and %s", ErrPackageMismatch, f.pkgname, pkgname)
		}
	}
	return err
}

// Load will send all the source files through the go/loader package
// for processing and type resolution.  This must be called prior
// to any type lookups.  If any source files change due to processing
// then Load() must be called again before type lookups
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

func (f *Tools) OrganizeAll() (err error) {
	filenames := []string{}
	for filename := range f.files {
		filenames = append(filenames, filename)
	}
	return f.OrganizeFiles(filenames...)
}

func (f *Tools) OrganizeFiles(files ...string) (err error) {
	for _, filename := range files {
		_, err = f.Organize(filename)
		if err != nil {
			break
		}
	}
	return
}

func (f *Tools) Organize(filename string) (output []byte, err error) {
	output, err = f.SeparateValues(filename)
	if err != nil {
		return
	}

	o := &organizer{
		formatter: &formatter{
			Tools:  f,
			file:   f.files[filename],
			src:    f.sources[filename],
			writer: bytes.NewBuffer(nil),
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

// SeparateValues analyzes the file and will group const and var
// blocks by type.  SeparateValues will only manipulate declarations
// that are within parenthesized blocks, ie:
//   const (
//    Int1 int = iota
//    Int2
//    Int3
//    Int4
//
//    Str1 string = "string1"
//    Str2        = "string2"
//    Str3        = "string3"
//    Str4        = "string4"
//   )
//
// Becomes:
//
//   const (
//     Int1 int = iota
//     Int2
//     Int3
//     Int4
//   )
//
//   const (
//     Str1 string = "string1"
//     Str2        = "string2"
//     Str3        = "string3"
//     Str4        = "string4"
//   )
func (f *Tools) SeparateValues(filename string) ([]byte, error) {
	file, found := f.files[filename]
	if !found {
		return nil, fmt.Errorf("%q: %w", filename, fs.ErrNotExist)
	}

	vf := &valueCleaner{
		formatter: &formatter{
			Tools:  f,
			file:   file,
			src:    f.sources[filename],
			writer: bytes.NewBuffer(nil),
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

type FileWriter func(filename string, data []byte) error

// ChangedFiles returns a list of filenames whose content has
// changed during the course of processing
func (f *Tools) ChangedFiles() (changed []string) {
	for filename := range f.changed {
		changed = append(changed, filename)
	}
	return changed
}

func (f *Tools) Changed() (changed []Change) {
	for _, change := range f.changed {
		changed = append(changed, change)
	}
	return changed
}

// WriteFiles will write all the changed files using the supplied
// FileWriter.  If an error is encountered processing stops and
// the error is returned
func (f *Tools) WriteFiles(writer FileWriter) (err error) {
	for filename := range f.changed {
		err = writer(filename, f.sources[filename])
		if err != nil {
			break
		}
	}
	return
}
