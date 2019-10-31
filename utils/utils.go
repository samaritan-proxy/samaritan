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
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/samaritan-proxy/samaritan/logger"
)

// IntMin returns the smaller int
// I can't believe that golang doesn't provide this.
// Yes, there is math.Min() but giving two int, you have to
//     math.Min(float64(a), float64(b))
// Converting TWO int to FLOAT64 EVERY TIME just to get the smaller one, that's crazy!
// There should be something like:
//     a < b ? a : b
// or:
//     a if a < b else b
// But there isn't! So I wrote this stupid func along with the above, I must be crazy.
//
// Hi, let's just blame that golang doesn't have generics.
func IntMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IntMax returns the greater int
func IntMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Round round up float to a specific precision.
// I'm getting a little bit used to this
// The implementation is dumb but safe hopefully
func Round(f float64, digits int) (float64, error) {
	return strconv.ParseFloat(fmt.Sprintf(fmt.Sprintf("%%.%df", digits), f), 64)
}

// WriteFileAtomically is the same as ioutil.WriteFile but it writes atomically
// by using os.Rename which is atomic as required by POSIX.
//
// Known issue: If the tempfile is write on another drive the rename will fail
// with EXDEV, in this case atomic write is not possible.
func WriteFileAtomically(filename string, data []byte, perm os.FileMode) error {
	dir, name := path.Split(filename)
	f, err := ioutil.TempFile(dir, name)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}

	if closeErr := f.Close(); err == nil {
		err = closeErr
	}

	if permErr := os.Chmod(f.Name(), perm); err == nil {
		err = permErr
	}

	if err == nil {
		err = os.Rename(f.Name(), filename)
	}

	// Any err should result in full cleanup.
	if err != nil {
		os.Remove(f.Name())
	}

	return err
}

// GetLocalIP returns a non-loopback local IP
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, address := range addrs {
		// find non-loopback
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", errors.New("no non-loopback IP address found")
}

// ReadPidFile reads content from given file path and return as a int
func ReadPidFile(pidFile string) (int, error) {
	pidStr, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return -1, fmt.Errorf("error while reading pid file: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidStr)))
	if err != nil {
		return -1, fmt.Errorf("malformed pid file: %v", err)
	}
	return pid, nil
}

// WithTempFile creates a tempfile with given content then calls f()
//
// The file will be deleted after calling f()
func WithTempFile(content string, f func(filename string)) {
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		logger.Fatal(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		logger.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		logger.Fatal(err)
	}

	f(tmpfile.Name())
}

// WithTempDir creates a temp dir then run the given callback(tempDir)
// finally remove the temp dir and any children it contains
func WithTempDir(f func(d string)) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		logger.Fatal(err)
	}

	defer os.RemoveAll(dir) // clean up
	f(dir)
}

// FlattenKey flatten the key for formatting, removes spaces and point.
func FlattenKey(key string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ':':
			fallthrough
		case ' ', '.':
			return '_'
		default:
			return r
		}
	}, key)
}

// IsAddrInUse returns whether the given error is known to report that the
// address is in use. e.g. syscall.EADDRINUSE
func IsAddrInUse(err error) bool {
	if err == syscall.EADDRINUSE {
		return true
	}
	if err, ok := err.(*net.OpError); ok {
		if err, ok := err.Err.(*os.SyscallError); ok {
			return err.Err == syscall.EADDRINUSE
		}
	}
	return false
}

// RunCmd run cmd and returns the execution result
func RunCmd(cmd string) (string, error) {
	out, err := exec.Command("/bin/sh", "-c", cmd).Output()
	return string(bytes.TrimSpace(out)), err
}

// LocateHAProxy returns HAProxy location
func LocateHAProxy() (string, error) {
	var err error

	binPath, _ := RunCmd("which haproxy")
	if _, err = os.Stat(binPath); err != nil {
		binPath, err = RunCmd("find /bin /sbin /usr/sbin /usr/local/bin /opt /etc/opt /var/opt -executable -type f -name haproxy | head -n 1")
	}
	return binPath, err
}

var (
	// ErrInvalidHTTPListen indicates that the given address is invalid
	ErrInvalidHTTPListen = errors.New("invalid TCP4 address")
	// ErrHTTPListenMissingPort indicates that the given address lack port
	ErrHTTPListenMissingPort = errors.New("port number must be speified")
)

// VerifyTCP4Address returns error if the given address is not a valid TCP4 address.
// e.g. 1.1.1.1:      -> NG
//      1.1.1.1:1000  -> OK
func VerifyTCP4Address(address string) error {
	if _, err := net.ResolveTCPAddr("tcp4", address); err != nil {
		return ErrInvalidHTTPListen
	}

	// SplitHostPort will called by ResolveTCPAddr, error always is nil at here.
	_, port, _ := net.SplitHostPort(address)
	if port == "" {
		return ErrHTTPListenMissingPort
	}

	return nil
}

// Copy returns a copy of given registry.
func Copy(src map[string]string) map[string]string {
	var dst = make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// DurationPtr return *time.Duration
func DurationPtr(d time.Duration) *time.Duration {
	return &d
}

// PickFirstNonEmptyStr return the first no empty string.
func PickFirstNonEmptyStr(str ...string) string {
	for _, s := range str {
		if s != "" {
			return s
		}
	}
	return ""
}

// NewUUID generates a random UUID according to RFC 4122
func NewUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
