package sdk

import (
	"encoding/json"
	"errors"
)

// ValidateSchema reports whether schema is well-formed JSON. Mixin authors
// embedding a schema.json with go:embed can call this in a test to catch a
// broken schema before Porter does.
func ValidateSchema(schema []byte) error {
	if !json.Valid(schema) {
		return errors.New("schema is not valid JSON")
	}
	return nil
}
