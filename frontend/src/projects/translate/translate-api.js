/*
 * Copyright 2026 Su Yang (soulteary)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Web 版翻译 API：通过 fetch 调用 /translate/api/*。
 * 使用绝对路径避免相对 URL 在不同 base 下解析到错误端点。
 */
const API_BASE = '/translate';

async function request(method, path, body) {
  const url = `${API_BASE}${path}`;
  const opt = { method, headers: {} };
  if (body != null) {
    opt.headers['Content-Type'] = 'application/json';
    opt.body = JSON.stringify(body);
  }
  const res = await fetch(url, opt);
  const text = await res.text();
  if (!res.ok) {
    let msg = text;
    try {
      const j = JSON.parse(text);
      if (j.error) msg = j.error;
    } catch (_) {}
    const err = new Error(msg);
    err.Error = msg;
    throw err;
  }
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch (_) {
    return text;
  }
}

/**
 * Legacy single translation (zh -> en, one model). Kept for backward compatibility.
 */
export async function TranslateZhToEn(text) {
  return request('POST', '/api/translate', { text });
}

/**
 * Translate with optional source/target/models. Returns either legacy shape
 * { input, output, model, ... } or multi shape { input, results: [{ model, targetLanguage, output, latencyMs, error? }], createdAt }.
 */
export async function Translate(payload) {
  return request('POST', '/api/translate', {
    text: payload.text,
    sourceLanguage: payload.sourceLanguage || undefined,
    targetLanguages: payload.targetLanguages && payload.targetLanguages.length ? payload.targetLanguages : undefined,
    models: payload.models && payload.models.length ? payload.models : undefined,
  });
}

export async function GetSettings() {
  return request('GET', '/api/settings');
}

export async function SaveSettings(settings) {
  return request('PUT', '/api/settings', settings);
}

export async function CopyToClipboard(text) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    await navigator.clipboard.writeText(text);
  } else {
    throw new Error('复制不可用');
  }
}

export function HideWindow() {}

export async function AddHistory(result) {
  return request('POST', '/api/history', result);
}

export async function GetHistory(query) {
  const q = query ? `?q=${encodeURIComponent(query)}` : '';
  return request('GET', `/api/history${q}`);
}

export async function ClearHistory() {
  return request('DELETE', '/api/history');
}

/**
 * 显示关于/启动页（splash）窗口。后端通过 Wails 事件通知主进程展示窗口。
 */
export async function ShowSplash() {
  return request('POST', '/api/splash');
}

/**
 * 使用系统 TTS 朗读文本（macOS 使用 say）。payload: { text, language? }。
 * 非 macOS 时后端返回 501，可据此提示“当前系统不支持朗读”。
 */
export async function Speak(payload) {
  return request('POST', '/api/speak', {
    text: payload.text ?? '',
    language: payload.language ?? '',
  });
}
