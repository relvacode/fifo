package fifo

import (
	"context"
	"golang.org/x/sync/errgroup"
	"io"
	"net/url"
	"os"
)

type SourcePipe struct {
	Name   string
	Path   string
	URL    *url.URL
	Stream io.ReadCloser
}

type Sources []*SourcePipe

// Copy starts copying data from each source stream to each existing named pipe
func (s Sources) Copy(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, src := range s {
		pipe, err := os.OpenFile(src.Path, os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}

		g.Go(func() error {
			_, err := io.Copy(pipe, src.Stream)
			return Catch(nil, err, pipe.Close(), src.Stream.Close()).ErrorOrNil()
		})
	}

	return g.Wait()
}

func (s Sources) Teardown() (mu *MultiError) {
	for _, src := range s {
		mu = Catch(mu, os.Remove(src.Path))
	}
	return
}

type TargetPipe struct {
	Name   string
	Path   string
	URL    *url.URL
	Stream WriteDestroyCloser
}

type Targets []*TargetPipe

func (t Targets) Copy(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, tg := range t {
		pipe, err := os.OpenFile(tg.Path, os.O_RDONLY, 0600)
		if err != nil {
			return err
		}

		g.Go(func() error {
			_, err := io.Copy(tg.Stream, pipe)
			return Catch(nil, err, tg.Stream.Close(), pipe.Close()).ErrorOrNil()
		})
	}

	return g.Wait()
}

func (t Targets) Teardown() (mu *MultiError) {
	for _, target := range t {
		mu = Catch(mu, os.Remove(target.Path))
	}
	return
}
