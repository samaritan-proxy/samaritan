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

package admin

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/samaritan-proxy/samaritan/config"
	"github.com/samaritan-proxy/samaritan/logger"
	"github.com/samaritan-proxy/samaritan/utils"
)

// Server is a HTTP Server
type Server struct {
	config *config.Config
	hs     *http.Server
	router *mux.Router
}

// New creates a HTTP Server with given config
func New(cfg *config.Config) *Server {
	r := mux.NewRouter()
	ip := cfg.Admin.Bind.Ip
	port := cfg.Admin.Bind.Port
	s := &Server{
		config: cfg,
		router: r,
		hs: &http.Server{
			Addr:    fmt.Sprintf("%s:%d", ip, port),
			Handler: r,
		},
	}
	s.addHandler()
	return s
}

// Start starts the server with a boolean indicating whether to retry
// if port is use.
func (s *Server) Start(portInUseRetry bool) error {
	l, err := s.tryListen(s.hs.Addr, portInUseRetry)
	if err != nil {
		return err
	}

	logger.Infof("Starting HTTP server at %s", s.hs.Addr)
	go func() {
		err := s.hs.Serve(l)
		switch err {
		case http.ErrServerClosed:
		default:
			logger.Warnf("HTTP Server exit with %s", err)
		}
	}()
	return nil
}

func (s *Server) tryListen(addr string, portInUseRetry bool) (net.Listener, error) {
	var once sync.Once
	for {
		l, err := net.Listen("tcp", addr)
		if utils.IsAddrInUse(err) && portInUseRetry {
			once.Do(func() {
				logger.Warnf("Error listening: %v, try again...", err)
			})
			time.Sleep(time.Second)
			continue
		}

		if err != nil {
			return nil, err
		}
		return l, nil
	}
}

// Stop stop the server
func (s *Server) Stop() {
	switch err := s.hs.Shutdown(nil); err {
	case nil:
	default:
		logger.Warnf("Failed to stop HTTP server, error: %s", err)
	}
}
