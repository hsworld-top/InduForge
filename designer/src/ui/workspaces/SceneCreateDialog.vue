<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import IconLucideBox from '~icons/lucide/box'
import IconLucideRoute from '~icons/lucide/route'
import IconLucideX from '~icons/lucide/x'
import type { SceneKind } from './scene-contract-api'

const props = defineProps<{
  kind: SceneKind
  loading: boolean
  error?: string
}>()

const emit = defineEmits<{
  cancel: []
  submit: [input: { sceneId: string; name: string }]
}>()

const sceneId = ref('')
const name = ref('')
const validationError = ref('')
const title = computed(() => `新建${props.kind === '2d' ? '2D 画面' : '3D 场景'}`)
const message = computed(() => validationError.value || props.error || '')

watch(
  () => props.kind,
  () => {
    sceneId.value = ''
    name.value = ''
    validationError.value = ''
  },
)

function submit(): void {
  const normalizedId = sceneId.value.trim()
  const normalizedName = name.value.trim()
  if (!/^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$/.test(normalizedId)) {
    validationError.value = '场景 ID 只能包含字母、数字、下划线和连字符'
    return
  }
  if (!normalizedName) {
    validationError.value = '请输入场景名称'
    return
  }
  validationError.value = ''
  emit('submit', { sceneId: normalizedId, name: normalizedName })
}
</script>

<template>
  <div
    class="scene-create-backdrop"
    @click.self="!loading && emit('cancel')"
    @keydown.esc="!loading && emit('cancel')"
  >
    <section
      class="scene-create-dialog"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="`scene-create-title-${kind}`"
    >
      <header>
        <div class="scene-create-title">
          <span><IconLucideRoute v-if="kind === '2d'" /><IconLucideBox v-else /></span>
          <h3 :id="`scene-create-title-${kind}`">{{ title }}</h3>
        </div>
        <button
          type="button"
          class="icon-button"
          title="关闭"
          aria-label="关闭新建场景"
          :disabled="loading"
          @click="emit('cancel')"
        >
          <IconLucideX />
        </button>
      </header>

      <form @submit.prevent="submit">
        <label>
          <span>场景 ID</span>
          <input
            v-model="sceneId"
            autofocus
            autocomplete="off"
            maxlength="128"
            placeholder="line-overview"
            :disabled="loading"
          />
        </label>
        <label>
          <span>名称</span>
          <input
            v-model="name"
            autocomplete="off"
            maxlength="128"
            placeholder="产线总览"
            :disabled="loading"
          />
        </label>
        <p v-if="message" class="scene-create-error" role="alert">{{ message }}</p>
        <footer>
          <button
            type="button"
            class="secondary-button"
            :disabled="loading"
            @click="emit('cancel')"
          >
            取消
          </button>
          <button type="submit" class="primary-button" :disabled="loading">
            {{ loading ? '创建中' : '创建并打开' }}
          </button>
        </footer>
      </form>
    </section>
  </div>
</template>

<style scoped>
.scene-create-backdrop {
  position: absolute;
  z-index: 20;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 16px;
  background: rgba(23, 32, 51, 0.34);
}

.scene-create-dialog {
  width: min(420px, 100%);
  overflow: hidden;
  border: 1px solid #cbd3df;
  border-radius: 7px;
  background: #fff;
  box-shadow: 0 18px 48px rgba(23, 32, 51, 0.22);
}

header,
.scene-create-title,
footer {
  display: flex;
  align-items: center;
}

header {
  justify-content: space-between;
  min-height: 52px;
  padding: 0 14px 0 16px;
  border-bottom: 1px solid #e1e6ed;
}

.scene-create-title {
  min-width: 0;
  gap: 10px;
}

.scene-create-title > span {
  width: 30px;
  height: 30px;
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 6px;
  color: #245fc2;
  background: #eaf1ff;
}

.scene-create-title svg,
.icon-button svg {
  width: 15px;
  height: 15px;
}

h3 {
  margin: 0;
  overflow: hidden;
  color: #172033;
  font-size: 14px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.icon-button {
  width: 30px;
  height: 30px;
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 5px;
  color: #64748b;
  background: transparent;
  cursor: pointer;
}

.icon-button:hover:not(:disabled) {
  border-color: #d8dee8;
  background: #f3f5f8;
}

form {
  display: grid;
  gap: 14px;
  padding: 18px;
}

label {
  display: grid;
  gap: 6px;
  color: #445168;
  font-size: 12px;
  font-weight: 600;
}

input {
  width: 100%;
  height: 36px;
  box-sizing: border-box;
  padding: 0 10px;
  border: 1px solid #cbd3df;
  border-radius: 5px;
  outline: none;
  color: #172033;
  background: #fff;
  font:
    12px Consolas,
    monospace;
}

input:focus {
  border-color: #2563eb;
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.12);
}

.scene-create-error {
  margin: -2px 0 0;
  color: #b42318;
  font-size: 11px;
}

footer {
  justify-content: flex-end;
  gap: 8px;
  margin-top: 2px;
}

footer button {
  min-width: 84px;
  height: 34px;
  padding: 0 12px;
  border-radius: 5px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.secondary-button {
  border: 1px solid #cbd3df;
  color: #445168;
  background: #fff;
}

.primary-button {
  border: 1px solid #1d4ed8;
  color: #fff;
  background: #2563eb;
}

button:disabled,
input:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
</style>
