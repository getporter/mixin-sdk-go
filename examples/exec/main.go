// Command exec is a proof-of-concept port of Porter's built-in exec mixin
// onto mixin-sdk-go, validating that the SDK's Mixin interface covers a real
// mixin's needs. It supports the same porter.yaml step shape as the
// original: description/command/dir/arguments/suffix-arguments/flags/envs/
// outputs/suppress-output/ignoreError.
package main

import sdk "github.com/getporter/mixin-sdk-go"

func main() {
	sdk.Run(&Mixin{})
}
