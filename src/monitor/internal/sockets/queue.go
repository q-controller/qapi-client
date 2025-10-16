package sockets

import (
	"iter"

	"golang.org/x/sys/unix"
)

type fdQueue struct {
	fd int
}

func (q *fdQueue) Wait() (iter.Seq[int], error) {
	fds, fdsErr := q.waitInternal()
	if fdsErr != nil {
		return nil, fdsErr
	}

	return func(yield func(int) bool) {
		if fds == nil {
			return
		}

		for fd := range fds {
			if !yield(fd) {
				return
			}
		}
	}, nil
}

func (q *fdQueue) Close() error {
	return unix.Close(q.fd)
}

func (q *fdQueue) Add(fd int) error {
	return q.addInternal(fd)
}

func (q *fdQueue) Delete(fd int) error {
	return q.deleteInternal(fd)
}

func NewFdQueue() (*fdQueue, error) {
	fd, fdErr := createQueueFd()

	if fdErr != nil {
		return nil, fdErr
	}

	return &fdQueue{
		fd: fd,
	}, nil
}
