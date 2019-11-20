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
	"net/http"
	_ "net/http/pprof"

	statshttp "github.com/kirk91/stats/http"
	"github.com/samaritan-proxy/samaritan/consts"
	"github.com/samaritan-proxy/samaritan/stats"
)

func (s *Server) addHandler() {
	// config
	s.router.Methods(http.MethodGet).
		Path("/config").
		HandlerFunc(s.handleGetConfig)

	// stats
	s.router.Methods(http.MethodGet).
		Path("/stats").
		Handler(statshttp.Handler(stats.Store()))
	s.router.Methods(http.MethodGet).
		Path("/stats/prometheus").
		Handler(statshttp.PrometheusHandler(stats.Store(), consts.AppName))

	// ops
	s.router.PathPrefix("/ops").
		Subrouter().
		Path("/shutdown").
		HandlerFunc(s.handleShutdown)

	// pprof
	s.router.PathPrefix("/debug/").
		Handler(http.DefaultServeMux)

	// TODO: add /help return help of all api
}
