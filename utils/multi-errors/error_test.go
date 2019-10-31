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

package multierror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWithError(t *testing.T) {
	singleError := errors.New("1")
	mError := New()
	mError.Add(singleError)

	// New with single error.
	dstError := NewWithError(singleError)
	assert.Equal(t, 1, len(dstError.errs))

	// New with multi-error.
	dstError = NewWithError(mError)
	assert.Equal(t, mError.errs, dstError.errs)

	dstError.Add(errors.New("new error"))
	assert.NotEqual(t, mError.errs, dstError.errs)
}

func TestDefaultFormatter(t *testing.T) {
	mError := New()
	mError.Add(errors.New("1"))
	mError.Add(errors.New("2"))

	expect := `1, 2`
	assert.Equal(t, expect, mError.Error())
}

func TestSetterFormatter(t *testing.T) {
	mError := New()
	mError.Add(errors.New("1"))
	mError.Add(errors.New("2"))

	fmt := func([]error) string {
		return "ERROR"
	}
	mError.SetFormatter(fmt)
	assert.Equal(t, "ERROR", mError.Error())
}

func TestErrorsErrorOrNil(t *testing.T) {
	mError := New()
	assert.Nil(t, mError.ErrorOrNil())

	mError.Add(errors.New("error"))
	assert.NotNil(t, mError.ErrorOrNil())
}

func TestErrorsAdd(t *testing.T) {
	mError := New()
	assert.Empty(t, mError.errs)

	mError.Add(errors.New("error"))
	assert.NotEmpty(t, mError.errs)
	assert.Equal(t, "error", mError.Error())
}

func TestErrorsAddNil(t *testing.T) {
	mError := New()
	assert.Empty(t, mError.errs)

	mError.Add(nil)
	assert.Empty(t, mError.errs)
}

func TestErrorsRawErrors(t *testing.T) {
	mError := New()
	assert.Equal(t, mError.errs, mError.RawError())

	mError.Add(errors.New("error"))
	assert.Equal(t, mError.errs, mError.RawError())
}
