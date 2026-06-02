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

package speech

import (
	"context"
	"os/exec"
	"strings"
	"sync"
)

// languageCode -> macOS voice name (from "say -v?")
var defaultVoices = map[string]string{
	"zh":    "Tingting", // 简体中文
	"en":    "Alex",     // 英语
	"en-US": "Samantha", // 美式英语
	"ja":    "Kyoko",    // 日语
	"ko":    "Yuna",     // 韩语
	"de":    "Anna",
	"fr":    "Amelie",
	"es":    "Monica",
	"it":    "Alice",
	"pt":    "Luciana",
	"pt-BR": "Luciana",
	"ru":    "Milena",
	"nl":    "Xander",
	"pl":    "Zosia",
	"tr":    "Yelda",
	"sv":    "Alva",
	"da":    "Sara",
	"fi":    "Satu",
	"el":    "Melina",
	"id":    "Damayanti",
	"hu":    "Mariska",
	"ro":    "Ioana",
	"cs":    "Zuzana",
	"sk":    "Laura",
	"bg":    "Milena",
	"uk":    "Milena",
	"et":    "Liisu",
	"lv":    "Liga",
	"lt":    "Rasa",
	"sl":    "Lado",
}

// Single-flight control: a new Speak cancels the previous "say" process so
// utterances do not overlap and stale requests stop holding their connection.
var (
	speakMu     sync.Mutex
	speakCancel context.CancelFunc
	speakGen    uint64
)

// Speak uses macOS "say" to speak the given text with a voice appropriate for the language.
// If language is empty or unknown, the system default voice is used.
// Empty text is a no-op and returns nil.
//
// Only one utterance plays at a time: starting a new Speak cancels any in-flight
// one. Being preempted by a newer call is not reported as an error.
func Speak(ctx context.Context, text string, language string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	voice := ""
	if language != "" {
		voice = defaultVoices[language]
		// fallback: try base language (e.g. pt-BR -> pt)
		if voice == "" && len(language) > 2 && language[2] == '-' {
			voice = defaultVoices[language[:2]]
		}
	}
	args := []string{}
	if voice != "" {
		args = append(args, "-v", voice)
	}
	args = append(args, text)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	speakMu.Lock()
	speakGen++
	gen := speakGen
	if speakCancel != nil {
		speakCancel() // stop the previous utterance
	}
	speakCancel = cancel
	speakMu.Unlock()

	err := exec.CommandContext(runCtx, "say", args...).Run()

	speakMu.Lock()
	if speakGen == gen {
		speakCancel = nil
	}
	speakMu.Unlock()

	// If we were preempted by a newer Speak (runCtx cancelled but the caller's
	// ctx is still live), that is expected, not a failure.
	if runCtx.Err() != nil && ctx.Err() == nil {
		return nil
	}
	return err
}
