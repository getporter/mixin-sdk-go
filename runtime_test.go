package sdk

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// fakeMixin is a minimal, fully in-memory Mixin used to exercise the
// runtime dispatch logic without a real mixin implementation.
type fakeMixin struct {
	Context

	version   VersionInfo
	schema    []byte
	schemaErr error

	buildErr error
	gotBuild BuildInput

	installErr error
	gotInstall StepInput

	upgradeErr error
	gotUpgrade StepInput

	uninstallErr error
	gotUninstall StepInput

	invokeErr error
	gotAction string
	gotInvoke StepInput
}

func (m *fakeMixin) Version() VersionInfo { return m.version }

func (m *fakeMixin) Schema() ([]byte, error) { return m.schema, m.schemaErr }

func (m *fakeMixin) Build(cfg BuildInput, out io.Writer) error {
	m.gotBuild = cfg
	if m.buildErr != nil {
		return m.buildErr
	}
	_, err := io.WriteString(out, "RUN echo hi\n")
	return err
}

func (m *fakeMixin) Install(step StepInput) error {
	m.gotInstall = step
	return m.installErr
}

func (m *fakeMixin) Upgrade(step StepInput) error {
	m.gotUpgrade = step
	return m.upgradeErr
}

func (m *fakeMixin) Invoke(action string, step StepInput) error {
	m.gotAction, m.gotInvoke = action, step
	return m.invokeErr
}

func (m *fakeMixin) Uninstall(step StepInput) error {
	m.gotUninstall = step
	return m.uninstallErr
}

func newTestContext(stdin string) (*Context, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	rtCtx := &Context{
		In:         strings.NewReader(stdin),
		Out:        out,
		Err:        errOut,
		FileSystem: afero.NewMemMapFs(),
		Getenv:     func(string) string { return "" },
	}
	return rtCtx, out, errOut
}

func TestExecute_Version(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"plaintext default", nil, "hazmat v1.2.3 by porter\n"},
		{"plaintext explicit", []string{"-o", "plaintext"}, "hazmat v1.2.3 by porter\n"},
		{"json", []string{"-o", "json"}, `{
  "name": "hazmat",
  "author": "porter",
  "version": "v1.2.3"
}
`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &fakeMixin{version: VersionInfo{Name: "hazmat", Author: "porter", Version: "v1.2.3"}}
			rtCtx, out, _ := newTestContext("")
			code := Execute(m, append([]string{"version"}, tc.args...), rtCtx)
			if code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if out.String() != tc.want {
				t.Errorf("stdout = %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestExecute_Version_BadFormat(t *testing.T) {
	m := &fakeMixin{version: VersionInfo{Name: "hazmat"}}
	rtCtx, _, errOut := newTestContext("")
	code := Execute(m, []string{"version", "-o", "xml"}, rtCtx)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "unsupported output format") {
		t.Errorf("stderr = %q, want mention of unsupported format", errOut.String())
	}
}

func TestExecute_Schema(t *testing.T) {
	m := &fakeMixin{schema: []byte(`{"type":"object"}`)}
	rtCtx, out, _ := newTestContext("")
	code := Execute(m, []string{"schema"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if out.String() != `{"type":"object"}` {
		t.Errorf("stdout = %q", out.String())
	}
}

func TestExecute_Build(t *testing.T) {
	m := &fakeMixin{}
	stdin := "config:\n  clientVersion: 1.2.3\nactions:\n  install: []\n"
	rtCtx, out, _ := newTestContext(stdin)
	code := Execute(m, []string{"build"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if out.String() != "RUN echo hi\n" {
		t.Errorf("stdout = %q", out.String())
	}

	var cfg struct {
		ClientVersion string `yaml:"clientVersion"`
	}
	if err := m.gotBuild.Config.Unmarshal(&cfg); err != nil {
		t.Fatalf("Unmarshal config: %v", err)
	}
	if cfg.ClientVersion != "1.2.3" {
		t.Errorf("ClientVersion = %q, want 1.2.3", cfg.ClientVersion)
	}
}

func TestExecute_Build_NoConfig(t *testing.T) {
	m := &fakeMixin{}
	rtCtx, _, _ := newTestContext("actions:\n  install: []\n")
	code := Execute(m, []string{"build"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if m.gotBuild.Config != nil {
		t.Errorf("Config = %q, want nil", m.gotBuild.Config)
	}
}

const installStdin = `install:
- hazmat:
    description: "say hi"
    command: "echo hi"
`

func TestExecute_Install(t *testing.T) {
	m := &fakeMixin{}
	rtCtx, _, _ := newTestContext(installStdin)
	code := Execute(m, []string{"install"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if m.gotInstall.Action != "install" {
		t.Errorf("Action = %q, want install", m.gotInstall.Action)
	}

	var step struct {
		Description string `yaml:"description"`
		Command     string `yaml:"command"`
	}
	if err := m.gotInstall.Data.Unmarshal(&step); err != nil {
		t.Fatalf("Unmarshal data: %v", err)
	}
	if step.Command != "echo hi" {
		t.Errorf("Command = %q, want %q", step.Command, "echo hi")
	}
}

func TestExecute_Install_FromFile(t *testing.T) {
	m := &fakeMixin{}
	rtCtx, _, _ := newTestContext("")
	if err := afero.WriteFile(rtCtx.FileSystem, "step.yaml", []byte(installStdin), 0o644); err != nil {
		t.Fatal(err)
	}
	code := Execute(m, []string{"install", "-f", "step.yaml"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if m.gotInstall.Action != "install" {
		t.Errorf("Action = %q, want install", m.gotInstall.Action)
	}
}

func TestExecute_Upgrade(t *testing.T) {
	m := &fakeMixin{}
	stdin := "upgrade:\n- hazmat:\n    command: echo up\n"
	rtCtx, _, _ := newTestContext(stdin)
	code := Execute(m, []string{"upgrade"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if m.gotUpgrade.Action != "upgrade" {
		t.Errorf("Action = %q, want upgrade", m.gotUpgrade.Action)
	}
}

func TestExecute_Uninstall(t *testing.T) {
	m := &fakeMixin{}
	stdin := "uninstall:\n- hazmat:\n    command: echo down\n"
	rtCtx, _, _ := newTestContext(stdin)
	code := Execute(m, []string{"uninstall"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if m.gotUninstall.Action != "uninstall" {
		t.Errorf("Action = %q, want uninstall", m.gotUninstall.Action)
	}
}

func TestExecute_Invoke(t *testing.T) {
	m := &fakeMixin{}
	stdin := "status:\n- hazmat:\n    command: echo status\n"
	rtCtx, _, _ := newTestContext(stdin)
	code := Execute(m, []string{"invoke", "--action", "status"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if m.gotAction != "status" {
		t.Errorf("action = %q, want status", m.gotAction)
	}
	if m.gotInvoke.Action != "status" {
		t.Errorf("StepInput.Action = %q, want status", m.gotInvoke.Action)
	}
}

func TestExecute_StepErrors(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{"no actions", []string{"install"}, "", "expected exactly one action"},
		{"two actions", []string{"install"}, "install:\n- hazmat: {}\nupgrade:\n- hazmat: {}\n", "expected exactly one action"},
		{"two steps", []string{"install"}, "install:\n- hazmat: {}\n- hazmat: {}\n", "expected exactly one step"},
		{"malformed step", []string{"install"}, "install:\n- hazmat: {}\n  other: {}\n", "malformed step"},
		{"missing file", []string{"install", "-f", "missing.yaml"}, "", "could not read step input"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &fakeMixin{}
			rtCtx, _, errOut := newTestContext(tc.stdin)
			code := Execute(m, tc.args, rtCtx)
			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if !strings.Contains(errOut.String(), tc.want) {
				t.Errorf("stderr = %q, want substring %q", errOut.String(), tc.want)
			}
		})
	}
}

func TestExecute_MixinErrorPropagates(t *testing.T) {
	wantErr := "boom"
	m := &fakeMixin{installErr: errString(wantErr)}
	rtCtx, _, errOut := newTestContext(installStdin)
	code := Execute(m, []string{"install"}, rtCtx)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), wantErr) {
		t.Errorf("stderr = %q, want substring %q", errOut.String(), wantErr)
	}
}

func TestExecute_ContextInjected(t *testing.T) {
	m := &fakeMixin{}
	rtCtx, _, _ := newTestContext(installStdin)
	rtCtx.Debug = true
	code := Execute(m, []string{"install"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !m.Debug {
		t.Error("embedded Context.Debug was not propagated to the mixin")
	}
	if m.Out != rtCtx.Out {
		t.Error("embedded Context.Out was not wired up")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
