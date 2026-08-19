package sdk

import (
	"encoding/json"
	"fmt"
	"io"
)

// VersionInfo identifies a mixin and its build version, returned by
// Mixin.Version and printed by the `version` command.
type VersionInfo struct {
	Name    string `json:"name"`
	Author  string `json:"author,omitempty"`
	Version string `json:"version"` // populated at build time via ldflags, not hardcoded
}

// printVersion implements the `version` command's stdout contract: a
// human-readable line for the plaintext format (the default), or a JSON
// object for the json format.
func printVersion(out io.Writer, info VersionInfo, format string) error {
	switch format {
	case "", "plaintext":
		authorship := ""
		if info.Author != "" {
			authorship = " by " + info.Author
		}
		_, err := fmt.Fprintf(out, "%s %s%s\n", info.Name, info.Version, authorship)
		return err
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	default:
		return fmt.Errorf("unsupported output format %q: expected json or plaintext", format)
	}
}
