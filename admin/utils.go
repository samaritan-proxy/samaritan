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

package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ContentTypes.
const (
	ContentPlainText       = "text/plain; charset=UTF-8"
	ContentApplicationJSON = "application/json; charset=UTF-8"
)

func setContent(w http.ResponseWriter, v string) {
	w.Header().Set("Content-Type", v)
}

func writeMessage(w http.ResponseWriter, code int, msg string) error {
	setContent(w, ContentApplicationJSON)
	w.WriteHeader(code)
	_, err := w.Write([]byte(fmt.Sprintf(`{"msg": "%s"}`, msg)))
	if err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, body interface{}) error {
	setContent(w, ContentApplicationJSON)
	s, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		goto Failed
	}
	_, err = w.Write(s)
	if err != nil {
		goto Failed
	}
	return nil
Failed:
	writeMessage(w, http.StatusInternalServerError, err.Error())
	return err
}
