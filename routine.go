package fifo

import (
	"context"
	"sync"
	"sync/atomic"
)

type ErrorFunc func() *MultiError


func NewGroup(ctx context.Context) (context.Context, *ErrorGroup) {
	ctx, cancel := context.WithCancel(ctx)
	var count int64
	return ctx, &ErrorGroup{
		cancel: cancel,
		mu:     new(sync.Mutex),
		count:  &count,
		done:   make(chan struct{}),
	}
}

type ErrorGroup struct {
	cancel context.CancelFunc
	error  *MultiError
	mu     *sync.Mutex

	count *int64
	done  chan struct{}
}

func (g *ErrorGroup) Wait() *MultiError {
	if atomic.LoadInt64(g.count) != 0 {
		<-g.done
	}
	g.cancel()
	return g.error
}

func (g *ErrorGroup) start(f ErrorFunc) {
	err := f()
	if err != nil && len(err.err) > 0 {
		g.mu.Lock()
		g.error = Catch(g.error, err)
		g.mu.Unlock()
		g.cancel()
	}

	n := atomic.AddInt64(g.count, -1)
	if n == 0 {
		close(g.done)
	}
}

func (g *ErrorGroup) Go(f ErrorFunc) {
	atomic.AddInt64(g.count, 1)
	go g.start(f)
}
