package client

import (
	"iter"

	"golang.org/x/sys/unix"
)

type Epoll struct {
	fd int
}

func (q *Epoll) Wait() (iter.Seq[int], error) {
	events := make([]unix.EpollEvent, 10)
	n, err := unix.EpollWait(q.fd, events, -1) // Block until events occur
	if err != nil {
		if err == unix.EINTR {
			return nil, nil
		}

		return nil, err
	}

	return func(yield func(int) bool) {
		for i := range n {
			fd := int(events[i].Fd)
			if !yield(fd) {
				return
			}
		}
	}, nil
}

func (q *Epoll) Close() error {
	return unix.Close(q.fd)
}

func (q *Epoll) Add(fd int) error {
	event := unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLET,
		Fd:     int32(fd),
	}

	if err := unix.EpollCtl(q.fd, unix.EPOLL_CTL_ADD, fd, &event); err != nil {
		return err
	}

	return nil
}

func (q *Epoll) Delete(fd int) error {
	return unix.EpollCtl(q.fd, unix.EPOLL_CTL_DEL, fd, nil)
}

func NewEventQueue() (EventQueue, error) {
	fd, err := unix.EpollCreate1(0)

	if err != nil {
		return nil, nil
	}

	return &Epoll{
		fd: fd,
	}, nil
}
