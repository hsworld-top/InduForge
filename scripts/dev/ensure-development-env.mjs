import { copyFile, stat } from 'node:fs/promises'
import { constants } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(scriptDir, '../..')
const envFile = path.join(repoRoot, '.env')
const developmentTemplate = path.join(repoRoot, '.env.development.example')

async function exists(file) {
  try {
    await stat(file)
    return true
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return false
    }
    throw error
  }
}

if (await exists(envFile)) {
  console.log('使用现有根目录 .env。')
} else {
  await stat(developmentTemplate)
  await copyFile(developmentTemplate, envFile, constants.COPYFILE_EXCL)
  console.log('根目录 .env 不存在，已从 .env.development.example 生成。')
}
