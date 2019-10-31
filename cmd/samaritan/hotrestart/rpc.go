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
	"encoding/json"
	"errors"
	"net"
)

// messageType represents the rpc message type for hot-restart.
type messageType uint8

const (
	shutdownAdminReq messageType = iota + 1
	shutdownAdminReply
	shutdownLocalConfReq
	shutdownLocalConfReply
	drainListenersReq
	drainListenersReply
	terminateReq
	terminateReply
	unknownReply
)

// message is a rpc message.
type message struct {
	Type messageType
	Len  uint16
	Data []byte
}

type shutdownParentAdminRequest struct{}
type shutdownParentAdminResponse struct{}
type shutdownParentLocalConfRequest struct{}
type shutdownParentLocalConfResponse struct{}
type drainParentListenersRequest struct{}
type drainParentListenersResponse struct{}
type terminateParentRequest struct{}
type terminateParentResponse struct{}

func newMessage(typ messageType, i interface{}) (*message, error) {
	var data []byte
	if i != nil {
		var err error
		data, err = json.Marshal(i)
		if err != nil {
			return nil, err
		}
	}
	msg := &message{
		Type: typ,
		Len:  uint16(len(data)),
		Data: data,
	}
	return msg, nil
}

func newShutdownParentAdminRequest() *message {
	req, err := newMessage(shutdownAdminReq, new(shutdownParentAdminRequest))
	if err != nil {
		panic(err)
	}
	return req
}

func newShutdownParentAdminResponse() *message {
	resp, err := newMessage(shutdownAdminReply, new(shutdownParentAdminResponse))
	if err != nil {
		panic(err)
	}
	return resp
}

func newShutdownParentLocalConfRequest() *message {
	req, err := newMessage(shutdownLocalConfReq, new(shutdownParentLocalConfRequest))
	if err != nil {
		panic(err)
	}
	return req
}

func newShutdownParentLocalConfResponse() *message {
	resp, err := newMessage(shutdownLocalConfReply, new(shutdownParentLocalConfResponse))
	if err != nil {
		panic(err)
	}
	return resp
}

func newDrainParentListenersRequest() *message {
	req, err := newMessage(drainListenersReq, new(drainParentListenersRequest))
	if err != nil {
		panic(err)
	}
	return req
}

func newDrainParentListenersResponse() *message {
	resp, err := newMessage(drainListenersReply, new(drainParentListenersResponse))
	if err != nil {
		panic(err)
	}
	return resp
}

func newTerminateParentRequest() *message {
	req, err := newMessage(terminateReq, new(terminateParentRequest))
	if err != nil {
		panic(err)
	}
	return req
}

func newTerminateParentResponse() *message {
	resp, err := newMessage(terminateReply, new(terminateParentResponse))
	if err != nil {
		panic(err)
	}
	return resp
}

func newUnknownResponse() *message {
	resp, err := newMessage(unknownReply, nil)
	if err != nil {
		panic(err)
	}
	return resp
}

func sendMessage(conn *net.UnixConn, msg *message) error {
	b := make([]byte, 3+msg.Len)
	b[0] = byte(msg.Type)
	b[1] = byte(msg.Len >> 8) // big endian
	b[2] = byte(msg.Len)
	copy(b[3:], msg.Data)
	// TODO(kirk91): add notes for using WriteMsgUnix directly
	_, _, err := conn.WriteMsgUnix(b, nil, nil)
	return err
}

func readMessage(conn *net.UnixConn) (*message, error) {
	b := make([]byte, 4096)
	n, _, _, _, err := conn.ReadMsgUnix(b, nil)
	if err != nil {
		return nil, err
	}

	// header: 1byte(type) + 2byte(len)
	if n < 3 {
		return nil, errors.New("invalid header")
	}
	msg := new(message)
	msg.Type = messageType(b[0])
	msg.Len = uint16(b[1])<<8 | uint16(b[2]) // big endian
	if uint16(len(b[2:n])) < msg.Len {
		return nil, errors.New("incomplete data")
	}
	msg.Data = b[3 : 3+msg.Len]
	return msg, nil
}
