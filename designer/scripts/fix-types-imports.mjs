import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve('src')
const TARGET_FILE_RE = /\.(?:js|vue|ts)$/
const PATH_REPLACEMENTS = [
  ['./types.js', './types'],
  ['../document/types.js', '../document/types'],
  ['../../document/types.js', '../../document/types'],
  ['"./document/types.js"', '"./document/types"'],
  ["'./document/types.js'", "'./document/types'"],
  ["'../editor-core/types.js'", "'../editor-core/document/types'"],
  ['"../editor-core/types.js"', '"../editor-core/document/types"'],
]

function walk(dir) {
  for (const name of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, name.name)
    if (name.isDirectory()) {
      walk(p)
    } else if (TARGET_FILE_RE.test(name.name)) {
      let c = fs.readFileSync(p, 'utf8')
      const o = c
      for (const [from, to] of PATH_REPLACEMENTS) {
        c = c.replaceAll(from, to)
      }
      if (c !== o) fs.writeFileSync(p, c)
    }
  }
}

walk(root)
