package client

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

type Monitor struct {
	queue       EventQueue
	dispatchers map[string]*Dispatcher
	cmdCh       chan Command
	readEnd     *os.File
	writeEnd    *os.File
}

func (m *Monitor) Close() error {
	resp := m.Do(ActionClose, nil)
	return resp.Error
}

func NewMonitor() (*Monitor, error) {
	queue, queueErr := NewEventQueue()
	if queueErr != nil {
		return nil, queueErr
	}

	fds := make([]int, 2)
	if pipeErr := unix.Pipe(fds); pipeErr != nil {
		return nil, pipeErr
	}
	for _, fd := range fds {
		if err := unix.SetNonblock(fd, true); err != nil {
			return nil, err
		}
	}

	monitor := &Monitor{
		queue:       queue,
		dispatchers: make(map[string]*Dispatcher),
		cmdCh:       make(chan Command, 128),
		readEnd:     os.NewFile(uintptr(fds[0]), "pipe-r"),
		writeEnd:    os.NewFile(uintptr(fds[1]), "pipe-w"),
	}

	if addErr := queue.Add(int(monitor.readEnd.Fd())); addErr != nil {
		return nil, addErr
	}

	return monitor, nil
}

func (m *Monitor) Add(name, socketPath string) error {
	session, sessionErr := NewSession(socketPath, 2*time.Second, 300*time.Second)
	if sessionErr != nil {
		return sessionErr
	}

	resp := m.Do(ActionAdd, AddPayload{
		Session: session,
		Id:      name,
	})

	return resp.Error
}

func (m *Monitor) Do(action Action, payload interface{}) ActionResponse {
	var resp ActionResponse
	done := make(chan ActionResponse)
	defer close(done)
AddLoop:
	for {
		select {
		case m.cmdCh <- Command{
			Action:  action,
			Payload: payload,
			Done:    done,
		}:
			unix.Write(int(m.writeEnd.Fd()), []byte{1})
			resp = <-done
			break AddLoop
		default:
			unix.Write(int(m.writeEnd.Fd()), []byte{1})
		}
	}

	return resp
}

func (m *Monitor) Execute(name string, request Request) (<-chan Result, error) {
	resp := m.Do(ActionExecute, ExecutePayload{
		Id:      name,
		Request: request,
	})
	if resp.Error != nil {
		return nil, resp.Error
	}
	if ch, ok := resp.Payload.(<-chan Result); ok {
		return ch, nil
	}
	return nil, fmt.Errorf("wrong reponse's payload for the command EXECUTE")
}

func (m *Monitor) Start() <-chan Message {
	eventCh := make(chan Message)
	go func() {
		defer close(m.cmdCh)
		defer close(eventCh)
	MainLoop:
		for {
			events, err := m.queue.Wait() // Block until events occur
			if err != nil {
				slog.Error("EventQueue.Wait error", "error", err)
				return
			}

			if events == nil {
				continue
			}

			for event := range events {
				if event == int(m.readEnd.Fd()) {
					buf := make([]byte, 64)
					for {
						_, err := unix.Read(event, buf)
						if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
							break
						} else if err != nil {
							fmt.Printf("pipe read error: %v", err)
							break
						}
					}

				CommandLoop:
					for {
						select {
						case cmd := <-m.cmdCh:
							switch cmd.Action {
							case ActionAdd:
								if payload, payloadOk := cmd.Payload.(AddPayload); payloadOk {
									if err := m.queue.Add(payload.Session.fd); err != nil {
										cmd.Done <- ActionResponse{
											Error: err,
										}
										payload.Session.Close()
									} else {
										m.dispatchers[payload.Id] = &Dispatcher{
											Session: payload.Session,
											Pubsub:  NewPubSub(),
										}
										cmd.Done <- ActionResponse{
											Error: err,
										}
									}
								} else {
									cmd.Done <- ActionResponse{
										Error: fmt.Errorf("check payload for ADD command"),
									}
								}
							case ActionExecute:
								if payload, payloadOk := cmd.Payload.(ExecutePayload); payloadOk {
									if dispatcher, found := m.dispatchers[payload.Id]; found {
										payload, err := dispatcher.Execute(payload.Request)
										cmd.Done <- ActionResponse{
											Error:   err,
											Payload: payload,
										}
									} else {
										cmd.Done <- ActionResponse{Error: nil}
									}
								} else {
									cmd.Done <- ActionResponse{Error: fmt.Errorf("check payload for EXECUTE command")}
								}
							case ActionClose:
								errs := []error{}
								for _, dispatcher := range m.dispatchers {
									dispatcher.Close()
								}
								m.dispatchers = make(map[string]*Dispatcher)
								if closeErr := m.queue.Close(); closeErr != nil {
									errs = append(errs, closeErr)
								}

								m.readEnd.Close()
								m.writeEnd.Close()

								cmd.Done <- ActionResponse{
									Error: errors.Join(errs...),
								}
								break MainLoop
							}
						default:
							// No more commands
							break CommandLoop
						}
					}

					continue
				}
				for name, dispatcher := range m.dispatchers {
					if dispatcher.Session.fd == event {
						slog.Debug("new event arrived", "instance", name)
						if handleErr := dispatcher.ReadAndPublish(name, eventCh); handleErr == io.EOF {
							m.queue.Delete(dispatcher.Session.fd)
							dispatcher.Close()
							delete(m.dispatchers, name)
						}
						break
					}
				}
			}
		}
	}()

	return eventCh
}
