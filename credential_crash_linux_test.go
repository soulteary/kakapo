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
	"bufio"
	"errors"
	"github.com/soulteary/kakapo/internal/config"
	"github.com/soulteary/kakapo/internal/persistence"
	"github.com/soulteary/kakapo/internal/translate"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// diskCredentials is a synthetic credential service backed by real private files.
// It exercises process persistence, not the native macOS Keychain implementation.
type diskCredentials struct {
	path     string
	afterSet func()
}

func (s *diskCredentials) Get(string, string) (string, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return "", nil
	}
	return string(data), err
}
func (s *diskCredentials) Set(_, _, secret string) error {
	if err := persistence.WriteFile(s.path, []byte(secret)); err != nil {
		return err
	}
	if s.afterSet != nil {
		s.afterSet()
	}
	return nil
}
func (s *diskCredentials) Delete(string, string) error {
	err := os.Remove(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
func TestCredentialJournalSurvivesProcessTermination(t *testing.T) {
	phase := os.Getenv("KAKAPO_CREDENTIAL_CRASH_PHASE")
	if phase != "" {
		a := NewTranslateApp()
		store := &diskCredentials{path: os.Getenv("KAKAPO_CREDENTIAL_CRASH_KEYFILE")}
		a.credentials = store
		if phase == "writer" {
			store.afterSet = func() {
				_, _ = os.Stdout.WriteString("ready\n")
				for {
					time.Sleep(time.Hour)
				}
			}
			if err := a.SaveSettings(&SettingsDTO{Providers: []ProviderDTO{{ID: config.DefaultProviderID, BaseURL: os.Getenv("KAKAPO_CREDENTIAL_CRASH_NEXT_URL"), Models: []string{"m"}, Enabled: true, SetAPIKey: "new-synthetic-key"}}}); err != nil {
				t.Fatal(err)
			}
			t.Fatal("credential commit boundary was not reached")
		}
		if _, err := a.TranslateZhToEn("source"); !errors.Is(err, translate.ErrSettingsPending) {
			t.Fatalf("restart did not block single translation: %v", err)
		}
		if _, err := a.TranslateParallel("source", "", nil, nil); !errors.Is(err, translate.ErrSettingsPending) {
			t.Fatalf("restart did not block parallel translation: %v", err)
		}
		status, err := a.GetSettings()
		if err != nil || len(status.PendingCredentialIDs) != 1 || !status.Providers[0].APIKeyNeedsConfirmation {
			t.Fatalf("restart recovery status: %+v %v", status, err)
		}
		return
	}
	dir := isolatedConfig(t)
	requests := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- struct{}{}
		io.WriteString(w, `{"choices":[{"message":{"content":"unexpected"}}]}`)
	}))
	defer server.Close()
	c := config.Default()
	c.Providers[0].BaseURL = server.URL
	writeConfig(t, c)
	keyfile := filepath.Join(t.TempDir(), "synthetic-credential")
	if err := persistence.WriteFile(keyfile, []byte("old-synthetic-key")); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "KAKAPO_CREDENTIAL_CRASH_KEYFILE="+keyfile, "KAKAPO_CREDENTIAL_CRASH_NEXT_URL="+server.URL+"/next")
	writer := exec.Command(os.Args[0], "-test.run", "^TestCredentialJournalSurvivesProcessTermination$")
	writer.Env = append(env, "KAKAPO_CREDENTIAL_CRASH_PHASE=writer")
	stdout, err := writer.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Process.Kill(); _ = writer.Wait() }()
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if line != "ready\n" {
			t.Fatalf("writer exited before mutation: %q", line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("writer did not reach credential commit")
	}
	if err := writer.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	err = writer.Wait()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
		t.Fatalf("unexpected writer result: %v", err)
	}
	secret, err := os.ReadFile(keyfile)
	if err != nil || string(secret) != "new-synthetic-key" {
		t.Fatal("synthetic credential did not survive process termination")
	}
	persisted, err := config.Load()
	if err != nil || len(persisted.PendingCredentialIDs) != 1 || persisted.Providers[0].BaseURL != server.URL {
		t.Fatalf("durable pending state: %+v %v", persisted, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "settings.json")); err != nil {
		t.Fatal(err)
	}
	reader := exec.Command(os.Args[0], "-test.run", "^TestCredentialJournalSurvivesProcessTermination$")
	reader.Env = append(env, "KAKAPO_CREDENTIAL_CRASH_PHASE=reader")
	if out, err := reader.CombinedOutput(); err != nil {
		t.Fatalf("fresh-process recovery: %v\n%s", err, out)
	}
	select {
	case <-requests:
		t.Fatal("endpoint received a request while credentials were pending")
	default:
	}
}
