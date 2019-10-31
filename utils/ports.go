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

package utils

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/kavu/go_reuseport"
)

// CheckPort return true if port not bind by other process.
func CheckPort(port int) bool {
	ln, err := net.Listen("tcp4", fmt.Sprintf(":%d", port))
	defer func() {
		if err == nil && ln != nil {
			ln.Close()
		}
	}()
	return err == nil
}

// CheckPortWithReusePort return true if port not bind by other process.
func CheckPortWithReusePort(port int) bool {
	ln, err := reuseport.Listen("tcp4", fmt.Sprintf(":%d", port))
	defer func() {
		if err == nil && ln != nil {
			ln.Close()
		}
	}()
	return err == nil
}

// genRandomNumInRange return a random number in [start, end)
func genRandomNumInRange(start, end int) int {
	return rand.Intn(end-start) + start
}

// GetPortByRange return a random port in [start, end)
// if can not find a unused in timeout will return a error.
func GetPortByRange(start, end int, timeout time.Duration) (int, error) {
	t := time.NewTimer(timeout)
	for {
		select {
		case <-t.C:
			return -1, fmt.Errorf("failed to get a port, timeout")
		default:
			if port := genRandomNumInRange(start, end); CheckPort(port) {
				return port, nil
			}
		}
	}
}
