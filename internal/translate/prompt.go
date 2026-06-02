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

package translate

import "strings"

// SystemPrompt is the system message for 中译英 (legacy / default).
const SystemPrompt = "你是专业英文写作助手，将用户中文翻译为自然地道英文，保留原意，语气尽量商务中性。输出仅英文，不要解释。"

// built-in prompts for common language pairs (source -> target).
var builtinPrompts = map[string]string{
	"zh-en": "你是专业英文写作助手，将用户中文翻译为自然地道英文，保留原意，语气尽量商务中性。输出仅英文，不要解释。",
	"en-zh": "你是专业中文写作助手，将用户英文翻译为自然地道中文，保留原意，语气尽量商务中性。输出仅中文，不要解释。",
	"zh-ja": "你是专业日文写作助手，将用户中文翻译为自然地道日文，保留原意。输出仅日文，不要解释。",
	"ja-zh": "你是专业中文写作助手，将用户日文翻译为自然地道中文，保留原意。输出仅中文，不要解释。",
	"en-ja": "你是专业日文写作助手，将用户英文翻译为自然地道日文，保留原意。输出仅日文，不要解释。",
	"ja-en": "你是专业英文写作助手，将用户日文翻译为自然地道英文，保留原意。输出仅英文，不要解释。",
}

// BuildSystemPrompt returns the system prompt for the given source and target language pair.
// Language codes are normalized to lowercase. Unknown pairs get a generic template.
func BuildSystemPrompt(source, target string) string {
	key := strings.ToLower(strings.TrimSpace(source)) + "-" + strings.ToLower(strings.TrimSpace(target))
	if key == "" {
		return SystemPrompt
	}
	if p, ok := builtinPrompts[key]; ok {
		return p
	}
	// Generic template for any other pair
	return "你是一名专业翻译。将用户的输入从 " + source + " 翻译成 " + target + "，保留原意，语气中性。只输出译文，不要解释。"
}
