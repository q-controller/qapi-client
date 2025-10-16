package sockets

import (
	"fmt"
	"iter"

	"golang.org/x/sys/unix"
)

func (q *fdQueue) waitInternal() (iter.Seq[int], error) {
	events := make([]unix.Kevent_t, 10)
	n, err := unix.Kevent(q.fd, nil, events, nil) // Block until events occur
	if err != nil {
		if err == unix.EINTR {
			return nil, nil
		}
		return nil, err
	}

	return func(yield func(int) bool) {
		for i := range n {
			fd := int(events[i].Ident)
			if !yield(fd) {
				return
			}
		}
	}, nil
}

func (q *fdQueue) addInternal(fd int) error {
	event := unix.Kevent_t{
		Ident:  uint64(fd),
		Filter: unix.EVFILT_READ,
		Flags:  unix.EV_ADD | unix.EV_ENABLE,
	}

	if _, err := unix.Kevent(q.fd, []unix.Kevent_t{event}, nil, nil); err != nil {
		return fmt.Errorf("kevent add failed: %v", err)
	}

	return nil
}

func (q *fdQueue) deleteInternal(fd int) error {
	event := unix.Kevent_t{
		Ident:  uint64(fd),
		Filter: unix.EVFILT_READ,
		Flags:  unix.EV_DELETE,
	}
	_, err := unix.Kevent(q.fd, []unix.Kevent_t{event}, nil, nil)
	return err
}

func createQueueFd() (int, error) {
	return unix.Kqueue()
}
