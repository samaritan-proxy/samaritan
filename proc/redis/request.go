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
	"fmt"
	"time"

	"go.uber.org/atomic"
)

// rawRequest represents the raw rawRequest.
type rawRequest struct {
	createdAt  time.Time
	finishedAt time.Time
	body       *RespValue
	resp       *RespValue
	hooks      []func(*rawRequest)
	done       chan struct{}
}

func newRawRequest(v *RespValue) *rawRequest {
	return &rawRequest{
		createdAt: time.Now(),
		body:      v,
		// pre-alloc
		hooks: make([]func(*rawRequest), 0, 4),
		done:  make(chan struct{}),
	}
}

func (r *rawRequest) IsValid() (valid bool) {
	b := r.body
	// rawRequest body must be mulk string array
	if b.Type != Array || len(b.Array) == 0 {
		return
	}
	for _, v := range b.Array {
		if v.Type != BulkString {
			return
		}
	}
	return true
}

func (r *rawRequest) Body() *RespValue {
	return r.body
}

func (r *rawRequest) Duration() time.Duration {
	return r.finishedAt.Sub(r.createdAt)
}

// RegisterHook registers a hook which will be called when rawRequest is done.
func (r *rawRequest) RegisterHook(hook func(*rawRequest)) {
	r.hooks = append(r.hooks, hook)
}

func (r *rawRequest) Wait() {
	<-r.done
}

func (r *rawRequest) SetResponse(v *RespValue) {
	r.finishedAt = time.Now()
	r.resp = v
	// call hook by order, LIFO
	for i := len(r.hooks) - 1; i >= 0; i-- {
		hook := r.hooks[i]
		hook(r)
	}
	close(r.done)
}

func (r *rawRequest) Response() *RespValue {
	return r.resp
}

type simpleRequest struct {
	createdAt  time.Time
	finishedAt time.Time
	body       *RespValue
	resp       *RespValue
	hooks      []func(*simpleRequest)
	done       chan struct{}
}

func newSimpleRequest(v *RespValue) *simpleRequest {
	return &simpleRequest{
		createdAt: time.Now(),
		body:      v,
		// pre-alloc
		hooks: make([]func(*simpleRequest), 0, 4),
		done:  make(chan struct{}),
	}
}

func (r *simpleRequest) Body() *RespValue {
	return r.body
}

func (r *simpleRequest) Duration() time.Duration {
	return r.finishedAt.Sub(r.createdAt)
}

// RegisterHook registers a hook which will be called when rawRequest is done.
func (r *simpleRequest) RegisterHook(hook func(*simpleRequest)) {
	r.hooks = append(r.hooks, hook)
}

func (r *simpleRequest) SetResponse(resp *RespValue) {
	r.finishedAt = time.Now()
	r.resp = resp
	// call hook by order, LIFO
	for i := len(r.hooks) - 1; i >= 0; i-- {
		hook := r.hooks[i]
		hook(r)
	}
	close(r.done)
}

func (r *simpleRequest) Wait() {
	<-r.done
}

func (r *simpleRequest) Response() *RespValue {
	return r.resp
}

type msetRequest struct {
	raw       *rawRequest
	children  []*simpleRequest
	childWait *atomic.Int32
}

func newMSetRequest(raw *rawRequest) (*msetRequest, error) {
	v := raw.Body().Array
	if len(v) == 1 || len(v)%2 != 1 {
		return nil, errors.New(invalidRequest)
	}
	r := &msetRequest{
		raw:       raw,
		childWait: atomic.NewInt32(0),
	}
	return r, nil
}

func (r *msetRequest) Split() []*simpleRequest {
	if r.children != nil {
		return r.children
	}

	// split into simple requests
	v := r.raw.Body().Array
	sreqs := make([]*simpleRequest, 0, len(v)/2)
	for i := 0; i < len(v)/2; i++ {
		sv := &RespValue{
			Type: Array,
			Array: []RespValue{
				{Type: BulkString, Text: []byte("set")},
				v[2*i+1],
				v[2*(i+1)],
			},
		}
		sreq := newSimpleRequest(sv)
		sreq.RegisterHook(r.onChildDone)
		sreqs = append(sreqs, sreq)
	}

	r.children = sreqs
	r.childWait.Store(int32(len(sreqs)))
	return sreqs
}

func (r *msetRequest) onChildDone(simpleReq *simpleRequest) {
	wait := r.childWait.Dec()
	if wait == 0 {
		r.raw.SetResponse(respOK)
	}
}

type mgetRequest struct {
	raw       *rawRequest
	children  []*simpleRequest
	childWait *atomic.Int32
}

func newMGetRequest(raw *rawRequest) (*mgetRequest, error) {
	v := raw.Body().Array
	if len(v) < 2 {
		return nil, errors.New(invalidRequest)
	}
	r := &mgetRequest{
		raw:       raw,
		childWait: atomic.NewInt32(0),
	}
	return r, nil
}

func (r *mgetRequest) Split() []*simpleRequest {
	if r.children != nil {
		return r.children
	}

	// split into simple requests
	v := r.raw.Body().Array
	sreqs := make([]*simpleRequest, 0, len(v)-1)
	for i := 1; i < len(v); i++ {
		sv := &RespValue{
			Type: Array,
			Array: []RespValue{
				{Type: BulkString, Text: []byte("get")},
				v[i],
			},
		}
		sreq := newSimpleRequest(sv)
		sreq.RegisterHook(r.onChildDone)
		sreqs = append(sreqs, sreq)
	}

	r.children = sreqs
	r.childWait.Store(int32(len(sreqs)))
	return sreqs
}

func (r *mgetRequest) onChildDone(simpleReq *simpleRequest) {
	wait := r.childWait.Dec()
	if wait == 0 {
		r.setResponse()
	}
}

func (r *mgetRequest) setResponse() {
	v := make([]RespValue, len(r.children))
	for i, child := range r.children {
		v[i] = *child.Response()
	}
	r.raw.SetResponse(&RespValue{
		Type:  Array,
		Array: v,
	})
}

type sumResultRequest struct {
	raw       *rawRequest
	children  []*simpleRequest
	childWait *atomic.Int32
}

func newSumResultRequest(raw *rawRequest) (*sumResultRequest, error) {
	v := raw.Body().Array
	if len(v) < 2 {
		return nil, errors.New(invalidRequest)
	}
	r := &sumResultRequest{
		raw:       raw,
		childWait: atomic.NewInt32(0),
	}
	return r, nil
}

func (r *sumResultRequest) Split() []*simpleRequest {
	if r.children != nil {
		return r.children
	}

	// split into simple requests
	v := r.raw.Body().Array
	sreqs := make([]*simpleRequest, 0, len(v)-1)
	for i := 1; i < len(v); i++ {
		sv := &RespValue{
			Type: Array,
			Array: []RespValue{
				v[0],
				v[i],
			},
		}
		sreq := newSimpleRequest(sv)
		sreq.RegisterHook(r.onChildDone)
		sreqs = append(sreqs, sreq)
	}

	r.children = sreqs
	r.childWait.Store(int32(len(sreqs)))
	return sreqs
}

func (r *sumResultRequest) onChildDone(simpleReq *simpleRequest) {
	wait := r.childWait.Dec()
	if wait == 0 {
		r.setResponse()
	}
}

func (r *sumResultRequest) setResponse() {
	total := int64(0)
	errCount := 0
	for _, child := range r.children {
		resp := child.Response()
		switch resp.Type {
		case Integer:
			total += resp.Int
		default:
			errCount++
		}
	}

	if errCount == 0 {
		r.raw.SetResponse(newInteger(total))
	} else {
		r.raw.SetResponse(newError(fmt.Sprintf("finished with %d error(s)", errCount)))
	}
}
