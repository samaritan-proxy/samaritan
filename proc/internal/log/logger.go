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

package log

import (
	"fmt"

	"github.com/samaritan-proxy/samaritan/logger"
)

// InfoLogger represents a logger with Info* APIs.
type InfoLogger interface {
	Info(...interface{})
	Infof(string, ...interface{})
}

// WarnLogger represents a logger with Warn* APIs.
type WarnLogger interface {
	Warn(...interface{})
	Warnf(string, ...interface{})
}

// DebugLogger represents a logger with Debug* APIs.
type DebugLogger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
}

// FatalLogger represents a logger with Fatal* APIs.
type FatalLogger interface {
	Fatal(...interface{})
	Fatalf(string, ...interface{})
}

// Logger represents a full-featured logger.
type Logger interface {
	InfoLogger
	WarnLogger
	DebugLogger
	FatalLogger
}

type log struct {
	prefix string
}

// New returns *log.
func New(prefix string) Logger {
	return &log{
		prefix: prefix,
	}
}

// Infof wraps the logger.Infof method.
func (log *log) Infof(f string, a ...interface{}) {
	f = log.attachPrefix(f)
	logger.InfofDepth(1, f, a...)
}

// Info wraps the logger.Info method.
func (log *log) Info(a ...interface{}) {
	a = append([]interface{}{log.attachPrefix("")}, a...)
	logger.InfoDepth(1, a...)
}

// Warnf wraps the logger.Warnf method.
func (log *log) Warnf(f string, a ...interface{}) {
	f = log.attachPrefix(f)
	logger.WarnfDepth(1, f, a...)
}

// Warn wraps the logger.Warn method.
func (log *log) Warn(a ...interface{}) {
	a = append([]interface{}{log.attachPrefix("")}, a...)
	logger.WarnDepth(1, a...)
}

// Debugf wraps the logger.Debugf method.
func (log *log) Debugf(f string, a ...interface{}) {
	f = log.attachPrefix(f)
	logger.DebugfDepth(1, f, a...)
}

// Debug wraps the logger.Debug method.
func (log *log) Debug(a ...interface{}) {
	a = append([]interface{}{log.attachPrefix("")}, a...)
	logger.DebugDepth(1, a...)
}

// Fatalf wraps the logger.Fatalf method.
func (log *log) Fatalf(f string, a ...interface{}) {
	f = log.attachPrefix(f)
	logger.FatalfDepth(1, f, a...)
}

// Fatal wraps the logger.Fatal method.
func (log *log) Fatal(a ...interface{}) {
	a = append([]interface{}{log.attachPrefix("")}, a...)
	logger.FatalDepth(1, a...)
}

func (log *log) attachPrefix(f string) string {
	return fmt.Sprintf("%s %s", log.prefix, f)
}
