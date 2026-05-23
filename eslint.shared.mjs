import js from '@eslint/js'
import tsEslintPlugin from '@typescript-eslint/eslint-plugin'
import tsParser from '@typescript-eslint/parser'
import prettier from '@vue/eslint-config-prettier'
import vue from 'eslint-plugin-vue'

export const frontendGlobals = {
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
  'prefer-const': 'error',
  'no-var': 'error',
  'no-useless-catch': 'off',
}

export function defineVueFrontendConfig({ name, ignores = [], rules = [] } = {}) {
  const jsRules = {
    ...frontendRules,
    'no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
  }

  const typedRules = {
    ...frontendRules,
    'no-unused-vars': 'off',
    '@typescript-eslint/no-unused-vars': [
      'error',
      { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
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
