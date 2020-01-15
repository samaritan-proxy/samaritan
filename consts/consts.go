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

package consts

import (
	"time"

	"github.com/coreos/go-semver/semver"
)

var (
	// Build contains build information, it's fill in makefile
	// e.g. 2016-05-10_15:17:04@6eaa797bd159d7e82ca247de9eda3d53029cebe3
	Build string
)

func init() {
	checkVersion()
}

// Basic consts
const (
	Version = "1.0.0-rc1"
	// AppName is used in metrics
	AppName = "samaritan"
)

// checkVersion will panic if Version isn't semantic version
func checkVersion() {
	semver.New(Version)
}

// Default values
const (
	DefaultDataPath    = "/data/run/" + AppName + "/"
	DefaultPidFileName = AppName + ".pid"

	// DefaultParentTerminateTime is the default time that child
	// will wait before terminate the parent
	DefaultParentTerminateTime = time.Minute * 6

	DefaultConfigFile = "/etc/" + AppName + ".yaml"
)

const (
	// ParentPidEnv is the name of environment variable contains PID of parent Sam
	// instance
	ParentPidEnv = "__Samaritan_Parent__"

	// ParentTerminateTimeEnv is the name of environment variable contains the time
	// that child will wait before terminate the parent during a hot restart
	ParentTerminateTimeEnv = "__Samaritan_Parent_Terminate_Time__"
)
