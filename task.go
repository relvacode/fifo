package fifo

import (
	"github.com/pkg/errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type Call struct {
	Binary string
	Args   []string
	Shell  string
}

func (c Call) Cmdline() (string, []string) {
	if c.Shell != "" {
		return c.Shell, []string{
			"-c",
			strings.Join(append([]string{c.Binary}, c.Args...), " "),
		}
	}

	return c.Binary, c.Args
}

type Task struct {
	Call Call
	// Preserve created target objects on failure
	Preserve       bool
	MountDirectory string
	Providers      []Provider

	// Sources provides a mapping of directory local named pipes to their equivalent URL
	Sources map[string]string

	Targets map[string]string

	Stdin *string

	// Stdout is the target URL (if defined) for the output of the command
	Stdout *string
	Stderr *string
}

func (t *Task) SetupSourcePipes() (Sources, error) {
	sources := make(Sources, len(t.Sources))
	for name, urn := range t.Sources {
		u, err := url.Parse(urn)
		if err != nil {
			return nil, err
		}

		rel := filepath.Join(t.MountDirectory, name)

		s, err := ProvideSource(u, t.Providers...)
		if err != nil {
			return nil, errors.Wrap(err, name)
		}

		err = syscall.Mkfifo(rel, 0666)
		if err != nil {
			return nil, errors.Wrapf(err, "mkfifo on %q", name)
		}

		sources[name] = &SourcePipe{
			Path:   rel,
			URL:    u,
			Stream: s,
		}
	}

	return sources, nil
}

func (t *Task) SetupTargetPipes() (Targets, error) {
	targets := make(Targets, len(t.Targets))
	for name, urn := range t.Targets {
		u, err := url.Parse(urn)
		if err != nil {
			return nil, err
		}

		rel := filepath.Join(t.MountDirectory, name)

		s, err := ProvideTarget(u, t.Providers...)
		if err != nil {
			return nil, errors.Wrap(err, name)
		}

		err = syscall.Mkfifo(rel, 0666)
		if err != nil {
			return nil, errors.Wrapf(err, "mkfifo on %q", name)
		}

		targets[name] = &TargetPipe{
			Path:   rel,
			URL:    u,
			Stream: s,
		}
	}

	return targets, nil
}

func (t *Task) SetupInput() (io.ReadCloser, error) {
	if t.Stdin == nil {
		return os.Stdin, nil
	}

	u, err := url.Parse(*t.Stdin)
	if err != nil {
		return nil, err
	}

	return ProvideSource(u, t.Providers...)
}

type NoOpWriteDestroyCloser struct {
	io.Writer
}

func (NoOpWriteDestroyCloser) Close() error {
	return nil
}

func (NoOpWriteDestroyCloser) Destroy() error {
	return nil
}

func (t *Task) SetupOutput() (stdout WriteDestroyCloser, stderr WriteDestroyCloser, err error) {
	var u *url.URL
	if t.Stdout != nil {
		u, err = url.Parse(*t.Stdout)
		if err != nil {
			return
		}
		stdout, err = ProvideTarget(u, t.Providers...)
		if err != nil {
			return
		}
	} else {
		stdout = &NoOpWriteDestroyCloser{Writer: os.Stdout}
	}

	if t.Stderr != nil {
		u, err = url.Parse(*t.Stderr)
		if err != nil {
			return
		}
		stderr, err = ProvideTarget(u, t.Providers...)
		if err != nil {
			return
		}
	} else {
		stderr = &NoOpWriteDestroyCloser{Writer: os.Stderr}
	}

	return
}
