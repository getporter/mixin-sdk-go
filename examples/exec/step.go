package main

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// Step is the shape of a single exec instruction in porter.yaml, e.g.:
//
//	install:
//	- exec:
//	    description: "..."
//	    command: "..."
//
// StepInput.Data (see sdk.StepInput) holds everything under the "exec" key,
// so Step maps directly onto it without an extra wrapping layer.
type Step struct {
	Description     string            `yaml:"description"`
	Command         string            `yaml:"command"`
	WorkingDir      string            `yaml:"dir,omitempty"`
	Arguments       []string          `yaml:"arguments,omitempty"`
	SuffixArguments []string          `yaml:"suffix-arguments,omitempty"`
	Flags           Flags             `yaml:"flags,omitempty"`
	EnvironmentVars map[string]string `yaml:"envs,omitempty"`
	Outputs         []Output          `yaml:"outputs,omitempty"`
	SuppressOutput  bool              `yaml:"suppress-output,omitempty"`
	IgnoreError     IgnoreError       `yaml:"ignoreError,omitempty"`
}

// Output describes how to capture one named output from the command's
// stdout (via regex or jsonPath) or from a file it wrote (via path).
// Exactly one of these is expected to be set per output.
type Output struct {
	Name     string `yaml:"name"`
	FilePath string `yaml:"path,omitempty"`
	JSONPath string `yaml:"jsonPath,omitempty"`
	Regex    string `yaml:"regex,omitempty"`
}

// IgnoreError lets a step tolerate a failed command under conditions given
// in porter.yaml, instead of always failing the bundle action.
type IgnoreError struct {
	All       bool  `yaml:"all,omitempty"`
	ExitCodes []int `yaml:"exitCodes,omitempty"`
	Output    struct {
		Contains []string `yaml:"contains,omitempty"`
		Regex    []string `yaml:"regex,omitempty"`
	} `yaml:"output,omitempty"`
}

// Flags holds a step's command-line flags. Values are a slice because a
// flag may repeat (flags: {tag: [a, b]} -> --tag a --tag b).
type Flags map[string][]string

// UnmarshalYAML accepts both a scalar and a list per flag:
//
//	flags:
//	  detach: true
//	  tag: [a, b]
func (f *Flags) UnmarshalYAML(value *yaml.Node) error {
	var raw map[string]yaml.Node
	if err := value.Decode(&raw); err != nil {
		return fmt.Errorf("could not unmarshal flags: %w", err)
	}

	*f = make(Flags, len(raw))
	for name, node := range raw {
		if node.Kind == yaml.SequenceNode {
			var values []string
			if err := node.Decode(&values); err != nil {
				return fmt.Errorf("could not unmarshal flag %q: %w", name, err)
			}
			(*f)[name] = values
			continue
		}

		var v string
		if err := node.Decode(&v); err != nil {
			return fmt.Errorf("could not unmarshal flag %q: %w", name, err)
		}
		(*f)[name] = []string{v}
	}
	return nil
}

// ToSlice renders flags as command-line arguments, e.g. {"tag": ["a"]} ->
// ["--tag", "a"], sorted by name for deterministic output.
func (f Flags) ToSlice() []string {
	names := make([]string, 0, len(f))
	for name := range f {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]string, 0, 2*len(f))
	for _, name := range names {
		dash := "--"
		if len(name) == 1 {
			dash = "-"
		}
		flag := dash + name

		values := f[name]
		if len(values) == 0 {
			result = append(result, flag)
			continue
		}
		for _, v := range values {
			result = append(result, flag, v)
		}
	}
	return result
}
