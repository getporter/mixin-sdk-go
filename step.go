package sdk

// StepInput is the input passed to Install, Upgrade, Invoke, and Uninstall,
// decoded from a single step of the corresponding action in porter.yaml,
// e.g. for:
//
//	install:
//	- hazmat:
//	    description: "..."
//	    command: "..."
//
// Data holds the raw bytes of the mixin-specific block (see RawMessage) —
// here, the description/command/... fields. Use Data.Unmarshal to decode it
// into a step type of your own design.
type StepInput struct {
	// Action is the porter.yaml action this step belongs to: "install",
	// "upgrade", "uninstall", or the custom action name for Invoke.
	Action string

	Data RawMessage
}
