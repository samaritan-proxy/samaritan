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
	"bytes"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/samaritan-proxy/samaritan/logger"
	"github.com/samaritan-proxy/samaritan/proc/internal/hc/errs"
)

// ErrPayloadTooLarge indicates the payload is too large.
var ErrPayloadTooLarge = errors.New("payload too large")

// ErrPayloadEmpty indicates the payload is empty.
var ErrPayloadEmpty = errors.New("payload can not be empty")

// TCPAction represents an action of advanced TCP.
type TCPAction interface {
	Perform(conn *BufferedConn, deadline time.Time) error
}

// TCPSend sends specific payload.
type TCPSend struct {
	payload []byte
}

func validatePayload(payload string) error {
	if len(payload) == 0 {
		return ErrPayloadEmpty
	}
	if len(payload) > MaxBufferSize {
		return ErrPayloadTooLarge
	}
	return nil
}

func decodePayload(b []byte) (string, error) {
	var isHexData = b[0] == 'b'
	if !isHexData {
		return strconv.Unquote(string(b))
	}

	var err error
	var rawData = string(b[1:])
	if rawData, err = strconv.Unquote(rawData); err != nil {
		return "", err
	}

	var payload []byte
	payload, err = hex.DecodeString(rawData)
	if err != nil {
		return "", err
	}
	return string(payload), err
}

// NewTCPSend creates TCPSend with given payload unquoted.
func NewTCPSend(b []byte) (*TCPSend, error) {
	payload, err := decodePayload(b)
	if err != nil {
		return nil, err
	}
	if err := validatePayload(payload); err != nil {
		return nil, err
	}
	return &TCPSend{payload: []byte(payload)}, err
}

// Perform performs TCPSend.
func (a *TCPSend) Perform(conn *BufferedConn, deadline time.Time) error {
	return SendBytesWithDeadline(conn, a.payload, deadline)
}

// TCPExpect expects specific payload from connection.
type TCPExpect struct {
	payload []byte
}

// NewTCPExpect creates TCPExpect with given payload unquoted.
func NewTCPExpect(b []byte) (*TCPExpect, error) {
	payload, err := decodePayload(b)
	if err != nil {
		return nil, err
	}
	if err := validatePayload(payload); err != nil {
		return nil, err
	}
	return &TCPExpect{payload: []byte(payload)}, err
}

// endsWithPartially searches partial payload as suffix in buf.
// The return value is the index of payload[0] found in buf.
func endsWithPartially(buf []byte, payload []byte) int {
	// payload: [] [] [] []
	//       i:        <-⬆
	for i := len(payload) - 1; i > 0; i-- {
		if bytes.HasSuffix(buf, payload[0:i]) {
			return len(buf) - i
		}
	}
	return -1
}

func (a *TCPExpect) searchPayload(buf []byte, atEOF bool) (skip int, found bool) {
	// logger.Debugf("Buf: '%s' Payload: '%s'", buf, a.payload)
	index := bytes.Index(buf, a.payload)
	// Found.
	if index != -1 {
		logger.Debugf("TCPExpect: Found[%d]", index)
		return index + len(a.payload), true
	}
	if atEOF {
		// Did not found at all.
		logger.Debug("TCPExpect: Did not found at all")
		return -1, false
	}
	// Partially found.
	if firstByteLastPos := endsWithPartially(buf, a.payload); firstByteLastPos != -1 {
		// payload[0:firstByteLastPos] was found in buf.
		// But a complete payload was not found yet.
		// Bytes before firstByteLastPos could be skipped.
		logger.Debugf("TCPExpect: First byte[%c] found[%d]", a.payload[0], firstByteLastPos)
		return firstByteLastPos, false
	}
	// Did not found a single matched byte while not EOF.
	// Skip all.
	logger.Debugf("TCPExpect: Skip all[%d]", len(buf))
	return len(buf), false
}

// Perform performs TCPExpect.
func (a *TCPExpect) Perform(conn *BufferedConn, deadline time.Time) error {
	if conn.Error() != nil {
		return conn.Error()
	}
	logger.Debugf("Expect: '%s'", a.payload)
	return doWithDeadline(deadline, func() error {
		for {
			skip, found := a.searchPayload(conn.Buffer(), conn.AtEOF())
			if skip > 0 {
				if err := conn.SkipBuffer(skip); err != nil {
					return err
				}
			}
			// Found.
			if found {
				return nil
			}
			// Did not found at all, we are doomed(EOF).
			if skip == -1 {
				return errs.ErrUnexpectedResponse
			}
			// Not found yet, more data is needed(available).
			conn.ReadMore()
			if err := conn.Error(); err != nil {
				return err
			}
		}
	})
}

// TODO:
// type TCPExpectRegex struct {
// 	re *regexp.Regexp
// }
