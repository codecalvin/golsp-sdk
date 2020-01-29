package server

import (
	"context"
	"log"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

// cancel manages $/cancelRequest by keeping track of running commands
type cancel struct {
	mu sync.Mutex
	m  map[jsonrpc2.ID]func()
}

// withCancel is like context.withCancel, except you can also cancel via
// calling c.cancel with the same id.
func (c *cancel) withCancel(ctx context.Context, id jsonrpc2.ID) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	if c.m == nil {
		c.m = make(map[jsonrpc2.ID]func())
	}
	c.m[id] = cancel
	c.mu.Unlock()
	return ctx, func() {
		c.mu.Lock()
		delete(c.m, id)
		c.mu.Unlock()
		cancel()
	}
}

// cancel will cancel the request with id. If the request has already been
// cancelled or not been tracked before, cancel is a noop.
func (c *cancel) cancel(id jsonrpc2.ID) {
	var cancel func()
	c.mu.Lock()
	if c.m != nil {
		cancel = c.m[id]
		delete(c.m, id)
	}
	c.mu.Unlock()
	if cancel != nil {
		log.Printf("cancelling request %s\n", id)
		cancel()
		log.Printf("cancelled request %s\n", id)
	}
}
