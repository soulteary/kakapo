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

//go:build !darwin
// +build !darwin

package secrets

// KeychainService matches the darwin name for API compatibility (stub ignores it).
const KeychainService = "com.soulteary.kakapo"

// KeychainStore stub for non-darwin (no Keychain).
type KeychainStore struct{}

func (KeychainStore) Get(service, account string) (string, error) { return "", nil }
func (KeychainStore) Set(service, account, secret string) error   { return nil }
func (KeychainStore) Delete(service, account string) error        { return nil }
