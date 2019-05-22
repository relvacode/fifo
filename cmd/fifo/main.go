package main

import (
	"context"
	"github.com/jessevdk/go-flags"
	"github.com/relvacode/fifo"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"os"
)

type Options struct {
	Sources map[string]string `short:"s" long:"source" description:"Describe input sources"`
	Targets map[string]string `short:"t" long:"target" description:"Describe targets"`
	Shell   string            `long:"shell" default:"sh" description:"Command shell"`

	Preserve      bool   `long:"preserve" description:"Preserve created targets on command failure"`
	PipeDirectory string `long:"pipes" description:"Location on filesystem to mount pipes (default: tmp)"`

	Stdin  *string `long:"stdin" description:"Read command STDIN from this target (default: STDIN)"`
	Stdout *string `long:"stdout" description:"Write command STDOUT to this target (default: STDOUT)"`
	Stderr *string `long:"stderr" description:"Write command STDERR to this target (default: STDERR)"`
}

func Main() error {
	o := new(Options)
	p := flags.NewParser(o, flags.PassDoubleDash|flags.HelpFlag)
	p.Name = "FiFo"
	p.ShortDescription = "Native Cloud Streaming for Legacy Tooling"
	args, err := p.Parse()
	if err != nil {
		return err
	}

	// Setup directory to mount pipes
	var mountOn = o.PipeDirectory
	if mountOn == "" {
		d, err := ioutil.TempDir("", "fifo")
		if err != nil {
			return err
		}

		defer os.RemoveAll(d)
	}

	t := &fifo.Task{
		Call: fifo.Call{
			Binary: args[0],
			Args:   args[1:],
			Shell:  o.Shell,
		},
		Preserve:       o.Preserve,
		MountDirectory: mountOn,
		Providers: []fifo.Provider{
			fifo.FileProvider{
				Create: os.FileMode(0666),
			},
			&fifo.S3Provider{
				AccessKeyID: os.Getenv("AWS_ACCESS_KEY_ID"),
				SecretKey:   os.Getenv("AWS_SECRET_ACCESS_KEY"),
				Endpoint:    os.Getenv("AWS_ENDPOINT"),
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
		return err
	}

	errs := c.Start(context.Background()).Errors()
	if len(errs) > 0 {
		for _, err = range errs[:len(errs)-1] {
			logrus.Error(err)
		}
		return errs[len(errs)-1]
	}
	return nil
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	err := Main()
	if err != nil {
		log.Fatal(err)
	}
}
