import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const repoRoot = path.resolve(__dirname, '..', '..')

const targets = [
  {
    fileName: '.claudeignore',
    title: 'Claude Agent 忽略文件',
    extra: '用于生态兼容；Claude Code 实际权限仍以 .claude/settings.json 为准。',
  },
  {
    fileName: '.codexignore',
    title: 'Codex Agent 忽略文件',
    extra: '用于减少 Codex 索引与上下文检索体积。',
  },
  {
    fileName: '.cursorignore',
    title: 'Cursor Agent 忽略文件',
    extra: '用于减少 Cursor 索引与上下文检索体积。',
  },
]

/**
 * 读取共享忽略规则正文。
 *
 * @returns {Promise<string>} 共享规则文本
 */
async function readSharedRules() {
  const sharedPath = path.join(__dirname, 'ignore.shared.txt')
  return readFile(sharedPath, 'utf8')
}

/**
 * 生成单个工具的忽略文件内容。
 *
 * @param {{ title: string, extra: string }} target 目标文件配置
 * @param {string} sharedRules 共享规则正文
 * @returns {string} 目标文件内容
 */
function buildIgnoreContent(target, sharedRules) {
  return [
    `# ${target.title}`,
    '# 由 scripts/ai-rules/sync-ignore-files.mjs 自动生成，请勿手改。',
    `# ${target.extra}`,
    '',
    sharedRules.trim(),
    '',
  ].join('\n')
}

/**
 * 将共享忽略规则同步到所有目标文件。
 *
 * @returns {Promise<void>}
 */
async function syncIgnoreFiles() {
  const sharedRules = await readSharedRules()
  await mkdir(repoRoot, { recursive: true })

  for (const target of targets) {
    const outputPath = path.join(repoRoot, target.fileName)
    const content = buildIgnoreContent(target, sharedRules)
    await writeFile(outputPath, content, 'utf8')
  }
}

await syncIgnoreFiles()
