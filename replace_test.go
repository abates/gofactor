package tools

import (
	"go/format"
	"testing"
)

func TestReplace(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		replaceName    string
		replaceType    string
		replaceContent string
		want           string
	}{
		{
			name: "simple",
			input: `package foo

			func myfunc() { println() }
			func main() {
				myfunc()
			}
			`,
			replaceName:    "myfunc",
			replaceContent: "func myfunc(arg int) {\nprintln(arg)\n}\n",
			want: `package foo

			func myfunc(arg int) {
				println(arg)
			}

      func main() {
        myfunc()
      }`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := Replace("test.go", []byte(test.input)).Func(test.replaceName, test.replaceType, []byte(test.replaceContent))
			if r.Err == nil {
				wantBytes, err := format.Source([]byte(test.want))
				test.want = string(wantBytes)
				if err != nil {
					t.Fatalf("Failed to format wanted source: %v", err)
				}
				got := string(r.Content())
				if test.want != got {
					t.Errorf("Wanted\n%s\nGot:\n%s", test.want, got)
				}
			} else {
				t.Errorf("Unexpected error: %v", r.Err)
			}
		})
	}
}
