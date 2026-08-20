// Package mage provides Build/Test/Publish helpers for a mixin's own
// magefile.go, so it can be a few lines that delegate here instead of
// reimplementing the mixin build/release flow from scratch:
//
//	//go:build mage
//	package main
//
//	import sdkmage "github.com/getporter/mixin-sdk-go/mage"
//
//	var version = "dev" // set by CI, e.g. from `git describe`
//
//	var m = sdkmage.Magefile{Dir: ".", Pkg: ".", Name: "hazmat", Version: version, BinDir: "bin/mixins/hazmat"}
//
//	func Build() error   { return m.Build() }
//	func Test() error    { return m.Test() }
//	func Publish() error { return m.Publish() }
//
// Publish cross-compiles for Platforms and writes each binary plus a
// SHA-256 checksum file to BinDir/dist/ — it does not push anywhere.
// Wire your own CI release step (e.g. `gh release create`) to upload
// dist/'s contents.
package mage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

// Platform is one GOOS/GOARCH pair to cross-compile for.
type Platform struct {
	OS, Arch string
}

// Platforms is Porter's standard mixin release matrix.
var Platforms = []Platform{
	{"linux", "amd64"}, {"linux", "arm64"},
	{"darwin", "amd64"}, {"darwin", "arm64"},
	{"windows", "amd64"}, {"windows", "arm64"},
}

// Magefile holds what Build/Test/Publish need for one mixin.
type Magefile struct {
	// Dir is the mixin's module root, where go.mod lives. Almost always
	// "." — a magefile.go's target functions already run with the
	// process's working directory at the repo root.
	Dir string

	// Pkg is the package to build, relative to Dir. "." for a
	// single-binary mixin — mixin-init's default layout.
	Pkg string

	// Name is the mixin's name: the output binary's filename and the
	// prefix for Publish's per-platform dist filenames.
	Name string

	// Version is embedded into the binary via
	// -ldflags "-X main.version=<Version>", the convention mixin-init's
	// generated main.go expects.
	Version string

	// BinDir is where Build and Publish write output, conventionally
	// "bin/mixins/<name>".
	BinDir string
}

// Build compiles m.Pkg for the current GOOS/GOARCH into
// m.BinDir/m.Name[.exe].
func (m Magefile) Build() error {
	outPath := filepath.Join(m.BinDir, m.Name+exeExt(runtime.GOOS))
	return m.compile(outPath, runtime.GOOS, runtime.GOARCH)
}

// Test runs `go test ./...` for the mixin, streaming output to
// stdout/stderr.
func (m Magefile) Test() error {
	cmd := exec.Command("go", "-C", m.Dir, "test", "./...")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}
	return nil
}

// Publish cross-compiles m.Pkg for every platform in Platforms — in
// parallel, one `go build` per platform — and writes each binary plus a
// <name>-<goos>-<goarch>[.exe].sha256sum checksum file to m.BinDir/dist/.
// It does not push anywhere.
func (m Magefile) Publish() error {
	dir := filepath.Join(m.BinDir, "dist")
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("could not clean %s: %w", dir, err)
	}

	var wg sync.WaitGroup
	errs := make([]error, len(Platforms))
	for i, p := range Platforms {
		wg.Add(1)
		go func(i int, p Platform) {
			defer wg.Done()
			outPath := filepath.Join(dir, fmt.Sprintf("%s-%s-%s%s", m.Name, p.OS, p.Arch, exeExt(p.OS)))
			if err := m.compile(outPath, p.OS, p.Arch); err != nil {
				errs[i] = err
				return
			}
			errs[i] = writeChecksum(outPath)
		}(i, p)
	}
	wg.Wait()

	return errors.Join(errs...)
}

func (m Magefile) compile(outPath, goos, goarch string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("could not create %s: %w", filepath.Dir(outPath), err)
	}

	absOutPath, err := filepath.Abs(outPath)
	if err != nil {
		return fmt.Errorf("could not resolve %s: %w", outPath, err)
	}

	cmd := exec.Command("go", "-C", m.Dir, "build",
		"-ldflags", "-w -X main.version="+m.Version,
		"-o", absOutPath, m.Pkg)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build (%s/%s): %w", goos, goarch, err)
	}
	return nil
}

func exeExt(goos string) string {
	if goos == "windows" {
		return ".exe"
	}
	return ""
}

func writeChecksum(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open %s for checksumming: %w", path, err)
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return fmt.Errorf("could not checksum %s: %w", path, err)
	}

	sum := hex.EncodeToString(hash.Sum(nil)) + "  " + filepath.Base(path) + "\n"
	if err := os.WriteFile(path+".sha256sum", []byte(sum), 0o644); err != nil {
		return fmt.Errorf("could not write checksum for %s: %w", path, err)
	}
	return nil
}
