package testing

import sdk "github.com/getporter/mixin-sdk-go"

// StepInput builds an sdk.StepInput for testing, from the step's raw wire
// YAML — the mixin-specific fields your Mixin's Install/Upgrade/Invoke/
// Uninstall would actually receive, after Porter has already stripped off
// the {action: [{mixinName: ...}]} wrapping. For example, given
// porter.yaml:
//
//	install:
//	- hazmat:
//	    command: launch
//
// StepInput("install", "command: launch") produces what your Mixin's
// Install method would receive.
func StepInput(action, yamlBody string) sdk.StepInput {
	return sdk.StepInput{Action: action, Data: sdk.RawMessage(yamlBody)}
}

// BuildInput builds an sdk.BuildInput for testing, from YAML config and
// actions bodies (either may be empty), matching what Build and Lint
// receive.
func BuildInput(configYAML, actionsYAML string) sdk.BuildInput {
	return sdk.BuildInput{Config: sdk.RawMessage(configYAML), Actions: sdk.RawMessage(actionsYAML)}
}

// Execute runs m through the same command dispatch Run uses — building the
// cobra command tree, parsing args and stdin, calling the matching Mixin
// method — against ctx, without spawning a subprocess. It returns the exit
// code sdk.Execute would have returned; check ctx.Stdout()/ctx.Stderr()
// for output. Use this for a black-box test of the full CLI surface; call
// Mixin methods directly for a white-box unit test of one method.
func Execute(m sdk.Mixin, args []string, ctx *Context) int {
	return sdk.Execute(m, args, ctx.Context)
}
