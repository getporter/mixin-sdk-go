package main

import (
	"embed"
	"fmt"
	"io"

	sdk "github.com/getporter/mixin-sdk-go"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

//go:embed schema.json
var schemaFile embed.FS

// Mixin implements sdk.Mixin. Embedding sdk.Context by value gets
// In/Out/Err/FileSystem wired up automatically before any command runs.
type Mixin struct {
	sdk.Context
}

func (m *Mixin) Version() sdk.VersionInfo {
	return sdk.VersionInfo{Name: "exec", Author: "Porter Authors", Version: version}
}

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

func (m *Mixin) Build(cfg sdk.BuildInput, out io.Writer) error {
	_, err := fmt.Fprintln(out, "# exec mixin has no buildtime dependencies")
	return err
}

func (m *Mixin) Install(step sdk.StepInput) error { return m.runStep(step) }

func (m *Mixin) Upgrade(step sdk.StepInput) error { return m.runStep(step) }

func (m *Mixin) Uninstall(step sdk.StepInput) error { return m.runStep(step) }

func (m *Mixin) Invoke(action string, step sdk.StepInput) error { return m.runStep(step) }

func (m *Mixin) runStep(input sdk.StepInput) error {
	var step Step
	if err := input.Data.Unmarshal(&step); err != nil {
		return fmt.Errorf("could not parse %s step: %w", input.Action, err)
	}
	return m.execute(step)
}
