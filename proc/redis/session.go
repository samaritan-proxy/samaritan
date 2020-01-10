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

package redis

import (
	"io"
	"net"
	"sync"
)

type session struct {
	p              *redisProc
	conn           net.Conn
	dec            *decoder
	enc            *encoder
	processingReqs chan *rawRequest

	quitOnce sync.Once
	quit     chan struct{}
	done     chan struct{}
}

func newSession(p *redisProc, conn net.Conn) *session {
	return &session{
		p:              p,
		conn:           conn,
		enc:            newEncoder(conn, 8192),
		dec:            newDecoder(conn, 4096),
		processingReqs: make(chan *rawRequest, 32),
		quit:           make(chan struct{}),
		done:           make(chan struct{}),
	}
}

func (s *session) Serve() {
	writeDone := make(chan struct{})
	go func() {
		s.loopWrite()
		s.conn.Close()
		s.doQuit()
		close(writeDone)
	}()

	s.loopRead()
	s.conn.Close()
	s.doQuit()
	<-writeDone
	close(s.done)
}

func (s *session) Close() {
	s.doQuit()
	s.conn.Close()
	<-s.done
}

func (s *session) doQuit() {
	s.quitOnce.Do(func() {
		close(s.quit)
	})
}

func (s *session) loopRead() {
	for {
		v, err := s.dec.Decode()
		if err != nil {
			if err != io.EOF {
				s.p.logger.Warnf("loop read exit: %v", err)
			}
			return
		}

		req := newRawRequest(v)
		s.p.handleRequest(req)

		select {
		case s.processingReqs <- req:
		case <-s.quit:
			return
		}
	}
}

func (s *session) loopWrite() {
	var (
		req *rawRequest
		err error
	)
	for {
		select {
		case <-s.quit:
			return
		case req = <-s.processingReqs:
		}

		req.Wait()
		// TODO(kirk91): abstract response
		resp := req.Response()
		if err = s.enc.Encode(resp); err != nil {
			goto FAIL
		}

		// there are some processing requests, we could flush later to
		// reduce the amount of write syscalls.
		if len(s.processingReqs) != 0 {
			continue
		}
		// flush the buffer data to the underlying connection.
		if err = s.enc.Flush(); err != nil {
			goto FAIL
		}
	}

FAIL:
	s.p.logger.Warnf("loop write exit: %v", err)
}
