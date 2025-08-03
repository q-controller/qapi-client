package client

import (
	"encoding/json"
	"fmt"
)

type Dispatcher struct {
	Session *Session
	Pubsub  *PubSub
}

func (d *Dispatcher) Close() {
	d.Session.Close()
	d.Pubsub.Close()
}

func (d *Dispatcher) Execute(request Request) (<-chan Result, error) {
	execErr := d.Session.SendCommand(request)
	if execErr != nil {
		return nil, execErr
	}

	return d.Pubsub.Subscribe(request.Id), nil
}

func (d *Dispatcher) ReadAndPublish(instanceName string, eventCh chan<- Message) error {
	dataArray, readErr := d.Session.ReadResponse()
	if readErr != nil {
		fmt.Printf("Failed to read from session: %v\n", readErr)
		return readErr
	}
	for _, data := range dataArray {
		var env RawResponse
		if err := json.Unmarshal([]byte(data), &env); err != nil {
			fmt.Printf("Failed to decode response: %v\n", err)
			continue
		}

		msg := Message{
			Instance: instanceName,
		}
		switch {
		case env.Event != "" && env.Data != nil && env.Timestamp != nil:
			msg.Type = MessageEvent
			var event QAPIEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				fmt.Printf("Failed to decode QAPIEvent: %v\n", err)
				break
			}
			msg.Event = &event
			select {
			case eventCh <- msg:
			default:
			}
		case env.Return != nil:
			var result QAPIResult
			if err := json.Unmarshal([]byte(data), &result); err != nil {
				fmt.Printf("Failed to decode QAPIResult: %v\n", err)
				break
			}
			d.Pubsub.Publish(Result{
				Raw:      result,
				Instance: instanceName,
			})
		default:
			msg.Type = MessageGeneric
			msg.Generic = []byte(data)
			select {
			case eventCh <- msg:
			default:
			}
		}
	}
	return nil
}
