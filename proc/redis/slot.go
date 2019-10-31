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
	"errors"
	"strconv"
	"strings"
)

type redisHost struct {
	ID       string
	Addr     string
	MasterID string
	Replicas []*redisHost
	Slots    []int
}

var errInvalidClusterNodes = errors.New("invalid cluster nodes")

func parseClusterNodes(data string) (map[string]*redisHost, error) {
	lines := strings.Split(data, "\n")
	hosts := make(map[string]*redisHost)
	for _, line := range lines {
		fields := strings.Fields(line)
		// the last line is empty
		if len(fields) == 0 {
			continue
		}
		if len(fields) < 8 {
			return nil, errInvalidClusterNodes
		}

		id := fields[0]
		addr := strings.Split(fields[1], "@")[0]
		if len(strings.Split(addr, ":")) != 2 {
			return nil, errInvalidClusterNodes
		}
		// TODO: detect flags
		host := &redisHost{ID: id, Addr: addr}
		hosts[id] = host

		isMaster := fields[3] == "-"
		if !isMaster {
			host.MasterID = fields[3]
			continue
		}

		// attach slots to master node
		if len(fields) < 9 {
			return nil, errInvalidClusterNodes
		}
		slots, err := parseClusterNodesSlot(fields[8:])
		if err != nil {
			return nil, err
		}
		host.Slots = slots
	}

	// restructure replicas
	for id, host := range hosts {
		if host.MasterID == "" {
			continue
		}
		master := hosts[host.MasterID]
		master.Replicas = append(master.Replicas, host)
		delete(hosts, id)
	}
	return hosts, nil
}

func parseClusterNodesSlot(segements []string) ([]int, error) {
	// Format: <the first 8 fields> <slot> <slot> ...
	// <slot> has 3 cases:
	// (1) single slot number: "233"
	// (2) slot range: "233-666"
	// (3) importing and migrating slot: [slot_number-<-importing_from_node_id]
	// We will ignore the (3) case here.
	slots := make([]int, 0)
	for _, seg := range segements {
		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			// Ignore the importing and migrating
			continue
		}
		parts := strings.Split(seg, "-")
		if len(parts) == 2 {
			start, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, errInvalidClusterNodes
			}
			end, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, errInvalidClusterNodes
			}
			for i := start; i <= end; i++ {
				slots = append(slots, i)
			}
		} else if len(parts) == 1 {
			slot, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, errInvalidClusterNodes
			}
			slots = append(slots, slot)
		} else {
			return nil, errInvalidClusterNodes
		}
	}
	return slots, nil
}
