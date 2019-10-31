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
	"io"
	"net"
	"time"

	"github.com/samaritan-proxy/samaritan/proc/internal/hc/errs"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
)

// Checker could perform customized actions for checking.
type Checker struct {
	actions []TCPAction
	config  *hc.ATCPChecker
}

// NewChecker creates AdvancedTCPChecker with given config.
func NewChecker(config *hc.ATCPChecker) (*Checker, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	actions := make([]TCPAction, 0, len(config.GetAction())*2)
	for _, obj := range config.GetAction() {
		s, err := NewTCPSend(obj.Send)
		if err != nil {
			return nil, err
		}
		actions = append(actions, s)
		e, err := NewTCPExpect(obj.Expect)
		if err != nil {
			return nil, err
		}
		actions = append(actions, e)
	}
	return &Checker{
		config:  config,
		actions: actions,
	}, nil
}

// Check does advanced TCP check to given addr with timeout.
func (c *Checker) Check(addr string, timeout time.Duration) error {
	var err error
	var deadline = time.Now().Add(timeout)

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	bufferedConn := NewBufferedConn(conn)
	defer bufferedConn.Close()

	for _, act := range c.actions {
		if err = act.Perform(bufferedConn, deadline); err != nil {
			return err
		}
	}
	return nil
}

// SendBytesWithDeadline sends payload to w before deadline hits.
func SendBytesWithDeadline(w io.Writer, payload []byte, deadline time.Time) error {
	return doWithDeadline(deadline, func() error {
		nw, err := w.Write(payload)
		if err != nil {
			return err
		}
		if nw != len(payload) {
			return io.ErrShortWrite
		}
		return nil
	})
}

// doWithDeadline runs fn and return what fn returns.
// NOTE: ErrTimeout is returned if deadline is hit before fn returns.
func doWithDeadline(deadline time.Time, fn func() error) error {
	var ret = make(chan error, 1)
	go func() {
		ret <- fn()
	}()
	var durationLeft = time.Until(deadline)
	select {
	case err := <-ret:
		return err
	case <-time.After(durationLeft):
		return errs.ErrTimeout
	}
}

// ExpectBytesWithDeadline expect given payload from r before deadline hits.
func ExpectBytesWithDeadline(r io.Reader, payload []byte, deadline time.Time) error {
	return doWithDeadline(deadline, func() error {
		var resp = make([]byte, len(payload))
		_, err := io.ReadFull(r, resp)
		if err != nil {
			return err
		}

		if !bytes.Equal(resp, payload) {
			return errs.ErrUnexpectedResponse
		}
		return nil
	})
}
