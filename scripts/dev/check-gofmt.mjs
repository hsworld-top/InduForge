import { spawnSync } from 'node:child_process'

const mode = process.argv[2] ?? '--check'
if (!['--check', '--write'].includes(mode)) {
  throw new Error('仅支持 --check 或 --write')
}

const filesResult = spawnSync('git', ['ls-files', '--', '*.go'], {
  cwd: process.cwd(),
  encoding: 'utf8',
})
if (filesResult.error) {
  throw filesResult.error
}
if (filesResult.status !== 0) {
  process.stderr.write(filesResult.stderr)
  process.exit(filesResult.status ?? 1)
}

const goFiles = filesResult.stdout.split(/\r?\n/).filter(Boolean)
const gofmtArgs = mode === '--write' ? ['-w', ...goFiles] : ['-l', ...goFiles]
const gofmtResult = spawnSync('gofmt', gofmtArgs, {
  cwd: process.cwd(),
  encoding: 'utf8',
})
if (gofmtResult.error) {
  throw gofmtResult.error
}
if (gofmtResult.status !== 0) {
  process.stderr.write(gofmtResult.stderr)
  process.exit(gofmtResult.status ?? 1)
}

if (mode === '--check' && gofmtResult.stdout.trim()) {
  console.error(`以下 Go 文件未格式化:\n${gofmtResult.stdout.trim()}`)
  process.exit(1)
}
