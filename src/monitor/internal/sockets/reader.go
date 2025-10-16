package sockets

import (
	"io"
	"strings"

	"github.com/q-controller/qapi-client/src/utils"
	"golang.org/x/sys/unix"
)

type fdReader struct {
	fd         int
	dataBuffer strings.Builder
}

func (r *fdReader) Read() ([]string, error) {
	var readErr error = nil
	temp := make([]byte, 1024) // Temporary buffer for reading

	for {
		n, err := unix.Read(r.fd, temp)
		if n > 0 {
			// Append the read data to the buffer
			r.dataBuffer.Write(temp[:n])
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
	jsonStrings, remaining, _ := utils.ParseJSONObjects(r.dataBuffer.String())

	// Reset buffer and store any remaining unparsed data
	r.dataBuffer.Reset()
	if remaining != "" {
		r.dataBuffer.WriteString(remaining)
	}

	return jsonStrings, readErr
}
