package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/abates/gofactor"
	"github.com/abates/gofactor/internal/diff"
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

func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		return
	}

	for _, arg := range args {
		input, err := ioutil.ReadFile(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read file %q: %v\n", arg, err)
			os.Exit(1)
		}

		output, err := gofactor.Organize(arg, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to organize %q: %v\n", arg, err)
			os.Exit(1)
		}

		if *write {
			err = ioutil.WriteFile(arg, output, 0)
		} else if *doDiff || *list {
			if d, err := diff.Diff("", input, output); err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't perform diff on %s: %v", arg, err)
			} else if len(d) > 0 {
				if *list {
					fmt.Fprintf(os.Stdout, "%s\n", arg)
				} else {
					fmt.Fprintf(os.Stdout, "%s\n%s", arg, string(d))
				}
			}
		} else {
			fmt.Fprintln(os.Stdout, string(output))
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: gofmt [flags] [path ...]\n")
	flag.PrintDefaults()
}
