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
	"errors"
	"github.com/soulteary/kakapo/internal/config"
	"github.com/soulteary/kakapo/internal/translate"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type interruptedCredentials struct {
	*testCredentials
	afterSet    func() error
	afterDelete func() error
}

func (s *interruptedCredentials) Set(service, account, key string) error {
	if err := s.testCredentials.Set(service, account, key); err != nil {
		return err
	}
	if s.afterSet != nil {
		return s.afterSet()
	}
	return nil
}
func (s *interruptedCredentials) Delete(service, account string) error {
	if err := s.testCredentials.Delete(service, account); err != nil {
		return err
	}
	if s.afterDelete != nil {
		return s.afterDelete()
	}
	return nil
}
func TestCredentialCommitFailureBlocksMismatchedTranslation(t *testing.T) {
	dir := isolatedConfig(t)
	oldRequests := make(chan string, 1)
	old := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oldRequests <- r.Header.Get("Authorization")
		io.WriteString(w, `{"choices":[{"message":{"content":"old endpoint"}}]}`)
	}))
	defer old.Close()
	nextRequests := make(chan string, 1)
	next := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextRequests <- r.Header.Get("Authorization")
		io.WriteString(w, `{"choices":[{"message":{"content":"new endpoint"}}]}`)
	}))
	defer next.Close()
	c := config.Default()
	c.Providers[0].BaseURL = old.URL
	writeConfig(t, c)
	store := &interruptedCredentials{testCredentials: &testCredentials{keys: map[string]string{config.DefaultProviderID: "old-synthetic-key"}}}
	a := NewTranslateApp()
	a.credentials = store
	path := filepath.Join(dir, "settings.json")
	store.afterSet = func() error { return os.Chmod(path, 0400) }
	dto := &SettingsDTO{Providers: []ProviderDTO{{ID: config.DefaultProviderID, BaseURL: next.URL, Models: []string{"m"}, Enabled: true, SetAPIKey: "new-synthetic-key"}}}
	if err := a.SaveSettings(dto); err == nil {
		t.Fatal("expected real settings write failure after credential mutation")
	}
	if _, err := a.TranslateZhToEn("synthetic source"); err == nil {
		t.Fatal("translation accepted mismatched settings and credential after failed save")
	}
	select {
	case <-oldRequests:
		t.Fatal("old endpoint received request after failed credential/settings commit")
	default:
	}
	persisted, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(persisted), "pendingCredentialIDs") || strings.Contains(string(persisted), "synthetic-key") {
		t.Fatal("durable marker missing or contained a credential")
	}
	restarted := NewTranslateApp()
	restarted.credentials = store
	if _, err := restarted.TranslateParallel("source", "", nil, nil); !errors.Is(err, translate.ErrSettingsPending) {
		t.Fatalf("restart/parallel bypassed pending state: %v", err)
	}
	status, err := restarted.GetSettings()
	if err != nil || len(status.PendingCredentialIDs) != 1 || !status.Providers[0].APIKeyNeedsConfirmation {
		t.Fatalf("recovery status missing: %+v %v", status, err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	store.afterSet = nil
	incomplete := *dto
	incomplete.Providers = append([]ProviderDTO(nil), dto.Providers...)
	incomplete.Providers[0].SetAPIKey = ""
	if err := restarted.SaveSettings(&incomplete); !errors.Is(err, translate.ErrSettingsPending) {
		t.Fatalf("pending state cleared without confirmation: %v", err)
	}
	if err := a.SaveSettings(dto); err != nil {
		t.Fatalf("explicit credential recovery failed: %v", err)
	}
	result, err := a.TranslateZhToEn("source")
	if err != nil || result.Output != "new endpoint" {
		t.Fatalf("recovered translation: %+v %v", result, err)
	}
	if header := <-nextRequests; header != "Bearer new-synthetic-key" {
		t.Fatal("recovered endpoint received wrong credential")
	}
}

func TestAmbiguousCredentialAcknowledgementAndClearRecovery(t *testing.T) {
	isolatedConfig(t)
	c := config.Default()
	writeConfig(t, c)
	base := &testCredentials{keys: map[string]string{config.DefaultProviderID: "old-synthetic"}}
	store := &interruptedCredentials{testCredentials: base, afterSet: func() error { return errors.New("synthetic lost acknowledgement") }}
	a := NewTranslateApp()
	a.credentials = store
	dto := &SettingsDTO{Providers: []ProviderDTO{{ID: config.DefaultProviderID, BaseURL: c.Providers[0].BaseURL, Models: []string{"m"}, SetAPIKey: "new-synthetic", Enabled: true}}}
	if err := a.SaveSettings(dto); !errors.Is(err, translate.ErrSettingsPending) {
		t.Fatalf("ambiguous Set accepted: %v", err)
	}
	key, _ := base.Get("", config.DefaultProviderID)
	if key != "new-synthetic" {
		t.Fatal("fault did not occur after credential commit")
	}
	if _, err := a.TranslateZhToEn("x"); !errors.Is(err, translate.ErrSettingsPending) {
		t.Fatal("ambiguous key was used")
	}
	store.afterSet = nil
	store.afterDelete = func() error { return errors.New("synthetic lost delete acknowledgement") }
	dto.Providers[0].SetAPIKey = ""
	dto.Providers[0].ClearAPIKey = true
	if err := a.SaveSettings(dto); !errors.Is(err, translate.ErrSettingsPending) {
		t.Fatalf("ambiguous Delete accepted: %v", err)
	}
	key, _ = base.Get("", config.DefaultProviderID)
	if key != "" {
		t.Fatal("delete fault happened before deletion")
	}
	store.afterDelete = nil
	if err := a.SaveSettings(dto); err != nil {
		t.Fatalf("idempotent clear recovery: %v", err)
	}
	current, err := config.Load()
	if err != nil || len(current.PendingCredentialIDs) != 0 {
		t.Fatalf("recovery marker retained: %+v %v", current, err)
	}
}
func TestCredentialMarkerFailureAndInvalidIdentifiers(t *testing.T) {
	dir := isolatedConfig(t)
	c := config.Default()
	writeConfig(t, c)
	base := &testCredentials{keys: map[string]string{config.DefaultProviderID: "old"}}
	a := NewTranslateApp()
	a.credentials = base
	path := filepath.Join(dir, "settings.json")
	dto := &SettingsDTO{Providers: []ProviderDTO{{ID: config.DefaultProviderID, BaseURL: c.Providers[0].BaseURL, Models: []string{"m"}, SetAPIKey: "new"}}}
	bad := *dto
	bad.Temperature = math.NaN()
	if err := a.SaveSettings(&bad); !errors.Is(err, translate.ErrInvalidConfig) {
		t.Fatal("non-finite input was accepted")
	}
	keyBefore, _ := base.Get("", config.DefaultProviderID)
	if keyBefore != "old" {
		t.Fatal("invalid input mutated credential")
	}
	if err := os.Chmod(path, 0400); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveSettings(dto); err == nil {
		t.Fatal("unwritable marker accepted")
	}
	key, _ := base.Get("", config.DefaultProviderID)
	if key != "old" {
		t.Fatal("credential changed before durable marker")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	duplicate := *dto
	duplicate.Providers = append([]ProviderDTO(nil), dto.Providers...)
	duplicate.Providers = append(duplicate.Providers, dto.Providers[0])
	if err := a.SaveSettings(&duplicate); !errors.Is(err, translate.ErrInvalidConfig) {
		t.Fatalf("duplicate provider accepted: %v", err)
	}
	for _, invalid := range []config.Settings{{Providers: []config.ProviderConfig{{ID: ""}}}, {Providers: []config.ProviderConfig{{ID: "x"}, {ID: "x"}}}, {Providers: c.Providers, PendingCredentialIDs: []string{""}}} {
		writeConfig(t, invalid)
		if err := a.SaveSettings(dto); !errors.Is(err, translate.ErrInvalidConfig) {
			t.Fatalf("invalid previous state accepted: %v", err)
		}
	}
	c.PendingCredentialIDs = []string{"removed"}
	writeConfig(t, c)
	base.keys["removed"] = "synthetic"
	recovery, err := a.GetSettings()
	if err != nil || len(recovery.Providers) != 2 || recovery.Providers[1].ID != "removed" || recovery.Providers[1].BaseURL != "" || !recovery.Providers[1].APIKeyNeedsConfirmation {
		t.Fatalf("uncommitted provider recovery card missing: %+v %v", recovery, err)
	}

	if err := a.SaveSettings(&SettingsDTO{}); err != nil {
		t.Fatalf("removed-provider recovery: %v", err)
	}
	key, _ = base.Get("", "removed")
	if key != "" {
		t.Fatal("removed pending credential retained")
	}
}

type snapshotCredentials struct {
	*testCredentials
	entered chan struct{}
	release chan struct{}
	updated chan struct{}
	once    sync.Once
	setOnce sync.Once
}

func (s *snapshotCredentials) Get(service, account string) (string, error) {
	s.once.Do(func() { close(s.entered); <-s.release })
	return s.testCredentials.Get(service, account)
}
func (s *snapshotCredentials) Set(service, account, key string) error {
	err := s.testCredentials.Set(service, account, key)
	s.setOnce.Do(func() { close(s.updated) })
	return err
}
func TestConcurrentSettingsAndCredentialSnapshot(t *testing.T) {
	isolatedConfig(t)
	oldHeaders := make(chan string, 2)
	old := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oldHeaders <- r.Header.Get("Authorization")
		io.WriteString(w, `{"choices":[{"message":{"content":"old"}}]}`)
	}))
	defer old.Close()
	nextHeaders := make(chan string, 2)
	next := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextHeaders <- r.Header.Get("Authorization")
		io.WriteString(w, `{"choices":[{"message":{"content":"new"}}]}`)
	}))
	defer next.Close()
	c := config.Default()
	c.Providers[0].BaseURL = old.URL
	writeConfig(t, c)
	store := &snapshotCredentials{testCredentials: &testCredentials{keys: map[string]string{config.DefaultProviderID: "old-key"}}, entered: make(chan struct{}), release: make(chan struct{}), updated: make(chan struct{})}
	a := NewTranslateApp()
	a.credentials = store
	translated := make(chan error, 1)
	go func() { _, err := a.TranslateZhToEn("source"); translated <- err }()
	select {
	case <-store.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("key snapshot was not reached")
	}
	saved := make(chan error, 1)
	go func() {
		saved <- a.SaveSettings(&SettingsDTO{Providers: []ProviderDTO{{ID: config.DefaultProviderID, BaseURL: next.URL, Models: []string{"m"}, Enabled: true, SetAPIKey: "new-key"}}})
	}()
	select {
	case <-store.updated:
	case <-time.After(100 * time.Millisecond):
	}
	close(store.release)
	if err := <-translated; err != nil {
		t.Fatal(err)
	}
	if err := <-saved; err != nil {
		t.Fatal(err)
	}
	if header := <-oldHeaders; header != "Bearer old-key" {
		t.Fatal("old endpoint received new credential from a torn snapshot")
	}
	if _, err := a.TranslateZhToEn("source"); err != nil {
		t.Fatal(err)
	}
	if header := <-nextHeaders; header != "Bearer new-key" {
		t.Fatal("new endpoint received old credential")
	}
}
