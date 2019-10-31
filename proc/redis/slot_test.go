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
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseClusterNodesSlot(t *testing.T) {
	tests := []struct {
		segements   []string
		expectSlots []int
		hasError    bool
	}{
		{[]string{"ab"}, []int{}, true},
		{[]string{"1"}, []int{1}, false},
		{[]string{"1-5"}, []int{1, 2, 3, 4, 5}, false},
		{[]string{"[233-<-importing_from_node_id]"}, []int{}, false},
		{[]string{"1", "3-6", "100", "[233-<-importing_from_node_id]"}, []int{1, 3, 4, 5, 6, 100}, false},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			slots, err := parseClusterNodesSlot(test.segements)
			if test.hasError {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, test.expectSlots, slots)
			assert.Nil(t, err)
		})
	}
}

var redis3SampleClusterNodes = `ffffffffffffffffffff00000000980000000822 10.101.90.224:7017 master - 0 1528688887753 7 connected 12288-16383
ffffffffffffffffffff00000000980000000820 10.101.90.227:7000 master - 0 1528688888260 11 connected 4096-8191
ffffffffffffffffffff00000000980000000819 10.101.90.227:7001 slave ffffffffffffffffffff00000000980000000816 0 1528688885722 45 connected
ffffffffffffffffffff00000000980000000818 10.101.90.228:7001 master - 0 1528688884715 3 connected 8192-12287
ffffffffffffffffffff00000000980000000821 10.101.90.224:7016 slave ffffffffffffffffffff00000000980000000818 0 1528688886734 3 connected
ffffffffffffffffffff00000000980000000816 10.101.90.226:7028 myself,master - 0 0 45 connected 0-4095
ffffffffffffffffffff00000000980000000815 10.101.90.226:7029 slave ffffffffffffffffffff00000000980000000820 0 1528688883704 11 connected
ffffffffffffffffffff00000000980000000817 10.101.90.228:7000 slave ffffffffffffffffffff00000000980000000822 0 1528688882695 7 connected
` // The last empty line

var redis4SampleClusterNodes = `f71e056667319f629e228744c249d6fabb5945c0 10.101.67.28:7000@17000 master - 0 1530599123540 11 connected 0-4094
868cd2a259045dca3b38beeb4f65eefe6031584e 10.101.67.29:7000@17000 master - 0 1530599123035 2 connected 8192-12287
3a01120c0e6ef8bfb171c3425ffb7df3f23b0172 10.101.67.27:7001@17001 slave f71e056667319f629e228744c249d6fabb5945c0 0 1530599123035 11 connected
4c4e3841f44dd0e8998185be80cf718f804d8826 10.101.67.27:7000@17000 myself,master - 0 1530599122000 12 connected 4095-8191
051678e70b5836840d87e9ba2c4162f4ae057374 10.101.67.28:7001@17001 master - 0 1530599124543 3 connected 12288-16383
570d030037fba0dd96b03e60bd8579e55cb8da32 10.101.67.27:7002@17002 slave 051678e70b5836840d87e9ba2c4162f4ae057374 0 1530599123000 4 connected
1cb23688b66837e0839d7bc23400f7ee33a9f6b6 10.101.67.28:7002@17002 slave 868cd2a259045dca3b38beeb4f65eefe6031584e 0 1530599123000 6 connected
246d6946b63622eed33ded4abd6187c3c9a23a18 10.101.67.29:7001@17001 slave 4c4e3841f44dd0e8998185be80cf718f804d8826 0 1530599124543 12 connected
`

func TestParseClusterNodes(t *testing.T) {
	t.Run("redis3", func(t *testing.T) {
		testParseClusterNodes(t, redis3SampleClusterNodes)
	})
	t.Run("redis4", func(t *testing.T) {
		testParseClusterNodes(t, redis4SampleClusterNodes)
	})
}

func testParseClusterNodes(t *testing.T, data string) {
	t.Helper()
	hosts, err := parseClusterNodes(data)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(hosts))
	for _, host := range hosts {
		assert.NotEmpty(t, host.ID)
		assert.NotEmpty(t, host.Addr)
		assert.Empty(t, host.MasterID)
		assert.Equal(t, 1, len(host.Replicas))
		assert.NotEmpty(t, host.Slots)
		for _, replica := range host.Replicas {
			assert.Equal(t, host.ID, replica.MasterID)
			assert.NotEmpty(t, replica.ID)
			assert.NotEmpty(t, replica.Addr)
			assert.Empty(t, replica.Replicas)
		}
	}
}
