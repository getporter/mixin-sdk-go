package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	sdk "github.com/getporter/mixin-sdk-go"
	"github.com/spf13/afero"
)

func (m *Mixin) execute(step Step) error {
	cmd := exec.Command(step.Command, buildArgs(step)...)
	if step.WorkingDir != "" && step.WorkingDir != "." {
		cmd.Dir = step.WorkingDir
	}
	if len(step.EnvironmentVars) > 0 {
		cmd.Env = append(os.Environ(), envPairs(step.EnvironmentVars)...)
	}

	var stdout, stderr bytes.Buffer
	if step.SuppressOutput {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	} else {
		cmd.Stdout = io.MultiWriter(m.Out, &stdout)
		cmd.Stderr = io.MultiWriter(m.Err, &stderr)
	}

	runErr := cmd.Run()
	if exitErr, ok := runErr.(*exec.ExitError); ok && step.IgnoreError.shouldIgnore(exitErr.ExitCode(), stderr.String()) {
		runErr = nil
	}
	if runErr != nil {
		return fmt.Errorf("error running command %q: %w", step.Command, runErr)
	}

	return m.writeOutputs(step, stdout.String())
}

func buildArgs(step Step) []string {
	args := make([]string, 0, len(step.Arguments)+2*len(step.Flags)+len(step.SuffixArguments))
	args = append(args, step.Arguments...)
	args = append(args, step.Flags.ToSlice()...)
	args = append(args, step.SuffixArguments...)
	return args
}

func envPairs(vars map[string]string) []string {
	pairs := make([]string, 0, len(vars))
	for k, v := range vars {
		pairs = append(pairs, k+"="+v)
	}
	return pairs
}

func (h IgnoreError) shouldIgnore(exitCode int, stderr string) bool {
	if h.All {
		return true
	}
	if slices.Contains(h.ExitCodes, exitCode) {
		return true
	}
	for _, s := range h.Output.Contains {
		if strings.Contains(stderr, s) {
			return true
		}
	}
	for _, pattern := range h.Output.Regex {
		if re, err := regexp.Compile(pattern); err == nil && re.MatchString(stderr) {
			return true
		}
	}
	return false
}

// writeOutputs resolves each of step's outputs (from stdout via regex or
// jsonPath, or from a file the command wrote) and writes it to sdk.OutputsDir
// under its output name, in the shape Porter expects to find it in.
func (m *Mixin) writeOutputs(step Step, stdout string) error {
	if len(step.Outputs) == 0 {
		return nil
	}

	var stdoutJSON any
	var decodeErr error
	decodeStdoutJSON := func() (any, error) {
		if stdoutJSON == nil && decodeErr == nil && stdout != "" {
			decodeErr = json.Unmarshal([]byte(stdout), &stdoutJSON)
		}
		return stdoutJSON, decodeErr
	}

	if err := m.FileSystem.MkdirAll(sdk.OutputsDir, 0755); err != nil {
		return fmt.Errorf("could not create outputs directory %s: %w", sdk.OutputsDir, err)
	}

	for _, o := range step.Outputs {
		var value []byte
		switch {
		case o.JSONPath != "":
			doc, err := decodeStdoutJSON()
			if err != nil {
				return fmt.Errorf("output %q: stdout is not valid JSON: %w", o.Name, err)
			}
			result, err := jsonpath.Get(o.JSONPath, doc)
			if err != nil {
				return fmt.Errorf("output %q: jsonPath %q: %w", o.Name, o.JSONPath, err)
			}
			if s, ok := result.(string); ok {
				value = []byte(s)
			} else if value, err = json.Marshal(result); err != nil {
				return fmt.Errorf("output %q: %w", o.Name, err)
			}

		case o.Regex != "":
			re, err := regexp.Compile(o.Regex)
			if err != nil {
				return fmt.Errorf("output %q: invalid regex %q: %w", o.Name, o.Regex, err)
			}
			var matches []string
			for _, sm := range re.FindAllStringSubmatch(stdout, -1) {
				if len(sm) > 1 {
					matches = append(matches, sm[1:]...)
				}
			}
			value = []byte(strings.Join(matches, "\n"))

		case o.FilePath != "":
			b, err := afero.ReadFile(m.FileSystem, o.FilePath)
			if err != nil {
				return fmt.Errorf("output %q: reading %s: %w", o.Name, o.FilePath, err)
			}
			value = b

		default:
			continue
		}

		if err := afero.WriteFile(m.FileSystem, filepath.Join(sdk.OutputsDir, o.Name), value, 0644); err != nil {
			return fmt.Errorf("could not write output %q: %w", o.Name, err)
		}
	}
	return nil
}
