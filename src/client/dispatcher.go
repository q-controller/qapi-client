package client

import (
	"context"
)

// Data represents a message with an identifier and a payload of generic type T.
// Used to deliver results to subscribers.
type Data[T any] struct {
	Id      string // Subscription identifier
	Payload T      // The actual data to deliver
}

// Subscription represents a request to subscribe to a result with a given Id.
// The result will be delivered on the Payload channel.
type Subscription[T any] struct {
	Id      string // Subscription identifier
	Payload chan T // Channel to receive the result
}

// Dispatcher is a generic, thread-safe, single-use subscription manager.
// It allows clients to subscribe for a result by Id and asynchronously receive
// the result when it becomes available (e.g., from a socket-based service).
type Dispatcher[T any] struct {
	dataCh         chan Data[T]
	subscriptionCh chan Subscription[T]
}

// Enqueue registers a new subscription for the given Id.
// It returns a channel that will receive the result when available.
// The channel will be closed after the result is sent or if the subscription is cancelled.
// Only one outstanding subscription per Id is allowed at a time.
func (l *Dispatcher[T]) Enqueue(id string) <-chan T {
	ch := make(chan T, 1)
	l.subscriptionCh <- Subscription[T]{
		Id:      id,
		Payload: ch,
	}
	return ch
}

// Post delivers a result to the subscription with the matching Id.
// If a subscription exists, the result is sent and the channel is closed.
func (l *Dispatcher[T]) Post(data Data[T]) {
	l.dataCh <- data
}

// Run starts the subscription event loop in a new goroutine.
// It listens for new subscriptions and results, and matches them by Id.
// When the context is cancelled, all open subscription channels are closed and the loop exits.
// Returns a CancelFunc to stop the loop and a channel that is closed when the loop exits.
func (l *Dispatcher[T]) Run(ctx context.Context) (context.CancelFunc, <-chan struct{}) {
	done := make(chan struct{})
	runCtx, cancel := context.WithCancel(ctx)
	go func(c context.Context) {
		subscriptions := map[string]chan T{}
		for {
			select {
			case <-c.Done():
				for _, subCh := range subscriptions {
					close(subCh)
				}
				close(done)
				return
			case d := <-l.dataCh:
				if subCh, exists := subscriptions[d.Id]; exists {
					select {
					case subCh <- d.Payload:
					default:
					}
					close(subCh)
					delete(subscriptions, d.Id)
				}
			case s := <-l.subscriptionCh:
				if _, exists := subscriptions[s.Id]; exists {
					close(s.Payload)
				} else {
					subscriptions[s.Id] = s.Payload
				}
			}
		}
	}(runCtx)
	return cancel, done
}

// NewDispatcher creates a new Dispatcher with the specified buffer size for internal channels.
// The buffer size should be chosen based on expected concurrency and throughput.
func NewDispatcher[T any](bufferSize int) *Dispatcher[T] {
	return &Dispatcher[T]{
		dataCh:         make(chan Data[T], bufferSize),
		subscriptionCh: make(chan Subscription[T], bufferSize),
	}
}
