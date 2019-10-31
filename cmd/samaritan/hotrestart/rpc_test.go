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

package hotrestart

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageTypes(t *testing.T) {
	assert.Equal(t, messageType(1), shutdownAdminReq)
	assert.Equal(t, messageType(2), shutdownAdminReply)
	assert.Equal(t, messageType(3), shutdownLocalConfReq)
	assert.Equal(t, messageType(4), shutdownLocalConfReply)
	assert.Equal(t, messageType(5), drainListenersReq)
	assert.Equal(t, messageType(6), drainListenersReply)
	assert.Equal(t, messageType(7), terminateReq)
	assert.Equal(t, messageType(8), terminateReply)
	assert.Equal(t, messageType(9), unknownReply)
}

func TestNewShutdownParentAdminRequest(t *testing.T) {
	req := newShutdownParentAdminRequest()
	assert.Equal(t, shutdownAdminReq, req.Type)
	assert.Equal(t, uint16(2), req.Len)
	assert.Equal(t, []byte(`{}`), req.Data)
}

func TestNewShutdownParentAdminResponse(t *testing.T) {
	resp := newShutdownParentAdminResponse()
	assert.Equal(t, shutdownAdminReply, resp.Type)
	assert.Equal(t, uint16(2), resp.Len)
	assert.Equal(t, []byte(`{}`), resp.Data)
}

func TestNewShutdownParentLocalConfRequest(t *testing.T) {
	req := newShutdownParentLocalConfRequest()
	assert.Equal(t, shutdownLocalConfReq, req.Type)
	assert.Equal(t, uint16(2), req.Len)
	assert.Equal(t, []byte(`{}`), req.Data)
}

func TestNewShutdownParentLocalConfResponse(t *testing.T) {
	resp := newShutdownParentLocalConfResponse()
	assert.Equal(t, shutdownLocalConfReply, resp.Type)
	assert.Equal(t, uint16(2), resp.Len)
	assert.Equal(t, []byte(`{}`), resp.Data)
}

func TestNewDrainParentListenersRequest(t *testing.T) {
	req := newDrainParentListenersRequest()
	assert.Equal(t, drainListenersReq, req.Type)
	assert.Equal(t, uint16(2), req.Len)
	assert.Equal(t, []byte(`{}`), req.Data)
}

func TestNewDrainParentListenersResponse(t *testing.T) {
	resp := newDrainParentListenersResponse()
	assert.Equal(t, drainListenersReply, resp.Type)
	assert.Equal(t, uint16(2), resp.Len)
	assert.Equal(t, []byte(`{}`), resp.Data)
}

func TestNewTerminateParentRequest(t *testing.T) {
	req := newTerminateParentRequest()
	assert.Equal(t, terminateReq, req.Type)
	assert.Equal(t, uint16(2), req.Len)
	assert.Equal(t, []byte(`{}`), req.Data)
}

func TestNewTerminateParentResponse(t *testing.T) {
	resp := newTerminateParentResponse()
	assert.Equal(t, terminateReply, resp.Type)
	assert.Equal(t, uint16(2), resp.Len)
	assert.Equal(t, []byte(`{}`), resp.Data)
}

func TestNewUnknownResponse(t *testing.T) {
	resp := newUnknownResponse()
	assert.Equal(t, unknownReply, resp.Type)
	assert.Equal(t, uint16(0), resp.Len)
	assert.Equal(t, []byte(nil), resp.Data)
}

func TestReadMessageWithInvalidHeader(t *testing.T) {
	sockName := genDomainSocketName(123456)
	defer removeDomainSocket(sockName)
	lis, err := net.Listen("unix", sockName)
	if err != nil {
		t.Error(err)
		return
	}
	defer lis.Close()

	go func() {
		conn, _ := lis.Accept()
		conn.Write([]byte{'h'})
	}()

	conn, err := net.Dial("unix", sockName)
	assert.NoError(t, err)
	_, err = readMessage(conn.(*net.UnixConn))
	assert.Error(t, err)
}

func TestReadMessageWithIncompleteData(t *testing.T) {
	sockName := genDomainSocketName(123456)
	defer removeDomainSocket(sockName)
	lis, err := net.Listen("unix", sockName)
	if err != nil {
		t.Error(err)
		return
	}
	defer lis.Close()

	go func() {
		conn, _ := lis.Accept()
		conn.Write([]byte{1, 2, 1, 'h'})
	}()

	conn, err := net.Dial("unix", sockName)
	assert.NoError(t, err)
	_, err = readMessage(conn.(*net.UnixConn))
	assert.Error(t, err)
}

func TestReadMessge(t *testing.T) {
	sockName := genDomainSocketName(123456)
	defer removeDomainSocket(sockName)
	lis, err := net.Listen("unix", sockName)
	if err != nil {
		t.Error(err)
		return
	}
	defer lis.Close()

	go func() {
		conn, _ := lis.Accept()
		b := []byte{1, 2, 1}
		for i := 0; i < 513; i++ {
			b = append(b, 'h')
		}
		conn.Write(b)
	}()

	conn, err := net.Dial("unix", sockName)
	assert.NoError(t, err)
	msg, err := readMessage(conn.(*net.UnixConn))
	assert.Nil(t, err)
	assert.Equal(t, messageType(1), msg.Type)
	assert.Equal(t, uint16(513), msg.Len)
	var data []byte
	for i := 0; i < 513; i++ {
		data = append(data, 'h')
	}
	assert.Equal(t, data, msg.Data)
}

func TestWriteMessage(t *testing.T) {
	sockName := genDomainSocketName(123456)
	defer removeDomainSocket(sockName)
	lis, err := net.Listen("unix", sockName)
	if err != nil {
		t.Error(err)
		return
	}
	defer lis.Close()

	go func() {
		conn, _ := lis.Accept()
		msg := &message{
			Type: terminateReq,
			Len:  5,
			Data: []byte("hello"),
		}
		sendMessage(conn.(*net.UnixConn), msg)
	}()

	conn, err := net.Dial("unix", sockName)
	assert.NoError(t, err)
	b := make([]byte, 128)
	n, err := conn.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, 8, n)
	assert.Equal(t, []byte{byte(terminateReq), 0, 5, 'h', 'e', 'l', 'l', 'o'}, b[:n])
}
