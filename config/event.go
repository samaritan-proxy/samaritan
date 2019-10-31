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

package config

import (
	"github.com/samaritan-proxy/samaritan/pb/config/service"
)

var (
	_ Event = new(SvcAddEvent)
	_ Event = new(SvcRemoveEvent)
	_ Event = new(SvcConfigEvent)
	_ Event = new(SvcEndpointEvent)
)

// Event represents an event indicating config change.
type Event interface {
	// String() string
}

// SvcAddEvent represents a service add event.
type SvcAddEvent struct {
	Name      string
	Config    *service.Config
	Endpoints []*service.Endpoint
}

// SvcRemoveEvent represents a service remove event.
type SvcRemoveEvent struct {
	Name string
}

// SvcConfigEvent represents a service config change event.
type SvcConfigEvent struct {
	Name   string
	Config *service.Config
}

// SvcEndpointEvent represents a service endpoints change event.
type SvcEndpointEvent struct {
	Name    string
	Added   []*service.Endpoint
	Removed []*service.Endpoint
}
