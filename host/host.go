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

package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/atomic"

	"github.com/samaritan-proxy/samaritan/utils"
	multierror "github.com/samaritan-proxy/samaritan/utils/multi-errors"
)

// Type indicates the type of host.
type Type int

// Available host types.
const (
	TypeMain Type = iota
	TypeBackup
)

// String returns the string representation of the host's type.
func (typ Type) String() string {
	switch typ {
	case TypeMain:
		return "Main"
	case TypeBackup:
		return "Backup"
	default:
		return "Unknown"
	}
}

var errUnknownType = errors.New("unknown type")

// UnmarshalJSON implements json.Unmarshaler interface.
func (typ *Type) UnmarshalJSON(data []byte) error {
	// compatiable with old version
	if i, err := strconv.Atoi(string(data)); err == nil {
		switch Type(i) {
		case TypeMain:
			*typ = TypeMain
		case TypeBackup:
			*typ = TypeBackup
		default:
			return errUnknownType
		}
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case TypeMain.String():
		*typ = TypeMain
	case TypeBackup.String():
		*typ = TypeBackup
	default:
		return errUnknownType
	}
	return nil
}

// MarshalJSON implements json.Marshaler interface.
func (typ Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(typ.String())
}

// ParseType always returns a valid Type.
// If given "backup", it returns TypeBackup.
// Otherwise it returns TypeMain.
func ParseType(typ string) Type {
	if strings.EqualFold(typ, "backup") {
		return TypeBackup
	}
	return TypeMain
}

// IsEqual returns whether two hosts are the same.
func IsEqual(h1, h2 *Host) bool {
	if h1 == nil || h2 == nil {
		return false
	}
	return h1.Addr == h2.Addr && h1.Type == h2.Type
}

// Host represents a backend host.
type Host struct {
	Addr string
	Type Type
	*Stats

	removeOnce sync.Once
	removeCh   chan struct{}
}

// New creates a host instance.
func New(addr string) *Host {
	return NewWithType(addr, TypeMain)
}

// NewWithType creates a host instance with specified type.
func NewWithType(addr string, typ Type) *Host {
	return &Host{
		Addr:     addr,
		Type:     typ,
		Stats:    NewStats(),
		removeCh: make(chan struct{}),
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (h *Host) MarshalJSON() ([]byte, error) {
	v := struct {
		Addr string `json:"Addr"`
		Type Type   `json:"Type"`
	}{
		Addr: h.Addr,
		Type: h.Type,
	}
	return json.Marshal(v)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (h *Host) UnmarshalJSON(data []byte) error {
	v := struct {
		Addr string `json:"Addr"`
		Type Type   `json:"Type"`
	}{}
	err := json.Unmarshal(data, &v)
	if err == nil {
		*h = *NewWithType(v.Addr, v.Type)
	}
	return err
}

// String returns the string representation of Host.
func (h *Host) String() string {
	return fmt.Sprintf("%s(%s)", h.Addr, h.Type)
}

// IsValid returns true if host is valid.
func (h *Host) IsValid() bool {
	return h.Validate() == nil
}

// Validate validates inner address and type and returns the first error.
func (h *Host) Validate() error {
	mErr := multierror.NewWithError(utils.VerifyTCP4Address(h.Addr))
	if h.Type != TypeMain && h.Type != TypeBackup {
		mErr.Add(errors.New("unknown host type"))
	}
	return mErr.ErrorOrNil()
}

// IsHealthy returns whether the host is healthy.
func (h *Host) IsHealthy() bool {
	return h.isHealthy.Load()
}

// WaitRemoved close when this host removed.
func (h *Host) WaitRemoved() <-chan struct{} {
	return h.removeCh
}

func (h *Host) markRemoved() {
	h.removeOnce.Do(func() {
		close(h.removeCh)
	})
}

// Stats describes the stats of a host.
type Stats struct {
	isHealthy       atomic.Bool
	connTotal       atomic.Uint64
	connActive      atomic.Uint64
	connDestroy     atomic.Uint64
	connBytesIn     atomic.Uint64
	connBytesOut    atomic.Uint64
	failedCount     atomic.Uint64
	successfulCount atomic.Uint64
}

// NewStats creates a stats container for host.
func NewStats() *Stats {
	stats := new(Stats)
	stats.isHealthy.Store(true)
	return stats
}

// MarshalJSON implements the json.Marshaler interface.
func (stats *Stats) MarshalJSON() ([]byte, error) {
	v := struct {
		IsHealthy    bool   `json:"is_healthy"`
		ConnTotal    uint64 `json:"cx_total"`
		ConnActive   uint64 `json:"cx_active"`
		ConnDestroy  uint64 `json:"cx_destroy"`
		ConnBytesIn  uint64 `json:"cx_rx_bytes_total"`
		ConnBytesOut uint64 `json:"cx_tx_bytes_total"`
	}{
		IsHealthy:    stats.isHealthy.Load(),
		ConnTotal:    stats.connTotal.Load(),
		ConnActive:   stats.connActive.Load(),
		ConnDestroy:  stats.connDestroy.Load(),
		ConnBytesIn:  stats.connBytesIn.Load(),
		ConnBytesOut: stats.connBytesOut.Load(),
	}
	return json.Marshal(v)
}

// IncConnCount increases the connection count by 1.
func (stats *Stats) IncConnCount() {
	stats.connTotal.Inc()
	stats.connActive.Inc()
}

// DecConnCount decreases the connection count by 1.
func (stats *Stats) DecConnCount() {
	stats.connDestroy.Inc()
	stats.connActive.Dec()
}

// ConnCount returns the connection count.
func (stats *Stats) ConnCount() uint64 {
	return stats.connActive.Load()
}

// IncFailedCount increase failed count by 1 and clears successful count.
func (stats *Stats) IncFailedCount() uint64 {
	stats.successfulCount.Store(0)
	return stats.failedCount.Inc()
}

// IncSuccessfulCount increase successful count by 1 and clears failed count.
func (stats *Stats) IncSuccessfulCount() uint64 {
	stats.failedCount.Store(0)
	return stats.successfulCount.Inc()
}

func (stats *Stats) setHealthy() bool {
	stats.successfulCount.Store(0)
	stats.failedCount.Store(0)
	return stats.isHealthy.CAS(false, true)
}

func (stats *Stats) setUnhealthy() bool {
	stats.successfulCount.Store(0)
	stats.failedCount.Store(0)
	return stats.isHealthy.CAS(true, false)
}

// ConnBytesInCounter return input bytes counter.
func (stats *Stats) ConnBytesInCounter() *atomic.Uint64 {
	return &stats.connBytesIn
}

// ConnBytesOutCounter return output bytes counter.
func (stats *Stats) ConnBytesOutCounter() *atomic.Uint64 {
	return &stats.connBytesOut
}

// Set is a hosts container.
type Set struct {
	sync.RWMutex
	all           map[string]*Host
	healthyMain   map[string]*Host
	healthyBackup map[string]*Host
	// the cache of current healthy hosts
	healthyCache atomic.Value // []*Host
}

// NewSet creates a set.
func NewSet(hosts ...*Host) *Set {
	s := &Set{
		all:           make(map[string]*Host),
		healthyMain:   make(map[string]*Host),
		healthyBackup: make(map[string]*Host),
	}
	s.add(hosts...)
	return s
}

func (set *Set) addToHealthy(host ...*Host) {
	if len(host) == 0 {
		return
	}
	for _, h := range host {
		if h == nil {
			continue
		}
		switch h.Type {
		case TypeMain:
			set.healthyMain[h.Addr] = h
		case TypeBackup:
			set.healthyBackup[h.Addr] = h
		default:
			continue
		}
	}
	set.buildHealthyCache()
}

func (set *Set) removeFromHealthy(host ...*Host) {
	if len(host) == 0 {
		return
	}
	for _, h := range host {
		if h == nil {
			continue
		}
		switch h.Type {
		case TypeMain:
			delete(set.healthyMain, h.Addr)
		case TypeBackup:
			delete(set.healthyBackup, h.Addr)
		default:
			continue
		}
	}
	set.buildHealthyCache()
}

func (set *Set) buildHealthyCache() {
	hostMap := set.healthy()

	keys := make([]string, 0, len(hostMap))
	for k := range hostMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	hosts := make([]*Host, 0, len(hostMap))
	for _, k := range keys {
		hosts = append(hosts, hostMap[k])
	}

	set.healthyCache.Store(hosts)
}

// Add adds host to the set.
func (set *Set) Add(hosts ...*Host) {
	set.Lock()
	defer set.Unlock()
	set.add(hosts...)
}

func (set *Set) add(hosts ...*Host) {
	if len(hosts) == 0 {
		return
	}
	for _, host := range hosts {
		set.all[host.Addr] = host
	}
	set.addToHealthy(hosts...)
}

// Remove removes host from the set.
func (set *Set) Remove(hosts ...*Host) {
	set.Lock()
	defer set.Unlock()
	set.remove(hosts...)
}

func (set *Set) remove(hosts ...*Host) {
	if len(hosts) == 0 {
		return
	}
	for _, host := range hosts {
		delete(set.all, host.Addr)
		host.markRemoved()
	}
	set.removeFromHealthy(hosts...)
}

// MarkHostHealthy marks the given host as healthy.
func (set *Set) MarkHostHealthy(host *Host) bool {
	if !host.setHealthy() {
		return false
	}
	set.Lock()
	defer set.Unlock()
	if _, ok := set.all[host.Addr]; !ok {
		return false
	}
	set.addToHealthy(host)
	return true
}

// MarkHostUnhealthy marks the given host as unhealthy.
func (set *Set) MarkHostUnhealthy(host *Host) bool {
	if !host.setUnhealthy() {
		return false
	}
	set.Lock()
	defer set.Unlock()
	if _, ok := set.all[host.Addr]; !ok {
		return false
	}
	set.removeFromHealthy(host)
	return true
}

func (set *Set) healthy() map[string]*Host {
	healthyHosts := set.healthyMain
	if len(healthyHosts) == 0 {
		healthyHosts = set.healthyBackup
	}
	return healthyHosts
}

// Healthy returns the healthy hosts.
func (set *Set) Healthy() []*Host {
	hosts, _ := set.healthyCache.Load().([]*Host)
	return hosts
}

// All returns the all hosts.
func (set *Set) All() []*Host {
	set.RLock()
	defer set.RUnlock()
	hosts := make([]*Host, 0, len(set.all))
	for _, host := range set.all {
		hosts = append(hosts, host)
	}
	return hosts
}

func (set *Set) Len() int {
	set.RLock()
	defer set.RUnlock()
	return len(set.all)
}

func (set *Set) Random() *Host {
	set.RLock()
	defer set.RUnlock()

	healthyHosts := set.healthy()
	l := len(healthyHosts)
	if l == 0 {
		return nil
	}
	i := 0
	target := rand.Intn(l)
	for _, v := range healthyHosts {
		if i == target {
			return v
		}
		if i < target {
			i++
			continue
		}
	}
	return nil
}

func (set *Set) Exist(addr string) bool {
	set.RLock()
	defer set.RUnlock()
	_, ok := set.all[addr]
	return ok
}

// ReplaceAll replaces the internal hosts atomicly.
func (set *Set) ReplaceAll(hosts []*Host) {
	set.Lock()
	defer set.Unlock()
	for _, host := range set.all {
		set.remove(host)
	}
	set.add(hosts...)
}
