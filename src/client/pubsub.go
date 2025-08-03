package client

import (
	"context"
	"sync"
)

type Result struct {
	Raw      QAPIResult
	Instance string
}

// PubSub manages subscriptions and publishing for request-response
type PubSub struct {
	mu     sync.RWMutex
	subs   map[string]chan Result
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPubSub creates a new PubSub instance
func NewPubSub() *PubSub {
	ctx, cancel := context.WithCancel(context.Background())
	return &PubSub{
		subs:   make(map[string]chan Result),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Subscribe creates a channel for a specific request ID
func (ps *PubSub) Subscribe(id string) <-chan Result {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ch := make(chan Result)
	ps.subs[id] = ch
	return ch
}

// Publish sends a response to the subscribed channel and closes it
func (ps *PubSub) Publish(res Result) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ch, exists := ps.subs[res.Raw.Id]; exists {
		select {
		case ch <- res:
			close(ch)
			delete(ps.subs, res.Raw.Id)
		default:
			// Handle case where channel is not ready
		}
	}
}

// Close shuts down the PubSub and all channels
func (ps *PubSub) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	for id, ch := range ps.subs {
		close(ch)
		delete(ps.subs, id)
	}
	ps.cancel()
}
