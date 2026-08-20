// Package sdk lets a Porter mixin author implement a small interface
// instead of copying and renaming the Skeletor template repo. Run wires a
// Mixin implementation up to Porter's mixin CLI protocol: it builds the
// cobra command tree, reads and decodes stdin, calls the matching Mixin
// method, and writes output to stdout in the shape Porter expects.
//
// This package does not change the Porter-to-mixin wire protocol — a mixin
// built with it is still a standalone binary invoked with build, schema,
// version, install, upgrade, invoke, uninstall, and lint over
// stdin/stdout.
package sdk

import "io"

// Mixin is the interface a mixin author implements.
//
// Install, Upgrade, Invoke, and Uninstall run in the bundle's execution
// environment and typically need to write files (e.g. step outputs to
// /cnab/app/porter/outputs) or stream subprocess output. A Mixin that
// embeds Context by value gets those wired up automatically before any
// command runs:
//
//	type HazmatMixin struct {
//	    sdk.Context
//	}
type Mixin interface {
	// Version returns the mixin's identity and version for the `version`
	// command.
	Version() VersionInfo

	// Schema returns the JSON schema describing this mixin's config and
	// step shapes, for the `schema` command.
	Schema() ([]byte, error)

	// Build writes Dockerfile lines needed at build time for the `build`
	// command. Mixins with no buildtime dependencies can leave out writing
	// anything to out.
	Build(cfg BuildInput, out io.Writer) error

	// Lint checks this mixin's steps in the manifest for problems, for the
	// `lint` command. Mixins with nothing to check can return nil, nil.
	Lint(cfg BuildInput) (LintResults, error)

	// Install executes the `install` action.
	Install(step StepInput) error

	// Upgrade executes the `upgrade` action.
	Upgrade(step StepInput) error

	// Invoke executes a custom action, e.g. `invoke --action status`.
	Invoke(action string, step StepInput) error

	// Uninstall executes the `uninstall` action.
	Uninstall(step StepInput) error
}
