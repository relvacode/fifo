package main

import (
	"context"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/relvacode/fifo"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type TaskOptions struct {
	Sources fifo.UrlMapping `short:"s" long:"source" description:"Describe input sources"`
	Targets fifo.UrlMapping `short:"t" long:"target" description:"Describe targets"`
	Shell   string          `long:"shell" default:"sh" description:"Command shell"`

	Preserve bool `long:"preserve" description:"Preserve created targets on command failure"`

	Stdin  *fifo.Url `long:"stdin" description:"Read command STDIN from this target (default: STDIN)"`
	Stdout *fifo.Url `long:"stdout" description:"Write command STDOUT to this target (default: STDOUT)"`
	Stderr *fifo.Url `long:"stderr" description:"Write command STDERR to this target (default: STDERR)"`
}

type Options struct {
	TaskOptions `group:"Task Options"`
}

func signalContext(ctx context.Context, signals ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	sig := make(chan os.Signal, len(signals))
	signal.Notify(sig, signals...)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-sig:
			cancel()
		}
	}()

	return ctx
}

func Main() (code int, mu *fifo.MultiError) {
	o := new(Options)
	p := flags.NewParser(o, flags.PassDoubleDash|flags.HelpFlag)
	p.Name = "fifo"
	p.LongDescription = "Native Cloud Streaming for Legacy Executables"
	args, err := p.Parse()
	if err != nil {
		mu = fifo.Catch(mu, err)
		return
	}

	// Setup directory to mount pipes
	temporaryLocation, err := ioutil.TempDir("", "fifo")
	if err != nil {
		mu = fifo.Catch(mu, err)
		return
	}

	defer func() {
		mu.Append(os.RemoveAll(temporaryLocation))
	}()

	t := &fifo.Task{
		Call: fifo.Call{
			Shell:       o.Shell,
			Executable:  args[0],
			Args:        args[1:],
			Environment: os.Environ(),
		},
		Preserve:       o.Preserve,
		MountDirectory: temporaryLocation,
		Providers: []fifo.Provider{
			fifo.FileProvider{
				Create: os.FileMode(0666),
			},
			&fifo.HTTPProvider{
				Client: http.DefaultClient,
			},
			&fifo.S3Provider{
				AccessKeyID:    os.Getenv("AWS_ACCESS_KEY_ID"),
				SecretKey:      os.Getenv("AWS_SECRET_ACCESS_KEY"),
				Endpoint:       os.Getenv("AWS_ENDPOINT"),
			},
		},

		Sources: o.Sources,
		Targets: o.Targets,

		Stdin:  o.Stdin,
		Stdout: o.Stdout,
		Stderr: o.Stderr,
	}

	c, err := fifo.NewCommand(t)
	if err != nil {
		mu = fifo.Catch(mu, err)
		return
	}

	// Handle cancellation signals
	ctx := signalContext(context.Background(), syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

	code, pmu := c.Start(ctx)
	mu = fifo.Catch(mu, pmu)

	return
}

func main() {
	code, err := Main()
	errs := err.Errors()
	if len(errs) > 0 {
		for _, e := range errs {
			_, _ = fmt.Fprintf(os.Stderr, "  * %v\n", e)
		}
		if code < 1 {
			code = 1
		}
	}

	os.Exit(code)
}
