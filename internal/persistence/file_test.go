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

package persistence

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func defaultOperations() operations {
	return operations{(*os.File).Sync, (*os.File).Close, os.Rename, (*os.File).Sync}
}
func TestReplacementAndPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := WriteFile(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("second")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "second" {
		t.Fatalf("replacement: %q %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("permissions: %v %v", info, err)
	}
}
func TestDestinationFailuresPreserveExistingData(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFile(filepath.Join(dir, "missing", "data"), []byte("x")); err == nil {
		t.Fatal("missing parent accepted")
	}
	if err := WriteFile(dir, []byte("x")); !errors.Is(err, ErrNotRegular) {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(link, []byte("new")); !errors.Is(err, ErrNotRegular) {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "old" {
		t.Fatal("symlink target changed")
	}
	if err := WriteFile(filepath.Join(target, "child"), []byte("x")); err == nil {
		t.Fatal("invalid parent accepted")
	}
}
func TestSyncCloseRenameAndCommitUncertainty(t *testing.T) {
	for _, phase := range []string{"sync", "close", "rename", "directory sync"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "data")
			if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
				t.Fatal(err)
			}
			ops := defaultOperations()
			fault := syscall.EIO
			switch phase {
			case "sync":
				ops.syncFile = func(*os.File) error { return fault }
			case "close":
				ops.closeFile = func(f *os.File) error {
					if err := f.Close(); err != nil {
						return err
					}
					return fault
				}
			case "rename":
				ops.rename = func(string, string) error { return fault }
			case "directory sync":
				ops.syncDir = func(*os.File) error { return fault }
			}
			err := writeFile(path, []byte("new"), ops)
			if !errors.Is(err, fault) {
				t.Fatalf("phase error lost: %v", err)
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			var committed *CommittedError
			if phase == "directory sync" {
				if string(data) != "new" || !errors.As(err, &committed) || !strings.Contains(err.Error(), "committed") {
					t.Fatalf("uncertain commit not disclosed: %q %v", data, err)
				}
			} else if string(data) != "old" || errors.As(err, &committed) {
				t.Fatalf("precommit failure changed data: %q %v", data, err)
			}
			files, err := filepath.Glob(filepath.Join(dir, ".kakapo-*.tmp"))
			if err != nil || len(files) != 0 {
				t.Fatalf("temporary files retained: %v %v", files, err)
			}
		})
	}
}

func TestReadOnlyDestinationIsNotReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data")
	if err := os.WriteFile(path, []byte("old"), 0400); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("new")); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("read-only destination accepted: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "old" {
		t.Fatal("read-only data changed")
	}
}
