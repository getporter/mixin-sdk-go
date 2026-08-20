// Package testing provides in-memory test doubles for unit testing a
// sdk.Mixin implementation without spawning a subprocess: no real stdin,
// stdout/stderr, or filesystem is touched. Since its import path's last
// element shadows the standard library's testing package, most callers
// import it under an alias:
//
//	mixintesting "github.com/getporter/mixin-sdk-go/testing"
package testing

import (
	"bytes"
	"path/filepath"
	"strings"

	sdk "github.com/getporter/mixin-sdk-go"
	"github.com/spf13/afero"
)

// Context is an in-memory sdk.Context: empty stdin (replace it with
// SetStdin), buffered stdout/stderr (read with Stdout/Stderr), and an
// in-memory filesystem.
type Context struct {
	*sdk.Context

	out *bytes.Buffer
	err *bytes.Buffer
}

// NewContext returns a fresh in-memory Context, suitable for embedding in
// a Mixin under test (via sdk.ContextAware / embedding sdk.Context by
// value) or passing to Execute.
func NewContext() *Context {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	return &Context{
		Context: &sdk.Context{
			In:         strings.NewReader(""),
			Out:        out,
			Err:        errOut,
			FileSystem: afero.NewMemMapFs(),
			Getenv:     func(string) string { return "" },
		},
		out: out,
		err: errOut,
	}
}

// SetStdin replaces stdin with content, e.g. the wire-format YAML for a
// StepInput or BuildInput built with this package's fixture helpers.
func (c *Context) SetStdin(content string) {
	c.In = strings.NewReader(content)
}

// Stdout returns everything written to Context.Out so far.
func (c *Context) Stdout() string { return c.out.String() }

// Stderr returns everything written to Context.Err so far.
func (c *Context) Stderr() string { return c.err.String() }

// WriteFile seeds the in-memory filesystem with content at path, e.g. a
// file a Mixin method under test is expected to read.
func (c *Context) WriteFile(path, content string) error {
	return afero.WriteFile(c.FileSystem, path, []byte(content), 0o644)
}

// ReadFile reads a file back from the in-memory filesystem.
func (c *Context) ReadFile(path string) (string, error) {
	b, err := afero.ReadFile(c.FileSystem, path)
	return string(b), err
}

// Output reads a single mixin output by name from sdk.OutputsDir, e.g.
// after calling a Mixin method expected to write one via
// afero.WriteFile(ctx.FileSystem, filepath.Join(sdk.OutputsDir, name), ...).
func (c *Context) Output(name string) (string, error) {
	return c.ReadFile(filepath.Join(sdk.OutputsDir, name))
}
