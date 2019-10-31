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

package proc

//go:generate mockgen -package mock -destination mock/mock.go $REPO_URI/$GOPACKAGE Proc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/proc/internal/log"
	"github.com/samaritan-proxy/samaritan/stats"
)

var (
	builderRegistry = make(map[protocol.Protocol]Builder)
)

// RegisterBuilder registers the given proc builder to the builder map.
func RegisterBuilder(p protocol.Protocol, b Builder) {
	builderRegistry[p] = b
}

func getBuilder(p protocol.Protocol) (Builder, bool) {
	b, ok := builderRegistry[p]
	return b, ok
}

type BuildParams struct {
	Name   string
	Cfg    *service.Config
	Hosts  []*host.Host
	Stats  *Stats
	Logger log.Logger
}

// Builder creates a processor.
type Builder interface {
	// Build builds a new processor.
	Build(params BuildParams) (Proc, error)
}

// Proc is used to handle and forward the network traffic.
type Proc interface {
	Name() string
	Address() string
	Config() *service.Config

	OnSvcHostAdd([]*host.Host) error
	OnSvcHostRemove([]*host.Host) error
	OnSvcAllHostReplace([]*host.Host) error

	OnSvcConfigUpdate(*service.Config) error

	Start() error
	StopListen() error
	Stop() error
}

type wrappedProc struct {
	Proc
}

func newWrappedProc(p Proc) Proc {
	return &wrappedProc{
		p,
	}
}

func (w *wrappedProc) OnSvcConfigUpdate(cfg *service.Config) error {
	setDefaultValue(cfg)
	return w.Proc.OnSvcConfigUpdate(cfg)
}

var (
	// ErrUnsupportedProtocol indicates the protocol is not supported.
	ErrUnsupportedProtocol = errors.New("unsupported protocol")
)

// New creates a processor with given config.
func New(name string, cfg *service.Config, hosts []*host.Host) (p Proc, err error) {
	b, ok := getBuilder(cfg.Protocol)
	if !ok {
		return nil, ErrUnsupportedProtocol
	}
	scopeName := fmt.Sprintf("service.%s.", strings.Replace(name, ".", "_", -1))
	s := NewStats(stats.CreateScope(scopeName))
	setDefaultValue(cfg)
	proc, err := b.Build(BuildParams{
		name,
		cfg,
		hosts,
		s,
		log.New(fmt.Sprintf("[%s]", name)),
	})
	if err != nil {
		return nil, err
	}
	return newWrappedProc(proc), nil
}
