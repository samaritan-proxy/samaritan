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

package mysql

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/samaritan-proxy/samaritan/proc/internal/hc/errs"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
)

// Checker does health checking with MySQL protocol.
// It does a login and expecting an reply.
type Checker struct {
	handshakeThenQuit []byte
}

// NewChecker creates MySQLChecker.
func NewChecker(config *hc.MySQLChecker) (*Checker, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Checker{
		handshakeThenQuit: NewHealthCheckPacket(config.Username),
	}, nil
}

// Check checks given host with timeout.
func (c *Checker) Check(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetDeadline(deadline)
	// Send Handshake.
	if err := c.sendHandshake(conn); err != nil {
		return err
	}
	reader := bufio.NewReader(conn)
	// Expect Greeting.
	if err = c.expectGreeting(reader); err != nil {
		return err
	}
	// Expect OK.
	if err = c.expectOK(reader); err != nil {
		return err
	}
	if time.Now().After(deadline) {
		return errs.ErrTimeout
	}
	return err
}

// sendHandshake sends c.handshakeThenQuit to given writer.
func (c *Checker) sendHandshake(writer io.Writer) error {
	written, err := writer.Write(c.handshakeThenQuit)
	if err != nil {
		return fmt.Errorf("write error: %s", err)
	}
	if written != len(c.handshakeThenQuit) {
		return fmt.Errorf("write error: %s", io.ErrShortWrite)
	}

	return nil
}

// expectGreeting reads an greeting packet from reader.
// NOTE: Error is returned if the packet is not an MySQL packet.
// Greeting packet is the first packet from server.
func (c *Checker) expectGreeting(reader *bufio.Reader) error {
	header, err := NewMySQLHeaderFromReader(reader)
	if err != nil {
		return err
	}

	_, err = reader.Discard(header.PayloadLength())

	return err
}

// expectOK reads an MySQL packet from reader.
// NOTE: Error is returned if the packet is not an OK_Packet.
// https://dev.mysql.com/doc/internals/en/packet-OK_Packet.html
func (c *Checker) expectOK(reader io.Reader) error {
	header, err := NewMySQLHeaderFromReader(reader)
	if err != nil {
		return err
	}
	err = binary.Read(reader, binary.LittleEndian, &header)
	if err != nil {
		return err
	}
	var payloadHeader byte
	err = binary.Read(reader, binary.LittleEndian, &payloadHeader)
	if err == nil && payloadHeader != 0 && payloadHeader != 0xfe {
		err = fmt.Errorf("unexpected payload header[%#x] for an OK packet", payloadHeader)
	}

	return err
}
