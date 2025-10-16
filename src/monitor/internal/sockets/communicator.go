package sockets

import (
	"sync"

	"golang.org/x/sys/unix"
)

type fdCommunicator struct {
	*fdReader
	*fdWriter
	once sync.Once
}

func (c *fdCommunicator) Read() ([]string, error) {
	return c.fdReader.Read()
}

func (c *fdCommunicator) Write(data []byte) error {
	return c.fdWriter.Write(data)
}

func (c *fdCommunicator) Close() {
	c.once.Do(func() {
		_ = unix.Shutdown(c.fdWriter.fd, unix.SHUT_RDWR)
		_ = unix.Close(c.fdWriter.fd)
		if c.fdWriter.fd != c.fdReader.fd {
			_ = unix.Shutdown(c.fdReader.fd, unix.SHUT_RDWR)
			_ = unix.Close(c.fdReader.fd)
		}
	})
}

func newFdCommunicator(readFd, writeFd int) Communicator {
	return &fdCommunicator{
		fdReader: &fdReader{fd: readFd},
		fdWriter: newWriter(writeFd),
	}
}
