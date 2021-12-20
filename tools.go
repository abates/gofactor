package tools

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"io/ioutil"
	"path/filepath"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

var (
	ErrDeclNotFound    = errors.New("Declaration not found")
	ErrPackageMismatch = errors.New("Different package declarations found")
)

func typStr(expr interface{}) (str string) {
	if expr == nil {
		return ""
	}

	switch t := expr.(type) {
	case *dst.BasicLit:
		str = t.Kind.String()
	case *dst.Ident:
		str = t.Name
	case *dst.StarExpr:
		str = typStr(t.X)
	case *dst.CallExpr:
		str = typStr(t.Fun)
	}
	return str
}

type FileWriter func(filename string, data []byte) error

type Change struct {
	Filename string
	Orig     []byte
	Current  []byte
}

type Tools struct {
	changed map[string]Change
	dfiles  map[string]*dst.File
	pkgname string
}

func New() *Tools {
	f := &Tools{
		changed: make(map[string]Change),
		dfiles:  make(map[string]*dst.File),
	}
	return f
}

// Add will attempt to add the given source to the file set
// and parse the content. If the filename already exists in the
// file set then fs.ErrExist is returned. Otherwise any parse
// errors are returned
func (f *Tools) Add(filename string, src []byte) error {
	if _, found := f.dfiles[filename]; found {
		return fs.ErrExist
	}

	err := f.parse(filename, src)
	if err == nil {
		dstFile := f.dfiles[filename]
		pkgname := dstFile.Name.Name
		if f.pkgname == "" {
			f.pkgname = pkgname
		} else if f.pkgname != pkgname {
			return fmt.Errorf("%w: %s and %s", ErrPackageMismatch, f.pkgname, pkgname)
		}
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

// AddFiles will add all the files supplied
func (f *Tools) AddFiles(files ...string) (err error) {
	for i := 0; i < len(files) && err == nil; i++ {
		err = f.AddFile(files[i])
	}
	return
}

// Changes returns a slice of Change structs containing information
// about each file that was changed
func (f *Tools) Changes() (changed []Change) {
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

func (f *Tools) format(filename string, cb func()) ([]byte, error) {
	buf := &bytes.Buffer{}
	decorator.Fprint(buf, f.dfiles[filename])
	start := buf.Bytes()

	cb()

	output := &bytes.Buffer{}
	decorator.Fprint(output, f.dfiles[filename])
	end, err := format.Source(output.Bytes())
	if err != nil {
		end = output.Bytes()
	}

	if !bytes.Equal(start, end) {
		change, found := f.changed[filename]
		if !found {
			change = Change{
				Filename: filename,
				Orig:     start,
			}
		}
		change.Current = end
		f.changed[filename] = change
	}
	return end, err
}

func (f *Tools) Organize(filename string) (output []byte, err error) {
	_, err = f.SeparateValues(filename)
	if err != nil {
		return
	}

	output, err = f.format(filename, func() {
		organizer := organizer{
			file: f.dfiles[filename],
		}

		f.dfiles[filename] = organizer.organize()
	})

	return output, err
}

func (f *Tools) OrganizeAll() (err error) {
	filenames := []string{}
	for filename := range f.dfiles {
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
	dstFile, err := decorator.Parse(src)
	if err == nil {
		f.dfiles[filename] = dstFile
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
	dfile, found := f.dfiles[filename]
	if !found {
		return nil, fmt.Errorf("%q: %w", filename, fs.ErrNotExist)
	}

	output, err := f.format(filename, func() {
		vf := &valueCleaner{
			file: dfile,
		}

		f.dfiles[filename] = vf.separateValDecls()
	})
	return output, err
}

// WriteFiles will write all the changed files using the supplied
// FileWriter.  If an error is encountered processing stops and
// the error is returned
func (f *Tools) WriteFiles(writer FileWriter) (err error) {
	for filename := range f.changed {
		buf := &bytes.Buffer{}
		err = decorator.Fprint(buf, f.dfiles[filename])
		if err == nil {
			err = writer(filename, buf.Bytes())
		}

		if err != nil {
			break
		}
	}
	return
}
