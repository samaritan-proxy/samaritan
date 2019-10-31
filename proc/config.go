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
	"time"

	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/utils"
)

const (
	defaultConnectTimeout = 3 * time.Second
	defaultIdleTimeout    = 10 * time.Minute
)

func setDefaultValue(cfg *service.Config) {
	if cfg == nil {
		return
	}
	if cfg.ConnectTimeout == nil {
		cfg.ConnectTimeout = utils.DurationPtr(defaultConnectTimeout)
	}
	if cfg.IdleTimeout == nil {
		cfg.IdleTimeout = utils.DurationPtr(defaultIdleTimeout)
	}
}
