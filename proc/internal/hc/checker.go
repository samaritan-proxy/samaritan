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
	"time"

	"github.com/samaritan-proxy/samaritan/pb/config/hc"
	"github.com/samaritan-proxy/samaritan/proc/internal/hc/atcp"
	"github.com/samaritan-proxy/samaritan/proc/internal/hc/mysql"
	"github.com/samaritan-proxy/samaritan/proc/internal/hc/redis"
	"github.com/samaritan-proxy/samaritan/proc/internal/hc/tcp"
)

// checker represents a checker of specific protocol.
type checker interface {
	Check(addr string, timeout time.Duration) error
}

// newChecker creates Checker with given Protocol.
// NOTE: TCPChecker is returned if protocol is unrecognized.
func newChecker(config *hc.HealthCheck) (checker, error) {
	if config == nil {
		return tcp.NewChecker(), nil
	}
	switch config.GetChecker().(type) {
	case *hc.HealthCheck_TcpChecker:
		return tcp.NewChecker(), nil
	case *hc.HealthCheck_MysqlChecker:
		return mysql.NewChecker(config.GetMysqlChecker())
	case *hc.HealthCheck_AtcpChecker:
		return atcp.NewChecker(config.GetAtcpChecker())
	case *hc.HealthCheck_RedisChecker:
		return redis.NewChecker(config.GetRedisChecker()), nil
	default:
		return tcp.NewChecker(), nil
	}
}
