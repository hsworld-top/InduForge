import js from '@eslint/js'
import vue from 'eslint-plugin-vue'
import prettier from '@vue/eslint-config-prettier'
import tsParser from '@typescript-eslint/parser'

const sharedGlobals = {
  // 浏览器全局变量
  document: 'readonly',
  window: 'readonly',
  localStorage: 'readonly',
  sessionStorage: 'readonly',
  console: 'readonly',
  // Node.js 全局变量
  URL: 'readonly',
  import: 'readonly',
  process: 'readonly',
  // Element Plus 组件
  ElMessage: 'readonly',
  ElMessageBox: 'readonly',
}

const sharedRules = {
  'vue/multi-word-component-names': 'off',
  'no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
  'prefer-const': 'error',
  'no-var': 'error',
  'no-useless-catch': 'off', // 在某些情况下 try/catch 是必要的
}

export default [
  js.configs.recommended,
  ...vue.configs['flat/essential'],
  prettier,
  {
    name: 'projectide-frontend-js',
    files: ['**/*.{js,mjs,cjs}'],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: sharedGlobals,
    },
    rules: sharedRules,
  },
  {
    name: 'projectide-frontend-ts',
    files: ['**/*.{ts,mts,cts,tsx}', '**/*.d.ts'],
    languageOptions: {
      parser: tsParser,
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: sharedGlobals,
    },
    rules: sharedRules,
  },
  {
    name: 'projectide-frontend-vue',
    files: ['**/*.vue'],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: sharedGlobals,
      parserOptions: {
        parser: tsParser,
        ecmaVersion: 'latest',
        sourceType: 'module',
      },
    },
    rules: sharedRules,
  },
  {
    ignores: ['dist/**', 'node_modules/**', '.vite/**'],
  },
]
