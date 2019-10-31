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
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/samaritan-proxy/samaritan/pb/common"
	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	hostpkg "github.com/samaritan-proxy/samaritan/host"
	"github.com/samaritan-proxy/samaritan/logger"
	"github.com/samaritan-proxy/samaritan/proc"
	_ "github.com/samaritan-proxy/samaritan/proc/redis"
	"github.com/samaritan-proxy/samaritan/utils"
)

func init() {
	rand.Seed(time.Now().Unix())
}

var (
	redisSvrPorts = [4]int{8000, 8001, 8002, 8003}

	proxyPort int
	processor proc.Proc

	defaultClusterManager = NewRedisCliClusterManager()
)

type ClusterManager interface {
	Start(addrs ...*net.TCPAddr) error
	NodeID(addr *net.TCPAddr) (string, error)
	AddMaster(addr *net.TCPAddr) error
	AddSlave(slave, master *net.TCPAddr) error
	Failover(slave *net.TCPAddr) error
	DelNode(addr *net.TCPAddr) error
	ReShard(from, to *net.TCPAddr, slot int) error
	AddAuth(password string) error
	DisableAuth() error
	Shutdown() error
}

type Node struct {
	ID   string
	Addr *net.TCPAddr

	Master *Node
}

type RedisCliClusterManager struct {
	nodes    map[string]*Node // <addr, Node>
	password string
}

func NewRedisCliClusterManager() ClusterManager {
	return &RedisCliClusterManager{
		nodes: make(map[string]*Node),
	}
}

func (m *RedisCliClusterManager) Start(addrs ...*net.TCPAddr) error {
	if len(m.nodes) != 0 {
		return fmt.Errorf("cluster is running")
	}
	if len(addrs) == 0 {
		return fmt.Errorf("addrs is none")
	}

	_, err := exec.Command("/bin/sh", "-c", "which redis-server").Output()
	if err != nil {
		return fmt.Errorf("unable to find redis-server: %v", err)
	}
	_, err = exec.Command("/bin/sh", "-c", "which redis-cli").Output()
	if err != nil {
		return fmt.Errorf("unable to find redis-cli: %v", err)
	}

	for _, addr := range addrs {
		if err := m.startRedisServer(addr.Port); err != nil {
			return err
		}

		m.doUntilSuccess(func() bool {
			return m.isNodeReady(addr)
		})

		nodeID, err := m.getNodeID(addr)
		if err != nil {
			return err
		}
		m.nodes[addr.String()] = &Node{
			ID:   nodeID,
			Addr: addr,
		}
	}

	params := []string{
		"--cluster",
		"create",
	}
	for _, addr := range addrs {
		params = append(params, addr.String())
	}
	params = append(params, "--cluster-yes")

	if _, err := exec.Command("redis-cli", params...).Output(); err != nil {
		return fmt.Errorf("failed to init redis cluster: %v", err)
	}
	m.doUntilSuccess(m.isClusterReady)
	return nil
}

func (m *RedisCliClusterManager) Shutdown() error {
	for _, node := range m.nodes {
		m.flushAll(node.Addr)
	}
	for _, node := range m.nodes {
		m.clusterReset(node.Addr, true)
	}
	for _, node := range m.nodes {
		m.shutdown(node.Addr, false)
	}
	m.nodes = make(map[string]*Node)
	m.password = ""
	return nil
}

func (m *RedisCliClusterManager) isClusterReady() bool {
	count := 0
	for _, node := range m.nodes {
		b, err := m.sendCommand(node.Addr, "CLUSTER", "INFO")
		if err != nil {
			return false
		}
		res := regexp.MustCompile(`cluster_state:(\S+)`).FindSubmatch(b)
		if len(res) != 2 {
			return false
		}
		if string(res[1]) != "ok" {
			return false
		}

		b, err = m.sendCommand(node.Addr, "CLUSTER", "NODES")
		if err != nil {
			return false
		}
		for _, _node := range m.nodes {
			if bytes.Contains(b, []byte(_node.ID)) {
				count++
			}
		}
	}
	return count == len(m.nodes)*len(m.nodes)
}

func (m *RedisCliClusterManager) isNodeReady(addr *net.TCPAddr) bool {
	_, err := m.sendCommand(addr, "PING")
	return err == nil
}

func (m *RedisCliClusterManager) doUntilSuccess(fn func() bool) {
	for {
		if fn() {
			return
		}
		logger.Warnf("call func[%s] failed, retry after 1s", runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name())
		time.Sleep(time.Second)
	}
}

func (m *RedisCliClusterManager) NodeID(addr *net.TCPAddr) (string, error) {
	node, ok := m.nodes[addr.String()]
	if !ok {
		return "", fmt.Errorf("node[%s] not in cluster", addr.String())
	}
	return node.ID, nil
}

func (m *RedisCliClusterManager) getRandomNode() *Node {
	nodes := make([]*Node, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	return nodes[rand.Intn(len(nodes))]
}

func (m *RedisCliClusterManager) AddMaster(addr *net.TCPAddr) error {
	if _, ok := m.nodes[addr.String()]; ok {
		return nil
	}

	if err := m.startRedisServer(addr.Port); err != nil {
		return err
	}

	m.doUntilSuccess(func() bool {
		return m.isNodeReady(addr)
	})

	nodeID, err := m.getNodeID(addr)
	if err != nil {
		return err
	}
	_, err = exec.Command(
		"redis-cli",
		"--cluster",
		"add-node",
		addr.String(),
		m.getRandomNode().Addr.String(),
	).Output()
	if err != nil {
		return err
	}
	m.nodes[addr.String()] = &Node{
		ID:   nodeID,
		Addr: addr,
	}
	m.doUntilSuccess(m.isClusterReady)
	return nil
}

func (m *RedisCliClusterManager) AddSlave(slave, master *net.TCPAddr) error {
	if _, ok := m.nodes[slave.String()]; ok {
		return fmt.Errorf("slave node exist")
	}
	masterNode, ok := m.nodes[master.String()]
	if !ok {
		return fmt.Errorf("master node not exist")
	}

	if err := m.startRedisServer(slave.Port); err != nil {
		return err
	}
	m.doUntilSuccess(func() bool {
		return m.isNodeReady(slave)
	})

	nodeID, err := m.getNodeID(slave)
	if err != nil {
		return err
	}

	_, err = exec.Command(
		"redis-cli",
		"--cluster",
		"add-node",
		slave.String(),
		master.String(),
		"--cluster-slave",
		"--cluster-master-id",
		masterNode.ID,
	).Output()
	if err != nil {
		return err
	}
	slaveNode := &Node{
		ID:     nodeID,
		Addr:   slave,
		Master: masterNode,
	}
	m.nodes[slave.String()] = slaveNode
	m.doUntilSuccess(m.isClusterReady)
	return nil
}

func (m *RedisCliClusterManager) Failover(slave *net.TCPAddr) error {
	slaveNode, ok := m.nodes[slave.String()]
	if !ok {
		return fmt.Errorf("slave not node exist")
	}
	masterNode := slaveNode.Master
	if masterNode == nil {
		return fmt.Errorf("not a slave node")
	}

	_, err := m.sendCommand(slave, "CLUSTER", "FAILOVER")
	if err != nil {
		return err
	}

	// switch
	masterNode, slaveNode = slaveNode, masterNode
	slaveNode.Master = masterNode
	masterNode.Master = nil

	m.doUntilSuccess(m.isClusterReady)
	return nil
}

func (m *RedisCliClusterManager) DelNode(addr *net.TCPAddr) error {
	node, ok := m.nodes[addr.String()]
	if !ok {
		return nil
	}
	// del-node will shutdown server and return exit code 1,
	// ignore error here.
	// https://github.com/antirez/redis/issues/5687
	// https://github.com/antirez/redis/issues/5815
	exec.Command(
		"redis-cli",
		"--cluster",
		"del-node",
		addr.String(),
		node.ID,
	).Output()
	delete(m.nodes, addr.String())
	m.doUntilSuccess(m.isClusterReady)
	return nil
}

func (m *RedisCliClusterManager) ReShard(from, to *net.TCPAddr, slot int) error {
	fromNode, ok := m.nodes[from.String()]
	if !ok {
		return nil
	}
	toNode, ok := m.nodes[to.String()]
	if !ok {
		return nil
	}
	_, err := exec.Command(
		"redis-cli",
		"--cluster",
		"reshard",
		from.String(),
		"--cluster-from", fromNode.ID,
		"--cluster-to", toNode.ID,
		"--cluster-slots", strconv.Itoa(slot),
		"--cluster-yes",
	).Output()
	return err
}

func (m *RedisCliClusterManager) startRedisServer(port int) error {
	dataDir := fmt.Sprintf("/var/lib/redis/cluster/%d", port)
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		os.RemoveAll(dataDir)
	}
	if err := os.MkdirAll(dataDir, 0666); err != nil {
		return fmt.Errorf("unable to make data dir %s: %v", dataDir, err)
	}

	configDir := "/etc/redis"
	if err := os.MkdirAll(configDir, 0666); err != nil {
		return fmt.Errorf("unable to make config dir %s: %v", configDir, err)
	}

	configPath := fmt.Sprintf("%s/%d.conf", configDir, port)
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %v", configPath, err)
	}

	config := fmt.Sprintf("port %d\ndir %s\ncluster-enabled yes\n", port, dataDir)
	if _, err := f.WriteString(config); err != nil {
		f.Close()
		return fmt.Errorf("unable to write config to file %s: %v", configPath, err)
	}
	f.Close()

	redisCmd := exec.Command(
		"redis-server",
		configPath,
	)

	stdout, err := redisCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to run redis-server[%d]: %v", port, err)
	}
	stderr, err := redisCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to run redis-server[%d]: %v", port, err)
	}
	if err := redisCmd.Start(); err != nil {
		return fmt.Errorf("failed to run redis-server[%d]: %v", port, err)
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	go func() {
		if err := redisCmd.Wait(); err != nil {
			logger.Fatalf("Failed to run redis-server[%d]: %v", port, err)
		}
	}()
	return nil
}

func (m *RedisCliClusterManager) sendCommand(addr *net.TCPAddr, command ...string) ([]byte, error) {
	params := []string{
		"-h", addr.IP.String(),
		"-p", strconv.Itoa(addr.Port),
	}
	if m.password != "" {
		params = append(params, "-a", m.password)
	}
	params = append(params, command...)
	return exec.Command("redis-cli", params...).Output()
}

func (m *RedisCliClusterManager) getNodeID(addr *net.TCPAddr) (string, error) {
	b, err := m.sendCommand(addr, "CLUSTER", "MYID")
	if err != nil {
		return "", fmt.Errorf("failed to send cluster myid, node: %s", addr.String())
	}
	return strings.TrimSpace(string(b)), nil
}

func (m *RedisCliClusterManager) flushAll(addr *net.TCPAddr) error {
	_, err := m.sendCommand(addr, "FLUSHALL")
	if err != nil {
		return fmt.Errorf("failed to send flushall, node: %s", addr.String())
	}
	return nil
}

func (m *RedisCliClusterManager) clusterReset(addr *net.TCPAddr, hard bool) error {
	var err error
	if hard {
		_, err = m.sendCommand(addr, "CLUSTER", "RESET", "HARD")
	} else {
		_, err = m.sendCommand(addr, "CLUSTER", "RESET")
	}
	if err != nil {
		return fmt.Errorf("failed to send cluster reset, node: %s", addr.String())
	}
	return nil
}

func (m *RedisCliClusterManager) shutdown(addr *net.TCPAddr, save bool) error {
	var err error
	if save {
		_, err = m.sendCommand(addr, "SHUTDOWN", "SAVE")
	} else {
		_, err = m.sendCommand(addr, "SHUTDOWN", "NOSAVE")
	}
	if err != nil {
		return fmt.Errorf("failed to send shutdown, node: %s", addr.String())
	}
	return nil
}

func (m *RedisCliClusterManager) AddAuth(password string) error {
	for _, node := range m.nodes {
		_, err := m.sendCommand(node.Addr, "CONFIG", "SET", "REQUIREPASS", password)
		if err != nil {
			return fmt.Errorf("failed to set password, node: %s", node.Addr.String())
		}
	}
	m.password = password
	return nil
}

func (m *RedisCliClusterManager) DisableAuth() error {
	if m.password == "" {
		return nil
	}
	for _, node := range m.nodes {
		_, err := m.sendCommand(node.Addr, "CONFIG", "SET", "REQUIREPASS", "")
		if err != nil {
			return fmt.Errorf("failed to clean password, node: %s", node.Addr.String())
		}
	}
	m.password = ""
	return nil
}

func setupRedisProxy() {
	hosts := make([]*hostpkg.Host, 0)
	// the fourth redis server is alone, not the memeber of cluster.
	for i := 0; i < 3; i++ {
		addr := fmt.Sprintf("127.0.0.1:%d", redisSvrPorts[i])
		hosts = append(hosts, hostpkg.New(addr))
	}
	cfg := &service.Config{
		Listener: &service.Listener{
			Address: &common.Address{
				Ip:   "127.0.0.1",
				Port: 0,
			},
		},
		ConnectTimeout: utils.DurationPtr(time.Second),
		IdleTimeout:    utils.DurationPtr(time.Minute * 20),
		Protocol:       protocol.Redis,
		ProtocolOptions: &service.Config_RedisOption{
			RedisOption: &protocol.RedisOption{
				ReadStrategy: protocol.RedisOption_MASTER,
			},
		},
	}
	p, err := proc.New("redis-integration-test", cfg, hosts)
	if err != nil {
		logger.Fatalf("Unable to run redis proxy: %v", err)
	}

	if err := p.Start(); err != nil {
		logger.Fatalf("Unable to run redis proxy: %v", err)
	}

	time.Sleep(time.Second) // wait proxy started

	addr := p.Address()
	_, port, _ := net.SplitHostPort(addr)
	if port == "0" {
		logger.Fatal("Start redis proxy failed or timeout")
	}
	proxyPort, _ = strconv.Atoi(port)
	processor = p
}

func getProxyAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", proxyPort)
}

const AuthPwd = "abc"

func TestMain(m *testing.M) {
	nodes := make([]*net.TCPAddr, 0)
	for _, port := range redisSvrPorts[:3] {
		addr, _ := net.ResolveTCPAddr("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
		nodes = append(nodes, addr)
	}
	if err := defaultClusterManager.Start(nodes...); err != nil {
		logger.Fatal(err)
	}
	setupRedisProxy()
	initRedisClient()

	code := m.Run()

	defaultClusterManager.Shutdown()
	os.Exit(code)
}
