package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// Options controls what mixin-init scaffolds.
type Options struct {
	// Name identifies the mixin: its directory (unless Dir is set), its
	// module path (unless Module is set), and its VersionInfo.Name.
	Name string

	// Module is the Go module path for the new mixin. Defaults to Name,
	// which is a valid (if unpublishable) module path — good enough to
	// build and test locally; authors rename it with `go mod edit
	// -module` once they know where the mixin will be published.
	Module string

	// Dir is the directory to scaffold into. Defaults to "./"+Name.
	Dir string

	// Author is embedded in the generated Mixin.Version().
	Author string

	// SDKVersion is the mixin-sdk-go version requirement passed to `go
	// get`. Defaults to "latest".
	SDKVersion string

	// SDKPath, if set, replaces the mixin-sdk-go dependency with a local
	// checkout instead of fetching SDKVersion — for developing against an
	// unpublished or in-progress SDK checkout.
	SDKPath string

	// Force allows scaffolding into a directory that already exists and
	// is non-empty.
	Force bool
}

var nameRE = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func (o *Options) applyDefaults() error {
	if !nameRE.MatchString(o.Name) {
		return fmt.Errorf("invalid mixin name %q: must start with a lowercase letter and contain only lowercase letters, digits, and hyphens", o.Name)
	}
	if o.Module == "" {
		o.Module = o.Name
	}
	if o.Dir == "" {
		o.Dir = "./" + o.Name
	}
	if o.SDKVersion == "" {
		o.SDKVersion = "latest"
	}
	return nil
}

// Generate scaffolds a new mixin project: go.mod, a buildable main.go
// stub, a starter CI workflow, and LICENSE. It does not write a cmd/ or
// pkg/ tree — the mixin's logic is implemented against the sdk.Mixin
// interface, not copied from a template.
func Generate(opts Options) error {
	if err := opts.applyDefaults(); err != nil {
		return err
	}

	if err := prepareDir(opts.Dir, opts.Force); err != nil {
		return err
	}

	data := templateData{Name: opts.Name, Author: opts.Author}

	mainGo, err := renderTemplate("main.go.tmpl", data)
	if err != nil {
		return fmt.Errorf("could not render main.go: %w", err)
	}
	if err := os.WriteFile(filepath.Join(opts.Dir, "main.go"), mainGo, 0644); err != nil {
		return fmt.Errorf("could not write main.go: %w", err)
	}

	ciYML, err := renderTemplate("ci.yml.tmpl", data)
	if err != nil {
		return fmt.Errorf("could not render CI workflow: %w", err)
	}
	workflowDir := filepath.Join(opts.Dir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return fmt.Errorf("could not create %s: %w", workflowDir, err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, opts.Name+".yml"), ciYML, 0644); err != nil {
		return fmt.Errorf("could not write CI workflow: %w", err)
	}

	license, err := readLicense()
	if err != nil {
		return fmt.Errorf("could not read LICENSE template: %w", err)
	}
	if err := os.WriteFile(filepath.Join(opts.Dir, "LICENSE"), license, 0644); err != nil {
		return fmt.Errorf("could not write LICENSE: %w", err)
	}

	// go mod init refuses to run if go.mod is already there, which -force
	// deliberately allows (re-scaffolding into an existing directory).
	if opts.Force {
		if err := os.Remove(filepath.Join(opts.Dir, "go.mod")); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("could not remove existing go.mod: %w", err)
		}
	}
	if err := runGo(opts.Dir, "mod", "init", opts.Module); err != nil {
		return fmt.Errorf("could not initialize go.mod: %w", err)
	}

	if opts.SDKPath != "" {
		absSDKPath, err := filepath.Abs(opts.SDKPath)
		if err != nil {
			return fmt.Errorf("could not resolve sdk path %q: %w", opts.SDKPath, err)
		}
		if err := runGo(opts.Dir, "mod", "edit",
			"-require=github.com/getporter/mixin-sdk-go@v0.0.0",
			"-replace=github.com/getporter/mixin-sdk-go="+absSDKPath); err != nil {
			return fmt.Errorf("could not point go.mod at local sdk checkout: %w", err)
		}
	} else if err := runGo(opts.Dir, "get", "github.com/getporter/mixin-sdk-go@"+opts.SDKVersion); err != nil {
		return fmt.Errorf("could not add mixin-sdk-go dependency (add it yourself with `go get github.com/getporter/mixin-sdk-go` in %s): %w", opts.Dir, err)
	}

	if err := runGo(opts.Dir, "mod", "tidy"); err != nil {
		return fmt.Errorf("could not tidy go.mod (run `go mod tidy` yourself in %s): %w", opts.Dir, err)
	}

	return nil
}

func prepareDir(dir string, force bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0755)
		}
		return fmt.Errorf("could not inspect %s: %w", dir, err)
	}
	if len(entries) > 0 && !force {
		return fmt.Errorf("%s already exists and is not empty (use -force to scaffold into it anyway)", dir)
	}
	return nil
}

func runGo(dir string, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go %s: %w\n%s", args[0], err, out.String())
	}
	return nil
}
