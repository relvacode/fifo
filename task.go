package fifo

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type Call struct {
	Executable string
	Shell      string
	Args       []string
	Environment   []string
}

func (c Call) Cmdline() (string, []string) {
	if c.Shell != "" {
		return c.Shell, []string{
			"-c",
			strings.Join(append([]string{c.Executable}, c.Args...), " "),
		}
	}

	return c.Executable, c.Args
}

type Task struct {
	Call Call
	// Preserve created target objects on failure
	Preserve       bool
	MountDirectory string
	Providers      []Provider

	// Sources provides a mapping of directory local named pipes to their equivalent URL
	Sources UrlMapping
	Targets UrlMapping

	Stdin *Url

	// Stdout is the target URL (if defined) for the output of the command
	Stdout *Url
	Stderr *Url
}

func (t *Task) Source(u *url.URL) (*SourcePipe, error) {
	rel := filepath.Join(t.MountDirectory, urlToFilename(u))

	s, err := ProvideSource(u, t.Providers...)
	if err != nil {
		return nil, err
	}

	err = syscall.Mkfifo(rel, 0666)
	if err != nil {
		return nil, err
	}

	return &SourcePipe{
		Path:   rel,
		URL:    u,
		Stream: s,
	}, nil
}

func (t *Task) Target(u *url.URL) (*TargetPipe, error) {
	rel := filepath.Join(t.MountDirectory, urlToFilename(u))

	s, err := ProvideTarget(u, t.Providers...)
	if err != nil {
		return nil, err
	}

	err = syscall.Mkfifo(rel, 0666)
	if err != nil {
		return nil, err
	}

	return &TargetPipe{
		Path:   rel,
		URL:    u,
		Stream: s,
	}, nil
}

func (t *Task) SetupInput() (io.ReadCloser, error) {
	if t.Stdin == nil {
		return os.Stdin, nil
	}
	return ProvideSource((*url.URL)(t.Stdin), t.Providers...)
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
	if t.Stdout != nil {
		stdout, err = ProvideTarget((*url.URL)(t.Stdout), t.Providers...)
		if err != nil {
			return
		}
	} else {
		stdout = &NoOpWriteDestroyCloser{Writer: os.Stdout}
	}

	if t.Stderr != nil {
		stderr, err = ProvideTarget((*url.URL)(t.Stderr), t.Providers...)
		if err != nil {
			return
		}
	} else {
		stderr = &NoOpWriteDestroyCloser{Writer: os.Stderr}
	}

	return
}
