package client

import "encoding/json"

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
