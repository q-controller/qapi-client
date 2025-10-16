package sockets

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

var ErrWriterChannelFull = fmt.Errorf("writer channel full")
var ErrIncompleteWrite = fmt.Errorf("incomplete write")
var ErrSocketClosed = fmt.Errorf("socket closed")
var ErrWriterClosed = fmt.Errorf("writer closed")

type WriterRequest struct {
	Data []byte
	Done chan error
}

type fdWriter struct {
	ch   chan *WriterRequest
	fd   int
	done chan struct{}
}

func newWriter(fd int) *fdWriter {
	w := &fdWriter{
		ch:   make(chan *WriterRequest, 100),
		fd:   fd,
		done: make(chan struct{}),
	}
	go w.run()
	return w
}

func (w *fdWriter) run() {
OuterLoop:
	for req := range w.ch {
		payload := req.Data
		totalSent := 0
		for totalSent < len(payload) {
			n, err := unix.Write(w.fd, payload[totalSent:])
			if err != nil {
				if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
					// Write buffer full; retry after a short delay
					time.Sleep(10 * time.Millisecond)
					continue
				}
				req.Done <- err
				break OuterLoop
			}
			if n == 0 {
				req.Done <- ErrSocketClosed
				break OuterLoop
			}
			totalSent += n
		}

		req.Done <- nil
	}

	close(w.done)
}

func (w *fdWriter) Write(buf []byte) error {
	b := make([]byte, len(buf))
	copy(b, buf)
	done := make(chan error, 1)
	select {
	case <-w.done:
		return ErrWriterClosed
	case w.ch <- &WriterRequest{Data: b, Done: done}:
		return <-done
	default:
		return ErrWriterChannelFull
	}
}
