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

import { defineConfig } from 'vite';
import { readFileSync } from 'fs';
import { fileURLToPath } from 'url';
import path from 'path';
import wails from '@wailsio/runtime/plugins/vite';


const __dirname = path.dirname(fileURLToPath(import.meta.url));

// 多页面配置
const multiPageEntries = JSON.parse(
  readFileSync(path.resolve(__dirname, 'scripts/multiPages.json'), 'utf-8')
);

const npm_config_page = process.env.npm_config_page || '';

/** 子入口 = 页面目录。单页构建时 root 已设为页面目录，入口用绝对路径以便解析，产出仍会落在 dist/<page>/ 根下。 */
function getEnterPages() {
  if (npm_config_page) {
    return path.resolve(__dirname, `src/projects/${npm_config_page}/index.html`);
  }
  return multiPageEntries.reduce((acc, item) => {
    acc[item.page] = path.resolve(__dirname, `src/projects/${item.page}/index.html`);
    return acc;
  }, {});
}

const isSinglePageBuild = true || Boolean(npm_config_page);

// 单页构建时 root 设为当前页面目录，使产出的 index.html 直接落在 outDir 根（即 dist/<page>/index.html）
const buildRoot = npm_config_page
  ? path.resolve(__dirname, `src/projects/${npm_config_page}`)
  : path.resolve(__dirname, './src/projects/');

// https://vitejs.dev/config/
export default defineConfig({
  root: buildRoot,
  base: isSinglePageBuild ? './' : '/',
  publicDir: path.resolve(__dirname, 'public'),
  envDir: path.resolve(__dirname),
  plugins: [wails(path.resolve(__dirname, 'bindings'))],
  build: {
    outDir: path.resolve(
      __dirname,
      isSinglePageBuild ? `dist/${npm_config_page}` : 'dist'
    ),
    emptyOutDir: true,
    rollupOptions: {
      input: getEnterPages(),
      output: {
        assetFileNames: 'assets/[name]-[hash][extname]',
        chunkFileNames: 'assets/[name]-[hash].js',
        entryFileNames: 'assets/[name]-[hash].js',
        compact: true,
      },
    },
    target: 'esnext',
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@bindings': path.resolve(__dirname, 'bindings'),
    },
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    // open: npm_config_page ? `/` : true,
  },
});
