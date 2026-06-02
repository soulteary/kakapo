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

import { Events } from '@wailsio/runtime';

// 启动页（splash）默认展示加载文案；点击"关于"时由主进程发来事件，切换为"关于"视图（项目地址 + 介绍文章）。
const statusEl = document.getElementById('status');
const loadingEl = document.getElementById('loading');
const aboutEl = document.getElementById('about');

const MESSAGES = [
  'Preparing translation workspace…',
  'Loading providers & models…',
  'Warming up the kakapo…',
  'Almost ready…',
];

let rotateTimer = null;
if (statusEl) {
  let i = 0;
  rotateTimer = setInterval(() => {
    i = (i + 1) % MESSAGES.length;
    statusEl.textContent = MESSAGES[i];
  }, 1500);
}

function showAbout() {
  if (rotateTimer != null) {
    clearInterval(rotateTimer);
    rotateTimer = null;
  }
  if (loadingEl) loadingEl.hidden = true;
  if (aboutEl) aboutEl.hidden = false;
}

// 主进程在显示"关于"时发出该事件（见 main.go 的 EventSplashAbout）。
try {
  Events.On('splash:about', showAbout);
} catch (_) {
  // 非 Wails 环境（如浏览器预览）忽略。
}
