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

import './style.css';

import { Application } from '@wailsio/runtime';
import {
  Translate,
  GetSettings,
  SaveSettings,
  CopyToClipboard,
  HideWindow,
  AddHistory,
  GetHistory,
  ClearHistory,
  Speak,
  ShowSplash,
} from './translate-api.js';

// Built-in language options (code -> label)，完整语言列表
const LANG_OPTIONS = [
  { code: 'zh', label: '简体中文' },
  { code: 'en', label: '英语' },
  { code: 'en-US', label: '美式英语' },
  { code: 'ja', label: '日语' },
  { code: 'et', label: '爱沙尼亚语' },
  { code: 'pl', label: '波兰语' },
  { code: 'de', label: '德语' },
  { code: 'fr', label: '法语' },
  { code: 'nl', label: '荷兰语' },
  { code: 'lv', label: '拉脱维亚语' },
  { code: 'bg', label: '保加利亚语' },
  { code: 'da', label: '丹麦语' },
  { code: 'ru', label: '俄语' },
  { code: 'fi', label: '芬兰语' },
  { code: 'cs', label: '捷克语' },
  { code: 'lt', label: '立陶宛语' },
  { code: 'ro', label: '罗马尼亚语' },
  { code: 'pt', label: '葡萄牙语' },
  { code: 'pt-BR', label: '葡萄牙语(巴西)' },
  { code: 'sv', label: '瑞典语' },
  { code: 'sl', label: '斯洛文尼亚语' },
  { code: 'uk', label: '乌克兰语' },
  { code: 'el', label: '希腊语' },
  { code: 'sk', label: '斯洛伐克语' },
  { code: 'tr', label: '土耳其语' },
  { code: 'es', label: '西班牙语' },
  { code: 'hu', label: '匈牙利语' },
  { code: 'it', label: '意大利语' },
  { code: 'id', label: '印尼语' },
];

/** 复制时使用的语言标题（用于多语言格式） */
const LANG_COPY_NAMES = {
  zh: '简体中文', 'en': '英语', 'en-US': '美式英语', ja: '日语',
  et: '爱沙尼亚语', pl: '波兰语', de: '德语', fr: '法语', nl: '荷兰语',
  lv: '拉脱维亚语', bg: '保加利亚语', da: '丹麦语', ru: '俄语', fi: '芬兰语',
  cs: '捷克语', lt: '立陶宛语', ro: '罗马尼亚语', pt: '葡萄牙语', 'pt-BR': '葡萄牙语(巴西)',
  sv: '瑞典语', sl: '斯洛文尼亚语', uk: '乌克兰语', el: '希腊语', sk: '斯洛伐克语',
  tr: '土耳其语', es: '西班牙语', hu: '匈牙利语', it: '意大利语', id: '印尼语',
};

const inputEl = document.getElementById('input');
const outputPlaceholder = document.getElementById('resultsPlaceholder');
const resultsContainer = document.getElementById('resultsContainer');
const statusEl = document.getElementById('status');
const btnTranslate = document.getElementById('btnTranslate');
const btnSpeakSource = document.getElementById('btnSpeakSource');
const btnBackTranslate = document.getElementById('btnBackTranslate');
const btnCopy = document.getElementById('btnCopy');
const btnClear = document.getElementById('btnClear');
const btnSettings = document.getElementById('btnSettings');
const sourceLangEl = document.getElementById('sourceLang');
const targetLangEl = document.getElementById('targetLang');
const btnSwapLang = document.getElementById('btnSwapLang');
const toolbarLogoEl = document.getElementById('toolbarLogo');

const settingsOverlay = document.getElementById('settingsOverlay');
const btnSettingsClose = document.getElementById('btnSettingsClose');
const btnSettingsCancel = document.getElementById('btnSettingsCancel');
const btnSettingsSave = document.getElementById('btnSettingsSave');
const btnSettingsQuit = document.getElementById('btnSettingsQuit');
const btnSettingsAbout = document.getElementById('btnSettingsAbout');
const settingsProvidersList = document.getElementById('settingsProvidersList');
const btnAddProvider = document.getElementById('btnAddProvider');
const providerCardTemplate = document.getElementById('providerCardTemplate');
const settingsSourceLanguage = document.getElementById('settingsSourceLanguage');
const settingsTargetLanguages = document.getElementById('settingsTargetLanguages');
const settingsTemperature = document.getElementById('settingsTemperature');
const settingsTemperatureField = document.getElementById('settingsTemperatureField');
const settingsTimeout = document.getElementById('settingsTimeout');
const settingsMaxInputChars = document.getElementById('settingsMaxInputChars');
const settingsAutoCopy = document.getElementById('settingsAutoCopy');
const settingsClearOnOpen = document.getElementById('settingsClearOnOpen');

const historyOverlay = document.getElementById('historyOverlay');
const btnHistory = document.getElementById('btnHistory');
const btnHistoryClose = document.getElementById('btnHistoryClose');
const historySearch = document.getElementById('historySearch');
const btnHistoryClear = document.getElementById('btnHistoryClear');
const historyList = document.getElementById('historyList');

let loading = false;
let maxInputChars = 5000;
let sourceLanguage = 'zh';
let targetLanguages = ['en'];
let lastMultiResult = null;
let autoCopy = false;

/** 预设服务商：Base URL 与默认模型。custom 为自定义，不锁定 Base URL。 */
const PROVIDER_PRESETS = {
  kimi: { label: 'Kimi', baseURL: 'https://api.moonshot.cn/v1', model: 'kimi-k2.6' },
  deepseek: { label: 'DeepSeek', baseURL: 'https://api.deepseek.com', model: 'deepseek-chat' },
  openai: { label: 'OpenAI', baseURL: 'https://api.openai.com', model: 'gpt-4o-mini' },
};

/** 服务商类型 -> 展示名 */
function providerLabel(type) {
  return (PROVIDER_PRESETS[type] && PROVIDER_PRESETS[type].label) || (type === 'custom' ? '自定义' : (type || '服务商'));
}

/** 根据 Base URL 推断服务商类型（与后端 ProviderTypeFromBaseURL 保持一致） */
function detectProviderFromBaseURL(baseURL) {
  const u = String(baseURL || '').toLowerCase();
  if (u.includes('moonshot')) return 'kimi';
  if (u.includes('deepseek')) return 'deepseek';
  if (u.includes('openai')) return 'openai';
  return 'custom';
}

/** kimi-k2 / deepseek 系列使用云端默认温度，不发送 temperature */
function modelUsesCloudDefaultTemp(name) {
  const m = String(name || '').toLowerCase().trim();
  return m.startsWith('kimi-k2') || m.startsWith('deepseek');
}

/** 逗号分隔字符串 -> 去空模型数组 */
function parseModels(str) {
  return String(str || '').split(',').map((s) => s.trim()).filter(Boolean);
}

/** 无翻译/清空时 status 循环展示的文案 */
const STATUS_IDLE_MESSAGES = [
  '自豪的使用开源技术栈制作', 
  '基于 Golang 1.26 & Wails v3.0.0-alpha.97 构建',
  '你今日的克制与笨拙，正在悄悄长成锋芒。',
  '雨落在身上是冷的，落在心里就成了润泽。',
  '人生最难的不是翻山，是不把自己交给灰尘。',
  '走过无人喝彩的路，才配得上掌声的回响。',
  '把遗憾写成注脚，把勇敢写成标题。',
  '风把路吹远了些，我把脚步走坚定。',
  '低谷不是尽头，是让光学会更亮的地方。',
  '不必等万事俱备，先把自己点燃。',
  '慢一点没关系，别停下就好。',

  // 新增 50 句
  '把每一次开始都当作重生。',
  '你可以疲惫，但不要投降。',
  '把难熬的日子，熬成更好的自己。',
  '做不到完美也没关系，先做到继续。',
  '心里有火，脚下就有路。',
  '你所坚持的，终会反过来拥抱你。',
  '别急着证明什么，时间会替你说话。',
  '把今天走好，明天会来接应你。',
  '越是黑暗，越要练习发光。',
  '别怕慢，怕的是原地打转。',
  '每一次不放弃，都是在给未来铺垫。',
  '把焦虑交给行动，把结果交给时间。',
  '你不是没天赋，你只是还在蓄力。',
  '愿你有敢于重来的勇气，也有不再回头的决心。',
  '你走过的弯路，都会成为直路的一部分。',
  '把心安放稳，风浪就只是风浪。',
  '人一旦认真起来，命运就会让路。',
  '别让情绪决定方向，让目标决定步伐。',
  '你能扛住的，都会变成你的底气。',
  '最好的逆袭，是把自己活成答案。',
  '不怕失败，怕的是不再尝试。',
  '小步也算前进，持续就是胜利。',
  '在你看不见的地方，你正在变强。',
  '愿你历尽千帆，仍保留起航的热烈。',
  '把不被理解当作修炼，把被看见当作奖励。',
  '做你能做的，剩下的交给路。',
  '你要相信，努力不会白费，只会迟到。',
  '别把自己困在昨天，明天还很宽。',
  '你不必讨好世界，只要成全自己。',
  '跌倒并不可耻，可耻的是不再站起。',
  '把目标拆小，把坚持放大。',
  '你今天多走的一步，明天就少一分慌张。',
  '当你决定不退，困难就开始退。',
  '把眼泪擦干，继续把路走亮。',
  '你需要的不是灵感，是规律的练习。',
  '别等风来，先把帆升起。',
  '你不是在等待机会，你是在制造机会。',
  '别怕没人懂，你要先懂自己。',
  '把抱怨换成方法，把叹息换成计划。',
  '世界不欠你答案，你要自己写。',
  '你越自律，越接近自由。',
  '把恐惧看清，它就会缩小。',
  '不向命运低头，是你最硬的浪漫。',
  '再难也别放弃，因为你已经走了很远。',
  '你只管努力，幸运会在路上捡到你。',
  '把今天的进步记下来，明天会更勇敢。',
  '别和过去较劲，去和未来握手。',
  '你要的生活，会在你不懈的路上出现。',
  '愿你一边受伤，一边成长，一边闪光。',
  '把平凡的日子过出光，是一种本事。',
];

const STATUS_IDLE_INTERVAL_MS = 10 * 1000;      // 每条文案展示 10 秒
const STATUS_IDLE_TYPEWRITER_MS = 60;     // 打字机每字间隔（毫秒）
let statusIdleTimer = null;
let statusIdleTypewriterTimeout = null;
let statusIdleIndex = 0;
/** 仅第一次启动时为 true，用于 status 从第一条开始；清空后再次启动则随机 */
let statusIdleFirstStartup = true;

function stopStatusIdleCycle() {
  if (statusIdleTimer != null) {
    clearInterval(statusIdleTimer);
    statusIdleTimer = null;
  }
  if (statusIdleTypewriterTimeout != null) {
    clearTimeout(statusIdleTypewriterTimeout);
    statusIdleTypewriterTimeout = null;
  }
}

/** 打字机效果显示一条文案 */
function typewriterShowMessage(text, charIndex) {
  statusEl.textContent = text.slice(0, charIndex + 1);
  if (charIndex + 1 < text.length) {
    statusIdleTypewriterTimeout = setTimeout(() => {
      typewriterShowMessage(text, charIndex + 1);
    }, STATUS_IDLE_TYPEWRITER_MS);
  }
}

function startStatusIdleCycle(useRandomStart = false) {
  stopStatusIdleCycle();
  if (statusIdleFirstStartup) {
    statusIdleIndex = 0;
    statusIdleFirstStartup = false;
  } else if (useRandomStart) {
    statusIdleIndex = Math.floor(Math.random() * STATUS_IDLE_MESSAGES.length);
  } else {
    statusIdleIndex = 0;
  }
  statusEl.classList.remove('error');
  typewriterShowMessage(STATUS_IDLE_MESSAGES[statusIdleIndex], 0);
  statusIdleTimer = setInterval(() => {
    statusIdleIndex = (statusIdleIndex + 1) % STATUS_IDLE_MESSAGES.length;
    typewriterShowMessage(STATUS_IDLE_MESSAGES[statusIdleIndex], 0);
  }, STATUS_IDLE_INTERVAL_MS);
}

function setStatus(msg, isError = false, idleRandomStart = false) {
  stopStatusIdleCycle();
  statusEl.textContent = msg || '';
  statusEl.classList.toggle('error', !!isError);
  if (!msg) {
    startStatusIdleCycle(idleRandomStart);
  }
}

function setLoading(on) {
  loading = on;
  btnTranslate.disabled = on;
  btnBackTranslate.disabled = on;
  if (on) setStatus('翻译中…');
}

function getSelectedTargetLangs() {
  const opts = Array.from(targetLangEl.selectedOptions || []);
  if (opts.length) return opts.map((o) => o.value);
  return targetLanguages.length ? targetLanguages : ['en'];
}

function updateSwapLangState() {
  const selected = getSelectedTargetLangs();
  btnSwapLang.disabled = selected.length > 1;
}

/** 收集所有服务商卡片中配置的模型名（用于温度可见性判断） */
function getAllSettingsModels() {
  const out = [];
  settingsProvidersList.querySelectorAll('.provider-card').forEach((card) => {
    const modelsInput = card.querySelector('.provider-models');
    parseModels(modelsInput ? modelsInput.value : '').forEach((m) => out.push(m));
  });
  return out;
}

/** 当所有已配置模型都使用云端默认温度时，隐藏温度设置项 */
function updateTemperatureFieldVisibility() {
  if (!settingsTemperatureField) return;
  const all = getAllSettingsModels();
  const hide = all.length > 0 && all.every(modelUsesCloudDefaultTemp);
  settingsTemperatureField.classList.toggle('field-hidden', hide);
}

/** 将预设服务商应用到卡片：填充 Base URL（自定义除外则只读），可选填默认模型 */
function applyProviderToCard(card, providerKey, { fillModel = false } = {}) {
  const baseInput = card.querySelector('.provider-baseurl');
  const modelsInput = card.querySelector('.provider-models');
  const preset = PROVIDER_PRESETS[providerKey];
  if (preset) {
    baseInput.value = preset.baseURL;
    baseInput.readOnly = true;
    baseInput.classList.add('input-readonly');
    if (fillModel && modelsInput && !modelsInput.value.trim()) {
      modelsInput.value = preset.model;
    }
  } else {
    // custom：允许自由编辑 Base URL
    baseInput.readOnly = false;
    baseInput.classList.remove('input-readonly');
  }
  updateTemperatureFieldVisibility();
}

/** 渲染一张服务商卡片并挂载事件；provider 为 GetSettings 返回的对象（可空，表示新建） */
function renderProviderCard(provider) {
  const frag = providerCardTemplate.content.cloneNode(true);
  const card = frag.querySelector('.provider-card');
  const typeSel = card.querySelector('.provider-type');
  const baseInput = card.querySelector('.provider-baseurl');
  const modelsInput = card.querySelector('.provider-models');
  const enabledInput = card.querySelector('.provider-enabled-input');
  const apiKeyInput = card.querySelector('.provider-apikey');
  const apiKeyHint = card.querySelector('.provider-apikey-hint');
  const clearKeyInput = card.querySelector('.provider-clearkey');
  const removeBtn = card.querySelector('.provider-remove');

  const p = provider || {};
  card.dataset.id = p.id || '';
  const type = p.type || (p.baseURL ? detectProviderFromBaseURL(p.baseURL) : 'kimi');
  typeSel.value = type;
  enabledInput.checked = provider ? !!p.enabled : true;
  baseInput.value = p.baseURL || (PROVIDER_PRESETS[type] ? PROVIDER_PRESETS[type].baseURL : '');
  modelsInput.value = Array.isArray(p.models) ? p.models.join(', ') : (p.models || '');
  if (PROVIDER_PRESETS[type]) {
    baseInput.readOnly = true;
    baseInput.classList.add('input-readonly');
  }
  if (p.apiKeySet) {
    apiKeyHint.textContent = p.apiKeyMask ? `已设置 ${p.apiKeyMask}` : '已设置';
    apiKeyInput.placeholder = '输入新 Key 可覆盖';
  } else {
    apiKeyHint.textContent = '';
  }
  if (!provider) {
    // 新建卡片：套用默认模型
    applyProviderToCard(card, type, { fillModel: true });
  }

  typeSel.addEventListener('change', () => {
    applyProviderToCard(card, typeSel.value, { fillModel: true });
  });
  modelsInput.addEventListener('input', updateTemperatureFieldVisibility);
  removeBtn.addEventListener('click', () => {
    card.remove();
    updateTemperatureFieldVisibility();
  });

  settingsProvidersList.appendChild(card);
  return card;
}

/** 收集所有卡片为 providers[] 提交给后端 */
function collectProviders() {
  const providers = [];
  settingsProvidersList.querySelectorAll('.provider-card').forEach((card) => {
    const type = card.querySelector('.provider-type').value;
    const baseURL = card.querySelector('.provider-baseurl').value.trim();
    const modelsArr = parseModels(card.querySelector('.provider-models').value);
    const enabled = card.querySelector('.provider-enabled-input').checked;
    const setAPIKey = card.querySelector('.provider-apikey').value.trim();
    const clearAPIKey = card.querySelector('.provider-clearkey').checked;
    providers.push({
      id: card.dataset.id || '',
      type,
      baseURL,
      models: modelsArr,
      enabled,
      apiKeySet: false,
      setAPIKey: setAPIKey || undefined,
      clearAPIKey,
    });
  });
  return providers;
}

function renderLangOptions() {
  const sourceHtml = LANG_OPTIONS.map(
    (l) => `<option value="${l.code}" ${l.code === sourceLanguage ? 'selected' : ''}>${l.label}</option>`
  ).join('');
  sourceLangEl.innerHTML = sourceHtml;

  const targetHtml = LANG_OPTIONS.map(
    (l) =>
      `<option value="${l.code}" ${targetLanguages.includes(l.code) ? 'selected' : ''}>${l.label}</option>`
  ).join('');
  targetLangEl.innerHTML = targetHtml;
  updateSwapLangState();
}

function groupResultsByTargetLanguage(results) {
  const byLang = {};
  for (const r of results) {
    const lang = r.targetLanguage || r.TargetLanguage || 'en';
    if (!byLang[lang]) byLang[lang] = [];
    byLang[lang].push(r);
  }
  return byLang;
}

function escapeHtml(s) {
  if (s == null || s === '') return '';
  const div = document.createElement('div');
  div.textContent = s;
  return div.innerHTML;
}

/** 从单条结果里取出译文，兼容多种字段名与命名风格 */
function getItemOutput(item) {
  if (item == null) return '';
  return (
    item.output ??
    item.Output ??
    item.text ??
    item.Text ??
    item.translation ??
    item.Translation ??
    ''
  );
}

function renderResults(multiResult) {
  if (!multiResult || !multiResult.results || !multiResult.results.length) {
    resultsContainer.hidden = true;
    outputPlaceholder.hidden = false;
    return;
  }
  outputPlaceholder.hidden = true;
  resultsContainer.hidden = false;
  const grouped = groupResultsByTargetLanguage(multiResult.results);
  const langLabels = Object.fromEntries(LANG_OPTIONS.map((l) => [l.code, l.label]));

  resultsContainer.innerHTML = '';
  for (const [lang, items] of Object.entries(grouped)) {
    const label = langLabels[lang] || lang;
    const groupDiv = document.createElement('div');
    groupDiv.className = 'result-group';
    groupDiv.dataset.targetLang = lang;
    groupDiv.innerHTML = `<div class="result-group-title">→ ${escapeHtml(label)}</div><div class="result-group-cards"></div>`;
    const cardsWrap = groupDiv.querySelector('.result-group-cards');
    const outputsToSet = [];

    for (const item of items) {
      const err = item.error || item.Error || '';
      const out = String(getItemOutput(item) ?? '');
      const model = item.model || item.Model || '';
      const provider = item.provider || item.Provider || '';
      const latency = item.latencyMs ?? item.LatencyMs ?? 0;

      const card = document.createElement('div');
      card.className = 'result-card';
      card.dataset.model = model;
      card.dataset.provider = provider;
      card.dataset.targetLang = lang;
      card.dataset.latencyMs = String(latency);
      const headDiv = document.createElement('div');
      headDiv.className = 'result-card-head';
      const providerBadge = provider ? `<span class="result-card-provider">${escapeHtml(providerLabel(provider))}</span>` : '';
      headDiv.innerHTML = `${providerBadge}${escapeHtml(model)} ${err ? `<span class="result-error">${escapeHtml(err)}</span>` : `${latency}ms`}`;
      const backTranslateBtn = document.createElement('button');
      backTranslateBtn.type = 'button';
      backTranslateBtn.className = 'btn-backtranslate-card';
      backTranslateBtn.textContent = '回译';
      backTranslateBtn.title = '反向翻译本卡片译文并显示在下方供核对';
      if (!err) {
        backTranslateBtn.addEventListener('click', () => doBackTranslateForCard(card));
      } else {
        backTranslateBtn.disabled = true;
      }
      headDiv.appendChild(backTranslateBtn);
      const speakCardBtn = document.createElement('button');
      speakCardBtn.type = 'button';
      speakCardBtn.className = 'btn-speak-card';
      speakCardBtn.textContent = '朗读';
      speakCardBtn.title = '朗读本卡片译文';
      if (!err) {
        speakCardBtn.addEventListener('click', () => doSpeakCard(card));
      } else {
        speakCardBtn.disabled = true;
      }
      headDiv.appendChild(speakCardBtn);
      card.appendChild(headDiv);
      const ta = document.createElement('textarea');
      ta.className = 'result-card-text editable';
      ta.rows = 3;
      if (err) ta.readOnly = true;
      card.appendChild(ta);
      cardsWrap.appendChild(card);
      outputsToSet.push({ ta, out });
    }
    resultsContainer.appendChild(groupDiv);
    for (const { ta, out } of outputsToSet) {
      ta.value = out;
    }
  }
}

function getFirstEditableOutput() {
  const el = document.activeElement;
  if (el && el.classList && el.classList.contains('result-card-text') && !el.readOnly && resultsContainer.contains(el)) {
    const v = el.value.trim();
    if (v) return v;
  }
  const ta = resultsContainer.querySelector('.result-card-text.editable:not([readonly])');
  return ta ? ta.value.trim() : '';
}

/** 获取当前用于回译的译文所在卡片的 DOM 元素（用于在下方展示回译结果） */
function getFirstEditableOutputCard() {
  const el = document.activeElement;
  if (el && el.classList && el.classList.contains('result-card-text') && !el.readOnly && resultsContainer.contains(el)) {
    const card = el.closest('.result-card');
    if (card) return card;
  }
  const ta = resultsContainer.querySelector('.result-card-text.editable:not([readonly])');
  return ta ? ta.closest('.result-card') : null;
}

function getAllOutputsByTargetLang() {
  const byLang = {};
  resultsContainer.querySelectorAll('.result-group').forEach((group) => {
    const lang = group.dataset.targetLang;
    const texts = [];
    group.querySelectorAll('.result-card-text.editable').forEach((ta) => {
      if (!ta.readOnly) texts.push(ta.value.trim());
    });
    if (texts.length) byLang[lang] = texts;
  });
  return byLang;
}

/** 按目标语言收集每张卡片的模型、耗时、译文（用于完整复制模板） */
function getAllCardsByTargetLang() {
  const byLang = {};
  resultsContainer.querySelectorAll('.result-group').forEach((group) => {
    const lang = group.dataset.targetLang;
    const cards = [];
    group.querySelectorAll('.result-card').forEach((card) => {
      const ta = card.querySelector('.result-card-text.editable');
      if (!ta || ta.readOnly) return;
      const text = ta.value.trim();
      if (!text) return;
      const model = card.dataset.model || '';
      const latencyMs = card.dataset.latencyMs || '0';
      cards.push({ model, latencyMs, text });
    });
    if (cards.length) byLang[lang] = cards;
  });
  return byLang;
}

async function doTranslate() {
  const text = inputEl.value.trim();
  if (!text) {
    setStatus('请输入要翻译的内容', true);
    return;
  }
  if ([...text].length > maxInputChars) {
    setStatus(`输入超过 ${maxInputChars} 字，请缩短后重试`, true);
    return;
  }
  setLoading(true);
  try {
    const targetLangs = getSelectedTargetLangs();
    const payload = {
      text,
      sourceLanguage,
      targetLanguages: targetLangs,
    };
    const result = await Translate(payload);

    if (result == null) {
      setStatus('翻译服务无返回', true);
      return;
    }

    // 转为纯对象，避免 Wails/Proxy 等导致 result.results 或 result.results[].output 不可用
    const data = (result != null && typeof result === 'object')
      ? JSON.parse(JSON.stringify(result))
      : result;

    if (data.results != null) {
      lastMultiResult = data;
      renderResults(data);
      const total = data.results.length;
      const ok = data.results.filter((r) => !(r.error || r.Error)).length;
      setStatus(`完成 ${ok}/${total} 条，按目标语言分组展示`);
      const primaryOutput = getItemOutput(data.results[0] || null);
      if (primaryOutput && autoCopy) {
        try {
          await CopyToClipboard(primaryOutput);
          setStatus('已复制首条结果');
        } catch (_) {}
      }
      const resultModels = [];
      const seenModels = new Set();
      data.results.forEach((r) => {
        const m = r.model || r.Model;
        if (m && !seenModels.has(m)) { seenModels.add(m); resultModels.push(m); }
      });
      await AddHistory({
        input: data.input,
        output: primaryOutput,
        provider: data.results[0]?.provider || data.results[0]?.Provider || '',
        model: data.results[0]?.model || resultModels[0] || '',
        latencyMs: data.results.reduce((m, r) => Math.max(m, r.latencyMs ?? r.LatencyMs ?? 0), 0),
        createdAt: data.createdAt ?? Math.floor(Date.now() / 1000),
        sourceLanguage,
        targetLanguages: targetLangs,
        models: resultModels,
        results: data.results,
      }).catch((err) => setStatus('历史保存失败: ' + (err && (err.message || err.Error) || '未知'), true));
    } else {
      lastMultiResult = null;
      renderResults(null);
      const output = data.output ?? data.Output ?? '';
      setStatus(`耗时 ${data.latencyMs ?? data.LatencyMs ?? 0}ms`);
      await AddHistory({
        input: data.input,
        output,
        provider: data.provider ?? data.Provider ?? '',
        model: data.model,
        latencyMs: data.latencyMs ?? data.LatencyMs ?? 0,
        createdAt: data.createdAt ?? data.CreatedAt ?? Math.floor(Date.now() / 1000),
        sourceLanguage,
        targetLanguages: targetLangs,
      }).catch((err) => setStatus('历史保存失败: ' + (err && (err.message || err.Error) || '未知'), true));
      if (autoCopy && output) {
        try {
          await CopyToClipboard(output);
          setStatus('已复制');
        } catch (_) {}
      }
      const multiLike = {
        input: data.input,
        results: [
          {
            provider: data.provider ?? data.Provider ?? '',
            model: data.model,
            targetLanguage: targetLangs[0] || 'en',
            output,
            latencyMs: data.latencyMs ?? data.LatencyMs ?? 0,
          },
        ],
        createdAt: data.createdAt,
      };
      renderResults(multiLike);
      lastMultiResult = multiLike;
    }
  } catch (err) {
    const msg = err && (err.message || err.Error || String(err)) || '翻译失败';
    setStatus(msg, true);
    renderResults(null);
    lastMultiResult = null;
  } finally {
    setLoading(false);
  }
}

/** 对指定卡片执行回译，结果展示在该卡片下方。opts.skipLoading 为 true 时不控制 loading 状态（用于批量回译）。 */
async function doBackTranslateForCard(card, opts = {}) {
  const ta = card.querySelector('.result-card-text');
  if (!ta || ta.readOnly) {
    if (!opts.skipLoading) setStatus('该卡片无可回译内容', true);
    return;
  }
  const text = ta.value.trim();
  if (!text) {
    if (!opts.skipLoading) setStatus('该卡片译文为空', true);
    return;
  }
  const targetLang = card.dataset.targetLang || getSelectedTargetLangs()[0] || 'en';
  if (!opts.skipLoading) setLoading(true);
  try {
    const payload = {
      text,
      sourceLanguage: targetLang,
      targetLanguages: [sourceLanguage],
    };
    const result = await Translate(payload);
    if (result == null) {
      if (!opts.skipLoading) setStatus('回译服务无返回', true);
      return;
    }
    const data = result != null && typeof result === 'object' ? JSON.parse(JSON.stringify(result)) : result;
    let out = '';
    if (data.results != null && data.results.length > 0) {
      const first = data.results.find((r) => !(r.error || r.Error));
      out = first ? (first.output ?? first.Output ?? '') : '';
      if (!out && data.results[0]) {
        if (!opts.skipLoading) setStatus(data.results[0].error || data.results[0].Error || '回译失败', true);
        return;
      }
    } else {
      out = data.output ?? data.Output ?? '';
    }
    if (out) {
      let block = card.querySelector('.result-card-backtranslate');
      if (!block) {
        block = document.createElement('div');
        block.className = 'result-card-backtranslate';
        card.appendChild(block);
      }
      block.innerHTML = `<span class="result-card-backtranslate-label">反向翻译（供核对）</span><pre class="result-card-backtranslate-text">${escapeHtml(out)}</pre>`;
      block.hidden = false;
      if (!opts.skipLoading) setStatus('已在译文下方展示回译结果');
    } else {
      if (!opts.skipLoading) setStatus('回译失败', true);
    }
  } catch (err) {
    const msg = err && (err.message || err.Error || String(err)) || '回译失败';
    if (!opts.skipLoading) setStatus(msg, true);
  } finally {
    if (!opts.skipLoading) setLoading(false);
  }
}

/** 获取所有可回译的卡片（有可编辑且非空的译文） */
function getBackTranslateableCards() {
  const cards = resultsContainer.querySelectorAll('.result-card');
  return Array.from(cards).filter((card) => {
    const ta = card.querySelector('.result-card-text');
    return ta && !ta.readOnly && ta.value.trim().length > 0;
  });
}

async function doBackTranslate() {
  const cards = getBackTranslateableCards();
  if (cards.length === 0) {
    setStatus('请先翻译，或确保至少一张卡片有可回译的译文', true);
    return;
  }
  setLoading(true);
  try {
    for (const card of cards) {
      await doBackTranslateForCard(card, { skipLoading: true });
    }
    setStatus(`已对 ${cards.length} 张卡片完成回译`);
  } catch (_) {
    setStatus('回译过程中发生错误', true);
  } finally {
    setLoading(false);
  }
}

/** 根据当前结果生成要复制的文本（完整模板：元信息 + 各语言块含模型、耗时、译文） */
function getCopyText() {
  const byLang = getAllCardsByTargetLang();
  const langs = Object.keys(byLang).filter((lang) => byLang[lang].length > 0);
  if (langs.length === 0) return '';

  const langLabels = Object.fromEntries(LANG_OPTIONS.map((l) => [l.code, l.label]));
  const copyName = (code) => LANG_COPY_NAMES[code] || langLabels[code] || code;

  if (langs.length === 1) {
    const cards = byLang[langs[0]];
    const texts = cards.map((c) => c.text).filter(Boolean);
    return texts.join('\n\n');
  }

  const sourceLabel = copyName(sourceLanguage);
  const targetLabelList = langs.map(copyName).join('、');
  const resultList = (lastMultiResult && lastMultiResult.results) || [];
  const modelSet = [];
  const providerSet = [];
  resultList.forEach((r) => {
    const m = r.model || r.Model;
    if (m && !modelSet.includes(m)) modelSet.push(m);
    const pv = r.provider || r.Provider;
    if (pv && !providerSet.includes(pv)) providerSet.push(pv);
  });
  const modelList = modelSet.join('、') || '—';
  const providerLine = providerSet.length ? providerSet.map(providerLabel).join('、') : '—';
  const rawInput = (lastMultiResult && lastMultiResult.input) || inputEl.value.trim() || '';

  const headerLines = [
    '# 完整翻译内容',
    '',
    `- 翻译动作：${sourceLabel} => ${targetLabelList}`,
    `- 服务商：${providerLine}`,
    `- 选择模型：${modelList}`,
    '- 原始翻译内容：',
    '```text',
    rawInput,
    '```',
  ];
  const blocks = [headerLines.join('\n')];

  const sep = (index) => (index === 0 ? '---' : '----');
  langs.forEach((lang, index) => {
    const cards = byLang[lang];
    const langTitle = `${copyName(lang)}翻译结果`;
    cards.forEach((card, cardIndex) => {
      const blockLines = [
        `# ${langTitle}`,
        '',
        `- 模型名称：${card.model || '—'}`,
        `- 执行时间：${card.latencyMs}ms`,
        '- 翻译结果：',
        '```text',
        card.text,
        '```',
      ];
      blocks.push(blockLines.join('\n'));
    });
    if (index < langs.length - 1) blocks.push(sep(index));
  });

  return blocks.join('\n\n');
}

async function doCopy() {
  const text = getCopyText();
  if (!text) {
    setStatus('请先翻译或选中要复制的译文', true);
    return;
  }
  try {
    await CopyToClipboard(text);
    setStatus('已复制');
  } catch (err) {
    setStatus('复制失败', true);
  }
}

function doClear() {
  inputEl.value = '';
  resultsContainer.innerHTML = '';
  resultsContainer.hidden = true;
  outputPlaceholder.hidden = false;
  lastMultiResult = null;
  setStatus('', false, true); // 清空后 status 从随机一条开始
  inputEl.focus();
}

async function doSpeakSource() {
  const text = inputEl.value.trim();
  if (!text) {
    setStatus('请输入要朗读的内容', true);
    return;
  }
  try {
    await Speak({ text, language: sourceLanguage });
    setStatus('朗读中…');
  } catch (err) {
    const msg = (err && (err.message || err.Error)) || '朗读失败';
    setStatus(msg, true);
  }
}

async function doSpeakCard(card) {
  const ta = card.querySelector('.result-card-text');
  if (!ta || ta.readOnly) {
    setStatus('该卡片无可朗读内容', true);
    return;
  }
  const text = ta.value.trim();
  if (!text) {
    setStatus('该卡片译文为空', true);
    return;
  }
  const targetLang = card.dataset.targetLang || 'en';
  try {
    await Speak({ text, language: targetLang });
    setStatus('朗读中…');
  } catch (err) {
    const msg = (err && (err.message || err.Error)) || '朗读失败';
    setStatus(msg, true);
  }
}

function doHide() {
  HideWindow();
}

function openSettings() {
  settingsOverlay.hidden = false;
  GetSettings()
    .then((s) => {
      settingsProvidersList.innerHTML = '';
      const providers = Array.isArray(s.providers) ? s.providers : [];
      if (providers.length === 0) {
        renderProviderCard(null);
      } else {
        providers.forEach((p) => renderProviderCard(p));
      }
      settingsSourceLanguage.value = s.sourceLanguage || 'zh';
      settingsTargetLanguages.value = Array.isArray(s.targetLanguages) ? s.targetLanguages.join(', ') : (s.targetLanguages || 'en');
      settingsTemperature.value = s.temperature !== undefined && s.temperature !== null ? String(s.temperature) : '0.2';
      settingsTimeout.value = String(s.timeoutSeconds || 30);
      settingsMaxInputChars.value = String(s.maxInputChars || 5000);
      settingsAutoCopy.checked = !!s.autoCopy;
      settingsClearOnOpen.checked = !!s.clearOnOpen;
      updateTemperatureFieldVisibility();
    })
    .catch(() => {});
}

function closeSettings() {
  settingsOverlay.hidden = true;
}

async function saveSettings() {
  const providers = collectProviders();
  if (providers.length === 0) {
    setStatus('请至少添加一个服务商', true);
    return;
  }
  const invalid = providers.find((p) => !p.baseURL || p.models.length === 0);
  if (invalid) {
    setStatus('每个服务商都需要填写 Base URL 和至少一个模型', true);
    return;
  }
  const targetStr = settingsTargetLanguages.value.trim();
  const targetArr = targetStr ? targetStr.split(',').map((s) => s.trim()).filter(Boolean) : undefined;
  const tempNum = parseFloat(settingsTemperature.value);
  const temperature = Number.isNaN(tempNum) ? 0.2 : tempNum;
  try {
    await SaveSettings({
      providers,
      sourceLanguage: settingsSourceLanguage.value.trim() || 'zh',
      targetLanguages: targetArr,
      timeoutSeconds: parseInt(settingsTimeout.value, 10) || 30,
      temperature,
      maxInputChars: parseInt(settingsMaxInputChars.value, 10) || 5000,
      autoCopy: settingsAutoCopy.checked,
      clearOnOpen: settingsClearOnOpen.checked,
    });
    sourceLanguage = settingsSourceLanguage.value.trim() || 'zh';
    targetLanguages = targetArr && targetArr.length ? targetArr : ['en'];
    autoCopy = settingsAutoCopy.checked;
    renderLangOptions();
    setStatus('设置已保存');
    closeSettings();
  } catch (err) {
    const msg = err && (err.message || err.Error || String(err)) || '保存失败';
    setStatus(msg, true);
  }
}

function swapDirection() {
  const t = getSelectedTargetLangs();
  if (t.length === 0) return;
  const newSource = t[0];
  const newTargets = sourceLanguage ? [sourceLanguage] : ['zh'];
  sourceLanguage = newSource;
  targetLanguages = newTargets;
  renderLangOptions();
  setStatus('已交换源/目标语言');
}

btnTranslate.addEventListener('click', doTranslate);
btnSpeakSource.addEventListener('click', doSpeakSource);
btnBackTranslate.addEventListener('click', doBackTranslate);
btnCopy.addEventListener('click', doCopy);
btnClear.addEventListener('click', doClear);
btnSwapLang.addEventListener('click', swapDirection);

// Logo 彩蛋：每次点击随机切换悬停光标，点击满 5 次后 logo 剧烈晃动，再点一次恢复 pointer
const LOGO_EGG_CURSORS = [
  'grab', 'grabbing', 'crosshair', 'move', 'wait', 'help', 'progress',
  'copy', 'cell', 'context-menu', 'not-allowed', 'zoom-in', 'zoom-out',
  'alias', 'nesw-resize', 'nwse-resize', 'col-resize', 'row-resize',
  'text', 'vertical-text', 'all-scroll', 'none',
];
let logoEggClickCount = 0;
if (toolbarLogoEl) {
  toolbarLogoEl.addEventListener('click', () => {
    logoEggClickCount += 1;
    if (logoEggClickCount > 5) {
      toolbarLogoEl.style.cursor = 'pointer';
      logoEggClickCount = 0;
      return;
    }
    if (logoEggClickCount === 5) {
      const logoImg = toolbarLogoEl.querySelector('.toolbar-logo-img');
      if (logoImg) {
        logoImg.classList.remove('logo-shake-egg');
        logoImg.offsetHeight;
        logoImg.classList.add('logo-shake-egg');
        const onEnd = () => {
          logoImg.classList.remove('logo-shake-egg');
          logoImg.removeEventListener('animationend', onEnd);
        };
        logoImg.addEventListener('animationend', onEnd);
      }
    }
    const idx = Math.floor(Math.random() * LOGO_EGG_CURSORS.length);
    toolbarLogoEl.style.cursor = LOGO_EGG_CURSORS[idx];
  });
}

targetLangEl.addEventListener('keydown', (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
    e.preventDefault();
    const opts = targetLangEl.options;
    for (let i = 0; i < opts.length; i++) opts[i].selected = true;
    targetLanguages = LANG_OPTIONS.map((l) => l.code);
    updateSwapLangState();
  }
});
targetLangEl.addEventListener('change', updateSwapLangState);
btnSettings.addEventListener('click', openSettings);
btnSettingsClose.addEventListener('click', closeSettings);
btnSettingsCancel.addEventListener('click', closeSettings);
btnSettingsSave.addEventListener('click', saveSettings);
btnAddProvider.addEventListener('click', () => {
  renderProviderCard(null);
});
btnSettingsQuit.addEventListener('click', () => {
  Application.Quit().catch(() => {});
});
if (btnSettingsAbout) {
  btnSettingsAbout.addEventListener('click', () => {
    ShowSplash().catch((err) => {
      const msg = (err && (err.message || err.Error)) || '无法显示关于页';
      setStatus(msg, true);
    });
  });
}

settingsOverlay.addEventListener('click', (e) => {
  if (e.target === settingsOverlay) closeSettings();
});

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    if (!settingsOverlay.hidden) {
      closeSettings();
    } else if (!historyOverlay.hidden) {
      closeHistory();
    } else {
      doHide();
    }
    return;
  }
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault();
    doTranslate();
  }
  if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key === 'C') {
    e.preventDefault();
    doCopy();
  }
});

inputEl.focus();

GetSettings().then((s) => {
  if (s && s.maxInputChars > 0) maxInputChars = s.maxInputChars;
  if (s && s.clearOnOpen) doClear();
  if (s && s.sourceLanguage) sourceLanguage = s.sourceLanguage;
  if (s && s.targetLanguages && s.targetLanguages.length) targetLanguages = s.targetLanguages;
  if (s) autoCopy = !!s.autoCopy;
  renderLangOptions();
}).catch(() => {});

startStatusIdleCycle();

let historySearchTimer = null;
function openHistory() {
  historyOverlay.hidden = false;
  historySearch.value = '';
  setStatus('');
  refreshHistoryList('');
}
function closeHistory() {
  historyOverlay.hidden = true;
}
function refreshHistoryList(query) {
  historyList.innerHTML = '';
  GetHistory(query || historySearch.value.trim())
    .then((raw) => {
      const entries = Array.isArray(raw) ? raw : (raw && raw.entries ? raw.entries : []);
      historyList.innerHTML = '';
      entries.forEach((entry) => {
        const input = String(entry.input ?? entry.Input ?? entry.text ?? entry.Text ?? '').trim();
        const output = String(entry.output ?? entry.Output ?? entry.translation ?? entry.Translation ?? '').trim();
        if (!input && !output && !(entry.results && entry.results.length)) return;
        const srcLang = entry.sourceLanguage ?? entry.SourceLanguage ?? '';
        const tgtLangs = entry.targetLanguages ?? entry.TargetLanguages;
        const tgtStr = Array.isArray(tgtLangs) ? tgtLangs.join(', ') : (tgtLangs || '');
        const langLabel = srcLang && tgtStr ? ` (${srcLang} → ${tgtStr})` : '';
        const models = entry.models ?? (entry.model ? [entry.model] : []);
        const latencyMs = entry.latencyMs ?? entry.LatencyMs ?? 0;
        const metaParts = [];
        if (models.length) metaParts.push(models.join(', '));
        if (latencyMs > 0) metaParts.push(`${latencyMs}ms`);
        const metaLabel = metaParts.length ? ` · ${metaParts.join(' · ')}` : '';
        const li = document.createElement('li');
        const truncate = (str, n) => (str.length <= n ? str : str.slice(0, n) + '…');
        const displayOutput = output || (entry.results && entry.results[0] && (entry.results[0].output ?? entry.results[0].Output)) || '';
        const inDiv = document.createElement('div');
        inDiv.className = 'history-input';
        inDiv.textContent = truncate(String(input), 40) + langLabel + metaLabel;
        const outDiv = document.createElement('div');
        outDiv.className = 'history-output';
        outDiv.textContent = truncate(String(displayOutput), 60);
        li.appendChild(inDiv);
        li.appendChild(outDiv);
        li.addEventListener('click', () => {
          inputEl.value = input;
          sourceLanguage = srcLang || sourceLanguage;
          targetLanguages = Array.isArray(tgtLangs) && tgtLangs.length ? tgtLangs : targetLanguages;
          renderLangOptions();
          const results = entry.results ?? entry.Results;
          if (results && results.length > 0) {
            lastMultiResult = {
              input,
              results,
              createdAt: entry.createdAt ?? entry.CreatedAt ?? Math.floor(Date.now() / 1000),
            };
            renderResults(lastMultiResult);
          } else {
            lastMultiResult = {
              input,
              results: [
                {
                  model: entry.model ?? entry.Model ?? '',
                  targetLanguage: (Array.isArray(tgtLangs) && tgtLangs[0]) || 'en',
                  output,
                  latencyMs: latencyMs || 0,
                },
              ],
              createdAt: entry.createdAt ?? entry.CreatedAt ?? Math.floor(Date.now() / 1000),
            };
            renderResults(lastMultiResult);
          }
          closeHistory();
        });
        historyList.appendChild(li);
      });
    })
    .catch((err) => {
      historyList.innerHTML = '';
      setStatus('加载历史失败: ' + (err && (err.message || err.Error) || '未知错误'), true);
    });
}
btnHistory.addEventListener('click', openHistory);
btnHistoryClose.addEventListener('click', closeHistory);
historySearch.addEventListener('input', () => {
  clearTimeout(historySearchTimer);
  historySearchTimer = setTimeout(() => refreshHistoryList(historySearch.value.trim()), 200);
});
btnHistoryClear.addEventListener('click', () => {
  ClearHistory().then(() => refreshHistoryList('')).catch(() => {});
});
historyOverlay.addEventListener('click', (e) => {
  if (e.target === historyOverlay) closeHistory();
});
