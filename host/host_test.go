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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseHostType(t *testing.T) {
	for typeName, expectType := range map[string]Type{
		"main":         TypeMain,
		"backup":       TypeBackup,
		"unknown type": TypeMain,
		"":             TypeMain,
	} {
		assert.Equal(t, expectType, ParseType(typeName),
			"typeName: %s, expected: %T(%d)", typeName, expectType, expectType)
	}
}

func TestMarshalHostType(t *testing.T) {
	for _, typ := range []Type{
		TypeMain,
		TypeBackup,
		Type(-1),
	} {
		b, err := json.Marshal(typ)
		assert.NoError(t, err)
		assert.Equal(t, strings.Replace(string(b), "\"", "", -1), typ.String())
	}
}

func TestUnmarshalHostType(t *testing.T) {
	tests := []struct {
		data []byte

		expected Type
		noError  bool
	}{
		{[]byte(strconv.Itoa(0)), TypeMain, true},
		{[]byte(strconv.Itoa(1)), TypeMain, true},
		{[]byte(strconv.Itoa(2)), TypeMain, false},
		{[]byte(`"Main"`), TypeMain, true},
		{[]byte(`"Backup"`), TypeMain, true},
		{[]byte(`"Dummy"`), TypeMain, false},
	}

	for _, test := range tests {
		var typ Type
		err := json.Unmarshal(test.data, &typ)
		if test.noError {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestHostString(t *testing.T) {
	for hostType, expectString := range map[Type]string{
		TypeMain:   "Main",
		TypeBackup: "Backup",
		Type(-1):   "Unknown",
	} {
		assert.Equal(t, expectString, hostType.String(), "%T(%d)", hostType, hostType)
	}
}

func TestMarshalHost(t *testing.T) {
	h := New("1.2.3.4:123")
	b, err := json.Marshal(h)
	assert.NoError(t, err)
	expected := fmt.Sprintf(`{
		"Addr": "%s",
		"Type": "%s"
	}`, h.Addr, h.Type)
	assert.JSONEq(t, expected, string(b))
}

func TestUnmarshalHost(t *testing.T) {
	addr := "1.2.3.4:123"
	typ := TypeMain
	data := []byte(fmt.Sprintf(`{
		"Addr": "%s",
		"Type": "%s"
	}`, addr, typ))
	var h Host
	err := json.Unmarshal(data, &h)
	assert.NoError(t, err)
	assert.Equal(t, h.Addr, addr)
	assert.Equal(t, h.Type, typ)
	assert.Equal(t, true, h.IsHealthy())
}

func TestHostValidateWithInvalidHost(t *testing.T) {
	host := &Host{
		Addr: "1.2.3.4:-1", // Invalid port
		Type: Type(-1),     // Unknown type
	}
	assert.Error(t, host.Validate())
	assert.False(t, host.IsValid())
}

func TestHostValidateWithValidHost(t *testing.T) {
	host := &Host{
		Addr: "1.2.3.4:5",
		Type: TypeMain,
	}
	assert.NoError(t, host.Validate())
	assert.True(t, host.IsValid())
}

func TestIsHostEqual(t *testing.T) {
	tests := []struct {
		h1, h2   *Host
		expected bool
	}{
		{h1: NewWithType(":123", TypeMain), h2: NewWithType(":124", TypeMain), expected: false},
		{h1: NewWithType(":123", TypeMain), h2: NewWithType(":123", TypeBackup), expected: false},
		{h1: NewWithType(":123", TypeBackup), h2: NewWithType(":123", TypeBackup), expected: true},
		{h1: NewWithType(":123", TypeMain), h2: NewWithType(":123", TypeMain), expected: true},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			fmt.Println(test.h1, test.h2)
			assert.Equal(t, IsEqual(test.h1, test.h2), test.expected)
		})
	}
}

func TestHostIncConnCount(t *testing.T) {
	host := New(":12345")
	var wg sync.WaitGroup
	n := 10
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			host.IncConnCount()
		}()
	}
	wg.Wait()
	assert.Equal(t, host.ConnCount(), uint64(n))
}

func TestHostDecConnCount(t *testing.T) {
	host := New(":12345")
	n := 10
	for i := 0; i < n; i++ {
		host.IncConnCount()
	}
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			host.DecConnCount()
		}()
	}
	wg.Wait()
	assert.Equal(t, host.ConnCount(), uint64(0))
}

func TestMarshalHostStats(t *testing.T) {
	h := New("1.2.3.4:123")
	h.IncConnCount()
	h.DecConnCount()
	h.connBytesIn.Store(1000)
	h.connBytesOut.Store(2000)

	stats := h.Stats
	data, err := json.Marshal(stats)
	assert.NoError(t, err)
	expected := `{
		"is_healthy": true,
		"cx_total": 1,
		"cx_active": 0,
		"cx_destroy": 1,
		"cx_rx_bytes_total": 1000,
		"cx_tx_bytes_total": 2000
	}`
	assert.JSONEq(t, expected, string(data))
}

func TestHostIncFailedCount(t *testing.T) {
	host := New(":12345")
	host.IncFailedCount()
	host.IncFailedCount()
	assert.EqualValues(t, host.IncFailedCount(), 3)
}

func TestHostIncSuccessfulCount(t *testing.T) {
	host := New(":12345")
	host.IncSuccessfulCount()
	host.IncSuccessfulCount()
	assert.EqualValues(t, host.IncSuccessfulCount(), 3)
}

func TestHostSetHealthy(t *testing.T) {
	host := New(":12345")
	host.IncSuccessfulCount()
	host.IncFailedCount()
	host.setHealthy()
	assert.Equal(t, host.isHealthy.Load(), true)
	assert.EqualValues(t, host.failedCount.Load(), 0)
	assert.EqualValues(t, host.successfulCount.Load(), 0)
}

func TestHostSetUnhealthy(t *testing.T) {
	host := New(":12345")
	host.IncSuccessfulCount()
	host.IncFailedCount()
	host.setUnhealthy()
	assert.Equal(t, host.isHealthy.Load(), false)
	assert.EqualValues(t, host.failedCount.Load(), 0)
	assert.EqualValues(t, host.successfulCount.Load(), 0)
}

func TestNewSet(t *testing.T) {
	hosts := []*Host{
		NewWithType(":12345", TypeMain),
		NewWithType(":12346", TypeBackup),
		NewWithType(":12347", TypeMain),
	}
	set := NewSet(hosts...)
	assert.Len(t, set.all, 3)
	assert.Len(t, set.healthyMain, 2)
	assert.Len(t, set.healthyBackup, 1)
}

func TestSetAddHost(t *testing.T) {
	set := NewSet()
	hosts := []*Host{
		NewWithType(":123", TypeMain),
		NewWithType(":124", TypeMain),
		NewWithType(":125", TypeBackup),
	}
	var wg sync.WaitGroup
	for _, host := range hosts {
		wg.Add(1)
		go func(host *Host) {
			defer wg.Done()
			set.Add(host)
		}(host)
	}
	wg.Wait()
	assert.Len(t, set.all, 3)
	assert.Len(t, set.healthyMain, 2)
	assert.Len(t, set.healthyBackup, 1)
}

func TestSetRemoveHost(t *testing.T) {
	hosts := []*Host{
		NewWithType(":123", TypeMain),
		NewWithType(":124", TypeMain),
		NewWithType(":125", TypeBackup),
	}
	set := NewSet(hosts...)
	var wg sync.WaitGroup
	for _, host := range hosts {
		wg.Add(1)
		go func(host *Host) {
			defer wg.Done()
			set.Remove(host)
		}(host)
	}
	wg.Wait()
	set.Remove(NewWithType(":126", TypeBackup))
	assert.Len(t, set.all, 0)
	assert.Len(t, set.healthyMain, 0)
	assert.Len(t, set.healthyBackup, 0)
}

func TestSetMarkHostUnhealthy(t *testing.T) {
	host1 := New(":123")
	host1.setUnhealthy()
	host2 := New(":124")
	host3 := New(":125")

	set := NewSet(host3)

	tests := []struct {
		host     *Host
		expected bool
	}{
		{host1, false},
		{host2, false},
		{host3, true},
	}

	var wg sync.WaitGroup
	for _, test := range tests {
		wg.Add(1)
		go func(host *Host, expected bool) {
			assert.Equal(t, set.MarkHostUnhealthy(host), expected)
			wg.Done()
		}(test.host, test.expected)
	}
	wg.Wait()

	hosts := set.Healthy()
	assert.Equal(t, 0, len(hosts))
}

func TestSetMarkHostHealthy(t *testing.T) {
	host1 := NewWithType(":123", TypeMain)
	host2 := NewWithType(":124", TypeMain)
	host2.setUnhealthy()
	host3 := NewWithType(":125", TypeMain)
	host3.setUnhealthy()

	set := NewSet(host3)

	tests := []struct {
		host     *Host
		expected bool
	}{
		{host1, false},
		{host2, false},
		{host3, true},
	}

	var wg sync.WaitGroup
	for _, test := range tests {
		wg.Add(1)
		go func(host *Host, expected bool) {
			assert.Equal(t, set.MarkHostHealthy(host), expected)
			wg.Done()
		}(test.host, test.expected)
	}
	wg.Wait()

	hosts := set.Healthy()
	assert.Equal(t, 1, len(hosts))
}

func TestSetGetHealthy(t *testing.T) {
	host1 := NewWithType(":123", TypeMain)
	host2 := NewWithType(":124", TypeBackup)
	host3 := NewWithType(":98", TypeMain)
	host4 := NewWithType(":201", TypeMain)
	hosts := []*Host{
		host1, host2, host3, host4,
	}
	set := NewSet(hosts...)

	healtyHosts := set.Healthy()
	for i := 0; i < 10; i++ {
		assert.Equal(t, healtyHosts, set.Healthy())
	}

	assert.Equal(t, healtyHosts, []*Host{host1, host4, host3})

	set.MarkHostUnhealthy(host1)
	set.MarkHostUnhealthy(host3)
	set.MarkHostUnhealthy(host4)
	healtyHosts = set.Healthy()
	assert.Equal(t, healtyHosts, []*Host{host2})

	set.MarkHostUnhealthy(host2)
	healtyHosts = set.Healthy()
	assert.Equal(t, healtyHosts, []*Host{})
}

func TestSetGetAll(t *testing.T) {
	host1 := NewWithType(":123", TypeMain)
	host2 := NewWithType(":124", TypeBackup)
	host3 := NewWithType(":125", TypeBackup)
	hosts := []*Host{
		host1, host2,
	}
	set := NewSet(hosts...)
	set.Add(host3)
	assert.Equal(t, len(set.All()), 3)
}

func TestSetLen(t *testing.T) {
	host1 := NewWithType(":123", TypeMain)
	host2 := NewWithType(":124", TypeBackup)
	host3 := NewWithType(":125", TypeBackup)

	set := NewSet(host1, host2)
	assert.Equal(t, set.Len(), 2)
	set.Add(host3)
	assert.Equal(t, set.Len(), 3)
	set.Remove(host1, host2)
	assert.Equal(t, set.Len(), 1)
}

func TestSetRandom(t *testing.T) {
	host1 := NewWithType(":123", TypeMain)
	host2 := NewWithType(":124", TypeBackup)
	host3 := NewWithType(":125", TypeMain)
	set := NewSet()
	assert.Nil(t, set.Random())
	set.Add(host1)
	assert.Equal(t, host1, set.Random())
	set.Remove(host1)
	set.Add(host2)
	assert.Equal(t, host2, set.Random())
	set.ReplaceAll([]*Host{host1, host3})
	if host := set.Random(); host != host1 && host != host3 {
		t.Fatalf("assert equal with host1 or host2")
	}
}

func TestSetExist(t *testing.T) {
	host1 := NewWithType(":123", TypeMain)
	host2 := NewWithType(":124", TypeBackup)
	host3 := NewWithType(":125", TypeMain)
	hosts := []*Host{host1, host2, host3}
	set := NewSet(hosts...)
	for _, h := range hosts {
		assert.True(t, set.Exist(h.Addr))
	}
}

func TestSetReplaceAll(t *testing.T) {
	host1 := NewWithType(":123", TypeMain)
	host2 := NewWithType(":124", TypeBackup)
	host3 := NewWithType(":125", TypeBackup)

	set := NewSet(host1, host2)
	assert.Equal(t, len(set.All()), 2)
	set.ReplaceAll([]*Host{host3})
	assert.Equal(t, len(set.All()), 1)
}

func TestMarkHostRemoved(t *testing.T) {
	host := NewWithType(":1234", TypeMain)
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-host.WaitRemoved()
	}()
	host.markRemoved()
	<-done

	// no panic
	host.markRemoved()
}
