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

// +build linux

package hotrestart

import "fmt"

func genDomainSocketName(id int) string {
	return fmt.Sprintf("@sam_domain_socket_%d", id)
}

func removeDomainSocket(name string) error {
	// NOTE: abstract domain socket will be removed by kernel when process exits.
	return nil
}
