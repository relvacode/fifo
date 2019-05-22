package fifo

import (
	"context"
	"github.com/pkg/errors"
	"io"
	"net/url"
	"os"
)

type SourcePipe struct {
	Path   string
	URL    *url.URL
	Stream io.ReadCloser
}

type Sources map[string]*SourcePipe

// Copy starts copying data from each source stream to each existing named pipe
func (s Sources) Copy(ctx context.Context) error {
	ctx, g := NewGroup(ctx)
	for name, src := range s {
		pipe, err := os.OpenFile(src.Path, os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return errors.Wrapf(err, "open %q", name)
		}

		g.Go(func() (mu *MultiError) {
			_, err := io.Copy(pipe, src.Stream)
			mu = Catch(mu, err)
			mu = Catch(mu, pipe.Close())
			return
		})
	}

	return g.Wait()
}

func (s Sources) Teardown() (mu *MultiError) {
	for _, src := range s {
		mu = Catch(mu, src.Stream.Close())
		mu = Catch(mu, os.Remove(src.Path))
	}
	return
}

type TargetPipe struct {
	Path   string
	URL    *url.URL
	Stream WriteDestroyCloser
}

type Targets map[string]*TargetPipe

func (t Targets) Copy(ctx context.Context) error {
	ctx, g := NewGroup(ctx)
	for name, tg := range t {
		pipe, err := os.OpenFile(tg.Path, os.O_RDONLY, 0666)
		if err != nil {
			return errors.Wrapf(err, "open %q", name)
		}

		g.Go(func() (mu *MultiError) {
			_, err := io.Copy(tg.Stream, pipe)
			mu = Catch(mu, err)
			mu = Catch(mu, pipe.Close())
			return
		})
	}

	return g.Wait()
}

func (t Targets) Teardown() (mu *MultiError) {
	for _, target := range t {
		mu = Catch(mu, target.Stream.Close())
		mu = Catch(mu, os.Remove(target.Path))
	}
	return
}
