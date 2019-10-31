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

package tcp

import (
	"context"
	"sync"
	"time"

	tcp "github.com/tevino/tcp-shaker"

	"github.com/samaritan-proxy/samaritan/logger"
)

var (
	mu            sync.Mutex
	sharedChecker *tcp.Checker
)

// initSharedCheckerIfNot initializes the global sharedChecker if it is not ready.
func initSharedCheckerIfNot() {
	mu.Lock()
	defer mu.Unlock()

	if sharedChecker == nil {
		sharedChecker = tcp.NewChecker()
	}
	if sharedChecker.IsReady() {
		return
	}

	go func() {
		logger.Info("Initializing TCP checker")
		if err := sharedChecker.CheckingLoop(context.TODO()); err != nil {
			logger.Warn("Initializing TCP checker failed: ", err)
		}
	}()

	logger.Info("Waiting for TCP checker to be ready")
	select {
	case <-sharedChecker.WaitReady():
		logger.Info("TCP checker is now ready")
	case <-time.After(time.Second * 10):
		logger.Fatal("Waiting for TCP checker to be ready timed out")
	}
}

// Checker doing health checking by TCP handshaking.
type Checker struct{}

// Check checks given host with timeout.
func (c *Checker) Check(addr string, timeout time.Duration) error {
	return sharedChecker.CheckAddr(addr, timeout)
}

// NewChecker creates TCPChecker.
func NewChecker() *Checker {
	initSharedCheckerIfNot()
	return &Checker{}
}
