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
	"errors"
	"io/ioutil"
	"log"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	net.DefaultResolver.PreferGo = true
}

func TestVerifyTCP4AddressWithInvalidAddress(t *testing.T) {
	for _, addr := range []string{
		"1.2.3.4:-1",      // Bad port
		"1.2.3.4:99999",   // Bad port
		"1.2.3.4:badPort", // Bad port
		"1.2.3.4.5:6",     // Bad IP
		"1.2.4:5",         // Bad IP
		"1.4:5",           // Bad IP
		"1:5",             // Bad IP
		"1.2.3:-1",        // Bad port and IP
		"1.2.3.4",         // Missing port
		"1.2.3.4:",        // Missing Port
		"1.2.3.4:5:6",     // Multiple port
		"::1",             // IPv6
	} {
		assert.Error(t, VerifyTCP4Address(addr), addr)
	}
}

func TestVerifyTCP4AddressWithValidAddress(t *testing.T) {
	for _, addr := range []string{
		"1.2.3.4:5", // 1.2.3.4:5
		":5",        // 0.0.0.0:5
	} {
		assert.NoError(t, VerifyTCP4Address(addr), addr)
	}
}

func TestMin(t *testing.T) {
	assert.Equal(t, IntMin(1, 2), 1)
	assert.Equal(t, IntMin(-1, -2), -2)
	assert.Equal(t, IntMin(1, -1), -1)
}

func TestMax(t *testing.T) {
	assert.Equal(t, IntMax(1, 2), 2)
	assert.Equal(t, IntMax(-1, -2), -1)
	assert.Equal(t, IntMax(1, -1), 1)
}

func TestReadInvalidPidFile(t *testing.T) {
	// non-existent file
	_, err := ReadPidFile("non-existent-file")
	assert.Error(t, err)

	// read file which contains incorrect content
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := []byte("ha")
	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}

	_, err = ReadPidFile(tmpfile.Name())
	assert.Error(t, err)
}

func TestReadPidFile(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := []byte("123")
	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}

	pid, err := ReadPidFile(tmpfile.Name())
	assert.Equal(t, pid, 123)
	assert.NoError(t, err)
}

func TestFlattenKey(t *testing.T) {
	assert.Equal(t, FlattenKey("a b.c:d-e"), "a_b_c_d-e")
}

func TestIsAddrInUse(t *testing.T) {
	scenarios := []struct {
		err error

		result bool
	}{
		{syscall.EADDRINUSE, true},
		{&net.OpError{Err: os.NewSyscallError("", syscall.EADDRINUSE)}, true},
		{&net.OpError{Err: syscall.EADDRINUSE}, false},
		{&net.OpError{Err: errors.New("fake error")}, false},
		{errors.New("fake error"), false},
	}
	for _, scenario := range scenarios {
		assert.Equal(t, IsAddrInUse(scenario.err), scenario.result)
	}
}

func TestWithTempFile(t *testing.T) {
	content := []byte("blablabla")
	var tmpFilename string
	f := func(filename string) {
		tmpFilename = filename
		p, err := ioutil.ReadFile(tmpFilename)
		assert.NoError(t, err)
		assert.Equal(t, p, content)
	}
	WithTempFile(string(content), f)
	if _, err := os.Stat(tmpFilename); !os.IsNotExist(err) {
		t.Fatalf("file %s shoud be removed", tmpFilename)
	}
}

func TestWithTempDir(t *testing.T) {
	var tmpDir string
	f := func(d string) {
		tmpDir = d
		_, err := os.Stat(tmpDir)
		assert.NoError(t, err)
	}
	WithTempDir(f)
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Fatalf("directory %s shoud be removed", tmpDir)
	}
}
