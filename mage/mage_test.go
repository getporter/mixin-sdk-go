package mage

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot locates this module's root (the parent of mage/), so tests can
// build the real examples/exec mixin instead of a synthetic fixture.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(wd) // mage -> root
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("could not locate module root from %s: %v", wd, err)
	}
	return root
}

func TestMagefile_Build(t *testing.T) {
	binDir := t.TempDir()
	m := Magefile{
		Dir:     repoRoot(t),
		Pkg:     "./examples/exec",
		Name:    "exec",
		Version: "v0.0.0-test",
		BinDir:  binDir,
	}

	if err := m.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}

	binPath := filepath.Join(binDir, "exec"+exeExt(runtime.GOOS))
	out, err := exec.Command(binPath, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("running built binary: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); !strings.Contains(got, "v0.0.0-test") {
		t.Errorf("version output = %q, want it to contain %q", got, "v0.0.0-test")
	}
}

func TestMagefile_Test(t *testing.T) {
	m := Magefile{Dir: filepath.Join(repoRoot(t), "examples", "exec")}
	if err := m.Test(); err != nil {
		t.Fatalf("Test: %v", err)
	}
}

func TestMagefile_Publish(t *testing.T) {
	binDir := t.TempDir()
	m := Magefile{
		Dir:     repoRoot(t),
		Pkg:     "./examples/exec",
		Name:    "exec",
		Version: "v0.0.0-test",
		BinDir:  binDir,
	}

	if err := m.Publish(); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	dist := filepath.Join(binDir, "dist")
	for _, p := range Platforms {
		binName := "exec-" + p.OS + "-" + p.Arch + exeExt(p.OS)
		binPath := filepath.Join(dist, binName)
		if _, err := os.Stat(binPath); err != nil {
			t.Errorf("missing %s: %v", binPath, err)
			continue
		}

		sumPath := binPath + ".sha256sum"
		sum, err := os.ReadFile(sumPath)
		if err != nil {
			t.Errorf("missing checksum for %s: %v", binPath, err)
			continue
		}
		if !strings.Contains(string(sum), binName) {
			t.Errorf("checksum file %s = %q, want it to mention %q", sumPath, sum, binName)
		}
		if len(strings.Fields(string(sum))[0]) != 64 {
			t.Errorf("checksum for %s doesn't look like a sha256 hex digest: %q", binPath, sum)
		}
	}
}

func TestMagefile_Publish_CleansPreviousDist(t *testing.T) {
	binDir := t.TempDir()
	dist := filepath.Join(binDir, "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dist, "stale-file")
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := Magefile{
		Dir:     repoRoot(t),
		Pkg:     "./examples/exec",
		Name:    "exec",
		Version: "v0.0.0-test",
		BinDir:  binDir,
	}
	if err := m.Publish(); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("expected stale dist file to be removed, stat err = %v", err)
	}
}
