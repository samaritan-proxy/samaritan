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

package flag

import (
	"flag"
	"os"
	"path"

	"github.com/samaritan-proxy/samaritan/consts"
)

// Command-line parameters, they should never be altered outside this package
var (
	ShowVersionOnly bool
	DataPath        string
	PidFile         string
	InstanceID      string
	BelongService   string
	ConfigFile      string
)

func init() {
	flagSet := flag.NewFlagSet(consts.AppName, flag.ExitOnError)
	flagSet.BoolVar(&ShowVersionOnly, "version", false, "Show Samaritan version")
	flagSet.StringVar(&DataPath, "data", consts.DefaultDataPath, "Folder for all local files, including pid")
	flagSet.StringVar(&PidFile, "pidfile", "", "Path to pid file")
	flagSet.StringVar(&InstanceID, "id", "", "Custom instance id")
	flagSet.StringVar(&BelongService, "service", "", "Custom service which instance belongs to")
	flagSet.StringVar(&ConfigFile, "config", consts.DefaultConfigFile, "Path to config file")

	// Ignore errors; flagSet is set to ExitOnError.
	flagSet.Parse(os.Args[1:])

	if PidFile == "" {
		PidFile = path.Join(DataPath, consts.DefaultPidFileName)
	}
}
