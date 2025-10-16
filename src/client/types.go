package client

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
)

type Request struct {
	Id        string          `json:"id,omitempty"`
	Execute   string          `json:"execute"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type Event struct {
	Id     string
	Data   []string
	Error  error
	Action *Action
}

type Action int

const (
	ActionAdd Action = iota
	ActionCancel
	ActionClose
	ActionExecute
)

type CommunicationType int

const (
	UnixDomain CommunicationType = iota
	Pipe
)

type UnixDomainConfig struct {
	SocketPath string
}

type CommunicationConfig struct {
	Type       CommunicationType `json:"type"`
	UnixDomain *UnixDomainConfig `json:"unix_domain,omitempty"`
}

var ErrUnknownCommunicationType = fmt.Errorf("unknown communication type")

type EventQueue interface {
	Wait(context context.Context) (iter.Seq[*Event], error)
	Add(id string, config CommunicationConfig) error
	Execute(id string, request Request) error
	Cancel(requestId string) error
	Close() error
}
