package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/q-controller/qapi-client/src/client"
	"github.com/q-controller/qapi-client/src/monitor/internal/sockets"
)

type AddRequestFuture struct {
	Id    string
	Error chan error
}

type AddRequestResult struct {
	Id    string
	Error error
}

type Monitor struct {
	queue client.EventQueue

	messagesCh  chan Message
	addLoop     *client.Dispatcher[error]
	requestLoop *client.Dispatcher[client.QAPIResult]
	stopCh      chan struct{}
}

func NewMonitor() (*Monitor, error) {
	queue, queueErr := sockets.NewAsyncQueue()
	if queueErr != nil {
		return nil, queueErr
	}

	messagesCh := make(chan Message)
	addLoop := client.NewDispatcher[error](0)
	requestLoop := client.NewDispatcher[client.QAPIResult](0)
	stopCh := make(chan struct{})

	go func() {
		if events, eventsErr := queue.Wait(context.Background()); eventsErr != nil {
			slog.Error("AsyncQueue.Wait error", "error", eventsErr)
			stopCh <- struct{}{}
			return
		} else {
			for event := range events {
				if event.Action != nil {
					switch *event.Action {
					case client.ActionAdd:
						addLoop.Post(client.Data[error]{
							Id:      event.Id,
							Payload: event.Error,
						})
					}
				} else {
					for _, data := range event.Data {
						var env client.RawResponse
						if err := json.Unmarshal([]byte(data), &env); err != nil {
							fmt.Printf("Failed to decode response: %v\n", err)
							continue
						}

						msg := Message{
							Instance: event.Id,
						}
						switch {
						case env.Event != "" && env.Data != nil && env.Timestamp != nil:
							msg.Type = MessageEvent
							var event client.QAPIEvent
							if err := json.Unmarshal([]byte(data), &event); err != nil {
								fmt.Printf("Failed to decode QAPIEvent: %v\n", err)
								break
							}
							msg.Event = &event
							select {
							case messagesCh <- msg:
							default:
							}
						case env.Return != nil:
							var result client.QAPIResult
							if err := json.Unmarshal([]byte(data), &result); err != nil {
								fmt.Printf("Failed to decode QAPIResult: %v\n", err)
								break
							}
							requestLoop.Post(client.Data[client.QAPIResult]{
								Id:      result.Id,
								Payload: result,
							})
						default:
							msg.Type = MessageGeneric
							msg.Generic = []byte(data)
							select {
							case messagesCh <- msg:
							default:
							}
						}
					}
				}
			}
		}
	}()

	go func() {
		addLoopCancel, _ := addLoop.Run(context.Background())
		requestCancel, _ := requestLoop.Run(context.Background())
		defer requestCancel()
		defer addLoopCancel()
		<-stopCh
	}()

	return &Monitor{
		queue:       queue,
		messagesCh:  messagesCh,
		stopCh:      stopCh,
		addLoop:     addLoop,
		requestLoop: requestLoop,
	}, nil
}

func (m *Monitor) Add(name, socketPath string) <-chan error {
	ch := m.addLoop.Enqueue(name)
	if err := m.queue.Add(name, client.CommunicationConfig{
		Type: client.UnixDomain,
		UnixDomain: &client.UnixDomainConfig{
			SocketPath: socketPath,
		},
	}); err != nil {
		m.addLoop.Post(client.Data[error]{
			Id:      name,
			Payload: err,
		})
	}

	return ch
}

func (m *Monitor) Cancel(requestId string) error {
	if requestId == "" {
		return nil
	}

	return m.queue.Cancel(requestId)
}

func (m *Monitor) Close() error {
	close(m.messagesCh)
	return m.queue.Close()
}

func (m *Monitor) Execute(name string, request client.Request) (*ExecuteResult, error) {
	ch := m.requestLoop.Enqueue(request.Id)

	return &ExecuteResult{
		resultCh: ch,
		instance: name,
	}, m.queue.Execute(name, request)
}

func (m *Monitor) Messages() <-chan Message {
	return m.messagesCh
}
