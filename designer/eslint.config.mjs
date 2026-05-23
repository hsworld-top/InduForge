import antfu from '@antfu/eslint-config'
import vueConfigPrettier from '@vue/eslint-config-prettier'
import { frontendGlobals, frontendIgnores } from '../eslint.shared.mjs'

export default antfu(
  {
    vue: true,
    typescript: true,
    lessOpinionated: true,
    stylistic: {
      indent: 2,
      quotes: 'single',
      semi: false,
    },
    ignores: [...frontendIgnores, 'public/**'],
    gitignore: true,
    jsonc: true,
    yaml: false,
  },

  {
    languageOptions: {
      globals: frontendGlobals,
    },
  },

  {
    rules: {
      'vue/multi-word-component-names': 'off',
      'prefer-const': 'error',
      'no-var': 'error',
      'no-useless-catch': 'off',
      // Migration baseline: keep gate green; tighten incrementally.
      'unused-imports/no-unused-vars': [
        'warn',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],
      'e18e/prefer-static-regex': 'warn',
      'ts/no-unused-expressions': 'warn',
      'no-unused-expressions': 'warn',
      'ts/no-use-before-define': 'warn',
      'no-use-before-define': 'warn',
      'no-new-func': 'warn',
      'regexp/no-unused-capturing-group': 'warn',
      'ts/ban-ts-comment': 'warn',
      'no-useless-return': 'warn',
      'no-restricted-globals': 'warn',
      'no-cond-assign': 'warn',
      'unicorn/prefer-number-properties': 'warn',
      'no-console': 'warn',
      'no-control-regex': 'warn',
      'no-template-curly-in-string': 'warn',
      'vue/no-use-v-if-with-v-for': 'warn',
      'regexp/no-super-linear-backtracking': 'warn',
      'vue/no-unused-vars': 'warn',
      'no-unused-vars': 'warn',
      'no-new': 'warn',
      'no-self-assign': 'warn',
      'no-useless-call': 'warn',
      'jsdoc/require-returns-description': 'off',
      'jsdoc/require-param-description': 'off',
      'jsdoc/require-description': 'off',
    },
  },

  {
    files: ['tsconfig.json'],
    rules: {
      'jsonc/sort-keys': 'off',
    },
  },

  vueConfigPrettier,
)
