package main

import (
	"strings"
	"testing"

	sdk "github.com/getporter/mixin-sdk-go"
)

func TestLint_FlagsEmbeddedBashMissingQuotes(t *testing.T) {
	input := sdk.BuildInput{
		Actions: sdk.RawMessage(`
install:
- exec:
    description: "run inline bash"
    command: bash
    flags:
      c: echo hi
`),
	}

	m := &Mixin{}
	results, err := m.Lint(input)
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2: %+v", len(results), results)
	}
	if results[0].Code != codeEmbeddedBash || results[0].Level != sdk.LintLevelWarning {
		t.Errorf("results[0] = %+v, want a %s warning", results[0], codeEmbeddedBash)
	}
	if results[1].Code != codeBashCArgMissingQuotes || results[1].Level != sdk.LintLevelError {
		t.Errorf("results[1] = %+v, want a %s error", results[1], codeBashCArgMissingQuotes)
	}
}

func TestLint_QuotedBashIsFine(t *testing.T) {
	input := sdk.BuildInput{
		Actions: sdk.RawMessage(`
install:
- exec:
    command: bash
    flags:
      c: '"echo ok"'
`),
	}

	m := &Mixin{}
	results, err := m.Lint(input)
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if len(results) != 1 || results[0].Code != codeEmbeddedBash {
		t.Errorf("results = %+v, want a single %s warning", results, codeEmbeddedBash)
	}
}

func TestLint_NonBashCommandsAreIgnored(t *testing.T) {
	input := sdk.BuildInput{
		Actions: sdk.RawMessage(`
install:
- exec:
    command: echo
    arguments: ["hi"]
`),
	}

	m := &Mixin{}
	results, err := m.Lint(input)
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("results = %+v, want none", results)
	}
}

// TestExecute_Lint drives lint through sdk.Execute, the same way Porter
// invokes the compiled binary.
func TestExecute_Lint(t *testing.T) {
	rtCtx, out := newTestContext()
	rtCtx.In = strings.NewReader(`
config:
actions:
  install:
  - exec:
      command: bash
      flags:
        c: echo hi
`)

	code := sdk.Execute(&Mixin{}, []string{"lint"}, rtCtx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; output: %s", code, out.String())
	}
	if got := out.String(); got == "" {
		t.Error("expected lint JSON on stdout, got nothing")
	}
}
