<!--
  新建/编辑变量分组（脚本侧与数据点侧共用表单壳）
-->
<script setup lang="ts">
import { useI18n } from 'vue-i18n'

interface VariableGroupOptionLike {
  id: string
  name: string
}

defineProps<{
  editMode?: boolean
  parentOptions?: VariableGroupOptionLike[]
}>()

const emit = defineEmits<{
  (event: 'confirm'): void
}>()

const visible = defineModel<boolean>({ default: false })
const name = defineModel<string>('name', { default: '' })
const parentId = defineModel<string | null>('parentId', { default: null })
const { t } = useI18n()

function handleCancel() {
  visible.value = false
}

function handleConfirm() {
  emit('confirm')
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="editMode ? t('variableGroupForm.editTitle') : t('variableGroupForm.createTitle')"
    width="420px"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <el-form label-width="70px">
      <el-form-item :label="t('variableGroupForm.name')">
        <el-input v-model="name" />
      </el-form-item>
      <el-form-item :label="t('variableGroupForm.parent')">
        <el-select v-model="parentId" :placeholder="t('variableGroupForm.root')" :clearable="false">
          <el-option :label="t('variableGroupForm.root')" :value="null" />
          <el-option
            v-for="group in parentOptions"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="handleCancel">{{ t('variableGroupForm.cancel') }}</el-button>
      <el-button type="primary" @click="handleConfirm">
        {{ t('variableGroupForm.confirm') }}
      </el-button>
    </template>
  </el-dialog>
</template>
