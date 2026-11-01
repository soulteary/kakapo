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

package secrets

import "testing"

func TestUnsupportedKeychainDoesNotPersist(t *testing.T) {
	var store Store = KeychainStore{}
	if KeychainService == "" {
		t.Fatal("missing service name")
	}
	if err := store.Set(KeychainService, "synthetic", "synthetic-key"); err != nil {
		t.Fatal(err)
	}
	value, err := store.Get(KeychainService, "synthetic")
	if err != nil || value != "" {
		t.Fatalf("non-macOS stub persisted a credential: %q %v", value, err)
	}
	if err := store.Delete(KeychainService, "synthetic"); err != nil {
		t.Fatal(err)
	}
}
