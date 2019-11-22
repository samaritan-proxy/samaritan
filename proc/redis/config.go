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

import "github.com/samaritan-proxy/samaritan/pb/config/service"

// Maybe will add some internal fields in the future, so define it.
type config struct {
	*service.Config
	slowReqThresholdInMicros uint64 // milliseconds
}

func newConfig(c *service.Config) *config {
	return &config{
		Config:                   c,
		slowReqThresholdInMicros: 50 * 1000, // 50ms
	}
}

func (c *config) Update(cfg *service.Config) {
	c.Config = cfg
}

func (c *config) Raw() *service.Config {
	return c.Config
}
