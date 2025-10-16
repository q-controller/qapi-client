package sockets

import "github.com/q-controller/qapi-client/src/client"

type Reader interface {
	Read() ([]string, error)
}

type Writer interface {
	Write([]byte) error
}

type Communicator interface {
	Reader
	Writer
	Close()
}

type AddConfig struct {
	Id     string                     `json:"id"`
	Config client.CommunicationConfig `json:"config"`
}

type CancelConfig struct {
	Id string `json:"id"`
}

type ExecuteConfig struct {
	Id      string         `json:"id"`
	Request client.Request `json:"request"`
}

type ManagementData struct {
	Action  client.Action  `json:"action"`
	Add     *AddConfig     `json:"add,omitempty"`
	Cancel  *CancelConfig  `json:"cancel,omitempty"`
	Execute *ExecuteConfig `json:"execute,omitempty"`
}
