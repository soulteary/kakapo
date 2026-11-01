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
	"github.com/wailsapp/wails/v3/pkg/application"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestEchoEventDoesNotLogPayload(t *testing.T) {
	if os.Getenv("KAKAPO_EVENT_LOG_CHILD") == "1" {
		var buffer bytes.Buffer
		app := application.New(application.Options{Name: "Event log test", Logger: slog.New(slog.NewJSONHandler(&buffer, nil))})
		service := NewEchoService()
		if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{}, 1)
		app.Event.On("echo-api-response", func(*application.CustomEvent) { done <- struct{}{} })
		app.Event.Emit("echo-api-event", map[string]string{"token": "synthetic-sensitive-event-body"})
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("event response missing")
		}
		output := buffer.String()
		if !strings.Contains(output, "Received event from frontend") {
			t.Fatal("event observation was disabled")
		}
		if strings.Contains(output, "synthetic-sensitive-event-body") {
			t.Fatal("arbitrary frontend event payload was logged")
		}
		return
	}
	child := exec.Command(os.Args[0], "-test.run", "^TestEchoEventDoesNotLogPayload$")
	child.Env = append(os.Environ(), "KAKAPO_EVENT_LOG_CHILD=1")
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("event log privacy: %v\n%s", err, output)
	}
}
