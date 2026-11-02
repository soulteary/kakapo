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

package events

import (
	"fmt"
	"github.com/wailsapp/wails/v3/pkg/application"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestEmitterLifecycleAndServiceSnapshot(t *testing.T) {
	Stop()
	defer func() { Stop(); mu.Lock(); emitters = nil; services = nil; mu.Unlock() }()
	first := application.NewService(&struct{}{})
	second := application.NewService(&struct{}{})
	AddService(first)
	var calls atomic.Int64
	AddEmitter(EventTime, 10*time.Millisecond, func() string { calls.Add(1); return "synthetic tick" }, WithServices(second))
	snapshot := Services()
	if len(snapshot) != 2 {
		t.Fatalf("service count %d", len(snapshot))
	}
	snapshot[0] = application.Service{}
	if reflect.DeepEqual(Services()[0], snapshot[0]) {
		t.Fatal("service snapshot aliases registry")
	}
	app := application.New(application.Options{Name: "Emitter unit test"})
	delivered := make(chan string, 10)
	app.Event.On(EventTime, func(event *application.CustomEvent) {
		select {
		case delivered <- fmt.Sprint(event.Data):
		default:
		}
	})
	Start(app)
	for i := 0; i < 2; i++ {
		select {
		case value := <-delivered:
			if !strings.Contains(value, "synthetic tick") {
				t.Fatalf("event value %q", value)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("emitter did not deliver event")
		}
	}
	Stop()
	before := calls.Load()
	time.Sleep(30 * time.Millisecond)
	if calls.Load() != before {
		t.Fatal("emitter continued after Stop")
	}
	Stop()
}
