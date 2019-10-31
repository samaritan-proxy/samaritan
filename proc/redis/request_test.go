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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRawRequestIsValid(t *testing.T) {
	tests := []struct {
		v      *RespValue
		expect bool
	}{
		{&RespValue{Type: SimpleString}, false},
		{&RespValue{Type: BulkString}, false},
		{&RespValue{Type: Error}, false},
		{&RespValue{Type: Integer}, false},
		{&RespValue{Type: Array}, false},
		{&RespValue{Type: Array, Array: []RespValue{*newSimpleString("PING")}}, false},
		{&RespValue{Type: Array, Array: []RespValue{*newBulkString("GET"), *newBulkString("A")}}, true},
	}
	for _, test := range tests {
		req := newRawRequest(test.v)
		valid := req.IsValid()
		assert.Equal(t, test.expect, valid)
	}
}

func TestRawRequestHook(t *testing.T) {
	req := newRawRequest(newSimpleString("PING"))
	var (
		hook1ExecAt, hook2ExecAt time.Time
		reqDuration              time.Duration
	)
	hook1 := func(req *rawRequest) {
		hook1ExecAt = time.Now()
		reqDuration = req.Duration()
	}
	hook2 := func(req *rawRequest) {
		hook2ExecAt = time.Now()
	}
	req.RegisterHook(hook1)
	req.RegisterHook(hook2)

	time.AfterFunc(time.Millisecond*500, func() {
		req.SetResponse(newSimpleString("PONG"))
	})
	req.Wait()
	assert.NotZero(t, hook1ExecAt)
	assert.NotZero(t, hook2ExecAt)
	assert.NotZero(t, reqDuration)
	assert.Equal(t, true, hook2ExecAt.Before(hook1ExecAt))
}

func TestSimpleRequestHook(t *testing.T) {
	req := newSimpleRequest(newSimpleString("PING"))
	var (
		hook1ExecAt, hook2ExecAt time.Time
		reqDuration              time.Duration
	)
	hook1 := func(req *simpleRequest) {
		hook1ExecAt = time.Now()
		reqDuration = req.Duration()
	}
	hook2 := func(req *simpleRequest) {
		hook2ExecAt = time.Now()
	}
	req.RegisterHook(hook1)
	req.RegisterHook(hook2)

	time.AfterFunc(time.Millisecond*500, func() {
		req.SetResponse(newSimpleString("PONG"))
	})
	req.Wait()
	assert.NotZero(t, hook1ExecAt)
	assert.NotZero(t, hook2ExecAt)
	assert.NotZero(t, reqDuration)
	assert.Equal(t, true, hook2ExecAt.Before(hook1ExecAt))
}

func TestNewMSetRequest(t *testing.T) {
	tests := []struct {
		raw     *rawRequest
		noError bool
	}{
		{
			raw: newRawRequest(newArray([]RespValue{*newBulkString("mset")})),
		},
		{
			raw: newRawRequest(newArray([]RespValue{
				*newBulkString("mset"),
				*newBulkString("a"),
			})),
		},
		{
			raw: newRawRequest(newArray([]RespValue{
				*newBulkString("mset"),
				*newBulkString("a"),
				*newBulkString("1"),
			})),
			noError: true,
		},
	}
	for _, test := range tests {
		msetReq, err := newMSetRequest(test.raw)
		if !test.noError {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.NotNil(t, msetReq)
	}
}

func TestMSetRequestSplit(t *testing.T) {
	raw := newRawRequest(newArray([]RespValue{
		*newBulkString("mset"),
		*newBulkString("a"),
		*newBulkString("1"),
		*newBulkString("b"),
		*newBulkString("2"),
	}))
	msetReq, _ := newMSetRequest(raw)
	reqs := msetReq.Split()
	assert.Equal(t, 2, len(reqs))
	assert.Equal(t, newArray([]RespValue{
		*newBulkString("set"),
		*newBulkString("a"),
		*newBulkString("1"),
	}), reqs[0].Body())
	assert.Equal(t, newArray([]RespValue{
		*newBulkString("set"),
		*newBulkString("b"),
		*newBulkString("2"),
	}), reqs[1].Body())
}

func TestMSetRequesChildDone(t *testing.T) {
	raw := newRawRequest(newArray([]RespValue{
		*newBulkString("mset"),
		*newBulkString("a"),
		*newBulkString("1"),
		*newBulkString("b"),
		*newBulkString("2"),
	}))

	msetReq, _ := newMSetRequest(raw)
	reqs := msetReq.Split()
	time.AfterFunc(time.Millisecond*100, func() {
		reqs[0].SetResponse(respOK)
	})
	time.AfterFunc(time.Millisecond*200, func() {
		reqs[1].SetResponse(respOK)
	})

	begin := time.Now()
	raw.Wait()
	assert.Equal(t, respOK, raw.Response())
	assert.Equal(t, true, time.Since(begin) > time.Millisecond*200)
}

func TestNewMGetRequest(t *testing.T) {
	tests := []struct {
		raw     *rawRequest
		noError bool
	}{
		{
			raw: newRawRequest(newArray([]RespValue{*newBulkString("mget")})),
		},
		{
			raw: newRawRequest(newArray([]RespValue{
				*newBulkString("mget"),
				*newBulkString("a"),
			})),
			noError: true,
		},
		{
			raw: newRawRequest(newArray([]RespValue{
				*newBulkString("mget"),
				*newBulkString("a"),
				*newBulkString("b"),
			})),
			noError: true,
		},
	}
	for _, test := range tests {
		mgetReq, err := newMGetRequest(test.raw)
		if !test.noError {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.NotNil(t, mgetReq)
	}
}

func TestMGetRequestSplit(t *testing.T) {
	raw := newRawRequest(newArray([]RespValue{
		*newBulkString("mget"),
		*newBulkString("a"),
		*newBulkString("b"),
	}))
	mgetReq, _ := newMGetRequest(raw)
	reqs := mgetReq.Split()
	assert.Equal(t, 2, len(reqs))
	assert.Equal(t, newArray([]RespValue{
		*newBulkString("get"),
		*newBulkString("a"),
	}), reqs[0].Body())
	assert.Equal(t, newArray([]RespValue{
		*newBulkString("get"),
		*newBulkString("b"),
	}), reqs[1].Body())
}

func TestMGetRequestChildDone(t *testing.T) {
	raw := newRawRequest(newArray([]RespValue{
		*newBulkString("mget"),
		*newBulkString("a"),
		*newBulkString("b"),
	}))

	mgetReq, _ := newMGetRequest(raw)
	reqs := mgetReq.Split()
	time.AfterFunc(time.Millisecond*100, func() {
		reqs[0].SetResponse(newBulkString("1"))
	})
	time.AfterFunc(time.Millisecond*100, func() {
		reqs[1].SetResponse(newBulkString("2"))
	})

	raw.Wait()
	expected := newArray([]RespValue{
		*newBulkString("1"),
		*newBulkString("2"),
	})
	assert.Equal(t, expected, raw.Response())
}

func TestNewSumResultRequest(t *testing.T) {
	tests := []struct {
		raw     *rawRequest
		noError bool
	}{
		{
			raw: newRawRequest(newArray([]RespValue{*newBulkString("touch")})),
		},
		{
			raw: newRawRequest(newArray([]RespValue{
				*newBulkString("touch"),
				*newBulkString("a"),
			})),
			noError: true,
		},
		{
			raw: newRawRequest(newArray([]RespValue{
				*newBulkString("touch"),
				*newBulkString("a"),
				*newBulkString("b"),
			})),
			noError: true,
		},
	}
	for _, test := range tests {
		sumResReq, err := newSumResultRequest(test.raw)
		if !test.noError {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.NotNil(t, sumResReq)
	}
}

func TestSumResultRequestSplit(t *testing.T) {
	raw := newRawRequest(newArray([]RespValue{
		*newBulkString("unlink"),
		*newBulkString("a"),
		*newBulkString("b"),
	}))
	sumResReq, _ := newSumResultRequest(raw)
	reqs := sumResReq.Split()
	assert.Equal(t, 2, len(reqs))
	assert.Equal(t, newArray([]RespValue{
		*newBulkString("unlink"),
		*newBulkString("a"),
	}), reqs[0].Body())
	assert.Equal(t, newArray([]RespValue{
		*newBulkString("unlink"),
		*newBulkString("b"),
	}), reqs[1].Body())
}

func TestSumResultRequestChildDone(t *testing.T) {
	raw := newRawRequest(newArray([]RespValue{
		*newBulkString("unlink"),
		*newBulkString("a"),
		*newBulkString("b"),
	}))
	sumResReq, _ := newSumResultRequest(raw)
	reqs := sumResReq.Split()
	time.AfterFunc(time.Millisecond*100, func() {
		reqs[0].SetResponse(newInteger(1))
		reqs[1].SetResponse(newInteger(1))
	})
	raw.Wait()
	assert.Equal(t, newInteger(2), raw.Response())
}

func TestSumResultRequestChildDoneWithError(t *testing.T) {
	raw := newRawRequest(newArray([]RespValue{
		*newBulkString("unlink"),
		*newBulkString("a"),
		*newBulkString("b"),
	}))
	sumResReq, _ := newSumResultRequest(raw)
	reqs := sumResReq.Split()
	time.AfterFunc(time.Millisecond*100, func() {
		reqs[0].SetResponse(newInteger(1))
		reqs[1].SetResponse(newError("unknown"))
	})
	raw.Wait()
	assert.Equal(t, Error, raw.Response().Type)
}
