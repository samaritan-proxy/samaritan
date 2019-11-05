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

package compressor

import (
	"errors"
	"fmt"
	"io"
)

//go:generate mockgen -package $GOPACKAGE -destination mock.go -source=$GOFILE

var (
	m = make(map[string]compressor)

	errUnsupportedAlgorithm = errors.New("compressor not exist")
)

// Register registers the compressor builder
func Register(typ string, c compressor) {
	if _, ok := m[typ]; ok {
		panic(fmt.Errorf("compressor: %s existed, duplicate registration", typ))
	}
	m[typ] = c
}

// UnRegister compressor builder, only for test
func UnRegister(typ string) {
	delete(m, typ)
}

// get returns the compressor with the name.
func get(typ string) (compressor, bool) {
	c, ok := m[typ]
	return c, ok
}

// NewReader return a reader for uncompressed.
func NewReader(typ string, r io.Reader) (io.Reader, error) {
	c, ok := get(typ)
	if !ok {
		return nil, errUnsupportedAlgorithm
	}
	return c.NewReader(r), nil
}

// NewWriter return a writer for compressed.
func NewWriter(typ string, w io.Writer) (io.WriteCloser, error) {
	c, ok := get(typ)
	if !ok {
		return nil, errUnsupportedAlgorithm
	}
	return c.NewWriter(w), nil
}

// compressor is used to compress & decompress values in command
type compressor interface {
	NewWriter(w io.Writer) io.WriteCloser
	NewReader(r io.Reader) io.Reader
}
