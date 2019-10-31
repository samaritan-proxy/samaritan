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

package hotrestart

import (
	"syscall"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRestarterWithoutParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	inst := NewMockInstance(ctrl)
	inst.EXPECT().ID().Return(1)
	inst.EXPECT().ParentID().Return(0)

	r, err := New(inst)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Shutdown()

	r.ShutdownParentLocalConf()
	r.ShutdownParentAdmin()
	r.DrainParentListeners()
	r.TerminateParent()
}

func TestRestarterWithWrongParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	inst := NewMockInstance(ctrl)
	inst.EXPECT().ID().Return(2)
	inst.EXPECT().ParentID().Return(1).AnyTimes()

	r, err := New(inst)
	if err != nil {
		t.Error(err)
		return
	}
	defer r.Shutdown()

	r.ShutdownParentLocalConf()
	r.ShutdownParentAdmin()
	r.DrainParentListeners()
	r.TerminateParent()
}

func TestRestarterWithParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	oldKill := kill
	defer func() { kill = oldKill }()
	killCalled := false
	kill = func(int, syscall.Signal) error {
		killCalled = true
		return nil
	}

	parent := NewMockInstance(ctrl)
	parent.EXPECT().ID().Return(1)
	parent.EXPECT().ParentID().Return(-1).AnyTimes()
	parent.EXPECT().ShutdownLocalConf().Return()
	parent.EXPECT().ShutdownAdmin().Return()
	parent.EXPECT().DrainListeners().Return()

	child := NewMockInstance(ctrl)
	child.EXPECT().ID().Return(2)
	child.EXPECT().ParentID().Return(1).AnyTimes()

	pr, err := New(parent)
	if err != nil {
		t.Error(err)
		return
	}
	defer pr.Shutdown()

	cr, err := New(child)
	if err != nil {
		t.Error(err)
		return
	}
	defer cr.Shutdown()

	cr.ShutdownParentLocalConf()
	cr.ShutdownParentAdmin()
	cr.DrainParentListeners()
	cr.TerminateParent()

	//parent.AssertExpectations(t) ???
	assert.True(t, killCalled)
}

func TestRestarterWithParentUnexpectedExit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parent := NewMockInstance(ctrl)
	parent.EXPECT().ID().Return(1)
	parent.EXPECT().ParentID().Return(-1)
	parent.EXPECT().ShutdownLocalConf().Return()
	parent.EXPECT().ShutdownAdmin().Return()

	child := NewMockInstance(ctrl)
	child.EXPECT().ID().Return(2)
	child.EXPECT().ParentID().Return(1).AnyTimes()

	pr, err := New(parent)
	if err != nil {
		t.Error(err)
		return
	}
	defer pr.Shutdown()

	cr, err := New(child)
	if err != nil {
		t.Error(err)
		return
	}
	defer cr.Shutdown()

	cr.ShutdownParentLocalConf()
	cr.ShutdownParentAdmin()

	// parent unexpected exit
	pr.Shutdown()

	cr.DrainParentListeners()
	cr.TerminateParent()
}
