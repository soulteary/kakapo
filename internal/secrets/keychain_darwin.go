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
// +build darwin

package secrets

import (
	"github.com/keybase/go-keychain"
)

// KeychainService is the service name for API Key items. The account is the
// provider ID (see config.ProviderConfig.ID), so each provider has its own key.
const KeychainService = "com.soulteary.kakapo"

// KeychainStore implements Store using macOS Keychain.
type KeychainStore struct{}

// Get returns the secret for the given service/account, or empty if not found.
func (KeychainStore) Get(service, account string) (string, error) {
	password, err := keychain.GetGenericPassword(service, account, "", "")
	if err != nil {
		if err == keychain.ErrorItemNotFound {
			return "", nil
		}
		return "", err
	}
	return string(password), nil
}

// Set stores the secret. Replaces existing item.
func (KeychainStore) Set(service, account, secret string) error {
	item := keychain.NewGenericPassword(service, account, "Kakapo API Key", []byte(secret), "")
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	err := keychain.AddItem(item)
	if err == keychain.ErrorDuplicateItem {
		_ = keychain.DeleteGenericPasswordItem(service, account)
		return keychain.AddItem(item)
	}
	return err
}

// Delete removes the secret.
func (KeychainStore) Delete(service, account string) error {
	return keychain.DeleteGenericPasswordItem(service, account)
}
