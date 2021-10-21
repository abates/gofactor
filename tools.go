package tools

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"io/ioutil"
	"path/filepath"
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
	case *ast.BasicLit:
		str = t.Kind.String()
	case *ast.Ident:
		str = t.Name
	case *ast.StarExpr:
		str = typStr(t.X)
	case *ast.CallExpr:
		str = typStr(t.Fun)
	default:
		println(fmt.Sprintf("TYPE: %T", expr))
		str = fmt.Sprintf("%s", expr)
	}
	return str
}

type Change struct {
	Filename string
	Orig     []byte
	Current  []byte
}

type FileWriter func(filename string, data []byte) error

type Tools struct {
	changed map[string]Change
	files   map[string]*ast.File
	sources map[string][]byte
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

// AddDir will add all the *.go files in the given directory
// to the local file set.  Each file will be parsed.  If any
// parsing errors occur processing stops and the associated
// error is returned
func (f *Tools) AddDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err == nil {
		err = f.AddFiles(files...)
	}
	return err
}

// AddFile will read the file and add it to the local fileset
func (f *Tools) AddFile(filename string) error {
	src, err := ioutil.ReadFile(filename)
	if err == nil {
		err = f.Add(filename, src)
	}
	return err
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

// AddFiles will add all the files supplied
func (f *Tools) AddFiles(files ...string) (err error) {
	for i := 0; i < len(files) && err == nil; i++ {
		err = f.AddFile(files[i])
	}
	return
}

func (f *Tools) Changed() (changed []Change) {
	for _, change := range f.changed {
		changed = append(changed, change)
	}
	return changed
}

// ChangedFiles returns a list of filenames whose content has
// changed during the course of processing
func (f *Tools) ChangedFiles() (changed []string) {
	for filename := range f.changed {
		changed = append(changed, filename)
	}
	return changed
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
		err = f.addFile(filename, output)
	} else {
		output = src
	}
	return output, err
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
