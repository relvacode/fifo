package fifo

import (
	"context"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"os/exec"
	"syscall"
)

func NewCommand(t *Task) (*Command, error) {
	return &Command{
		t: t,
	}, nil
}

type Command struct {
	t *Task
}

func destroyWhenError(mu *MultiError, targets ...WriteDestroyCloser) {
	if mu != nil && len(mu.err) > 0 {
		for _, tg := range targets {
			mu = Catch(mu, tg.Destroy())
		}
	}
}

// wait for a given command to finish and collect its exit code.
func wait(p *exec.Cmd) (int, error) {
	err := p.Wait()
	if err == nil {
		return 0, nil
	}

	ex, ok := err.(*exec.ExitError)
	if !ok {
		return 1, err
	}

	return ex.ExitCode(), nil
}

func (c *Command) Start(ctx context.Context) (code int, mu *MultiError) {
	mu = new(MultiError)

	gen := &TemplateGenerator{
		Provider:   c.t,
		SourceTags: c.t.Sources,
		TargetTags: c.t.Targets,
	}

	args, err := gen.Replace(c.t.Call.Args)
	if err != nil {
		mu.Append(err)
		return
	}

	defer mu.CatchMulti(gen.Sources.Teardown)

	// Destroy any created targets on failure
	defer func() {
		if len(gen.Targets) > 0 && !c.t.Preserve {
			for _, tg := range gen.Targets {
				destroyWhenError(mu, tg.Stream)
			}
		}
	}()

	defer mu.CatchMulti(gen.Targets.Teardown)

	stdin, err := c.t.SetupInput()
	if err != nil {
		mu.Append(errors.Wrap(err, "unable to setup input"))
		return
	}

	defer mu.Catch(stdin.Close)

	// Setup output for stdout and stderr
	stdout, stderr, err := c.t.SetupOutput()
	if err != nil {
		mu.Append(errors.Wrap(err, "unable to setup output"))
		return
	}

	// Destroy stdout and stderr on error
	defer destroyWhenError(mu, stdout, stderr)

	// Close stdout and stderr when done
	defer mu.Catch(stdout.Close, stderr.Close)

	p := exec.CommandContext(ctx, c.t.Call.Executable, args...)
	p.Stdin = stdin
	p.Stdout = stdout
	p.Stderr = stderr
	p.Env = c.t.Call.Environment
	if c.t.Call.WorkingDirectory != "" {
		p.Dir = c.t.Call.WorkingDirectory
	}

	g, ctx := errgroup.WithContext(ctx)

	if len(gen.Targets) > 0 {
		// the named pipe for receiving data from the command needs to be setup before the command starts
		g.Go(func() error {
			return gen.Targets.Copy(ctx)
		})
	}

	if mu.Catch(p.Start) {
		return
	}

	if len(gen.Sources) > 0 {
		// named pipe for writing data to the command needs to be setup after the command starts
		g.Go(func() error {
			return gen.Sources.Copy(ctx)
		})
	}

	// wait from the copy groups to complete
	mu.Append(g.Wait())

	// If we had errors processing IO then signal the process to prematurely SIGTERM
	if len(mu.Errors()) > 0 {
		mu.Append(p.Process.Signal(syscall.SIGTERM))
	}

	// wait for the command to complete and capture the error code
	code, err = wait(p)
	mu.Append(err)
	return
}
