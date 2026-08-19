package sdk

import (
	"io"
	"os"

	"github.com/spf13/afero"
)

// Context provides the stdin/stdout/stderr streams and filesystem access a
// Mixin needs while a command runs. Run wires one up to the real OS before
// dispatching a command; the testing package provides an in-memory
// substitute for unit tests.
type Context struct {
	In         io.Reader
	Out        io.Writer
	Err        io.Writer
	FileSystem afero.Fs
	Getenv     func(string) string

	// Debug reports whether the mixin was invoked with --debug.
	Debug bool
}

// newOSContext returns a Context wired to the real stdin/stdout/stderr and
// OS filesystem.
func newOSContext() *Context {
	return &Context{
		In:         os.Stdin,
		Out:        os.Stdout,
		Err:        os.Stderr,
		FileSystem: afero.NewOsFs(),
		Getenv:     os.Getenv,
	}
}

// ContextAware is implemented by a Mixin that wants Run's wired-up Context
// injected before any command executes. Embedding Context by value in your
// Mixin struct satisfies this automatically.
type ContextAware interface {
	SetContext(*Context)
}

// SetContext installs other's fields into c, so a Mixin that embeds Context
// by value satisfies ContextAware without writing its own method.
func (c *Context) SetContext(other *Context) {
	*c = *other
}
