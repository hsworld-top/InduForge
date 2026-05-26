<template>
  <DcDialog v-model="visible" :title="mode === 'create' ? '新建 Topic 分组' : '编辑 Topic 分组'" width="420px" :close-disabled="loading">
    <el-form label-position="top" class="kafka-topic-group-dialog">
      <el-form-item label="分组名称" required>
        <el-input v-model="form.name" maxlength="100" show-word-limit placeholder="输入分组名称" @keyup.enter="submit" />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="kafka-topic-group-dialog__footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="loading" :disabled="!form.name.trim()" @click="submit">
          保存
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { KafkaTopicGroup } from './types'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    mode: 'create' | 'edit'
    group?: KafkaTopicGroup | null
    loading?: boolean
  }>(),
  {
    group: null,
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: { name: string }): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const form = reactive({ name: '' })

const resetForm = () => {
  form.name = props.group?.name || ''
}

const submit = () => {
  const name = form.name.trim()
  if (!name || props.loading) return
  emit('submit', { name })
}

watch(
  () => [props.modelValue, props.group?.id] as const,
  () => {
    if (props.modelValue) resetForm()
  },
  { immediate: true },
)
</script>

<style scoped>
.kafka-topic-group-dialog {
  display: grid;
  gap: 2px;
}

.kafka-topic-group-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
