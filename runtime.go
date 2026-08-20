package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Run wires m up to Porter's mixin CLI protocol, reading os.Args and
// os.Stdin and writing to os.Stdout/os.Stderr, then calls os.Exit with the
// resulting exit code. Call it from main only:
//
//	func main() {
//	    sdk.Run(&HazmatMixin{})
//	}
func Run(m Mixin) {
	os.Exit(Execute(m, os.Args[1:], newOSContext()))
}

// Execute builds m's cobra command tree and runs it with the given args and
// Context, returning a process exit code (0 on success, 1 on error). Unlike
// Run, it never calls os.Exit, which makes it suitable for tests.
func Execute(m Mixin, args []string, rtCtx *Context) int {
	cmd := newRootCommand(m, rtCtx)
	cmd.SetArgs(args)
	cmd.SetIn(rtCtx.In)
	cmd.SetOut(rtCtx.Out)
	cmd.SetErr(rtCtx.Err)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(rtCtx.Err, err)
		return 1
	}
	return 0
}

func newRootCommand(m Mixin, rtCtx *Context) *cobra.Command {
	if ca, ok := m.(ContextAware); ok {
		ca.SetContext(rtCtx)
	}

	cmd := &cobra.Command{
		Use:           m.Version().Name,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().BoolVar(&rtCtx.Debug, "debug", false, "Enable debug logging")

	cmd.AddCommand(
		newVersionCommand(m, rtCtx),
		newSchemaCommand(m, rtCtx),
		newBuildCommand(m, rtCtx),
		newLintCommand(m, rtCtx),
		newStepCommand("install", rtCtx, m.Install),
		newStepCommand("upgrade", rtCtx, m.Upgrade),
		newStepCommand("uninstall", rtCtx, m.Uninstall),
		newInvokeCommand(m, rtCtx),
	)

	return cmd
}

func newVersionCommand(m Mixin, rtCtx *Context) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the mixin version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printVersion(rtCtx.Out, m.Version(), format)
		},
	}
	cmd.Flags().StringVarP(&format, "output", "o", "plaintext", "Specify an output format. Allowed values: json, plaintext")
	return cmd
}

func newSchemaCommand(m Mixin, rtCtx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print the JSON schema for the mixin",
		RunE: func(cmd *cobra.Command, args []string) error {
			schema, err := m.Schema()
			if err != nil {
				return err
			}
			_, err = rtCtx.Out.Write(schema)
			return err
		},
	}
}

func newBuildCommand(m Mixin, rtCtx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Generate Dockerfile lines for the bundle invocation image",
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := readBuildInput(rtCtx)
			if err != nil {
				return err
			}
			return m.Build(input, rtCtx.Out)
		},
	}
}

func newLintCommand(m Mixin, rtCtx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "lint",
		Short: "Check sections of the bundle associated with this mixin for problems",
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := readBuildInput(rtCtx)
			if err != nil {
				return err
			}
			results, err := m.Lint(input)
			if err != nil {
				return err
			}
			return json.NewEncoder(rtCtx.Out).Encode(results)
		},
	}
}

// readBuildInput parses the stdin shape Porter sends both build and lint:
//
//	config: {...}
//	actions:
//	  install: [...]
//	  upgrade: [...]
func readBuildInput(rtCtx *Context) (BuildInput, error) {
	doc, err := io.ReadAll(rtCtx.In)
	if err != nil {
		return BuildInput{}, fmt.Errorf("could not read build input from stdin: %w", err)
	}
	cfg, err := extractRawYAML(doc, "config")
	if err != nil {
		return BuildInput{}, fmt.Errorf("could not parse build input: %w", err)
	}
	actions, err := extractRawYAML(doc, "actions")
	if err != nil {
		return BuildInput{}, fmt.Errorf("could not parse build input: %w", err)
	}
	return BuildInput{Config: cfg, Actions: actions}, nil
}

// newStepCommand builds the install/upgrade/uninstall commands, which all
// share the same stdin shape and dispatch to a StepInput-taking method.
func newStepCommand(name string, rtCtx *Context, handler func(StepInput) error) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Execute the %s functionality of this mixin", name),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := readStepDoc(rtCtx, file)
			if err != nil {
				return err
			}
			input, err := decodeStepInput(doc)
			if err != nil {
				return err
			}
			return handler(input)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to a file containing the step definition, instead of reading from stdin")
	return cmd
}

func newInvokeCommand(m Mixin, rtCtx *Context) *cobra.Command {
	var file, action string
	cmd := &cobra.Command{
		Use:   "invoke",
		Short: "Execute a custom action",
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := readStepDoc(rtCtx, file)
			if err != nil {
				return err
			}
			input, err := decodeStepInput(doc)
			if err != nil {
				return err
			}
			input.Action = action
			return m.Invoke(action, input)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to a file containing the step definition, instead of reading from stdin")
	cmd.Flags().StringVar(&action, "action", "", "Custom action name to invoke")
	return cmd
}

func readStepDoc(rtCtx *Context, file string) ([]byte, error) {
	if file == "" {
		doc, err := io.ReadAll(rtCtx.In)
		if err != nil {
			return nil, fmt.Errorf("could not read step input from stdin: %w", err)
		}
		return doc, nil
	}
	doc, err := afero.ReadFile(rtCtx.FileSystem, file)
	if err != nil {
		return nil, fmt.Errorf("could not read step input from %s: %w", file, err)
	}
	return doc, nil
}

// extractRawYAML pulls the raw bytes of doc[key] back out as their own YAML
// document, without decoding into a concrete type.
func extractRawYAML(doc []byte, key string) (RawMessage, error) {
	if len(doc) == 0 {
		return nil, nil
	}
	var root map[string]yaml.Node
	if err := yaml.Unmarshal(doc, &root); err != nil {
		return nil, err
	}
	node, ok := root[key]
	if !ok {
		return nil, nil
	}
	b, err := yaml.Marshal(&node)
	if err != nil {
		return nil, err
	}
	return RawMessage(b), nil
}

// decodeStepInput parses the wire shape Porter sends install/upgrade/
// uninstall/invoke on stdin:
//
//	<action>:
//	- <mixinName>:
//	    <step fields...>
//
// exactly one action, containing exactly one step, as Porter always
// resolves a manifest down to a single step before invoking the mixin.
func decodeStepInput(doc []byte) (StepInput, error) {
	var root map[string]yaml.Node
	if err := yaml.Unmarshal(doc, &root); err != nil {
		return StepInput{}, fmt.Errorf("could not parse step input: %w", err)
	}
	if len(root) != 1 {
		return StepInput{}, fmt.Errorf("expected exactly one action in step input, got %d", len(root))
	}

	var action string
	var stepsNode yaml.Node
	for k, v := range root {
		action, stepsNode = k, v
	}

	if stepsNode.Kind != yaml.SequenceNode || len(stepsNode.Content) != 1 {
		return StepInput{}, fmt.Errorf("expected exactly one step for action %q, got %d", action, len(stepsNode.Content))
	}

	stepNode := stepsNode.Content[0]
	if stepNode.Kind != yaml.MappingNode || len(stepNode.Content) != 2 {
		return StepInput{}, fmt.Errorf("malformed step for action %q: expected a single {mixin: {...}} entry", action)
	}

	// stepNode.Content is [keyNode, valueNode]: the mixin name and its
	// instruction data. The mixin name itself is redundant here — this
	// binary only ever runs its own steps.
	data, err := yaml.Marshal(stepNode.Content[1])
	if err != nil {
		return StepInput{}, err
	}
	return StepInput{Action: action, Data: data}, nil
}
