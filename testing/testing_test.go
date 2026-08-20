package testing

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/getporter/mixin-sdk-go"
	"github.com/spf13/afero"
)

// fakeMixin is a minimal Mixin used to exercise this package's helpers.
type fakeMixin struct {
	sdk.Context

	installErr error
	gotInstall sdk.StepInput
}

func (m *fakeMixin) Version() sdk.VersionInfo { return sdk.VersionInfo{Name: "fake", Version: "v0"} }
func (m *fakeMixin) Schema() ([]byte, error)  { return []byte(`{}`), nil }

func (m *fakeMixin) Build(cfg sdk.BuildInput, out io.Writer) error { return nil }

func (m *fakeMixin) Lint(cfg sdk.BuildInput) (sdk.LintResults, error) { return nil, nil }

// Install writes its command to stdout and a fixed output, the way a real
// Mixin would, so the tests below have something real to observe.
func (m *fakeMixin) Install(step sdk.StepInput) error {
	m.gotInstall = step
	if m.installErr != nil {
		return m.installErr
	}

	var s struct {
		Command string `yaml:"command"`
	}
	if err := step.Data.Unmarshal(&s); err != nil {
		return err
	}
	if _, err := io.WriteString(m.Out, s.Command+"\n"); err != nil {
		return err
	}

	if err := m.FileSystem.MkdirAll(sdk.OutputsDir, 0o755); err != nil {
		return err
	}
	return afero.WriteFile(m.FileSystem, filepath.Join(sdk.OutputsDir, "result"), []byte("ok"), 0o644)
}

func (m *fakeMixin) Upgrade(step sdk.StepInput) error               { return nil }
func (m *fakeMixin) Uninstall(step sdk.StepInput) error             { return nil }
func (m *fakeMixin) Invoke(action string, step sdk.StepInput) error { return nil }

func TestContext_CallingMixinMethodDirectly(t *testing.T) {
	ctx := NewContext()
	m := &fakeMixin{Context: *ctx.Context}

	step := StepInput("install", "command: echo hi")
	if err := m.Install(step); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if m.gotInstall.Action != "install" {
		t.Errorf("Action = %q, want install", m.gotInstall.Action)
	}
	if got := ctx.Stdout(); got != "echo hi\n" {
		t.Errorf("Stdout() = %q, want %q", got, "echo hi\n")
	}

	result, err := ctx.Output("result")
	if err != nil {
		t.Fatalf("Output: %v", err)
	}
	if result != "ok" {
		t.Errorf("Output(%q) = %q, want %q", "result", result, "ok")
	}
}

func TestContext_ReadWriteFile(t *testing.T) {
	ctx := NewContext()
	if err := ctx.WriteFile("in.txt", "hello"); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := ctx.ReadFile("in.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got != "hello" {
		t.Errorf("ReadFile = %q, want %q", got, "hello")
	}
}

func TestExecute_ViaRuntime(t *testing.T) {
	ctx := NewContext()
	ctx.SetStdin(strings.TrimSpace(`
install:
- fake:
    command: echo hi
`) + "\n")

	m := &fakeMixin{}
	code := Execute(m, []string{"install"}, ctx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, ctx.Stderr())
	}
	if got := ctx.Stdout(); got != "echo hi\n" {
		t.Errorf("Stdout() = %q, want %q", got, "echo hi\n")
	}
}

func TestBuildInput(t *testing.T) {
	input := BuildInput("clientVersion: 1.2.3", "install: []")

	var cfg struct {
		ClientVersion string `yaml:"clientVersion"`
	}
	if err := input.Config.Unmarshal(&cfg); err != nil {
		t.Fatalf("Unmarshal config: %v", err)
	}
	if cfg.ClientVersion != "1.2.3" {
		t.Errorf("ClientVersion = %q, want 1.2.3", cfg.ClientVersion)
	}
	if len(input.Actions) == 0 {
		t.Error("Actions was empty")
	}
}
