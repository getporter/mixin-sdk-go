package main

import (
	"bytes"
	"strings"
	"testing"

	sdk "github.com/getporter/mixin-sdk-go"
	"github.com/spf13/afero"
)

func newTestContext() (*sdk.Context, *bytes.Buffer) {
	var out bytes.Buffer
	return &sdk.Context{
		In:         strings.NewReader(""),
		Out:        &out,
		Err:        &out,
		FileSystem: afero.NewMemMapFs(),
		Getenv:     func(string) string { return "" },
	}, &out
}

func installStepInput(yaml string) sdk.StepInput {
	return sdk.StepInput{Action: "install", Data: sdk.RawMessage(yaml)}
}

func TestExecute_CapturesRegexOutput(t *testing.T) {
	rtCtx, out := newTestContext()
	m := &Mixin{Context: *rtCtx}

	step := installStepInput(`
command: sh
arguments:
- "-c"
- "echo hello-42"
outputs:
- name: number
  regex: "hello-(\\d+)"
`)

	if err := m.runStep(step); err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if !strings.Contains(out.String(), "hello-42") {
		t.Errorf("expected command stdout to be streamed, got %q", out.String())
	}

	number, err := afero.ReadFile(m.FileSystem, outputsDir+"/number")
	if err != nil {
		t.Fatalf("reading number output: %v", err)
	}
	if string(number) != "42" {
		t.Errorf("number output = %q, want %q", number, "42")
	}
}

// TestExecute_FileOutput covers the "path" output kind, which reads a file
// the command produced back through m.FileSystem. The subprocess itself
// always writes to the real OS filesystem (os/exec can't target an afero
// fs), so this seeds the file directly rather than shelling out — it's
// exercising ProcessFileOutputs's read/copy, not the subprocess.
func TestExecute_FileOutput(t *testing.T) {
	rtCtx, _ := newTestContext()
	m := &Mixin{Context: *rtCtx}

	if err := afero.WriteFile(m.FileSystem, "out.txt", []byte("file-contents\n"), 0644); err != nil {
		t.Fatalf("seeding out.txt: %v", err)
	}

	step := installStepInput(`
command: sh
arguments:
- "-c"
- "true"
outputs:
- name: fromfile
  path: out.txt
`)

	if err := m.runStep(step); err != nil {
		t.Fatalf("runStep: %v", err)
	}

	fromfile, err := afero.ReadFile(m.FileSystem, outputsDir+"/fromfile")
	if err != nil {
		t.Fatalf("reading fromfile output: %v", err)
	}
	if string(fromfile) != "file-contents\n" {
		t.Errorf("fromfile output = %q, want %q", fromfile, "file-contents\n")
	}
}

func TestExecute_JSONPathOutput(t *testing.T) {
	rtCtx, _ := newTestContext()
	m := &Mixin{Context: *rtCtx}

	step := installStepInput(`
command: sh
arguments:
- "-c"
- "echo '{\"nested\":{\"value\":\"jp-ok\"}}'"
outputs:
- name: nested
  jsonPath: "$.nested.value"
`)

	if err := m.runStep(step); err != nil {
		t.Fatalf("runStep: %v", err)
	}

	nested, err := afero.ReadFile(m.FileSystem, outputsDir+"/nested")
	if err != nil {
		t.Fatalf("reading nested output: %v", err)
	}
	if string(nested) != "jp-ok" {
		t.Errorf("nested output = %q, want %q", nested, "jp-ok")
	}
}

func TestExecute_FlagsAndEnv(t *testing.T) {
	rtCtx, out := newTestContext()
	m := &Mixin{Context: *rtCtx}

	step := installStepInput(`
command: sh
arguments:
- "-c"
- 'echo "$GREETING $*"'
- "sh"
flags:
  tag: hello
suffix-arguments:
- "world"
envs:
  GREETING: hi
`)

	if err := m.runStep(step); err != nil {
		t.Fatalf("runStep: %v", err)
	}
	if !strings.Contains(out.String(), "hi --tag hello world") {
		t.Errorf("expected flags/env/suffix-arguments in output, got %q", out.String())
	}
}

func TestExecute_IgnoreErrorAll(t *testing.T) {
	rtCtx, _ := newTestContext()
	m := &Mixin{Context: *rtCtx}

	step := installStepInput(`
command: sh
arguments:
- "-c"
- "exit 7"
ignoreError:
  all: true
`)

	if err := m.runStep(step); err != nil {
		t.Fatalf("expected error to be ignored, got: %v", err)
	}
}

func TestExecute_FailsOnUnignoredError(t *testing.T) {
	rtCtx, _ := newTestContext()
	m := &Mixin{Context: *rtCtx}

	step := installStepInput(`
command: sh
arguments:
- "-c"
- "exit 3"
`)

	if err := m.runStep(step); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestExecute_ViaRuntime drives the mixin through sdk.Execute end-to-end,
// the same way Porter invokes the compiled binary: args + stdin in, exit
// code out. This is the integration-test role milestone 3 calls for.
func TestExecute_ViaRuntime(t *testing.T) {
	rtCtx, out := newTestContext()
	rtCtx.In = strings.NewReader(`
install:
- exec:
    command: sh
    arguments:
    - "-c"
    - "echo from-runtime"
`)

	code := sdk.Execute(&Mixin{}, []string{"install"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; output: %s", code, out.String())
	}
	if !strings.Contains(out.String(), "from-runtime") {
		t.Errorf("expected command output, got %q", out.String())
	}
}
