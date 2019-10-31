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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/proc"
	"github.com/samaritan-proxy/samaritan/proc/internal/log"
	"github.com/samaritan-proxy/samaritan/stats"
)

func newRedisTestProc(t *testing.T) *redisProc {
	svcName := "test"
	svcCfg := &service.Config{
		Listener: &service.Listener{
			Address: &common.Address{
				Ip:   "127.0.0.1",
				Port: 0,
			},
			ConnectionLimit: 0,
		},
	}
	svcHosts := []*host.Host{}
	logger := log.New("[" + svcName + "]")
	stats := proc.NewStats(stats.CreateScope("service." + svcName))
	p, err := newRedisProc(svcName, svcCfg, svcHosts, stats, logger)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConfigHook(t *testing.T) {
	p := newRedisTestProc(t)
	called := false
	p.registerConfigHook(func(i *config) { called = true })
	assert.NoError(t, p.OnSvcConfigUpdate(nil))
	assert.True(t, called)

}

func TestOnSvcConfigUpdateWithBadConfig(t *testing.T) {
	p := newRedisTestProc(t)
	called := false
	p.registerConfigHook(func(i *config) { called = true })
	assert.Error(t, p.OnSvcConfigUpdate(&service.Config{}))
	assert.False(t, called)
}

func TestRedisProc_Name(t *testing.T) {
	p := newRedisTestProc(t)
	assert.Equal(t, "test", p.Name())
}

func TestRedisProc_Config(t *testing.T) {
	p := newRedisTestProc(t)
	assert.True(t, p.cfg.Equal(p.Config()))
}
