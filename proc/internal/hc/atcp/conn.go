// Copyright 2019 Samaritan Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package atcp

import (
	"errors"
	"io"
	"sync"

	"github.com/samaritan-proxy/samaritan/logger"
)

// type Conn interface {
// 	io.Writer
// 	Buffer() []byte
// 	AtEOF() bool
// 	SkipBuffer()
// 	ReadMore() error
// }

// MaxBufferSize is the maximum size of buffer.
const MaxBufferSize = 2048

var bufferPool = &sync.Pool{New: func() interface{} {
	buf := make([]byte, MaxBufferSize)
	return &buf
}}

// ErrSkippedTooFar indicates SkipBuffer is called with an invalid integer.
var ErrSkippedTooFar = errors.New("SkipBuffer: skipped beyond buffer size")

// BufferedConn is a buffered connection shared among actions.
type BufferedConn struct {
	conn   io.ReadWriteCloser
	buffer []byte
	start  int
	end    int
	err    error
}

// NewBufferedConn creates BufferedConn from raw net.Conn.
func NewBufferedConn(conn io.ReadWriteCloser) *BufferedConn {
	return &BufferedConn{
		conn:   conn,
		buffer: *(bufferPool.Get().(*[]byte)),
	}
}

// Buffer returns unread buffer which could be empty.
func (c *BufferedConn) Buffer() []byte {
	if c.start == c.end {
		c.ReadMore()
	}
	logger.Debugf("Buffer: %d, %d", c.start, c.end)
	return c.buffer[c.start:c.end]
}

func (c *BufferedConn) setErr(e error) {
	if e == nil {
		return
	}
	logger.Debug("Error: ", e)
	if c.err == nil || c.err == io.EOF {
		c.err = e
	}
}

// Error returns the first non-EOF error encountered.
func (c *BufferedConn) Error() error {
	if c.err == io.EOF {
		return nil
	}
	return c.err
}

// AtEOF returns true if any error exists besides EOF.
func (c *BufferedConn) AtEOF() bool {
	return c.err != nil
}

// SkipBuffer skips the first n bytes of the buffer.
func (c *BufferedConn) SkipBuffer(n int) error {
	if n <= 0 {
		return nil
	}
	if c.start+n <= c.end {
		c.start += n
		if c.start == c.end {
			// The whole buffer is skipped.
			// reset to the beginning by the way.
			c.start = 0
			c.end = 0
		}
		return nil
	}
	return ErrSkippedTooFar
}

func (c *BufferedConn) shiftBufferIfNeeded() {
	if c.end >= len(c.buffer) {
		copied := copy(c.buffer[0:], c.buffer[c.start:c.end])
		logger.Debugf("Shift: %d, %d -> 0, %d", c.start, c.end, copied)
		c.start = 0
		c.end = copied
	}
}

// ReadMore reads more data into buffer.
func (c *BufferedConn) ReadMore() {
	c.shiftBufferIfNeeded()
	nr, err := c.conn.Read(c.buffer[c.end:])
	if err == nil || err == io.EOF {
		c.end += nr
	}
	c.setErr(err)
}

// Read calls conn.Read.
func (c *BufferedConn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

// Write calls conn.Write.
func (c *BufferedConn) Write(p []byte) (int, error) {
	return c.conn.Write(p)
}

// Close Calls conn.Close then releases buffer.
func (c *BufferedConn) Close() {
	c.conn.Close()
	bufferPool.Put(&c.buffer)
}
