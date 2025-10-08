package client

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func isValidJSON(s string) bool {
	// Trim whitespace to handle common issues
	s = strings.TrimSpace(s)

	return json.Valid([]byte(s))
}

// Session manages QEMU Guest Agent communication
type Session struct {
	socketPath string
	fd         int
	dataBuffer strings.Builder
}

type Request struct {
	Id        string          `json:"id,omitempty"`
	Execute   string          `json:"execute"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// NewSession initializes a QAPI session and starts readiness polling
func NewSession(socketPath string, pingInterval, timeout time.Duration) (*Session, error) {
	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		return nil, err
	}

	// Set non-blocking
	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return nil, err
	}

	// Connect to QMP socket
	addr := &unix.SockaddrUnix{Name: socketPath}
	if err := unix.Connect(fd, addr); err != nil {
		unix.Close(fd)
		return nil, err
	}

	session := &Session{
		socketPath: socketPath,
		fd:         fd,
	}

	return session, nil
}

func (s *Session) SendCommand(request Request) error {
	payload, _ := json.Marshal(request)
	totalSent := 0
	for totalSent < len(payload) {
		n, err := unix.Write(s.fd, payload[totalSent:])
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				// Write buffer full; retry after a short delay
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return err
		}
		if n == 0 {
			return fmt.Errorf("write to instance %s returned 0 bytes", s.socketPath)
		}
		totalSent += n
	}

	if totalSent != len(payload) {
		return fmt.Errorf("incomplete write to instance %s; sent %d/%d bytes", s.socketPath, totalSent, len(payload))
	}
	return nil
}

func (s *Session) ReadResponse() ([]string, error) {
	var readErr error = nil
	temp := make([]byte, 1024) // Temporary buffer for reading

	for {
		n, err := unix.Read(s.fd, temp)
		if n > 0 {
			// Append the read data to the buffer
			s.dataBuffer.Write(temp[:n])
		}
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				// No more data available; return accumulated data
				break
			}
			// Return accumulated data (if any) and the error
			readErr = err
		}
		if n == 0 {
			// EOF (connection closed)
			return nil, io.EOF
		}
	}

	// Parse complete JSON objects from the buffer
	jsonStrings, remaining, _ := parseJSONObjects(s.dataBuffer.String())

	// Reset buffer and store any remaining unparsed data
	s.dataBuffer.Reset()
	if remaining != "" {
		s.dataBuffer.WriteString(remaining)
	}

	return jsonStrings, readErr
}

// Close stops the session and cleans up
func (s *Session) Close() error {
	if err := unix.Shutdown(s.fd, unix.SHUT_RDWR); err != nil {
		return err
	}

	if err := unix.Close(s.fd); err != nil {
		return err
	}
	return nil
}
