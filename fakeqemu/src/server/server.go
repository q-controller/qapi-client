package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/krjakbrjak/fakeqemu/generated/qapi"
	"github.com/q-controller/qapi-client/src/client"
)

type Session interface {
	Close() error
	Read(ctx context.Context, v interface{}) error
	Write(ctx context.Context, v interface{}) error
}

type QAPISession struct {
	conn  net.Conn
	dec   *json.Decoder
	enc   *json.Encoder
	mutex sync.Mutex
}

func (s *QAPISession) Close() error {
	return s.conn.Close()
}

func (s *QAPISession) Read(ctx context.Context, v interface{}) error {
	return s.dec.Decode(v)
}

func (s *QAPISession) Write(ctx context.Context, v interface{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.enc.Encode(v)
}

type Initializer func(ctx context.Context, s Session) error

func QMPInitializer(ctx context.Context, s Session) error {
	type qmp struct {
		Version      qapi.VersionInfo       `json:"version"`
		Capabilities qapi.QMPCapabilityList `json:"capabilities"`
	}
	type greeting struct {
		QMP qmp `json:"QMP"`
	}

	if err := s.Write(ctx, &greeting{
		QMP: qmp{
			Version: qapi.VersionInfo{
				Qemu: qapi.VersionTriple{
					Major: 9,
					Minor: 0,
					Micro: 1,
				},
			},
			Capabilities: qapi.QMPCapabilityList{},
		},
	}); err != nil {
		slog.WarnContext(ctx, "failed to write greeting", "error", err)
		return err
	}

	for {
		var req client.Request
		if err := s.Read(ctx, &req); err != nil {
			if errors.Is(err, io.EOF) {
				slog.WarnContext(ctx, "Connection closed")
				return err
			}
			slog.WarnContext(ctx, "json decode error", "error", err)
			return err
		}
		if req.Execute == qapi.QmpCapabilitiesMethod {
			if err := s.Write(ctx, &client.QAPIResult{
				Id:     req.Id,
				Return: json.RawMessage(`{}`),
			}); err != nil {
				slog.WarnContext(ctx, "failed to write capabilities response", "error", err)
				return err
			}
			return nil
		}
		if err := s.Write(ctx, &client.QAPIResult{
			Id: req.Id,
			Error: &client.Error{
				Class:       "generic",
				Description: fmt.Sprintf("first call %s", qapi.QmpCapabilitiesMethod),
			},
		}); err != nil {
			slog.WarnContext(ctx, "failed to write response", "error", err)
			return err
		}
	}
}

type Handler func(args json.RawMessage) (json.RawMessage, *client.Error)
type Server struct {
	listener    net.Listener
	mapping     map[string]Handler
	mutex       sync.RWMutex
	initializer Initializer
	sessMutex   sync.Mutex
	sessions    []Session
}

func New(udSocketPath string, initializer Initializer) (*Server, error) {
	if err := os.Remove(udSocketPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	listener, connErr := net.Listen("unix", udSocketPath)
	if connErr != nil {
		return nil, connErr
	}
	s := &Server{
		listener:    listener,
		mapping:     make(map[string]Handler),
		initializer: initializer,
		sessions:    make([]Session, 0),
	}
	return s, nil
}

func (s *Server) Close() error {
	return s.listener.Close()
}

func (s *Server) When(method string, handler Handler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.mapping[method] = handler
}

func (s *Server) SendEvent(ctx context.Context, event client.QAPIEvent) error {
	if event.Timestamp == nil {
		now := time.Now()
		event.Timestamp = &client.Timestamp{
			Seconds:      now.Unix(),
			Microseconds: int64(now.Nanosecond() / 1000),
		}
	}
	s.sessMutex.Lock()
	defer s.sessMutex.Unlock()
	for _, sess := range s.sessions {
		if err := sess.Write(ctx, &event); err != nil {
			slog.WarnContext(ctx, "failed to send event", "error", err)
		}
	}
	return nil
}

func (s *Server) Listen(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = s.listener.Close()
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}

		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	sess := &QAPISession{
		conn: conn,
		dec:  json.NewDecoder(conn),
		enc:  json.NewEncoder(conn),
	}
	defer sess.Close()
	if s.initializer != nil {
		if err := s.initializer(ctx, sess); err != nil {
			slog.WarnContext(ctx, "initializer error", "error", err)
			return
		}
	}

	s.sessMutex.Lock()
	s.sessions = append(s.sessions, sess)
	s.sessMutex.Unlock()
	defer func() {
		s.sessMutex.Lock()
		for i, session := range s.sessions {
			if session == sess {
				s.sessions = append(s.sessions[:i], s.sessions[i+1:]...)
				break
			}
		}
		s.sessMutex.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var req client.Request
			if err := sess.Read(ctx, &req); err != nil {
				if errors.Is(err, io.EOF) {
					slog.WarnContext(ctx, "Connection closed")
					return
				}
				slog.WarnContext(ctx, "json decode error", "error", err)
				return
			}
			b, err := json.Marshal(req)
			if err != nil {
				slog.WarnContext(ctx, "request marshal error", "error", err)
				return
			}
			slog.InfoContext(ctx, "data copied", "bytes", len(b), "content", string(b))
			s.mutex.RLock()
			handler, ok := s.mapping[req.Execute]
			s.mutex.RUnlock()
			if ok {
				ret, err := handler(req.Arguments)
				if ret == nil && err == nil {
					slog.WarnContext(ctx, "Return and Error should not be nil at the same time.")
					return
				}
				if err := sess.Write(ctx, &client.QAPIResult{
					Id:     req.Id,
					Return: ret,
					Error:  err,
				}); err != nil {
					slog.WarnContext(ctx, "write response error", "error", err)
					return
				}
			}
		}
	}
}
