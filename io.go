package fifo

import (
	"context"
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
	ctx, g := NewGroup(ctx)
	for _, src := range s {
		pipe, err := os.OpenFile(src.Path, os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}

		g.Go(func() (mu *MultiError) {
			_, err := io.Copy(pipe, src.Stream)
			mu = Catch(mu, err)
			mu = Catch(mu, pipe.Close())
			mu = Catch(mu, src.Stream.Close())
			return
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
	ctx, g := NewGroup(ctx)
	for _, tg := range t {
		pipe, err := os.OpenFile(tg.Path, os.O_RDONLY, 0600)
		if err != nil {
			return err
		}

		g.Go(func() (mu *MultiError) {
			_, err := io.Copy(tg.Stream, pipe)
			mu = Catch(mu, err)
			mu = Catch(mu, tg.Stream.Close())
			mu = Catch(mu, pipe.Close())
			return
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
