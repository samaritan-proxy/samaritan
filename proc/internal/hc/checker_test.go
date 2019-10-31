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

package hc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/samaritan-proxy/samaritan/proc/internal/hc/atcp"
	"github.com/samaritan-proxy/samaritan/proc/internal/hc/mysql"
	"github.com/samaritan-proxy/samaritan/proc/internal/hc/redis"
	"github.com/samaritan-proxy/samaritan/proc/internal/hc/tcp"
	"github.com/samaritan-proxy/samaritan/pb/config/hc"
)

func TestNewChecker(t *testing.T) {
	// tcp checker
	c, err := newChecker(&hc.HealthCheck{
		Checker: &hc.HealthCheck_TcpChecker{
			TcpChecker: &hc.TCPChecker{},
		},
	})
	assert.NoError(t, err)
	_, ok := c.(*tcp.Checker)
	assert.Equal(t, ok, true)

	// atcp checker
	c, err = newChecker(&hc.HealthCheck{
		Checker: &hc.HealthCheck_AtcpChecker{
			AtcpChecker: &hc.ATCPChecker{
				Action: []*hc.ATCPChecker_Action{
					{
						Send:   []byte(`"PING\r\n"`),
						Expect: []byte(`"+PONG"`),
					},
				},
			},
		},
	})
	assert.NoError(t, err)
	_, ok = c.(*atcp.Checker)
	assert.Equal(t, ok, true)

	// redis checker
	c, err = newChecker(&hc.HealthCheck{
		Checker: &hc.HealthCheck_RedisChecker{
			RedisChecker: &hc.RedisChecker{},
		},
	})
	assert.NoError(t, err)
	_, ok = c.(*redis.Checker)
	assert.Equal(t, ok, true)

	// mysql checker
	c, err = newChecker(&hc.HealthCheck{
		Checker: &hc.HealthCheck_MysqlChecker{
			MysqlChecker: &hc.MySQLChecker{
				Username: "mysql",
			},
		},
	})
	assert.NoError(t, err)
	_, ok = c.(*mysql.Checker)
	assert.Equal(t, ok, true)
}
