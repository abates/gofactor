package gofactor

import (
	"io/ioutil"
	"path/filepath"
	"sort"
	"testing"
)

func TestSeparateValues(t *testing.T) {
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

	inputs := getFiles(filepath.Glob("testdata/value_separate_*.input"))
	sort.Strings(inputs)

	wants := getFiles(filepath.Glob("testdata/value_separate_*.want"))
	sort.Strings(wants)

	if len(inputs) != len(wants) {
		t.Fatalf("Wanted %d outputs got %d", len(inputs), len(wants))
	}

	test := func(inputfile, wantfile string) func(t *testing.T) {
		return func(t *testing.T) {
			want := string(readFile(wantfile))
			gotBytes, err := SeparateValues(readFile(inputfile))
			if err != nil {
				t.Fatalf("Failed to separate values: %v", err)
			}

			got := string(gotBytes)
			if want != got {
				t.Errorf("Wanted:\n%s\n\nGot:\n%s\n", want, got)
			}
		}
	}

	for i, inputfile := range inputs {
		t.Run(filepath.Base(inputfile), test(inputfile, wants[i]))
	}

}
