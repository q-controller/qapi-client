package monitor

import "github.com/q-controller/qapi-client/src/client"

type MessageType int

const (
	MessageGeneric MessageType = iota
	MessageEvent
)

type InstanceMessageType int

const (
	InstanceMessageAdd InstanceMessageType = iota
	InstanceMessageDelete
)

type InstanceMessage struct {
	Instance            string
	InstanceMessageType InstanceMessageType
}

type Message struct {
	Type     MessageType
	Instance string
	Event    *client.QAPIEvent
	Generic  []byte
}

type MonitorEvent struct {
	InstanceMessage *InstanceMessage
	Message         *Message
}
