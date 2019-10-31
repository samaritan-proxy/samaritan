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
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/proc/internal/hc/errs"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
	"github.com/samaritan-proxy/samaritan/utils"
)

func TestMultipleExpect(t *testing.T) {
	for _, c := range []struct {
		Action   []*hc.ATCPChecker_Action
		Payloads []string
		Err      error
	}{

		{
			[]*hc.ATCPChecker_Action{
				{
					Send:   []byte(`"ba"`),
					Expect: []byte(`"na"`),
				},
			}, []string{"babanana"}, nil,
		},
		{
			[]*hc.ATCPChecker_Action{
				{
					Send:   []byte(`"a\n"`),
					Expect: []byte(`"b"`),
				},
			}, []string{"aaaaa\nb"}, nil,
		},
		{
			[]*hc.ATCPChecker_Action{
				{
					Send:   []byte(`"aa"`),
					Expect: []byte(`"bb"`),
				},
			}, []string{"asdasdsad\naa:bb\nasdjaksld"}, nil,
		},
		{
			[]*hc.ATCPChecker_Action{
				{
					Send:   []byte(`"PING_1\r\nPING_2\r\n"`),
					Expect: []byte(`"+PONG_2"`),
				},
			}, []string{"+PONG_1", "+PONG_2"}, nil,
		},
	} {

		srv := utils.StartTestServer(t, "", func(conn *net.TCPConn) {
			for _, p := range c.Payloads {
				conn.Write([]byte(p))
				// Do not assert return value of write here.
				// Connection may be closed by peer before writing is finished.
				time.Sleep(time.Millisecond * 10)
			}
			conn.SetDeadline(time.Now().Add(time.Millisecond * 500))
			ioutil.ReadAll(conn)
			conn.Close()
		})
		checker, err := NewChecker(&hc.ATCPChecker{
			Action: c.Action,
		})
		assert.NoError(t, err, c.Action)

		err = checker.Check(srv.Addr().String(), time.Second)
		if c.Err == nil {
			assert.NoError(t, err, "Testing: %s", c.Action)
		} else {
			assert.Equal(t, c.Err, err)
		}
		srv.Close()
	}
}

type fakeReadWriter struct {
	buffer        *bytes.Buffer
	sleepDuration time.Duration
}

func (rw *fakeReadWriter) Write(p []byte) (int, error) {
	time.Sleep(rw.sleepDuration)
	return rw.buffer.Write(p)
}

func (rw *fakeReadWriter) Read(p []byte) (int, error) {
	time.Sleep(rw.sleepDuration)
	return rw.buffer.Read(p)
}

func newFakeReadWriter(sleepDuration time.Duration) *fakeReadWriter {
	return &fakeReadWriter{
		buffer:        bytes.NewBuffer([]byte{}),
		sleepDuration: sleepDuration,
	}
}

func TestSendBytesWithDeadlineAndTimeout(t *testing.T) {
	payload := []byte("payload for test")
	writer := newFakeReadWriter(200 * time.Millisecond)

	assert.Equal(t, errs.ErrTimeout, SendBytesWithDeadline(writer, payload, time.Now().Add(100*time.Millisecond)))
}

func TestSendBytesWithDeadline(t *testing.T) {
	payload := []byte("payload for test")
	writer := newFakeReadWriter(0)

	SendBytesWithDeadline(writer, payload, time.Now().Add(time.Hour))

	assert.Equal(t, payload, writer.buffer.Bytes())
}

func TestExpectBytesWithDeadlineAndTimeout(t *testing.T) {
	payload := []byte("payload for test")
	reader := newFakeReadWriter(200 * time.Millisecond)
	reader.buffer.Write(payload)

	assert.Equal(t, errs.ErrTimeout, ExpectBytesWithDeadline(reader, payload, time.Now().Add(100*time.Millisecond)))
}

func TestExpectBytesWithDeadline(t *testing.T) {
	payload := []byte("payload for test")
	reader := newFakeReadWriter(0)
	reader.buffer.Write(payload)

	assert.NoError(t, ExpectBytesWithDeadline(reader, payload, time.Now().Add(100*time.Millisecond)))
}

func TestExpectBytesWithDeadlineAndUnexpectedPayload(t *testing.T) {
	payload := []byte("payload for test")
	reader := newFakeReadWriter(0)
	reader.buffer.Write([]byte("Unexpected payload"))

	assert.Equal(t, errs.ErrUnexpectedResponse, ExpectBytesWithDeadline(reader, payload, time.Now().Add(100*time.Millisecond)))
}
