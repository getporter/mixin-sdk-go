package sdk

// BuildInput is the input passed to Mixin.Build and Linter.Lint — Porter
// sends both commands the same shape on stdin. Config holds the raw bytes
// of the mixin's config block in porter.yaml:
//
//	mixins:
//	- hazmat:
//	    clientVersion: "1.2.3"
//
// and is empty when the mixin has no config. Actions holds the raw bytes
// of every action's steps for this mixin, keyed by action name (install,
// upgrade, uninstall, or a custom action), e.g.:
//
//	install:
//	- hazmat: {...}
//	upgrade:
//	- hazmat: {...}
//
// Build typically only needs Config; Lint needs Actions to inspect step
// definitions for problems.
type BuildInput struct {
	Config  RawMessage
	Actions RawMessage
}
