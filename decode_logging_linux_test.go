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

package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/soulteary/kakapo/internal/persistence"
	"github.com/soulteary/kakapo/internal/translate"
	"github.com/wailsapp/wails/v3/pkg/application"
	"log"
	"os"
	"strings"
	"testing"
)

func TestMalformedHistoryJSONDoesNotEchoNumericPayload(t *testing.T) {
	isolatedConfig(t)
	service := NewTranslateWebService()
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	const numeric = "123456789012345678901234567890"
	response := request(t, service, "POST", "/api/history", `{"input":"synthetic","latencyMs":`+numeric+`}`, 400)
	if strings.Contains(response.Body.String(), numeric) || strings.Contains(logs.String(), numeric) {
		t.Fatal("invalid request payload was reflected in error response or logs")
	}
}

func TestUnexpectedErrorsDoNotExposePathsOrPayload(t *testing.T) {
	const private = "synthetic-private-detail"
	for _, err := range []error{errors.New(private), &os.PathError{Op: "write", Path: private, Err: os.ErrPermission}, &persistence.CommittedError{Err: errors.New(private)}} {
		message := safeFailureMessage(err)
		if message == "" || strings.Contains(message, private) {
			t.Fatalf("unsafe diagnostic for %T", err)
		}
	}
	if message := safeFailureMessage(&persistence.CommittedError{Err: errors.New(private)}); !strings.Contains(message, "committed") {
		t.Fatal("committed outcome was hidden")
	}
	validation := &translate.ValidationError{Message: "safe validation"}
	if validation.Error() != "safe validation" || translate.ErrorMessage(validation) != "safe validation" {
		t.Fatal("trusted validation message lost")
	}
}
