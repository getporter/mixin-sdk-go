# mixin-sdk-go

A Go library for building [Porter](https://porter.sh) mixins. Import it and
implement one interface instead of cloning
[`skeletor`](https://github.com/getporter/skeletor) and renaming boilerplate
across a `cmd/` + `pkg/` tree.

This does not change the Porter-to-mixin wire protocol. A mixin built with
this SDK is still a standalone binary that Porter invokes with `build`,
`schema`, `version`, `install`, `upgrade`, `invoke`, `uninstall`, and `lint`
over stdin/stdout — only how you build that binary changes.

New to the SDK? [**docs/tutorial.md**](./docs/tutorial.md) walks through
building a complete mixin step by step. What follows here is the concise
reference version.

## Install

```sh
go get github.com/getporter/mixin-sdk-go
```

## Usage

Implement the `Mixin` interface:

```go
// main.go
package main

import sdk "github.com/getporter/mixin-sdk-go"

func main() {
	sdk.Run(&HazmatMixin{})
}
```

```go
// hazmat.go
package main

import (
	"fmt"
	"io"

	sdk "github.com/getporter/mixin-sdk-go"
)

type HazmatMixin struct {
	sdk.Context // gives you Out/Err/FileSystem, wired up automatically by Run
}

func (m *HazmatMixin) Version() sdk.VersionInfo {
	return sdk.VersionInfo{Name: "hazmat", Author: "you", Version: "v0.1.0"}
}

func (m *HazmatMixin) Schema() ([]byte, error) { return schema, nil }

func (m *HazmatMixin) Build(cfg sdk.BuildInput, out io.Writer) error {
	_, err := fmt.Fprintln(out, "RUN apt-get install -y hazmat-cli")
	return err
}

// Lint checks this mixin's steps for problems; return nil, nil if there's
// nothing to check.
func (m *HazmatMixin) Lint(cfg sdk.BuildInput) (sdk.LintResults, error) {
	return nil, nil
}

func (m *HazmatMixin) Install(step sdk.StepInput) error   { return m.run(step) }
func (m *HazmatMixin) Upgrade(step sdk.StepInput) error   { return m.run(step) }
func (m *HazmatMixin) Uninstall(step sdk.StepInput) error { return m.run(step) }

func (m *HazmatMixin) Invoke(action string, step sdk.StepInput) error {
	return m.run(step)
}

func (m *HazmatMixin) run(step sdk.StepInput) error {
	var s struct {
		Command string `yaml:"command"`
	}
	if err := step.Data.Unmarshal(&s); err != nil {
		return err
	}
	fmt.Fprintln(m.Out, s.Command)
	return nil
}
```

Embedding `sdk.Context` is optional but is how you get stdout/stderr and an
`afero.Fs` for writing step outputs (e.g. to
`/cnab/app/porter/outputs`) — `Run` injects the real ones before any command
executes. Swap it for an in-memory `sdk.Context` in tests to call your
`Mixin` methods directly, without a subprocess.

### `BuildInput` / `StepInput`

Porter sends mixin input as YAML, not JSON. `BuildInput.Config` and
`StepInput.Data` are `sdk.RawMessage` — raw bytes with a `.Unmarshal(v any)
error` method (YAML-backed) — so you decode them into whatever shape your
mixin's config or step definition actually has:

```go
var cfg struct {
	ClientVersion string `yaml:"clientVersion"`
}
input.Config.Unmarshal(&cfg)
```

`StepInput.Action` is the porter.yaml action the step belongs to
(`"install"`, `"upgrade"`, `"uninstall"`, or the custom action name for
`Invoke`).

### Testing your Mixin

The `testing` subpackage gives you an in-memory `Context` (buffered
stdout/stderr, an in-memory filesystem, no real subprocess) plus fixture
builders for `StepInput`/`BuildInput`:

```go
import mixintesting "github.com/getporter/mixin-sdk-go/testing"

func TestInstall(t *testing.T) {
	ctx := mixintesting.NewContext()
	m := &HazmatMixin{Context: *ctx.Context}

	err := m.Install(mixintesting.StepInput("install", "command: launch"))
	// assert err, ctx.Stdout(), ctx.Output("someOutputName"), ...
}
```

Or drive the whole CLI surface — args, stdin, command dispatch — with
`mixintesting.Execute(m, []string{"install"}, ctx)`, the same entry point
`Run` uses, without spawning a subprocess.

### Building and releasing your Mixin

The `mage` subpackage gives you `Build`/`Test`/`Publish` for a thin
`magefile.go`, instead of copying Porter's full mage-based release
tooling:

```go
//go:build mage
package main

import sdkmage "github.com/getporter/mixin-sdk-go/mage"

var version = "dev" // set by CI, e.g. from `git describe`

var m = sdkmage.Magefile{Dir: ".", Pkg: ".", Name: "hazmat", Version: version, BinDir: "bin/mixins/hazmat"}

func Build() error   { return m.Build() }
func Test() error    { return m.Test() }
func Publish() error { return m.Publish() }
```

`Publish` cross-compiles for Porter's standard release platforms and
writes each binary plus a SHA-256 checksum to `bin/mixins/hazmat/dist/` —
it does not push anywhere; wire your own CI release step (e.g.
`gh release create`) to upload `dist/`'s contents.

## Status

This SDK is under active development.

## Versioning

See [docs/versioning.md](./docs/versioning.md) for the full policy. In
short: `mixin-sdk-go` is pre-1.0 (`v0.x.y`): breaking changes may land in a
minor release, per semver's own pre-1.0 rule. Pin an exact version once
your mixin ships, rather than tracking `@latest`, and review the diff
before bumping it.

Once the `Mixin` interface settles, this SDK will cut `v1.0.0`. After
that, anything slated for removal gets a deprecation window first: it's
marked `// Deprecated:` in godoc for at least one release before it's
actually removed.

A given Porter release may require a newer `mixin-sdk-go` version — e.g.
if the mixin wire protocol changes. If that happens, it's a breaking
change to this SDK too, and follows the same rule as any other: it isn't
required silently, it's called out as a breaking change in that release.

## License

[Apache 2.0](./LICENSE)
