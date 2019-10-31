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

package net

import (
	"net"
	"time"

	"go.uber.org/atomic"

	"github.com/samaritan-proxy/samaritan/stats"
)

// Stats represents the Conn stats.
type Stats struct {
	ReadTotal  *stats.Counter
	WriteTotal *stats.Counter
	Duration   *stats.Histogram
}

type connCounter struct {
	inBytesCounter  *atomic.Uint64
	outBytesCounter *atomic.Uint64
}

// Conn is a net connection wrapper.
type Conn struct {
	net.Conn
	connCounter
	Stats *Stats

	createdAt    time.Time
	readTimeout  time.Duration
	writeTimeout time.Duration

	isClosed           *atomic.Bool
	recordDurationHook func() // for test
}

func Dial(network, address string, timeout time.Duration) (*Conn, error) {
	// TODO: parse option
	c, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	return New(c), nil
}

// New returns a Conn wrapper.
func New(rawConn net.Conn) *Conn {
	if _, ok := rawConn.(*Conn); ok {
		return rawConn.(*Conn)
	}

	return &Conn{
		Conn:      rawConn,
		isClosed:  atomic.NewBool(false),
		createdAt: time.Now(),
	}
}

// SetReadTimeout sets the read timeout.
func (c *Conn) SetReadTimeout(timeout time.Duration) {
	c.readTimeout = timeout
}

// SetWriteTimeout sets the write timeout.
func (c *Conn) SetWriteTimeout(timeout time.Duration) {
	c.writeTimeout = timeout
}

// SetStats sets the stats.
func (c *Conn) SetStats(stats *Stats) {
	c.Stats = stats
}

// Read reads data from the connection.
func (c *Conn) Read(b []byte) (int, error) {
	if c.readTimeout > 0 {
		if err := c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return 0, err
		}
	}
	n, err := c.Conn.Read(b)
	if err == nil && c.Stats != nil {
		c.Stats.ReadTotal.Add(uint64(n))
	}
	c.incBytesIn(n)
	return n, err
}

// Write writes data to the connection.
func (c *Conn) Write(b []byte) (int, error) {
	if c.writeTimeout > 0 {
		if err := c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return 0, err
		}
	}
	n, err := c.Conn.Write(b)
	if err == nil && c.Stats != nil {
		c.Stats.WriteTotal.Add(uint64(n))
	}
	c.incBytesOut(n)
	return n, err
}

// Close closes the connection.
func (c *Conn) Close() error {
	if !c.isClosed.CAS(false, true) {
		return nil
	}

	if c.Stats != nil {
		c.Stats.Duration.Record(uint64(time.Since(c.createdAt) / time.Second))
		if c.recordDurationHook != nil {
			c.recordDurationHook()
		}
	}
	return c.Conn.Close()
}

// CloseWrite shuts down the writing side of the connection if support.
func (c *Conn) CloseWrite() error {
	if closer, ok := c.Conn.(interface {
		CloseWrite() error
	}); ok {
		return closer.CloseWrite()
	}
	return nil
}

// CloseRead shuts down the reading side of the connection if support.
func (c *Conn) CloseRead() error {
	if closer, ok := c.Conn.(interface {
		CloseRead() error
	}); ok {
		return closer.CloseRead()
	}
	return nil
}

// SetInBytesCounter set the input bytes counter.
func (c *connCounter) SetInBytesCounter(u *atomic.Uint64) {
	if u == nil {
		return
	}
	c.inBytesCounter = u
}

// SetOutBytesCounter set the output bytes counter.
func (c *connCounter) SetOutBytesCounter(u *atomic.Uint64) {
	if u == nil {
		return
	}
	c.outBytesCounter = u
}

func (c *connCounter) incBytesIn(i int) {
	if c.inBytesCounter == nil || i <= 0 {
		return
	}
	c.inBytesCounter.Add(uint64(i))
}

func (c *connCounter) incBytesOut(i int) {
	if c.outBytesCounter == nil || i <= 0 {
		return
	}
	c.outBytesCounter.Add(uint64(i))
}
