package tools

import (
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestGoFactor(t *testing.T) {
	testFuncs := map[string]func(string, []byte) ([]byte, error){
		"SeparateValues": SeparateValues,
		"Organize":       Organize,
	}

	getFiles := func(files []string, err error) []string {
		if err != nil {
			t.Fatalf("Failed to get files: %v", err)
		}
		return files
	}

	readFile := func(filename string) []byte {
		output, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatalf("Failed to read file %s: %v", filename, err)
		}
		return output
	}

	inputs := getFiles(filepath.Glob("testdata/*.input"))
	sort.Strings(inputs)

	wants := getFiles(filepath.Glob("testdata/*.want"))
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
