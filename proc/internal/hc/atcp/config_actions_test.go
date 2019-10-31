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
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/proc/internal/hc/errs"
)

type fakeConn struct {
	rBuf *bytes.Reader
	wBuf bytes.Buffer
}

func (c *fakeConn) Read(b []byte) (n int, err error) {
	return c.rBuf.Read(b)
}

func (c *fakeConn) Write(b []byte) (n int, err error) {
	return c.wBuf.Write(b)
}

func (c *fakeConn) Close() error {
	return nil
}

func newFakeConn(r string) (*BufferedConn, *bytes.Buffer) {
	fConn := &fakeConn{
		rBuf: bytes.NewReader([]byte(r)),
	}
	bConn := NewBufferedConn(fConn)
	return bConn, &fConn.wBuf
}

func TestFakeConn(t *testing.T) {
	conn, wBuf := newFakeConn("")
	buf, err := ioutil.ReadAll(conn)
	assert.NoError(t, err)
	assert.Equal(t, "", string(buf))

	nw, err := conn.Write([]byte("test"))
	assert.Nil(t, err)

	assert.Equal(t, len("test"), nw)
	assert.Equal(t, "test", wBuf.String())
}

func TestExpect(t *testing.T) {
	for _, c := range []struct {
		Expect string
		Actual string
		Err    error
	}{
		{`"banana"`, "babanana", nil},
		{`"bb"`, "aaaaa\nbb", nil},
		{`"aa:bb"`, "asdasdsad\naa:bb\nasdjaksld", nil},
		{`" "`, "ban ana", nil},
		{`b"504F4e47"`, "PONG", nil},
		{`"x"`, "", errs.ErrUnexpectedResponse},
		{`"aaaa"`, "", errs.ErrUnexpectedResponse},
		{`"banana"`, "bababa", errs.ErrUnexpectedResponse},
		{`"banana"`, "nanana", errs.ErrUnexpectedResponse},
		{`"banana"`, "banabana", errs.ErrUnexpectedResponse},
	} {
		t.Logf("Expecting '%s' in '%s'", c.Expect, c.Actual)
		exp, err := NewTCPExpect([]byte(c.Expect))
		assert.Nil(t, err)
		deadline := time.Now().Add(time.Second * 1)
		conn, _ := newFakeConn(c.Actual)
		err = exp.Perform(conn, deadline)
		assert.Equal(t, c.Err, err)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func chooseRandomChar(s string) byte {
	return byte(s[rand.Intn(len(s))])
}

// TestBufferShifting splits the expecting target on the border of MaxBufferSize
// in order to trigger buffer shifting.
func TestBufferShifting(t *testing.T) {
	exp, err := NewTCPExpect([]byte(`"abc"`))
	assert.NoError(t, err)
	buf := make([]byte, MaxBufferSize)
	for i := 0; i < len(buf); i++ {
		buf[i] = chooseRandomChar("abd!")
	}
	buf[len(buf)-1] = byte('a')
	conn, _ := newFakeConn(string(buf) + "bc")
	err = exp.Perform(conn, time.Now().Add(time.Second))
	assert.NoError(t, err)
}

func TestSend(t *testing.T) {
	for _, test := range []struct {
		payload string
		expect  string
	}{
		{payload: `"abc"`, expect: "abc"},
		{payload: `"abc\r\n\tCBA"`, expect: "abc\r\n\tCBA"},
		{payload: `"a"`, expect: "a"},
		{payload: `"\r"`, expect: "\r"},
		{payload: `"\n"`, expect: "\n"},
		{payload: `"\t"`, expect: "\t"},
		{payload: `b"50494e47"`, expect: "PING"},
	} {
		snd, err := NewTCPSend([]byte(test.payload))
		assert.Nil(t, err)

		assert.Nil(t, err)
		assert.Equal(t, []byte(test.expect), snd.payload)

		deadline := time.Now().Add(time.Second * 1)
		conn, buf := newFakeConn("")
		err = snd.Perform(conn, deadline)
		assert.Nil(t, err)
		assert.Equal(t, test.expect, buf.String())
	}
}

func TestDecodePayload(t *testing.T) {
	ast := assert.New(t)
	for _, payload := range []string{
		newTestPayload(1),
		newTestPayload(2048),
		newTestHexPayload(1),
		newTestHexPayload(2048),
	} {
		_, errSend := NewTCPSend([]byte(payload))
		_, errExpect := NewTCPExpect([]byte(payload))

		ast.NoError(errSend)
		ast.NoError(errExpect)
	}
}

func TestDecodeBadPayload(t *testing.T) {
	ast := assert.New(t)
	for _, payload := range []string{
		newTestPayload(0),
		newTestPayload(2049),
		`b "504F4E47"`,
		`b"0"`,
		`bb"504F4E47"`,
		newTestHexPayload(0),
		newTestPayload(2049),
	} {
		_, errSend := NewTCPSend([]byte(payload))
		_, errExpect := NewTCPExpect([]byte(payload))

		ast.Error(errSend)
		ast.Error(errExpect)
	}
}

func newTestPayload(length int) string {
	payload := ""
	for i := 0; i < length; i++ {
		payload += "0"
	}
	return fmt.Sprintf(`"%s"`, payload)
}

func newTestHexPayload(length int) string {
	payload := ""
	for i := 0; i < length; i++ {
		payload += "00"
	}
	return fmt.Sprintf(`b"%s"`, payload)
}
