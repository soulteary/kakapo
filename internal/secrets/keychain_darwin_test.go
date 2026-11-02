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

//go:build darwin

package secrets

import (
	"crypto/rand"
	"encoding/hex"
	"testing"
)

// TestNativeKeychainRoundTrip uses a unique synthetic service; it never reads or
// modifies the application's real credential namespace.
func TestNativeKeychainRoundTrip(t *testing.T) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	service := "com.soulteary.kakapo.integration." + hex.EncodeToString(nonce)
	account := "synthetic-account"
	store := KeychainStore{}
	if value, err := store.Get(service, account); err != nil || value != "" {
		t.Fatal("native fixture namespace was unavailable or not empty")
	}
	t.Cleanup(func() {
		if err := store.Delete(service, account); err != nil {
			t.Errorf("native fixture cleanup failed: %v", err)
		}
	})
	if err := store.Set(service, account, "synthetic-first-key"); err != nil {
		t.Fatalf("native initial write failed: %v", err)
	}
	if value, err := store.Get(service, account); err != nil || value != "synthetic-first-key" {
		t.Fatal("native initial read did not match")
	}
	if err := store.Set(service, account, "synthetic-second-key"); err != nil {
		t.Fatalf("native replacement failed: %v", err)
	}
	if value, err := store.Get(service, account); err != nil || value != "synthetic-second-key" {
		t.Fatal("native replacement read did not match")
	}
	if err := store.Delete(service, account); err != nil {
		t.Fatalf("native deletion failed: %v", err)
	}
	if value, err := store.Get(service, account); err != nil || value != "" {
		t.Fatal("native deleted credential remained visible")
	}
	if err := store.Delete(service, account); err != nil {
		t.Fatalf("native repeated deletion failed: %v", err)
	}
}
