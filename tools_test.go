package tools

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestAddFile(t *testing.T) {
	content := `package main
  func main() {
	  println("nope")
  }`

	tools := New()
	err := tools.Add("main.go", []byte(content))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = tools.Add("main.go", []byte(content))
	if !errors.Is(err, fs.ErrExist) {
		t.Errorf("Expected error %v got %v", fs.ErrExist, err)
	}

	content = `package bar
  func foo() {
	  println("foo")
	}`

	err = tools.Add("foo.go", []byte(content))
	if !errors.Is(err, ErrPackageMismatch) {
		t.Errorf("Expected error %v got %v", ErrPackageMismatch, err)
	}
}

func TestAddDir(t *testing.T) {
	dir := os.DirFS("testdata/add_dir_test")
	tools := New()
	want, _ := fs.Glob(dir, "*.go")
	sort.Strings(want)
	err := tools.AddDir(dir, ".")
	if err == nil {
		got := []string{}
		for name := range tools.sources {
			got = append(got, name)
		}
		sort.Strings(got)
		if strings.Join(want, "") != strings.Join(got, "") {
			t.Errorf("Wanted files %v got %v", want, got)
		}
	} else {
		t.Errorf("Unexpected error: %v", err)
	}
}

type writeFile string

func (wf writeFile) WriteFile(name string, content []byte) error {
	return ioutil.WriteFile(filepath.Join(string(wf), name), content, 0644)
}

func TestWriteFiles(t *testing.T) {
	dir := os.DirFS("testdata/write_files_test/input")
	wants := make(map[string]string)
	wantChanged := []string{}
	sort.Strings(wantChanged)
	files, _ := filepath.Glob("testdata/write_files_test/want/*.go")
	for _, file := range files {
		want, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatalf("Failed to read test data %q: %v", file, err)
		}
		wants[filepath.Base(file)] = string(want)
		wantChanged = append(wantChanged, filepath.Base(file))
	}

	outDir, err := os.MkdirTemp("", "example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(outDir)

	tools := New()
	err = tools.AddDir(dir, ".")
	if err == nil {
		err = tools.OrganizeAll()
	}

	if err == nil {
		err = tools.WriteFiles(writeFile(outDir).WriteFile)
	}

	if err == nil {
		gots := make(map[string]string)
		gotChanged := tools.ChangedFiles()
		sort.Strings(gotChanged)
		files, _ := filepath.Glob(filepath.Join(outDir, "*.go"))
		for _, file := range files {
			got, err := ioutil.ReadFile(file)
			if err != nil {
				t.Fatalf("Failed to read output data %q: %v", file, err)
			}
			gots[filepath.Base(file)] = string(got)
		}

		if strings.Join(wantChanged, "") != strings.Join(gotChanged, "") {
			t.Errorf("Wanted list of changed files: %v got %v", wantChanged, gotChanged)
		}

		for filename, want := range wants {
			if want != gots[filename] {
				t.Errorf("%v: wanted\n%s\ngot\n%s", filename, want, gots[filename])
			}
		}
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGoTools(t *testing.T) {
	testFuncs := map[string]func(string, []byte) ([]byte, error){
		"SeparateValues": SeparateValues,
		"Organize":       Organize,
	}

	readFile := func(filename string) []byte {
		output, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatalf("Failed to read file %s: %v", filename, err)
		}
		return output
	}

	inputs, _ := filepath.Glob("testdata/tools_test/*.input")
	sort.Strings(inputs)

	wants, _ := filepath.Glob("testdata/tools_test/*.want")
	sort.Strings(wants)

	if len(inputs) != len(wants) {
		t.Fatalf("Wanted %d outputs got %d", len(inputs), len(wants))
	}

	test := func(testname, inputfile, wantfile string) func(t *testing.T) {
		return func(t *testing.T) {
			want := string(readFile(wantfile))
			names := strings.Split(testname, "_")
			if f, found := testFuncs[names[0]]; found {
				gotBytes, err := f(inputfile, readFile(inputfile))
				got := string(gotBytes)
				if err != nil {
					t.Errorf("Failed to execute %s: %v", names[0], err)
					t.Errorf("Output:\n%s\n", got)
				} else {
					if want != got {
						t.Errorf("Wanted:\n%s\n\nGot:\n%s\n", want, got)
					}
				}
			} else {
				t.Errorf("Unknown test function %q", names[0])
			}
		}
	}

	for i, inputfile := range inputs {
		testname := strings.TrimSuffix(filepath.Base(inputfile), filepath.Ext(inputfile))
		t.Run(testname, test(testname, inputfile, wants[i]))
	}

}
