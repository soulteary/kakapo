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

// APIError carries a sentinel kind and HTTP status without upstream body text.
// Provider responses may echo credentials or private input and must not become
// diagnostics that can be displayed, logged or persisted in history.
type APIError struct {
	Kind   error // one of the sentinel errors above
	Status int
	// Detail is retained for source compatibility. New errors leave it empty;
	// diagnostic formatting ignores it, including values created by older callers.
	Detail string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: HTTP %d", e.Kind.Error(), e.Status)
}

// Unwrap returns the sentinel kind so errors.Is keeps working.
func (e *APIError) Unwrap() error { return e.Kind }

// newAPIError builds an APIError for a non-2xx response.
func newAPIError(kind error, status int) error {
	return &APIError{Kind: kind, Status: status}
}

// ErrorMessage returns a user-facing message for a known error without exposing
// upstream response bodies or transport error details.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return baseMessage(err)
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
