package fifo

import (
	"bytes"
	"context"
	"github.com/pkg/errors"
	"os/exec"
	"text/template"
)

func NewCommand(t *Task) (*Command, error) {
	return &Command{
		t: t,
	}, nil
}

type Command struct {
	t *Task
}

// replace the given arguments with a template context
func replace(cx map[string]string, args []string) ([]string, error) {
	// TODO ensure that all context arguments are consumed
	replaced := make([]string, len(args))
	var b bytes.Buffer

	for i, a := range args {
		t, err := template.New("").Parse(a)
		if err != nil {
			return nil, err
		}

		err = t.Execute(&b, cx)
		if err != nil {
			return nil, err
		}

		replaced[i] = b.String()
		b.Reset()
	}

	return replaced, nil
}

func destroy(mu *MultiError, targets ...WriteDestroyCloser) {
	if mu != nil && len(mu.err) > 0 {
		for _, tg := range targets {
			mu = Catch(mu, tg.Destroy())
		}
	}
}

func (c *Command) Start(ctx context.Context) (mu *MultiError) {
	mu = new(MultiError)

	sources, err := c.t.SetupSourcePipes()
	if err != nil {
		mu.Append(err)
		return
	}

	defer mu.CatchMulti(sources.Teardown)

	targets, err := c.t.SetupTargetPipes()
	if err != nil {
		mu.Append(err)
		return
	}

	// Destroy any created targets on failure
	if len(targets) > 0 && !c.t.Preserve {
		var onDestroy []WriteDestroyCloser
		for _, tg := range targets {
			onDestroy = append(onDestroy, tg.Stream)
		}
		defer destroy(mu, onDestroy...)
	}

	defer mu.CatchMulti(targets.Teardown)

	bin, args := c.t.Call.Cmdline()

	// build template context from available sources and targets
	cx := make(map[string]string)
	for k, v := range sources {
		cx[k] = v.Path
	}
	for k, v := range targets {
		_, ok := cx[k]
		if ok {
			mu.Append(errors.Errorf("%q cannot be defined as both a source and a target", k))
			return
		}
		cx[k] = v.Path
	}

	args, err = replace(cx, args)
	if err != nil {
		mu.Append(err)
		return
	}

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
	defer destroy(mu, stdout, stderr)
	// Close stdout and stderr when done
	defer mu.Catch(stdout.Close, stderr.Close)

	p := exec.CommandContext(ctx, bin, args...)
	p.Stdin = stdin
	p.Stdout = stdout
	p.Stderr = stderr

	ctx, g := NewGroup(ctx)

	if len(targets) > 0 {
		// the named pipe for receiving data from the command needs to be setup before the command starts
		g.Go(func() (mu *MultiError) {
			mu = Catch(mu, targets.Copy(ctx))
			return
		})
	}

	if mu.Catch(p.Start) {
		return
	}

	if len(sources) > 0 {
		// named pipe for writing data to the command needs to be setup after the command starts
		g.Go(func() (mu *MultiError) {
			mu = Catch(mu, sources.Copy(ctx))
			return
		})
	}

	// wait from the copy groups to complete
	mu.Append(g.Wait())

	// wait for the command to complete
	mu.Append(p.Wait())
	return
}
