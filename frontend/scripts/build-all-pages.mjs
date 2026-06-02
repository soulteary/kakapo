#!/usr/bin/env bun
/**
 * 按 multiPages.json 对每个页面单独执行 vite build，并将 public/ 复制到每个页面产物目录根。
 * 产物：每个页面在独立目录 dist/<page>/，且该目录内包含完整资源（index.html + assets/ + public 静态文件）。
 * 用法：bun build-all-pages.mjs [dev]
 *   - 无参数或 production：--mode production
 *   - dev：--mode development --minify false（与 wails3 dev 使用的 build:dev 一致，但产出与 bun run build 相同的目录结构）
 */
import { readFileSync, cpSync, existsSync, readdirSync } from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { spawnSync } from 'child_process';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, '..');
const publicDir = path.join(root, 'public');
const multiPages = JSON.parse(
  readFileSync(path.resolve(root, 'scripts/multiPages.json'), 'utf-8')
);

const isDev = process.argv[2] === 'dev' || process.argv[2] === 'development';
const viteArgs = isDev
  ? ['vite', 'build', '--mode', 'development', '--minify', 'false']
  : ['vite', 'build', '--mode', 'production'];

/** 将 public 目录下的所有内容复制到 targetDir（不保留 public 这一层目录）。 */
function copyPublicTo(targetDir) {
  if (!existsSync(publicDir)) return;
  for (const entry of readdirSync(publicDir, { withFileTypes: true })) {
    const src = path.join(publicDir, entry.name);
    const dest = path.join(targetDir, entry.name);
    cpSync(src, dest, { recursive: true });
  }
}

for (const { page } of multiPages) {
  console.log(`\n[build] page: ${page}${isDev ? ' (dev)' : ''}`);
  const r = spawnSync('bunx', viteArgs, {
    cwd: root,
    env: { ...process.env, npm_config_page: page },
    stdio: 'inherit',
  });
  if (r.status !== 0) process.exit(r.status ?? 1);

  const pageOutDir = path.join(root, 'dist', page);
  copyPublicTo(pageOutDir);
  console.log(`[build] copied public/ → dist/${page}/`);
}

console.log('\n[build] all pages done.');
