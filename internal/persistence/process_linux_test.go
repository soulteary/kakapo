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
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestProcessTerminationAtCommitBoundary(t *testing.T) {
	if phase := os.Getenv("KAKAPO_ATOMIC_KILL_PHASE"); phase != "" {
		block := func(*os.File) error {
			_, _ = os.Stdout.WriteString("ready\n")
			for {
				time.Sleep(time.Hour)
			}
		}
		ops := defaultOperations()
		if phase == "before" {
			ops.syncFile = block
		} else {
			ops.syncDir = block
		}
		if err := writeFile(os.Getenv("KAKAPO_ATOMIC_KILL_PATH"), []byte("new"), ops); err != nil {
			t.Fatal(err)
		}
		t.Fatal("commit boundary was not reached")
	}
	for _, phase := range []string{"before", "after"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "data")
			if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run", "^TestProcessTerminationAtCommitBoundary$")
			cmd.Env = append(os.Environ(), "KAKAPO_ATOMIC_KILL_PHASE="+phase, "KAKAPO_ATOMIC_KILL_PATH="+path)
			pipe, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer cmd.Process.Kill()
			ready := make(chan string, 1)
			go func() { line, _ := bufio.NewReader(pipe).ReadString('\n'); ready <- line }()
			select {
			case line := <-ready:
				if line != "ready\n" {
					t.Fatalf("child failed before boundary: %q", line)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("child did not reach commit boundary")
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			err = cmd.Wait()
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
				t.Fatalf("unexpected process result: %v", err)
			}
			data, err := os.ReadFile(path)
			want := "old"
			if phase == "after" {
				want = "new"
			}
			if err != nil || string(data) != want {
				t.Fatalf("phase=%s data=%q want=%q err=%v", phase, data, want, err)
			}
			leftovers, err := filepath.Glob(filepath.Join(dir, ".kakapo-*.tmp"))
			if err != nil {
				t.Fatal(err)
			}
			if phase == "before" && len(leftovers) != 1 {
				t.Fatalf("expected a precommit orphan, got %v", leftovers)
			}
			if phase == "after" && len(leftovers) != 0 {
				t.Fatalf("postcommit temp retained: %v", leftovers)
			}
			for _, name := range leftovers {
				info, err := os.Stat(name)
				if err != nil || info.Mode().Perm()&0077 != 0 {
					t.Fatalf("orphan permissions: %v %v", info, err)
				}
			}
		})
	}
}
