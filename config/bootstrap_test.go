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
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/pb/config/bootstrap"
	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/utils"
)

func TestNewBootstrapFromYAMLFile(t *testing.T) {
	c := &bootstrap.Bootstrap{
		Instance: &common.Instance{
			Id:     "example.appid@10.0.0.1",
			Belong: "example.appid",
		},
		Admin: &bootstrap.Admin{
			Bind: &common.Address{
				Ip:   "0.0.0.0",
				Port: 12345,
			},
		},
		Log: bootstrap.Log{
			Level: bootstrap.DEBUG,
			Output: bootstrap.Log_Output{
				Type:   bootstrap.SYSLOG,
				Target: "127.0.0.1:518",
			},
		},
		Stats: bootstrap.Stats{
			Sinks: []*bootstrap.Sink{
				{Type: bootstrap.STATSD, Endpoint: "statsd.example.com:8125"},
			},
		},
		StaticServices: []*bootstrap.StaticService{
			{
				Name: "fc.sam_test",
				Config: &service.Config{
					Listener: &service.Listener{
						Address: &common.Address{
							Ip:   "0.0.0.0",
							Port: 12321,
						},
						ConnectionLimit: 1000,
					},
					HealthCheck: &hc.HealthCheck{
						Interval:      10 * time.Second,
						Timeout:       3 * time.Second,
						FallThreshold: 3,
						RiseThreshold: 3,
						Checker: &hc.HealthCheck_AtcpChecker{
							AtcpChecker: &hc.ATCPChecker{
								Action: []*hc.ATCPChecker_Action{
									{
										Send:   []byte(`PING`),
										Expect: []byte(`PONG`),
									},
								},
							},
						},
					},
					ConnectTimeout: utils.DurationPtr(3 * time.Second),
					IdleTimeout:    utils.DurationPtr(10 * time.Minute),
					LbPolicy:       service.LoadBalancePolicy_LEAST_CONNECTION,
					Protocol:       protocol.TCP,
					ProtocolOptions: &service.Config_TcpOption{
						TcpOption: &protocol.TCPOption{},
					},
				},
				Endpoints: []*service.Endpoint{
					{
						Address: &common.Address{
							Ip:   "10.101.68.90",
							Port: 7332,
						},
					},
				},
			},
		},
	}
	assert.NoError(t, c.Validate())
	b, err := json.Marshal(c)
	assert.NoError(t, err)
	yml, err := yaml.JSONToYAML(b)
	assert.NoError(t, err)

	tmpFile, err := ioutil.TempFile("", "bootstrap")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(yml)
	assert.NoError(t, err)
	assert.NoError(t, tmpFile.Sync())

	newCfg, err := LoadBootstrap(tmpFile.Name())
	assert.NoError(t, err)
	assert.True(t, c.Equal(newCfg))
}

func TestNewBootstrapFromYAMLFileWithNonexistentFile(t *testing.T) {
	_, err := LoadBootstrap("foo")
	_, ok := err.(*os.PathError)
	assert.True(t, ok)
}

func TestNewBootstrapFromYAMLWithBadContent(t *testing.T) {
	_, err := newBootstrapFromYAML([]byte{0})
	assert.Error(t, err)
}
