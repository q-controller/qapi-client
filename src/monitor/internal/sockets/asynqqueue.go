package sockets

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"

	"github.com/q-controller/qapi-client/src/client"
	"golang.org/x/sys/unix"
)

const (
	cManagementEndpoint = "internal-management"
)

var ErrRequestCanceled = fmt.Errorf("request canceled")
var ErrAddFailed = fmt.Errorf("failed to add instance")

type AsyncQueue struct {
	queue          *fdQueue
	instances      map[string]Communicator
	fd2Id          map[int]string
	eventsCh       chan *client.Event
	managementComm Communicator
}

func (q *AsyncQueue) Wait(context context.Context) (iter.Seq[*client.Event], error) {
	// Returns an iterator function that yields events from the queue until the context is done or the channel is closed.
	// The provided 'yield' function is called for each event; returning false from 'yield' stops iteration.
	return func(yield func(*client.Event) bool) {
		for {
			select {
			case <-context.Done():
				return
			case event, ok := <-q.eventsCh:
				if !ok {
					return
				}
				if !yield(event) {
					return
				}
			}
		}
	}, nil
}

func (q *AsyncQueue) Close() error {
	return q.send(ManagementData{
		Action: client.ActionClose,
	})
}

func (q *AsyncQueue) send(data ManagementData) error {
	bytes, bytesErr := json.Marshal(data)
	if bytesErr != nil {
		slog.Error("could not marshal management data", "error", bytesErr)
		return bytesErr
	}
	return q.managementComm.Write(bytes)
}

func (q *AsyncQueue) Add(id string, config client.CommunicationConfig) error {
	return q.send(ManagementData{
		Action: client.ActionAdd,
		Add: &AddConfig{
			Id:     id,
			Config: config,
		},
	})
}

func (q *AsyncQueue) Execute(id string, request client.Request) error {
	return q.send(ManagementData{
		Action: client.ActionExecute,
		Execute: &ExecuteConfig{
			Id:      id,
			Request: request,
		},
	})
}

func (q *AsyncQueue) Cancel(requestId string) error {
	if requestId == "" {
		return nil
	}

	return q.send(ManagementData{
		Action: client.ActionCancel,
		Cancel: &CancelConfig{
			Id: requestId,
		},
	})
}

func (q *AsyncQueue) registerCommunicator(id string, config client.CommunicationConfig) (Communicator, error) {
	comm, commErr := buildCommunication(config)
	if commErr != nil {
		slog.Error("could not build communication", "error", commErr)
		return nil, commErr
	}
	readFd, writeFd, estErr := comm.Establish()
	if estErr != nil {
		return nil, estErr
	}

	if err := q.queue.Add(readFd); err != nil {
		_ = unix.Close(readFd)
		_ = unix.Close(writeFd)
		return nil, err
	}

	q.instances[id] = newFdCommunicator(readFd, writeFd)
	q.fd2Id[readFd] = id
	return q.instances[id], nil
}

func NewAsyncQueue() (client.EventQueue, error) {
	queue, queueErr := NewFdQueue()
	if queueErr != nil {
		return nil, queueErr
	}

	q := &AsyncQueue{
		queue:     queue,
		eventsCh:  make(chan *client.Event),
		instances: make(map[string]Communicator),
		fd2Id:     make(map[int]string),
	}
	if comm, err := q.registerCommunicator(cManagementEndpoint, client.CommunicationConfig{
		Type: client.Pipe,
	}); err != nil {
		q.Close()
		return nil, err
	} else {
		q.managementComm = comm
	}

	go func(queue *fdQueue) {
	MainLoop:
		for {
			fds, fdsErr := queue.Wait() // Block until events occur
			if fdsErr != nil {
				slog.Error("EventQueue.Wait error", "error", fdsErr)
				return
			}

			if fds == nil {
				continue
			}

			for fd := range fds {
				if id, idOk := q.fd2Id[fd]; idOk {
					slog.Info("new event arrived", "instance", id)
					if comm, commOk := q.instances[id]; commOk {
						objects, objectsErr := comm.Read()
						if objectsErr != nil {
							slog.Info("connection closed or read failed, removing instance", "instance", id, "error", objectsErr)
							if delErr := queue.Delete(fd); delErr != nil {
								slog.Error("could not remove fd from queue", "fd", fd, "error", delErr)
							}
							comm.Close()
							delete(q.instances, id)
							delete(q.fd2Id, fd)
							q.eventsCh <- &client.Event{
								Id:    id,
								Error: objectsErr,
								Data:  nil,
							}
							continue
						}
						for _, object := range objects {
							if id == cManagementEndpoint {
								var cmd ManagementData
								if marshalErr := json.Unmarshal([]byte(object), &cmd); marshalErr != nil {
									slog.Error("could not unmarshal event", "instance", id, "error", marshalErr)
									continue
								}
								switch cmd.Action {
								case client.ActionAdd:
									if cmd.Add != nil {
										action := client.ActionAdd
										_, communicatoErr := q.registerCommunicator(cmd.Add.Id, cmd.Add.Config)
										q.eventsCh <- &client.Event{
											Id:     cmd.Add.Id,
											Error:  communicatoErr,
											Action: &action,
										}
									} else {
										slog.Error("missing communication config for ADD action")
									}
								case client.ActionCancel:
									if cmd.Cancel != nil {
										q.eventsCh <- &client.Event{
											Id:    cmd.Cancel.Id,
											Error: ErrRequestCanceled,
										}
									} else {
										slog.Error("missing cancel config for CANCEL action")
									}
								case client.ActionExecute:
									if cmd.Execute != nil {
										if comm, commOk := q.instances[cmd.Execute.Id]; commOk {
											bytes, bytesErr := json.Marshal(cmd.Execute.Request)
											if bytesErr != nil {
												slog.Error("could not marshal execute request", "error", bytesErr)
												continue
											}
											writeErr := comm.Write(bytes)
											if writeErr != nil {
												slog.Error("could not write execute request", "error", writeErr)
											}
										}
									} else {
										slog.Error("missing execute config for EXECUTE action")
									}
								case client.ActionClose:
									queue.Close()
									for _, comm := range q.instances {
										comm.Close()
									}
									break MainLoop
								}
							} else {
								q.eventsCh <- &client.Event{
									Id:    id,
									Error: nil,
									Data:  []string{object},
								}
							}
						}
					}
				}
			}
		}
	}(queue)

	return q, nil
}
