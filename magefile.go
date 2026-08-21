//go:build mage

// Package main is this repo's own dev magefile — not to be confused with
// the mage/ package, which is what mixin authors import into *their*
// magefile.go. Requires mage: `go install github.com/magefile/mage@latest`.
// Run `mage -l` for the target list.
package main

import (
	"os"
	"os/exec"

	"get.porter.sh/magefiles/git"
)

// Build compiles every package in the module.
func Build() error {
	return run("go", "build", "./...")
}

// Vet runs go vet across the module.
func Vet() error {
	return run("go", "vet", "./...")
}

// Test runs the module's test suite.
func Test() error {
	return run("go", "test", "./...")
}

// Lint runs golangci-lint across the module. Requires golangci-lint to
// already be installed; CI installs it via golangci-lint-action.
func Lint() error {
	return run("golangci-lint", "run", "./...")
}

// SetupDCO configures your git repository to automatically sign off your
// commits to comply with this project's Developer Certificate of Origin.
func SetupDCO() error {
	return git.SetupDCO()
}

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}
