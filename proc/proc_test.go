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

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/utils"
)

type builder struct {
	isBuildCalled bool
}

func (b *builder) Build(params BuildParams) (p Proc, err error) {
	b.isBuildCalled = true
	return
}

func TestRegisterBuilder(t *testing.T) {
	for _, typ := range []protocol.Protocol{
		protocol.TCP,
		protocol.MySQL,
		protocol.Redis,
	} {
		b := new(builder)
		RegisterBuilder(typ, b)
	}
	assert.Equal(t, len(builderRegistry), 3)
}

func TestNew(t *testing.T) {
	cfg := &service.Config{
		HealthCheck: &hc.HealthCheck{
			Interval:      time.Second,
			Timeout:       time.Second,
			FallThreshold: 3,
			RiseThreshold: 2,
			Checker: &hc.HealthCheck_TcpChecker{
				TcpChecker: &hc.TCPChecker{},
			},
		},
		Listener: &service.Listener{
			Address: &common.Address{
				Ip:   "127.0.0.1",
				Port: 1551,
			},
			ConnectionLimit: 0,
		},
		ConnectTimeout: utils.DurationPtr(time.Second),
		IdleTimeout:    utils.DurationPtr(time.Second),
		LbPolicy:       service.LoadBalancePolicy_ROUND_ROBIN,
		Protocol:       protocol.TCP,
		ProtocolOptions: &service.Config_TcpOption{
			TcpOption: &protocol.TCPOption{},
		},
	}

	// register tcp processor builder.
	b := new(builder)
	RegisterBuilder(protocol.TCP, b)

	_, err := New("name", cfg, nil)
	assert.NoError(t, err)

	_, err = New("name", cfg, nil)
	assert.NoError(t, err)
	assert.Condition(t, func() bool {
		return b.isBuildCalled == true
	})
}
