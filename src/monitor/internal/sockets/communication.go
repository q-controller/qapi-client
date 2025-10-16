package sockets

import (
	"github.com/q-controller/qapi-client/src/client"
	"golang.org/x/sys/unix"
)

type Communication interface {
	Establish() (int, int, error)
}

func buildCommunication(c client.CommunicationConfig) (Communication, error) {
	switch c.Type {
	case client.UnixDomain:
		return &unixDomainCommunication{socketPath: c.UnixDomain.SocketPath}, nil
	case client.Pipe:
		return &pipeCommunication{}, nil
	default:
		return nil, client.ErrUnknownCommunicationType
	}
}

type unixDomainCommunication struct {
	socketPath string
}

func (u *unixDomainCommunication) Establish() (int, int, error) {
	fd, socketErr := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if socketErr != nil {
		return -1, -1, socketErr
	}

	// Set non-blocking
	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return -1, -1, err
	}

	// Connect to QMP socket
	addr := &unix.SockaddrUnix{Name: u.socketPath}
	if err := unix.Connect(fd, addr); err != nil {
		unix.Close(fd)
		return -1, -1, err
	}

	return fd, fd, nil
}

type pipeCommunication struct{}

func (p *pipeCommunication) Establish() (int, int, error) {
	fds := make([]int, 2)
	if pipeErr := unix.Pipe(fds); pipeErr != nil {
		return -1, -1, pipeErr
	}
	for _, fd := range fds {
		if err := unix.SetNonblock(fd, true); err != nil {
			return -1, -1, err
		}
	}

	return fds[0], fds[1], nil
}
