package monitor

import "github.com/q-controller/qapi-client/src/client"

type MessageType int

const (
	MessageGeneric MessageType = iota
	MessageEvent
)

type Message struct {
	Type     MessageType
	Instance string
	Event    *client.QAPIEvent
	Generic  []byte
}
