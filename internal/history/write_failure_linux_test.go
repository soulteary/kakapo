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

package history

import (
	"bytes"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestFailedWritePreservesHistory(t *testing.T) {
	if path := os.Getenv("KAKAPO_HISTORY_WRITE_FAILURE_CHILD"); path != "" {
		signal.Ignore(syscall.SIGXFSZ)
		var limit syscall.Rlimit
		if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			limit.Cur = limit.Max
			if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
				t.Error(err)
			}
		})
		s := &Store{path: path}
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Add(Entry{Input: "new", Output: strings.Repeat("synthetic", 512), CreatedAt: 2}); err == nil {
			t.Fatal("file size limit did not cause a write error")
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal("failed write changed previously committed history")
		}
		return
	}
	path := filepath.Join(t.TempDir(), "history.json")
	s := &Store{path: path}
	if err := s.Add(Entry{Input: "previous", Output: "committed", CreatedAt: 1}); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", `ulimit -S -f 1; exec "$1" -test.run '^TestFailedWritePreservesHistory$'`, "sh", os.Args[0])
	cmd.Env = append(os.Environ(), "KAKAPO_HISTORY_WRITE_FAILURE_CHILD="+path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("real filesystem write-limit regression: %v\n%s", err, out)
	}
}
