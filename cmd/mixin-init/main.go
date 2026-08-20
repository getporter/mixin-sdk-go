// Command mixin-init scaffolds a new Porter mixin project against
// mixin-sdk-go: go.mod, a buildable main.go stub implementing sdk.Mixin, a
// starter CI workflow, and LICENSE. It replaces cloning and renaming
// getporter/skeletor — there's no cmd/ or pkg/ tree to copy, and no
// placeholder strings to find-and-replace.
//
// Usage:
//
//	go run github.com/getporter/mixin-sdk-go/cmd/mixin-init@latest hazmat
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("mixin-init", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: mixin-init <name> [flags]")
		fmt.Fprintln(fs.Output(), "\nScaffolds a new Porter mixin named <name> against mixin-sdk-go.")
		fs.PrintDefaults()
	}

	var opts Options
	fs.StringVar(&opts.Module, "module", "", "Go module path for the new mixin (default: <name>)")
	fs.StringVar(&opts.Dir, "dir", "", "directory to scaffold into (default: ./<name>)")
	fs.StringVar(&opts.Author, "author", "", "author name embedded in the mixin's version info")
	fs.StringVar(&opts.SDKVersion, "sdk-version", "", "mixin-sdk-go version to depend on (default: latest)")
	fs.StringVar(&opts.SDKPath, "sdk-path", "", "use a local mixin-sdk-go checkout instead of fetching -sdk-version")
	fs.BoolVar(&opts.Force, "force", false, "scaffold into a directory that already exists and is non-empty")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one argument, the mixin name, got %d", fs.NArg())
	}
	opts.Name = fs.Arg(0)

	if err := Generate(opts); err != nil {
		return err
	}

	fmt.Printf("Scaffolded %s in %s\n", opts.Name, opts.Dir)
	return nil
}
