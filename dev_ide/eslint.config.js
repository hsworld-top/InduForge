import js from '@eslint/js'
import vue from 'eslint-plugin-vue'
import prettier from '@vue/eslint-config-prettier'

export default [
  js.configs.recommended,
  ...vue.configs['flat/essential'],
  prettier,
  {
    name: 'projectide-frontend',
    files: ['**/*.{js,mjs,cjs,vue}'],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: {
        // 浏览器全局变量
        document: 'readonly',
        window: 'readonly',
        localStorage: 'readonly',
        sessionStorage: 'readonly',
        console: 'readonly',
        // Node.js 全局变量
        URL: 'readonly',
        import: 'readonly',
        // Element Plus 组件
        ElMessage: 'readonly',
        ElMessageBox: 'readonly',
      },
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
      },
    },
    rules: {
      'vue/multi-word-component-names': 'off',
      'no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      'prefer-const': 'error',
      'no-var': 'error',
      'no-useless-catch': 'off', // 在某些情况下 try/catch 是必要的
    },
  },
  {
    ignores: ['dist/**', 'node_modules/**', '.vite/**'],
  },
]
