package sdk

// BuildInput is the input passed to Mixin.Build, decoded from the mixin's
// config block in porter.yaml:
//
//	mixins:
//	- hazmat:
//	    clientVersion: "1.2.3"
//
// Config holds the raw bytes of that block (see RawMessage), or is empty
// when the mixin has no config.
type BuildInput struct {
	Config RawMessage
}
