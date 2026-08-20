package main

import (
	"fmt"
	"strings"

	sdk "github.com/getporter/mixin-sdk-go"
)

// exec-100/101 mirror Porter's built-in exec mixin: flag a bare `bash -c`
// invocation as a best-practice warning, and hard-error when its argument
// isn't wrapped in quotes (a common source of subtle YAML/shell escaping
// bugs).
const (
	codeEmbeddedBash          = "exec-100"
	codeBashCArgMissingQuotes = "exec-101"
)

// actionEntry is one {exec: {...}} entry in an action's step list.
type actionEntry struct {
	Exec Step `yaml:"exec"`
}

func (m *Mixin) Lint(input sdk.BuildInput) (sdk.LintResults, error) {
	var actions map[string][]actionEntry
	if err := input.Actions.Unmarshal(&actions); err != nil {
		return nil, fmt.Errorf("could not parse actions for lint: %w", err)
	}

	var results sdk.LintResults
	for actionName, entries := range actions {
		for i, entry := range entries {
			step := entry.Exec
			if step.Command != "bash" {
				continue
			}
			cFlag, ok := step.Flags["c"]
			if !ok {
				continue
			}

			loc := sdk.LintLocation{
				Action:          actionName,
				Mixin:           "exec",
				StepNumber:      i + 1,
				StepDescription: step.Description,
			}
			results = append(results, sdk.LintResult{
				Level:    sdk.LintLevelWarning,
				Location: loc,
				Code:     codeEmbeddedBash,
				Title:    "Best Practice: Avoid Embedded Bash",
				URL:      "https://porter.sh/best-practices/exec-mixin/#use-scripts",
			})

			for _, v := range cFlag {
				if isQuoted(v) {
					continue
				}
				results = append(results, sdk.LintResult{
					Level:    sdk.LintLevelError,
					Location: loc,
					Code:     codeBashCArgMissingQuotes,
					Title:    "bash -c argument missing wrapping quotes",
					Message: `The bash -c flag argument must be wrapped in quotes, for example
exec:
  description: Say Hello
  command: bash
  flags:
    c: '"echo Hello World"'
`,
					URL: "https://porter.sh/best-practices/exec-mixin/#quoting-escaping-bash-and-yaml",
				})
				break
			}
		}
	}

	return results, nil
}

func isQuoted(s string) bool {
	return (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`))
}
