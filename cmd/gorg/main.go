package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	tools "github.com/abates/gotools"
	"github.com/abates/gotools/internal/diff"
)

var (
	// main operation modes
	list   = flag.Bool("l", false, "list files whose formatting differs from gofmt's")
	write  = flag.Bool("w", false, "write result to (source) file instead of stdout")
	doDiff = flag.Bool("d", false, "display diffs instead of rewriting files")
)

func isDiff(a, b []byte) (bool, error) {
	d, err := diff.Diff("", a, b)
	return len(d) == 0, err
}

func process(tools *tools.Tools, filenames []string) {
	for _, filename := range filenames {
		input, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read %s: %v", filename, err)
		}

		output, err := tools.Organize(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to organize %q: %v\n", filename, err)
			os.Exit(1)
		}

		if *write {
			err = ioutil.WriteFile(filename, output, 0)
		} else if *doDiff || *list {
			if d, err := diff.Diff("", input, output); err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't perform diff on %s: %v", filename, err)
			} else if len(d) > 0 {
				if *list {
					fmt.Fprintf(os.Stdout, "%s\n", filename)
				} else {
					fmt.Fprintf(os.Stdout, "%s\n%s", filename, string(d))
				}
			}
		} else {
			fmt.Fprintln(os.Stdout, string(output))
		}
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		return
	}

	files := []string{}
	tools := tools.New()
	for _, arg := range args {
		if fi, err := os.Stat(arg); err == nil {
			if fi.IsDir() {
				err = tools.AddDir(arg)
				filenames, err := filepath.Glob(filepath.Join(arg, "*.go"))
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to read directory %s: %v", arg, err)
					os.Exit(1)
				}
				files = append(files, filenames...)
			} else {
				err = tools.AddDir(filepath.Dir(arg))
				files = append(files, arg)
			}

			if err != nil && !errors.Is(err, fs.ErrExist) {
				fmt.Fprintf(os.Stderr, "Failed to add directory: %v\n", err)
				os.Exit(-1)
			}
		}
	}
	process(tools, files)
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: gofmt [flags] [path ...]\n")
	flag.PrintDefaults()
}
