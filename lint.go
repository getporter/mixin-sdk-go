package sdk

// LintLevel is the severity of a LintResult. The values (not just the
// names) are part of the wire contract: Porter decodes the `lint` command's
// JSON output straight into its own equivalent type, matching by these
// same numbers.
type LintLevel int

const (
	// LintLevelError indicates a problem that will prevent the bundle
	// from building properly.
	LintLevelError LintLevel = 0

	// LintLevelWarning indicates a best-practice or non-fatal problem.
	LintLevelWarning LintLevel = 2
)

// LintLocation identifies where in the manifest a LintResult applies.
type LintLocation struct {
	// Action containing the step, e.g. "install".
	Action string

	// Mixin name, e.g. "hazmat".
	Mixin string

	// StepNumber is the position of the step within the action, starting
	// from 1.
	StepNumber int

	// StepDescription is the step's description field from the manifest.
	StepDescription string
}

// LintResult is a single problem identified by Linter.Lint.
type LintResult struct {
	Level    LintLevel
	Location LintLocation

	// Code uniquely identifies the type of problem. Recommended pattern
	// is MIXIN-NUMBER, e.g. "hazmat-100", to avoid colliding with codes
	// from another mixin or from Porter itself.
	Code string

	// Title to display (80 chars).
	Title string

	// Message explaining the problem.
	Message string

	// URL with more information about this problem, if any.
	URL string
}

// LintResults is the full set of problems Linter.Lint identified.
type LintResults []LintResult
