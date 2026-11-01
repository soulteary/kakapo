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
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func BenchmarkHistoryDecode(b *testing.B) {
	for _, results := range []int{1, 6} {
		b.Run(fmt.Sprintf("results=%d", results), func(b *testing.B) {
			entries := make([]Entry, maxEntries)
			for i := range entries {
				entries[i] = Entry{Input: strings.Repeat("你好世界", 100), Output: strings.Repeat("translation ", 100), CreatedAt: int64(i), SourceLanguage: "zh", TargetLanguages: []string{"en"}, Models: []string{"model"}}
				for j := 0; j < results; j++ {
					entries[i].Results = append(entries[i].Results, ResultItem{Provider: "provider", Model: "model", TargetLanguage: "en", Output: strings.Repeat("translation ", 100), LatencyMs: 200})
				}
			}
			data, err := json.Marshal(entries)
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				var got []Entry
				if err := json.Unmarshal(data, &got); err != nil {
					b.Fatal(err)
				}
				if len(got) != maxEntries || len(got[0].Results) != results {
					b.Fatal("incomplete history")
				}
			}
		})
	}
}
