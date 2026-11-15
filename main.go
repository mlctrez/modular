package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mlctrez/modular/internal/impl"
)

const usage = `Usage: modular [-q] [bump | <version>] "<commit message>"

Commits the staged files, optionally tags, and pushes to origin.

  bump       increment the revision of the latest semver tag
  <version>  create the annotated tag v<version>, e.g. 1.0.0
  -q         suppress success output (errors still print)

Example:
  git add main.go
  modular bump "describe the change"`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes modular and returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("modular", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	quiet := fs.Bool("q", false, "suppress success output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stderr, usage)
			return 0
		}
		fmt.Fprintf(stderr, "%v\n\n%s\n", err, usage)
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, usage)
		return 1
	}
	out := stdout
	if *quiet {
		out = io.Discard
	}
	if err := impl.Run(fs.Args(), impl.Options{Out: out}); err != nil {
		fmt.Fprintln(stderr, err)
		var ue *impl.UsageError
		if errors.As(err, &ue) {
			fmt.Fprintln(stderr, "run modular -h for usage")
		}
		return 1
	}
	return 0
}
