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
)

func (s *Server) addHandler() {
	s.router.Methods(http.MethodGet).Path("/config").HandlerFunc(s.handleGetConfig)
	s.router.Methods(http.MethodGet).Path("/stats").HandlerFunc(s.handleGetStats)
	s.router.Methods(http.MethodGet).Path("/stats/prometheus").HandlerFunc(s.handleGetStatsForPrometheus)
	s.router.PathPrefix("/debug/").Handler(http.DefaultServeMux)
	s.router.PathPrefix("/ops").Subrouter().Path("/shutdown").HandlerFunc(s.handleShutdown)
	// TODO: add /help return help of all api
}
