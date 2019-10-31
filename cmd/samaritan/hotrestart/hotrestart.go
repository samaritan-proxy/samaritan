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

//go:generate mockgen -package $GOPACKAGE -destination hotrestart_mock_test.go $REPO_URI/cmd/samaritan/$GOPACKAGE Instance

import (
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/samaritan-proxy/samaritan/logger"
)

// Instance represents a sam instance.
type Instance interface {
	ID() int
	ParentID() int
	ShutdownAdmin()
	DrainListeners()
	ShutdownLocalConf()
	Shutdown()
}

// Restarter is used to hot-restart sam.
type Restarter struct {
	Instance
	sockName   string
	lis        net.Listener
	parentConn *net.UnixConn

	quit chan struct{}
	done chan struct{}

	shutdownOnce sync.Once
}

// New returns a restarter with given instance.
func New(inst Instance) (*Restarter, error) {
	sockName := genDomainSocketName(inst.ID())
	lis, err := net.Listen("unix", sockName)
	// TODO: handle temporary error
	if err != nil {
		return nil, err
	}

	r := &Restarter{
		Instance: inst,
		sockName: sockName,
		lis:      lis,
		quit:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	if inst.ParentID() > 0 {
		sockName := genDomainSocketName(inst.ParentID())
		parentConn, err := net.Dial("unix", sockName)
		if err != nil {
			logger.Warnf("Dial to parent domian socket %s failed: %v", sockName, err)
		} else {
			r.parentConn = parentConn.(*net.UnixConn)
		}
	}

	go func() {
		defer func() {
			removeDomainSocket(r.sockName)
			close(r.done)
		}()

		for {
			conn, err := r.lis.Accept()
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
					delay := time.Millisecond * 100
					logger.Warnf("Accept failed: %v; retrying in %s", err, delay)
					timer := time.NewTimer(delay)
					select {
					case <-timer.C:
					case <-r.quit:
						timer.Stop()
						return
					}
					continue
				}

				select {
				case <-r.quit:
				default:
					logger.Warnf("Accpet failed: %v", err)
				}
				return
			}
			r.handleChild(conn.(*net.UnixConn))
		}
	}()
	return r, nil
}

func (r *Restarter) handleChild(conn *net.UnixConn) {
	logger.Info("Child connected")
	defer func() {
		logger.Info("Child disconnected")
	}()

	go func() {
		// TODO: better way to close child conn
		<-r.quit
		conn.Close()
	}()

	for {
		select {
		case <-r.quit:
			return
		default:
		}

		msg, err := readMessage(conn)
		switch err {
		case nil:
			logger.Debugf("Receive message %v from child", msg)
		default:
			if ne, ok := err.(*net.OpError); ok && ne.Err == io.EOF {
				return
			}
			logger.Warnf("Read msg from child failed: %v", err)
			continue
		}

		var handle func(from *net.UnixConn, data []byte)
		switch msg.Type {
		case shutdownLocalConfReq:
			handle = r.handleShutdownLocalConfRequest
		case shutdownAdminReq:
			handle = r.handleShutdownAdminRequest
		case drainListenersReq:
			handle = r.handleDrainListenersRequest
		case terminateReq:
			handle = r.handleTerminateRequest
		default:
			handle = r.handleUnknownRequest
		}
		handle(conn, msg.Data)
	}
}

func (r *Restarter) handleShutdownLocalConfRequest(from *net.UnixConn, data []byte) {
	r.ShutdownLocalConf()
	resp := newShutdownParentLocalConfResponse()
	sendMessage(from, resp)
}

func (r *Restarter) handleShutdownAdminRequest(from *net.UnixConn, data []byte) {
	r.ShutdownAdmin()
	resp := newShutdownParentAdminResponse()
	sendMessage(from, resp)
}

func (r *Restarter) handleDrainListenersRequest(from *net.UnixConn, data []byte) {
	r.DrainListeners()
	resp := newDrainParentListenersResponse()
	sendMessage(from, resp)
}

var kill = syscall.Kill

func (r *Restarter) handleTerminateRequest(from *net.UnixConn, data []byte) {
	resp := newTerminateParentResponse()
	sendMessage(from, resp)
	kill(os.Getpid(), syscall.SIGTERM)
}

func (r *Restarter) handleUnknownRequest(from *net.UnixConn, data []byte) {
	resp := newUnknownResponse()
	sendMessage(from, resp)
}

// ShutdownParentAdmin shutdowns the parent admin.
func (r *Restarter) ShutdownParentAdmin() {
	if r.parentConn == nil {
		return
	}
	req := newShutdownParentAdminRequest()
	sendMessage(r.parentConn, req)
	// TODO: validate response
	readMessage(r.parentConn)
}

// ShutdownParentLocalConf shutdowns the parent local conf store.
func (r *Restarter) ShutdownParentLocalConf() {
	if r.parentConn == nil {
		return
	}
	req := newShutdownParentLocalConfRequest()
	sendMessage(r.parentConn, req)
	readMessage(r.parentConn)
}

// DrainParentListeners drains the parent listeners.
func (r *Restarter) DrainParentListeners() {
	if r.parentConn == nil {
		return
	}
	req := newDrainParentListenersRequest()
	sendMessage(r.parentConn, req)
	readMessage(r.parentConn)
}

// TerminateParent terminates the parent.
func (r *Restarter) TerminateParent() {
	if r.parentConn == nil {
		return
	}
	req := newTerminateParentRequest()
	sendMessage(r.parentConn, req)
	readMessage(r.parentConn)
	r.parentConn.Close()
}

// Shutdown shutdowns the restarter.
func (r *Restarter) Shutdown() {
	r.shutdownOnce.Do(func() {
		close(r.quit)
		r.lis.Close()
		<-r.done
	})
}
