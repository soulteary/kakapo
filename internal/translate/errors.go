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

import (
	"errors"
	"fmt"
)

// Sentinel errors for UI mapping (see ErrorMessage).
var (
	ErrNoAPIKey      = errors.New("err_no_api_key")
	ErrInvalidConfig = errors.New("err_invalid_config")
	ErrNetwork       = errors.New("err_network")
	ErrTimeout       = errors.New("err_timeout")
	ErrUnauthorized  = errors.New("err_unauthorized")
	ErrRateLimited   = errors.New("err_rate_limited")
	ErrServer        = errors.New("err_server")
	ErrBadResponse   = errors.New("err_bad_response")
)

// APIError wraps a sentinel error (Kind) with the HTTP status and the detail
// extracted from the upstream API response body, so the UI can surface the
// real reason (e.g. "model not found", "invalid api key").
type APIError struct {
	Kind   error // one of the sentinel errors above
	Status int
	Detail string
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: HTTP %d: %s", e.Kind.Error(), e.Status, e.Detail)
	}
	return fmt.Sprintf("%s: HTTP %d", e.Kind.Error(), e.Status)
}

// Unwrap returns the sentinel kind so errors.Is keeps working.
func (e *APIError) Unwrap() error { return e.Kind }

// newAPIError builds an APIError for a non-2xx response.
func newAPIError(kind error, status int, detail string) error {
	return &APIError{Kind: kind, Status: status, Detail: detail}
}

// ErrorMessage returns a user-facing message for a known error. When the error
// carries upstream detail (APIError), it is appended so the user sees the real
// cause rather than only a generic hint.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	base := baseMessage(err)
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Detail != "" {
		return fmt.Sprintf("%s（接口返回：%s）", base, apiErr.Detail)
	}
	return base
}

// baseMessage maps a known sentinel error to its generic user-facing message.
func baseMessage(err error) string {
	switch {
	case errors.Is(err, ErrNoAPIKey):
		return "未配置 API Key，请在设置中填写"
	case errors.Is(err, ErrInvalidConfig):
		return "配置无效，请检查 Base URL 和模型"
	case errors.Is(err, ErrNetwork):
		return "网络不可用，请检查连接后重试"
	case errors.Is(err, ErrTimeout):
		return "请求超时，请稍后重试"
	case errors.Is(err, ErrUnauthorized):
		return "API Key 无效或已过期，请在设置中更新"
	case errors.Is(err, ErrRateLimited):
		return "请求过于频繁，请稍后再试"
	case errors.Is(err, ErrServer):
		return "翻译服务暂时不可用，请稍后重试"
	case errors.Is(err, ErrBadResponse):
		return "翻译结果异常，请重试"
	default:
		return err.Error()
	}
}
