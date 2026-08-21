package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot locates this module's root (the parent of cmd/mixin-init), so
// tests can point -sdk-path at it instead of depending on mixin-sdk-go
// being published.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(wd)) // cmd/mixin-init -> cmd -> root
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("could not locate module root from %s: %v", wd, err)
	}
	return root
}

func TestGenerate_ScaffoldsBuildableMixin(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	dir := filepath.Join(t.TempDir(), "hazmat")
	opts := Options{
		Name:    "hazmat",
		Author:  "Test Author",
		Dir:     dir,
		SDKPath: repoRoot(t),
	}

	if err := Generate(opts); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	for _, f := range []string{"go.mod", "go.sum", "main.go", "LICENSE", filepath.Join(".github", "workflows", "hazmat.yml")} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected %s to exist: %v", f, err)
		}
	}

	build := exec.Command("go", "build", "-o", "hazmat", ".")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("generated mixin does not build: %v\n%s", err, out)
	}

	vet := exec.Command("go", "vet", "./...")
	vet.Dir = dir
	if out, err := vet.CombinedOutput(); err != nil {
		t.Fatalf("go vet failed on generated mixin: %v\n%s", err, out)
	}

	version := exec.Command("./hazmat", "version")
	version.Dir = dir
	out, err := version.CombinedOutput()
	if err != nil {
		t.Fatalf("running generated mixin: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "hazmat dev by Test Author" {
		t.Errorf("version output = %q, want %q", got, "hazmat dev by Test Author")
	}
}

func TestGenerate_RejectsInvalidName(t *testing.T) {
	err := Generate(Options{Name: "Not Valid!", Dir: t.TempDir()})
	if err == nil {
		t.Fatal("expected an error for an invalid name")
	}
}

func TestGenerate_RefusesNonEmptyDirWithoutForce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Generate(Options{Name: "hazmat", Dir: dir})
	if err == nil {
		t.Fatal("expected an error for a non-empty target directory")
	}
}

func TestGenerate_ForceRescaffoldsExistingProject(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	dir := filepath.Join(t.TempDir(), "hazmat")
	opts := Options{Name: "hazmat", Dir: dir, SDKPath: repoRoot(t)}

	if err := Generate(opts); err != nil {
		t.Fatalf("first Generate: %v", err)
	}

	opts.Force = true
	if err := Generate(opts); err != nil {
		t.Fatalf("second Generate with -force: %v", err)
	}
}
