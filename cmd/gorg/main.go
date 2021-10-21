package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tools "github.com/abates/gotools"
	"github.com/abates/gotools/internal/diff"
)

var (
	// main operation modes
	list   = flag.Bool("l", false, "list files whose formatting differs from gofmt's")
	write  = flag.Bool("w", false, "write result to (source) file instead of stdout")
	doDiff = flag.Bool("d", false, "display diffs instead of rewriting files")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		return
	}

	added := map[string]bool{}
	files := []string{}
	tools := tools.New()

	for _, arg := range args {
		if fi, err := os.Stat(arg); err == nil {
			if fi.IsDir() {
				f, _ := filepath.Glob(filepath.Join(arg, "*.go"))
				files = append(files, f...)
			} else {
				files = append(files, arg)
				arg = filepath.Dir(arg)
			}

			if _, found := added[arg]; !found {
				dir := os.DirFS(arg)
				err = tools.AddDir(dir, ".")
				added[arg] = true
			}

			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to add %q: %v\n", arg, err)
				os.Exit(-1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Failed to stat %s: %v\n", arg, err)
		}
	}

	err := tools.OrganizeFiles(files...)
	if err == nil {
		if *list {
			fmt.Fprintln(os.Stderr, strings.Join(tools.ChangedFiles(), "\n"))
		} else if *doDiff {
			for _, change := range tools.Changed() {
				if d, err := diff.Diff("", change.Orig, change.Current); err != nil {
					fmt.Fprintf(os.Stderr, "Couldn't perform diff on %s: %v", change.Filename, err)
				} else if len(d) > 0 {
					fmt.Fprintf(os.Stdout, "%s\n%s", change.Filename, string(d))
				}
			}
		} else if *write {
			//err = tools.WriteFiles()
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to organize: %v", err)
		os.Exit(-1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: gofmt [flags] [path ...]\n")
	flag.PrintDefaults()
}
