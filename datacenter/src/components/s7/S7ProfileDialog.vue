<template>
  <el-dialog :model-value="modelValue" title="配置 PLC 类型" width="760px" @close="$emit('update:modelValue', false)">
    <el-form class="s7-profile-dialog" label-position="top">
      <section>
        <h3>01 PLC 类型</h3>
        <el-form-item label="PLC 系列">
          <el-select v-model="form.plcFamily" style="width: 100%" @change="applyFamilyDefaults">
            <el-option v-for="family in familyOptions" :key="family" :label="family" :value="family" />
          </el-select>
        </el-form-item>
        <el-checkbox v-model="form.optimizedBlockAccess">优化 DB 访问风险提示</el-checkbox>
      </section>
      <section>
        <h3>02 连接参数</h3>
        <div class="s7-profile-dialog__grid">
          <el-form-item label="Host"><el-input v-model="form.host" /></el-form-item>
          <el-form-item label="Port"><el-input-number v-model="form.port" :min="1" :max="65535" style="width: 100%" /></el-form-item>
          <el-form-item label="通信方式">
            <el-select v-model="form.communicationMode" style="width: 100%">
              <el-option label="Rack / Slot" value="rack_slot" />
              <el-option label="TSAP" value="tsap" />
            </el-select>
          </el-form-item>
          <el-form-item label="Rack"><el-input-number v-model="form.rack" :min="0" style="width: 100%" /></el-form-item>
          <el-form-item label="Slot"><el-input-number v-model="form.slot" :min="0" style="width: 100%" /></el-form-item>
          <el-form-item label="Local TSAP"><el-input v-model="form.localTsap" /></el-form-item>
          <el-form-item label="Remote TSAP"><el-input v-model="form.remoteTsap" /></el-form-item>
        </div>
      </section>
      <section>
        <h3>03 读取能力</h3>
        <div class="s7-profile-dialog__grid">
          <el-form-item label="默认周期 ms"><el-input-number v-model="form.pollIntervalMs" :min="100" style="width: 100%" /></el-form-item>
          <el-form-item label="连接超时 ms"><el-input-number v-model="form.connectTimeoutMs" :min="100" style="width: 100%" /></el-form-item>
          <el-form-item label="读取超时 ms"><el-input-number v-model="form.readTimeoutMs" :min="100" style="width: 100%" /></el-form-item>
          <el-form-item label="PDU Size"><el-input-number v-model="form.pduSize" :min="0" style="width: 100%" /></el-form-item>
          <el-form-item label="最大读取 bytes"><el-input-number v-model="form.maxReadBytes" :min="0" style="width: 100%" /></el-form-item>
          <el-form-item label="合并间隙 bytes"><el-input-number v-model="form.maxGapBytes" :min="0" style="width: 100%" /></el-form-item>
          <el-form-item label="并发读取"><el-input-number v-model="form.maxConcurrentReads" :min="1" style="width: 100%" /></el-form-item>
        </div>
      </section>
      <section>
        <h3>04 地址能力</h3>
        <el-checkbox-group v-model="form.supportedAreas">
          <el-checkbox-button v-for="area in areaOptions" :key="area" :label="area" />
        </el-checkbox-group>
        <div class="s7-profile-dialog__checks">
          <el-checkbox v-model="form.allowAbsoluteAddress">允许绝对地址</el-checkbox>
          <el-checkbox v-model="form.allowSymbolAddress">允许符号地址</el-checkbox>
        </div>
      </section>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import type { S7Profile } from './types'

const props = defineProps<{ modelValue: boolean; profile?: S7Profile | null; loading?: boolean }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void; (event: 'submit', payload: Record<string, unknown>): void }>()
const familyOptions = ['S7-200', 'S7-200 SMART', 'S7-300', 'S7-400', 'S7-1200', 'S7-1500', 'S7 Compatible']
const areaOptions = ['DB', 'M', 'I', 'Q']
const form = reactive({
  plcFamily: 'S7 Compatible',
  communicationMode: 'rack_slot',
  host: '',
  port: 102,
  rack: 0,
  slot: 1,
  localTsap: '',
  remoteTsap: '',
  pollIntervalMs: 1000,
  connectTimeoutMs: 3000,
  readTimeoutMs: 3000,
  pduSize: 0,
  maxReadBytes: 0,
  maxGapBytes: 8,
  maxConcurrentReads: 1,
  optimizedBlockAccess: false,
  allowAbsoluteAddress: true,
  allowSymbolAddress: false,
  supportedAreas: ['DB', 'M', 'I', 'Q'],
})

const applyFamilyDefaults = () => {
  if (form.plcFamily === 'S7-300' || form.plcFamily === 'S7-400') {
    Object.assign(form, { communicationMode: 'rack_slot', rack: 0, slot: 2, maxGapBytes: 8, optimizedBlockAccess: false })
  } else if (form.plcFamily === 'S7-1200' || form.plcFamily === 'S7-1500') {
    Object.assign(form, { communicationMode: 'rack_slot', rack: 0, slot: 1, maxGapBytes: 16, optimizedBlockAccess: true })
  } else if (form.plcFamily === 'S7-200' || form.plcFamily === 'S7-200 SMART') {
    Object.assign(form, { communicationMode: 'tsap', maxGapBytes: 8, optimizedBlockAccess: false })
  } else {
    Object.assign(form, { communicationMode: 'rack_slot', rack: 0, slot: 1, maxGapBytes: 8, optimizedBlockAccess: false })
  }
}

watch(
  () => [props.modelValue, props.profile],
  () => {
    if (!props.profile) return
    Object.assign(form, {
      ...props.profile,
      localTsap: props.profile.localTsap || '',
      remoteTsap: props.profile.remoteTsap || '',
      pduSize: props.profile.pduSize || 0,
      maxReadBytes: props.profile.maxReadBytes || 0,
      supportedAreas: props.profile.supportedAreas?.length ? [...props.profile.supportedAreas] : ['DB', 'M', 'I', 'Q'],
    })
  },
  { immediate: true },
)

const submit = () => {
  emit('submit', {
    ...form,
    localTsap: form.localTsap || null,
    remoteTsap: form.remoteTsap || null,
    pduSize: form.pduSize || null,
    maxReadBytes: form.maxReadBytes || null,
  })
}
</script>

<style scoped>
.s7-profile-dialog {
  display: grid;
  gap: 12px;
}
.s7-profile-dialog section {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
}
.s7-profile-dialog h3 {
  margin: 0 0 10px;
  font-size: 13px;
}
.s7-profile-dialog__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0 12px;
}
.s7-profile-dialog__checks {
  margin-top: 10px;
  display: flex;
  gap: 16px;
}
</style>
