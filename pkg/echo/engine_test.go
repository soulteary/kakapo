// Copyright 2026 Su Yang (soulteary)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package echo

import (
	"bytes"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEngineLoggingAndRecovery(t *testing.T) {
	for _, value := range []string{"0", "FALSE", " off ", "no"} {
		t.Setenv("KAKAPO_HTTP_LOG", value)
		if httpLogEnabled() {
			t.Fatalf("logging enabled for %q", value)
		}
	}
	t.Setenv("KAKAPO_HTTP_LOG", "1")
	if !httpLogEnabled() {
		t.Fatal("logging disabled")
	}
	var buf bytes.Buffer
	original := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(original) })
	e := NewEngine(WithLogPrefix("[TEST]"))
	e.POST("/route", func(c echo.Context) error { return c.NoContent(201) })
	h := &Handler{Echo: e}
	r := httptest.NewRequest("POST", "/route?token=synthetic-secret", strings.NewReader("synthetic-private-body"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatal(w.Code)
	}
	output := buf.String()
	if !strings.Contains(output, "[TEST] POST /route 201") || strings.Contains(output, "synthetic") {
		t.Fatalf("access log: %q", output)
	}
	quiet := NewEngine()
	quiet.GET("/panic", func(echo.Context) error { panic("synthetic panic") })
	w = httptest.NewRecorder()
	quiet.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if w.Code != 500 {
		t.Fatalf("panic not recovered: %d", w.Code)
	}
	empty := &Handler{}
	w = httptest.NewRecorder()
	empty.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Body.Len() != 0 {
		t.Fatal("nil handler wrote response")
	}
}
