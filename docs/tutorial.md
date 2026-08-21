# Tutorial: Build a Porter Mixin

This walks through building a real mixin, `hazmat`, from scratch with
`mixin-sdk-go`. By the end it will have a config schema, an `install`/
`upgrade`/`uninstall`/custom-action implementation that runs a shell
command and captures its output, a lint check, unit tests, and a way to
build, test, and publish it.

If you just want the API reference, see the [README](../README.md). This
doc is the narrative version: what to do, in order, and why.

## Background: what a mixin actually is

A Porter mixin is a standalone binary that Porter invokes with `build`,
`schema`, `version`, `install`, `upgrade`, `invoke`, `uninstall`, and
`lint`, passing YAML on stdin and reading a result off stdout. This SDK
doesn't change that protocol — it just means you implement one Go
interface instead of hand-wiring a CLI and a stdin/stdout parser for
each command yourself. If you want the full per-command wire contract,
see Porter's [Mixin Commands
guide](https://porter.sh/mixin-dev-guide/commands/); this tutorial covers
the same ground from the SDK's side.

## Prerequisites

- Go installed (see `go.mod` for the minimum version this SDK requires).
- Some familiarity with a Porter `porter.yaml` manifest — you don't need
  to be a Porter expert, but you should know that a manifest has
  `install`/`upgrade`/`uninstall` sections made of mixin steps.

## 1. Scaffold the project

```sh
go run github.com/getporter/mixin-sdk-go/cmd/mixin-init@latest hazmat
cd hazmat
```

This writes a new `./hazmat` directory containing:

- `go.mod` — a real module, already `go get`-ing `mixin-sdk-go`.
- `main.go` — a buildable stub implementing every `sdk.Mixin` method as a
  `TODO`, so the project compiles from the first command.
- `.github/workflows/hazmat.yml` — a starter CI workflow (checkout,
  setup-go, build/vet/test).
- `LICENSE` — Apache 2.0.

Useful flags, all optional:

| Flag | Default | What it does |
| --- | --- | --- |
| `-module` | same as the name | Go module path, if you already know where this will be published |
| `-dir` | `./<name>` | where to scaffold into |
| `-author` | (empty) | embedded in the generated `Version()` |
| `-sdk-version` | `latest` | pin a specific `mixin-sdk-go` release |
| `-sdk-path` | (unset) | point at a local SDK checkout instead of fetching a release — useful if you're developing against an unpublished SDK change |
| `-force` | `false` | scaffold into a directory that already exists and is non-empty |

Confirm it builds before changing anything:

```sh
go build ./... && go run . version
# hazmat dev by
```

## 2. The interface you're implementing

Everything a mixin needs to do lives on one interface, [`sdk.Mixin`](../mixin.go):

```go
type Mixin interface {
	Version() VersionInfo
	Schema() ([]byte, error)
	Build(cfg BuildInput, out io.Writer) error
	Lint(cfg BuildInput) (LintResults, error)
	Install(step StepInput) error
	Upgrade(step StepInput) error
	Invoke(action string, step StepInput) error
	Uninstall(step StepInput) error
}
```

| Method | Porter calls it during | Purpose |
| --- | --- | --- |
| `Version` | `porter build`, `porter mixins list` | identify the mixin and its build version |
| `Schema` | `porter build`, `porter schema`, editor tooling | describe the config/step shapes for manifest validation and autocomplete |
| `Build` | `porter build`, on your machine | emit Dockerfile lines for anything the mixin needs baked into the bundle image |
| `Lint` | `porter lint` | flag problems in how this mixin is used in the manifest |
| `Install`/`Upgrade`/`Uninstall` | `porter run`, inside the bundle | do the actual work for that action |
| `Invoke` | `porter run`, for a custom action | do the work for any action name that isn't install/upgrade/uninstall |

`main.go`'s generated `Mixin` struct embeds `sdk.Context` by value:

```go
type Mixin struct {
	sdk.Context
}
```

`sdk.Run` (called from `main`) detects that and wires up real
stdin/stdout/stderr and an OS filesystem before any command runs — that's
where `m.Out`, `m.Err`, and `m.FileSystem` come from in the methods below.
You don't have to embed it (a mixin with no I/O needs doesn't need to),
but most do.

## 3. Give it an identity: `Version`

```go
func (m *Mixin) Version() sdk.VersionInfo {
	return sdk.VersionInfo{Name: "hazmat", Author: "you", Version: version}
}
```

`version` here is the package-level `var version = "dev"` the generated
`main.go` already has — leave it as a var, not a constant. `mage Build`/
`Publish` (see step 8) set it via `-ldflags -X main.version=...`, so a
release build reports its real version without you touching this line.

## 4. Define the manifest schema: `Schema`

Porter merges every mixin's JSON schema into one document to validate
`porter.yaml` and drive editor autocomplete. Your schema needs:

- A `config` definition, if your mixin accepts config under
  `mixins:\n- hazmat:\n    ...` — otherwise skip it.
- An `<action>Step` definition per action you support (`installStep`,
  `upgradeStep`, `uninstallStep`, plus `invokeStep` for custom actions),
  each wrapping a single property named after your mixin.

For a `hazmat` mixin whose steps look like:

```yaml
install:
- hazmat:
    description: "say hi"
    command: "echo hello"
```

save this as `schema.json`:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "definitions": {
    "config": {
      "type": "object",
      "properties": {
        "hazmat": {
          "type": "object",
          "properties": {
            "clientVersion": { "type": "string" }
          },
          "additionalProperties": false
        }
      },
      "additionalProperties": false
    },
    "installStep": {
      "type": "object",
      "properties": { "hazmat": { "$ref": "#/definitions/hazmat" } },
      "additionalProperties": false,
      "required": ["hazmat"]
    },
    "upgradeStep": {
      "type": "object",
      "properties": { "hazmat": { "$ref": "#/definitions/hazmat" } },
      "additionalProperties": false,
      "required": ["hazmat"]
    },
    "uninstallStep": {
      "type": "object",
      "properties": { "hazmat": { "$ref": "#/definitions/hazmat" } },
      "additionalProperties": false,
      "required": ["hazmat"]
    },
    "invokeStep": {
      "type": "object",
      "properties": { "hazmat": { "$ref": "#/definitions/hazmat" } },
      "additionalProperties": false,
      "required": ["hazmat"]
    },
    "hazmat": {
      "type": "object",
      "properties": {
        "description": { "type": "string" },
        "command": { "type": "string" }
      },
      "additionalProperties": false,
      "required": ["command"]
    }
  },
  "type": "object",
  "properties": {
    "install": { "type": "array", "items": { "$ref": "#/definitions/installStep" } },
    "upgrade": { "type": "array", "items": { "$ref": "#/definitions/upgradeStep" } },
    "uninstall": { "type": "array", "items": { "$ref": "#/definitions/uninstallStep" } }
  },
  "additionalProperties": {
    "type": "array",
    "items": { "$ref": "#/definitions/invokeStep" }
  }
}
```

A few schema-authoring rules Porter's merge step relies on (see the [Mixin
Commands guide](https://porter.sh/mixin-dev-guide/commands/#schema) for
the full list): only relative `$ref`s, no chained `$ref`s, no `$id`s, and
custom-action support is what the `additionalProperties: {...invokeStep}`
block at the bottom signals.

Embed and validate it in `main.go`:

```go
//go:embed schema.json
var schemaFile embed.FS

func (m *Mixin) Schema() ([]byte, error) {
	schema, err := schemaFile.ReadFile("schema.json")
	if err != nil {
		return nil, err
	}
	if err := sdk.ValidateSchema(schema); err != nil {
		return nil, err
	}
	return schema, nil
}
```

`sdk.ValidateSchema` only checks that the bytes are well-formed JSON — it
catches a broken `go:embed` or a typo'd trailing comma, not schema
correctness. Call it in a test too, so CI catches a broken schema before
Porter does:

```go
func TestSchema_IsValidJSON(t *testing.T) {
	m := &Mixin{}
	schema, err := m.Schema()
	if err != nil {
		t.Fatal(err)
	}
	if err := sdk.ValidateSchema(schema); err != nil {
		t.Fatal(err)
	}
}
```

## 5. Buildtime dependencies: `Build`

`Build` receives the mixin's config block (`BuildInput.Config`, raw
YAML — see step 6 for `RawMessage`) and writes Dockerfile lines for
anything the mixin needs baked into the bundle's invocation image. A
mixin with no buildtime dependency can leave this a no-op:

```go
func (m *Mixin) Build(cfg sdk.BuildInput, out io.Writer) error {
	var config struct {
		ClientVersion string `yaml:"clientVersion"`
	}
	if err := cfg.Config.Unmarshal(&config); err != nil {
		return err
	}
	if config.ClientVersion == "" {
		return nil
	}
	_, err := fmt.Fprintf(out, "RUN hazmat-cli install --version %s\n", config.ClientVersion)
	return err
}
```

## 6. The step shape and `RawMessage`

Porter sends `install`/`upgrade`/`uninstall`/`invoke` the raw YAML for a
single step as `StepInput.Data` — a `sdk.RawMessage` ([]byte with a
`.Unmarshal(v any) error` method; despite the name, it decodes YAML, not
JSON, because that's what Porter actually sends over the wire).
`StepInput.Action` is the action the step belongs to.

Define your own step type and decode into it — the SDK doesn't know or
care about your step's shape:

```go
// step.go
type Step struct {
	Description string `yaml:"description"`
	Command     string `yaml:"command"`
}
```

## 7. Implement the actions

`Install`, `Upgrade`, `Uninstall`, and `Invoke` all decode the same step
shape, so route them through one helper:

```go
func (m *Mixin) Install(step sdk.StepInput) error   { return m.runStep(step) }
func (m *Mixin) Upgrade(step sdk.StepInput) error   { return m.runStep(step) }
func (m *Mixin) Uninstall(step sdk.StepInput) error { return m.runStep(step) }

func (m *Mixin) Invoke(action string, step sdk.StepInput) error {
	return m.runStep(step)
}

func (m *Mixin) runStep(input sdk.StepInput) error {
	var step Step
	if err := input.Data.Unmarshal(&step); err != nil {
		return fmt.Errorf("could not parse %s step: %w", input.Action, err)
	}
	return m.execute(step)
}
```

And the actual work, in `execute.go`. Step outputs go to files under
`sdk.OutputsDir` (`/cnab/app/porter/outputs` — matches Porter's own
runtime convention) for Porter to collect after the step runs:

```go
func (m *Mixin) execute(step Step) error {
	cmd := exec.Command("sh", "-c", step.Command)

	var stdout bytes.Buffer
	cmd.Stdout = io.MultiWriter(m.Out, &stdout)
	cmd.Stderr = m.Err

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running command %q: %w", step.Command, err)
	}

	if err := m.FileSystem.MkdirAll(sdk.OutputsDir, 0o755); err != nil {
		return err
	}
	return afero.WriteFile(m.FileSystem, filepath.Join(sdk.OutputsDir, "result"), stdout.Bytes(), 0o644)
}
```

Try it from the command line — Porter pipes step YAML on stdin, so you
can too:

```sh
echo 'install:
- hazmat:
    command: "echo hello from hazmat"' | go run . install
```

You'll see `hello from hazmat` printed, then a `mkdir /cnab: permission
denied` error — expected, and worth understanding rather than working
around: `/cnab/app/porter/outputs` only exists inside a real bundle
container. Running the binary directly on your machine has nowhere to
write that output to. That's exactly what the `testing` package (step 9)
is for — it swaps in an in-memory filesystem so output-writing code is
testable without `/cnab` existing anywhere.

`install`/`upgrade`/`uninstall`/`invoke` all also accept `-f <path>`
instead of stdin (`go run . install -f step.yaml`), and every command
accepts `--debug` (sets `Context.Debug`, which your mixin can check —
the SDK doesn't act on it itself, that's between you and your logs).

## 8. Optional: `Lint`

`Lint` is required by the `Mixin` interface (so you don't forget it
exists), but a no-op `return nil, nil` is a complete, valid
implementation if your mixin has nothing worth flagging. If it does,
`BuildInput.Actions` gives you every action's step list to inspect:

```go
// lint.go
const codeEmptyCommand = "hazmat-100"

type actionEntry struct {
	Hazmat Step `yaml:"hazmat"`
}

func (m *Mixin) Lint(input sdk.BuildInput) (sdk.LintResults, error) {
	var actions map[string][]actionEntry
	if err := input.Actions.Unmarshal(&actions); err != nil {
		return nil, fmt.Errorf("could not parse actions for lint: %w", err)
	}

	var results sdk.LintResults
	for actionName, entries := range actions {
		for i, entry := range entries {
			if entry.Hazmat.Command != "" {
				continue
			}
			results = append(results, sdk.LintResult{
				Level: sdk.LintLevelError,
				Location: sdk.LintLocation{
					Action:          actionName,
					Mixin:           "hazmat",
					StepNumber:      i + 1,
					StepDescription: entry.Hazmat.Description,
				},
				Code:  codeEmptyCommand,
				Title: "command is required",
			})
		}
	}
	return results, nil
}
```

Prefix `Code` with your mixin's name (`hazmat-100`, not `100`) so it
can't collide with another mixin's or Porter's own codes. See
`examples/exec/lint.go` in this repo for a second worked example,
including a warning-level result.

## 9. Test it

The `testing` subpackage gives you an in-memory `Context` — no real
stdin, subprocess, or filesystem — so you can unit test a method
directly, including the output-writing code that failed on the command
line in step 7:

```go
import mixintesting "github.com/getporter/mixin-sdk-go/testing"

func TestInstall(t *testing.T) {
	ctx := mixintesting.NewContext()
	m := &Mixin{Context: *ctx.Context}

	step := mixintesting.StepInput("install", `command: "echo hello from hazmat"`)
	if err := m.Install(step); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if got, want := ctx.Stdout(), "hello from hazmat\n"; got != want {
		t.Errorf("Stdout() = %q, want %q", got, want)
	}

	result, err := ctx.Output("result")
	if err != nil {
		t.Fatalf("Output: %v", err)
	}
	if got, want := result, "hello from hazmat\n"; got != want {
		t.Errorf("Output(result) = %q, want %q", got, want)
	}
}
```

`mixintesting.StepInput(action, yamlBody)` builds a `StepInput` from just
the mixin-specific fields — the part your `Step` type decodes — so the
fixture matches exactly what `Install` receives after Porter has already
stripped off the `{action: [{hazmat: ...}]}` wrapping.

For a black-box test of the whole CLI surface (arg parsing, stdin
decoding, command dispatch) instead of calling a method directly, drive
it through `mixintesting.Execute` — the same entry point `sdk.Run` uses,
without a subprocess:

```go
func TestExecute_Install(t *testing.T) {
	ctx := mixintesting.NewContext()
	ctx.SetStdin("install:\n- hazmat:\n    command: \"echo hi\"\n")

	m := &Mixin{}
	code := mixintesting.Execute(m, []string{"install"}, ctx)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, ctx.Stderr())
	}
}
```

Run it:

```sh
go test ./...
```

## 10. Build, test, and publish with a magefile

Rather than copying Porter's full mage-based release tooling, the `mage`
subpackage gives you `Build`/`Test`/`Publish` for a magefile that's a few
lines:

```go
// magefile.go
//go:build mage
package main

import sdkmage "github.com/getporter/mixin-sdk-go/mage"

var version = "dev" // set by CI, e.g. from `git describe`

var m = sdkmage.Magefile{Dir: ".", Pkg: ".", Name: "hazmat", Version: version, BinDir: "bin/mixins/hazmat"}

func Build() error   { return m.Build() }
func Test() error    { return m.Test() }
func Publish() error { return m.Publish() }
```

This needs the `mage` tool: `go install github.com/magefile/mage@latest`.
Then:

```sh
mage Build     # bin/mixins/hazmat/hazmat, for the current GOOS/GOARCH
mage Test      # go test ./...
mage Publish   # cross-compiles Porter's 6-platform release matrix into
               # bin/mixins/hazmat/dist/, one <name>-<goos>-<goarch>[.exe]
               # plus a matching .sha256sum file per platform
```

`Publish` deliberately doesn't push anywhere — wire your own CI release
step (e.g. `gh release create bin/mixins/hazmat/dist/*`) to upload
`dist/`'s contents. The naming convention it produces
(`hazmat-linux-amd64`, `hazmat-windows-amd64.exe`, ...) matches what
Porter's own `porter mixin install` expects to find at a release URL —
see [Distributing
Mixins](https://porter.sh/docs/development/dist-a-mixin/) for the exact
layout and how `porter mixin install hazmat --version v0.1.0 --url
<your-release-url>` resolves it.

## 11. CI

`mixin-init` already wrote `.github/workflows/hazmat.yml` — checkout,
setup-go, `go build ./...`, `go vet ./...`, `go test ./...`. That's
enough to start; add a `mage Publish` step (gated on a tag push) once
you're ready to cut releases, and a lint step if you adopt
`golangci-lint` — this SDK's own `.github/workflows/` and `.golangci.yml`
are a reference if you want one.

## Where to go from here

- [`examples/exec`](../examples/exec) in this repo — a complete second
  mixin (config, schema, build, install/upgrade/uninstall/invoke, lint,
  tests) ported from Porter's real built-in `exec` mixin, worth reading
  end to end once this tutorial's version feels familiar.
- [README.md](../README.md) — concise API reference for everything
  covered here.
- [Porter's Mixin Commands guide](https://porter.sh/mixin-dev-guide/commands/) —
  the wire-protocol contract this SDK implements, if you need the
  ground truth for an edge case.
- [Porter's Distributing Mixins guide](https://porter.sh/docs/development/dist-a-mixin/) —
  publishing and installing a mixin once it's ready to share.
