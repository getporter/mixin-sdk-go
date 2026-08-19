package sdk

import "gopkg.in/yaml.v3"

// RawMessage holds a chunk of undecoded step or config data exactly as
// Porter sent it on stdin. The mixin wire protocol is YAML (Porter marshals
// the manifest with a YAML encoder before piping it to the mixin binary),
// so despite the name — kept for familiarity with encoding/json.RawMessage —
// Unmarshal decodes YAML, not JSON.
type RawMessage []byte

// Unmarshal decodes the raw YAML into v. It is a no-op returning nil when m
// is empty, so mixins with no config/step fields don't need a nil check.
func (m RawMessage) Unmarshal(v any) error {
	if len(m) == 0 {
		return nil
	}
	return yaml.Unmarshal(m, v)
}
