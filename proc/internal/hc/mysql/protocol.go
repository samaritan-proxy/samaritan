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
	"encoding/binary"
	"fmt"
	"io"
)

// Capability flags.
const (
	// Speaks 4.1 protocol.
	ClientProtocol41 = 0x00000200
	// Can do 4.1 authentication.
	ClientPluginAuth = 0x00008000
)

// Character sets.
const (
	UTF8GeneralCI = 0x21
)

// ComQUIT is a complete MySQL packet of COM_QUIT.
// https://dev.mysql.com/doc/internals/en/com-quit.html
var ComQUIT = []byte{1, 0, 0, 0, 1}

// NewHealthCheckPacket creates MySQL packet for health checking with given username.
// NOTE: The returned bytes contains two MySQL packets in the following order:
// 1. HandshakeResponse
// 2. Command QUIT
func NewHealthCheckPacket(username string) []byte {
	responsePacket := &HandshakeResponse{
		CapabilityFlags: ClientProtocol41 | ClientPluginAuth,
		MaxPacketSize:   16777216,
		CharacterSet:    UTF8GeneralCI,
		Username:        username,
	}
	return append(buildMySQLPacket(responsePacket.Bytes(), 1),
		ComQUIT...)
}

// HandshakeResponse is composed as follows:
// https://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::HandshakeResponse
// 4              capability flags, CLIENT_PROTOCOL_41 always set
// 4              max-packet size
// 1              character set
// string[23]     reserved (all [0])
// string[NUL]    username
// if capabilities & CLIENT_PLUGIN_AUTH_LENENC_CLIENT_DATA {
//   lenenc-int     length of auth-response
//   string[n]      auth-response
// } else if capabilities & CLIENT_SECURE_CONNECTION {
//   1              length of auth-response
//   string[n]      auth-response
// } else {
//   string[NUL]    auth-response
// }
// if capabilities & CLIENT_CONNECT_WITH_DB {
//   string[NUL]    database
// }
// if capabilities & CLIENT_PLUGIN_AUTH {
//   string[NUL]    auth plugin name
// }
// if capabilities & CLIENT_CONNECT_ATTRS {
//   lenenc-int     length of all key-values
//   lenenc-str     key
//   lenenc-str     value
//   if-more data in 'length of all key-values', more keys and value pairs
// }
type HandshakeResponse struct {
	CapabilityFlags uint32
	MaxPacketSize   uint32
	CharacterSet    uint8
	Reserved        [23]uint8
	Username        string
	// AuthResponse should be a string but password is not used yet, so it will always be a \0.
	AuthResponse uint8
}

// Bytes returns the payload as bytes.
func (r *HandshakeResponse) Bytes() []byte {
	payloadLength := 0 +
		4 + // r.CapabilityFlags
		4 + // r.MaxPacketSize
		1 + // r.CharacterSet
		len(r.Reserved) +
		(len(r.Username) + 1) +
		1 // r.AuthResponse

	var payload = make([]byte, payloadLength)
	var i = 0
	binary.LittleEndian.PutUint32(payload[i:], r.CapabilityFlags)
	i += 4
	binary.LittleEndian.PutUint32(payload[i:], r.MaxPacketSize)
	i += 4
	payload[i] = r.CharacterSet
	i++
	copy(payload[i:], r.Reserved[:])
	i += len(r.Reserved)
	copy(payload[i:], r.Username)
	// At this point, there are two zeros remaining at the end of payload.
	// The first one is the \0 for Username.
	// The second one represents empty password for AuthResponse.
	return payload
}

// buildMySQLPacket sends a MySQL packet with given payload.
// NOTE: payload should not bigger than 16MB.
// The layout of A MySQL Packet:
// https://dev.mysql.com/doc/internals/en/mysql-packet.html
//
// int<3>       payload_length  The number of bytes in the packet beyond the initial 4 bytes.
// int<1>       sequence_id	    Sequence ID.
// string<var>  payload	        [len=payload_length] payload of the packet.
func buildMySQLPacket(payload []byte, seq byte) []byte {
	var payloadLen = len(payload)
	return append([]byte{
		// payloadLen in little-endian order.
		byte(payloadLen),
		byte(payloadLen >> 8),
		byte(payloadLen >> 16),
		// Sequence ID.
		seq,
	}, payload...)
}

// Header maps the header of a MySQL packet.
// https://dev.mysql.com/doc/internals/en/mysql-packet.html
type Header struct {
	PayloadLen [3]uint8
	SequenceID uint8
}

// PayloadLength returns length of payload.
func (h *Header) PayloadLength() int {
	return int(uint32(h.PayloadLen[0]) |
		uint32(h.PayloadLen[1])<<8 |
		uint32(h.PayloadLen[2])<<16)
}

// NewMySQLHeaderFromReader reads a MySQLHeader from reader.
func NewMySQLHeaderFromReader(reader io.Reader) (Header, error) {
	var header Header
	err := binary.Read(reader, binary.LittleEndian, &header)
	if err != nil {
		err = fmt.Errorf("read packet header error: %s", err)
	}
	return header, err
}
