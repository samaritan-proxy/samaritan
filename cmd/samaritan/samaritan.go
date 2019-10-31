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

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kirk91/stats/sink/statsd"

	"github.com/samaritan-proxy/samaritan/admin"
	"github.com/samaritan-proxy/samaritan/cmd/samaritan/flag"
	"github.com/samaritan-proxy/samaritan/cmd/samaritan/hotrestart"
	"github.com/samaritan-proxy/samaritan/config"
	"github.com/samaritan-proxy/samaritan/consts"
	"github.com/samaritan-proxy/samaritan/controller"
	"github.com/samaritan-proxy/samaritan/logger"
	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/bootstrap"
	"github.com/samaritan-proxy/samaritan/stats"
	"github.com/samaritan-proxy/samaritan/utils"
)

var (
	parentPid int
	c         *config.Config
	inst      *instance
)

func init() {
	checkProcInherit()
}

func checkProcInherit() {
	ppid, err := strconv.Atoi(os.Getenv(consts.ParentPidEnv))
	if err != nil {
		parentPid = -1
	}
	parentPid = ppid
}

type instance struct {
	*hotrestart.Restarter
	id       int
	parentID int

	c     *config.Config
	ctl   *controller.Controller
	admin *admin.Server

	shutdownAdminOnce  sync.Once
	drainListenersOnce sync.Once
}

// initInstance inits the global instance.
func initInstance() {
	server := admin.New(c)
	ctl, err := controller.New(c.Subscribe())
	if err != nil {
		logger.Fatalf("Error while initializing controller: %v", err)
	}

	inst = &instance{
		id:       os.Getpid(),
		parentID: parentPid,
		c:        c,
		admin:    server,
		ctl:      ctl,
	}
	r, err := hotrestart.New(inst)
	if err != nil {
		logger.Fatalf("Error while initializing restarter: %v", err)
	}
	inst.Restarter = r
	inst.Run()
}

// ID returns the instance id.
func (inst *instance) ID() int {
	return inst.id
}

// ParentID returns the parent id of instance.
func (inst *instance) ParentID() int {
	return inst.parentID
}

// Run runs the instance.
func (inst *instance) Run() {
	if err := inst.ctl.Start(); err != nil {
		logger.Fatalf("Error while starting broker: %v", err)
	}

	inst.ShutdownParentAdmin()
	if err := inst.admin.Start(inst.parentID > 0); err != nil {
		logger.Fatalf("Error while starting admin api: %s", err)
	}

	inst.DrainParentListeners()
	if inst.parentID > 0 {
		go func() {
			d, err := time.ParseDuration(os.Getenv(consts.ParentTerminateTimeEnv))
			if err != nil {
				d = consts.DefaultParentTerminateTime
			}
			logger.Infof("Will terminate parent %d after %s", inst.parentID, d)
			time.Sleep(d)
			inst.TerminateParent()
		}()
	}
}

// ShutdownAdmin shutdowns the admin api.
func (inst *instance) ShutdownAdmin() {
	inst.shutdownAdminOnce.Do(func() {
		logger.Infof("Shutdown admin api...")
		inst.admin.Stop()
	})
}

// DrainListeners drains the listeners.
func (inst *instance) DrainListeners() {
	inst.drainListenersOnce.Do(func() {
		logger.Infof("Drain listeners...")
		inst.ctl.DrainListeners()
	})
}

// Shutdown shutdowns the instance.
func (inst *instance) Shutdown() {
	inst.admin.Stop()
	inst.ctl.Stop()
	inst.Restarter.Shutdown()
}

func writePid() {
	pid := os.Getpid()
	logger.Info("PID: ", pid)
	pidFile := flag.PidFile
	oldPid, err := ioutil.ReadFile(pidFile)
	if err == nil {
		p, err := strconv.Atoi(string(oldPid))
		if err == nil {
			err = syscall.Kill(p, 0)
			if err == nil {
				logger.Warnf("Possibly running samaritan process[%d] detected, overriding pid file with %d", p, pid)
			}
		}
	}
	if err := utils.WriteFileAtomically(pidFile, []byte(strconv.FormatInt(int64(pid), 10)), 0644); err != nil {
		logger.Warn("Failed to create pid file:", err)
	}
}

func handleSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2)
	for s := range c {
		logger.Info("Signal received: ", s)
		switch s {
		case syscall.SIGINT, syscall.SIGTERM: // exit
			inst.Shutdown()
			logger.Info("Ready to exit, bye bye...")
			os.Exit(0)
		default:
			logger.Warnf("Ignore signal: %s", s)
		}
	}
}

func showSamVersion() {
	fmt.Printf("%s %s\n%s\n", strings.Title(consts.AppName), consts.Version, consts.Build)
}

func parseFlag() {
	if flag.ShowVersionOnly {
		showSamVersion()
		os.Exit(0)
	}

	initConfig()
	initLogger()

	// display the numbers of CPU that samaritan used.
	logger.Info("Using CPUs: ", runtime.GOMAXPROCS(-1))
	logger.Info("Version: ", consts.Version)
	if len(consts.Build) != 0 {
		logger.Info("Build: ", consts.Build)
	}
}

func initConfig() {
	if flag.ConfigFile == "" {
		log.Fatal("Config file cannot be empty")
	}

	logger.Info("Config file: ", flag.ConfigFile)
	b, err := config.LoadBootstrap(flag.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to load bootstrap: %v", err)
	}

	// handle the metadata of instance.
	if b.Instance == nil {
		b.Instance = &common.Instance{}
	}
	b.Instance.Version = consts.Version
	b.Instance.Id = utils.PickFirstNonEmptyStr(
		flag.InstanceID,
		b.Instance.Id,
		genDefaultInstanceID(b.Admin.Bind.Port),
	)
	b.Instance.Belong = utils.PickFirstNonEmptyStr(
		flag.BelongService,
		b.Instance.Belong,
	)

	c, err = config.New(b)
	if err != nil {
		logger.Fatalf("Failed to init config: %v", err)
	}
}

func genDefaultInstanceID(port uint32) string {
	ip, err := utils.GetLocalIP()
	if err != nil {
		logger.Fatalf("Failed to get local ip: %s", err.Error())
	}
	return fmt.Sprintf("%s_%d", ip, port)
}

func initLogger() {
	level := c.Log.Level
	output := c.Log.Output
	switch output.Type {
	case bootstrap.STDOUT:
		// do nothing
	case bootstrap.SYSLOG:
		tag := fmt.Sprintf("%s@%s", consts.AppName, c.Instance.Id)
		logger.InitSyslog(output.Target, tag)
	}
	logger.SetLevel(level.String())
}

func initStats() {
	var sinks []stats.Sink
	for _, cfg := range c.Stats.Sinks {
		switch cfg.Type {
		case bootstrap.STATSD:
			sinks = append(sinks, statsd.New(cfg.Endpoint, consts.AppName))
		default:
			logger.Warnf("unknown sink: %s", cfg.String())
			continue
		}
	}

	if err := stats.Init(sinks...); err != nil {
		logger.Fatal(err)
	}
	go func() {
		scope := stats.CreateScope("")
		ticker := time.NewTicker(time.Second * 5)
		for range ticker.C {
			scope.Gauge("live").Set(1)
			// TODO(kirk91): add uptime metric
		}
	}()
}

func main() {
	parseFlag()
	initStats()

	initInstance()
	writePid()

	handleSignal()
}
