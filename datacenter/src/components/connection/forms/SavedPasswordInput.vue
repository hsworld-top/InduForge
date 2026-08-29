<template>
  <el-input
    :model-value="modelValue"
    :type="visible ? 'text' : 'password'"
    :placeholder="placeholder"
    autocomplete="new-password"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <template #suffix>
      <el-button
        class="saved-password-input__toggle"
        link
        :loading="loading"
        :aria-label="visible ? ui('隐藏密码', 'Hide Password') : ui('查看密码', 'Show Password')"
        @click.stop="toggleVisibility"
      >
        <IconTablerEyeOff v-if="visible" />
        <IconTablerEye v-else />
      </el-button>
    </template>
  </el-input>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import IconTablerEye from '~icons/tabler/eye'
import IconTablerEyeOff from '~icons/tabler/eye-off'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  savedPasswordConfigured: { type: Boolean, default: false },
  loading: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'reveal-saved-password'])
const visible = ref(false)
const waitingForSavedPassword = ref(false)

const toggleVisibility = () => {
  if (!props.modelValue && props.savedPasswordConfigured) {
    waitingForSavedPassword.value = true
    emit('reveal-saved-password')
    return
  }
  visible.value = !visible.value
}

watch(
  () => props.modelValue,
  (value) => {
    if (waitingForSavedPassword.value && value) {
      visible.value = true
      waitingForSavedPassword.value = false
    }
    if (!value && !props.loading) visible.value = false
  },
)
</script>

<style scoped>
.saved-password-input__toggle {
  width: 24px;
  height: 24px;
  padding: 0;
  color: var(--dc-text-muted);
}

.saved-password-input__toggle svg {
  width: 16px;
  height: 16px;
}
</style>
