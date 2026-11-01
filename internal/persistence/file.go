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

// Package persistence writes private JSON files without truncating committed data.
package persistence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotRegular rejects symlinks and non-regular destination files.
var ErrNotRegular = errors.New("persistence destination must be a regular file")

// CommittedError means replacement completed, but its directory could not be
// synced. The new file is visible; crash durability is uncertain. Inspect the
// current data before retrying an operation that appends or has other side effects.
type CommittedError struct{ Err error }

func (e *CommittedError) Error() string {
	return "file replacement committed; directory sync failed: " + e.Err.Error()
}
func (e *CommittedError) Unwrap() error { return e.Err }

type operations struct {
	syncFile  func(*os.File) error
	closeFile func(*os.File) error
	rename    func(string, string) error
	syncDir   func(*os.File) error
}

// WriteFile writes a private file and atomically replaces path on the same
// filesystem. Before replacement, every failure preserves the previous file.
// It does not provide locking across processes or transactions across files.
func WriteFile(path string, data []byte) error {
	return writeFile(path, data, operations{(*os.File).Sync, (*os.File).Close, os.Rename, (*os.File).Sync})
}

func writeFile(path string, data []byte, ops operations) error {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return ErrNotRegular
		}
		if info.Mode().Perm()&0200 == 0 {
			return os.ErrPermission
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	file, err := os.CreateTemp(filepath.Dir(path), ".kakapo-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := ops.syncFile(file); err != nil {
		return fmt.Errorf("sync replacement file: %w", err)
	}
	if err := ops.closeFile(file); err != nil {
		return fmt.Errorf("close replacement file: %w", err)
	}
	if err := ops.rename(file.Name(), path); err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	if err := ops.syncDir(dir); err != nil {
		return &CommittedError{Err: err}
	}
	return nil
}
