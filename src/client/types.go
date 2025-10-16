package client

import (
	"encoding/json"
	"iter"
)

type Null struct{}
type QEmpty struct{}

type Error struct {
	Class       string `json:"class"`
	Description string `json:"desc"`
}

type QAPIEvent struct {
	Event     string          `json:"event,omitempty"`
	Data      json.RawMessage `json:"data"`
	Timestamp *struct {
		Seconds      int64 `json:"seconds"`
		Microseconds int64 `json:"microseconds"`
	} `json:"timestamp"`
}

type QAPIResult struct {
	Id     string          `json:"id,omitempty"`
	Error  *Error          `json:"error,omitempty"`
	Return json.RawMessage `json:"return,omitempty"`
}

type RawResponse struct {
	QAPIResult
	QAPIEvent
}

type Response[T any] struct {
	Id     string `json:"id,omitempty"`
	Error  string `json:"error,omitempty"`
	Return T      `json:"return,omitempty"`
}

type MessageType int

const (
	MessageGeneric MessageType = iota
	MessageEvent
)

type Message struct {
	Type     MessageType
	Instance string
	Event    *QAPIEvent
	Generic  []byte
}

type EventQueue interface {
	Wait() (iter.Seq[int], error)
	Close() error
	Add(fd int) error
	Delete(fd int) error
}

type ActionResponse struct {
	Error   error
	Payload interface{}
}
type Action int

const (
	ActionAdd Action = iota
	ActionClose
	ActionExecute
)

type AddPayload struct {
	SocketPath string
	Id         string
}

type ExecutePayload struct {
	Id      string
	Request Request
}

type Command struct {
	Action  Action
	Payload interface{}
	Done    chan<- ActionResponse
}
