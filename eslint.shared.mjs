import js from '@eslint/js'
import tsEslintPlugin from '@typescript-eslint/eslint-plugin'
import tsParser from '@typescript-eslint/parser'
import prettier from '@vue/eslint-config-prettier'
import vue from 'eslint-plugin-vue'
import globals from 'globals'

export const frontendGlobals = {
  ...globals.browser,
  ...globals.es2021,
  ...globals.node,
  document: 'readonly',
  window: 'readonly',
  localStorage: 'readonly',
  sessionStorage: 'readonly',
  console: 'readonly',
  setInterval: 'readonly',
  clearInterval: 'readonly',
  setTimeout: 'readonly',
  clearTimeout: 'readonly',
  URL: 'readonly',
  import: 'readonly',
  process: 'readonly',
  ImportMetaEnv: 'readonly',
  ElementCreationOptions: 'readonly',
  ElMessage: 'readonly',
  ElMessageBox: 'readonly',
}

export const frontendIgnores = [
  'dist/**',
  'node_modules/**',
  '.vite/**',
  'coverage/**',
  'eslint-report.json',
]

export const frontendRules = {
  'vue/multi-word-component-names': 'off',
  'vue/no-reserved-component-names': 'warn',
  'vue/no-unused-components': 'warn',
  'vue/no-mutating-props': 'warn',
  // 前端项目混有 Vue 模板、TS 类型声明和构建脚本，未定义符号先由 typecheck 兜底。
  'no-undef': 'warn',
  'prefer-const': 'error',
  'no-var': 'error',
  'no-useless-catch': 'off',
}

export function defineVueFrontendConfig({ name, ignores = [], rules = [] } = {}) {
  const jsRules = {
    ...frontendRules,
    // 先把存量未使用项作为 warning 暴露出来，避免规则统一时一次性修改大量业务代码。
    'no-unused-vars': [
      'warn',
      {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        caughtErrorsIgnorePattern: '^_',
      },
    ],
  }

  const typedRules = {
    ...frontendRules,
    'no-unused-vars': 'off',
    '@typescript-eslint/no-unused-vars': [
      'warn',
      {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        caughtErrorsIgnorePattern: '^_',
      },
    ],
  }

  return [
    js.configs.recommended,
    ...vue.configs['flat/essential'],
    prettier,
    {
      name: `${name || 'frontend'}-js`,
      files: ['**/*.{js,mjs,cjs}'],
      languageOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        globals: frontendGlobals,
      },
      rules: jsRules,
    },
    {
      name: `${name || 'frontend'}-ts`,
      files: ['**/*.{ts,mts,cts,tsx}', '**/*.d.ts'],
      plugins: {
        '@typescript-eslint': tsEslintPlugin,
      },
      languageOptions: {
        parser: tsParser,
        ecmaVersion: 'latest',
        sourceType: 'module',
        globals: frontendGlobals,
      },
      rules: typedRules,
    },
    {
      name: `${name || 'frontend'}-vue`,
      files: ['**/*.vue'],
      plugins: {
        '@typescript-eslint': tsEslintPlugin,
      },
      languageOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        globals: frontendGlobals,
        parserOptions: {
          parser: tsParser,
          ecmaVersion: 'latest',
          sourceType: 'module',
        },
      },
      rules: typedRules,
    },
    ...rules,
    {
      ignores: [...frontendIgnores, ...ignores],
    },
  ]
}
