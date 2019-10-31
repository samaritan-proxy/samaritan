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

package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ghodss/yaml"

	"github.com/samaritan-proxy/samaritan/pb/config/bootstrap"
)

func newBootstrapFromJSON(b []byte) (*bootstrap.Bootstrap, error) {
	cfg := new(bootstrap.Bootstrap)
	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func newBootstrapFromYAML(b []byte) (*bootstrap.Bootstrap, error) {
	jsonBytes, err := yaml.YAMLToJSON(b)
	if err != nil {
		return nil, err
	}
	return newBootstrapFromJSON(jsonBytes)
}

func LoadBootstrap(file string) (*bootstrap.Bootstrap, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return newBootstrapFromYAML(b)
}
