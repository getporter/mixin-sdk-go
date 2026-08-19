# mixin-sdk-go

A Go library for building [Porter](https://porter.sh) mixins. Import it and
implement one interface instead of cloning
[`skeletor`](https://github.com/getporter/skeletor) and renaming boilerplate
across a `cmd/` + `pkg/` tree.

This does not change the Porter-to-mixin wire protocol. A mixin built with
this SDK is still a standalone binary that Porter invokes with `build`,
`schema`, `version`, `install`, `upgrade`, `invoke`, and `uninstall` over
stdin/stdout — only how you build that binary changes.

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

## Status

This SDK is under active development; see [mixin-sdk-plan.md](./mixin-sdk-plan.md)
for the roadmap. Done so far:

- [x] Core `Mixin` interface and types (`mixin.go`, `build.go`, `step.go`,
      `version.go`)
- [x] CLI/runtime dispatch (`runtime.go`, `context.go`)
- [ ] A real mixin ported to the SDK as a proof of concept
- [ ] `mixin-init` scaffolding generator
- [ ] In-memory testing helpers
- [ ] Docs/tutorial

## License

[Apache 2.0](./LICENSE)
