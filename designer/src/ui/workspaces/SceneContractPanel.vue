<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { getApiErrorMessage } from '@/utils/request'
import {
  sceneContractApi,
  type SceneContract,
  type SceneContractMember,
  type SceneKind,
} from './scene-contract-api'

const props = defineProps<{
  projectId: string
  kind: SceneKind
}>()

const emit = defineEmits<{ close: []; synced: [message: string] }>()

const contracts = ref<SceneContract[]>([])
const loading = ref(false)
const saving = ref(false)
const message = ref('')
const selectedKey = ref('')
const form = ref<SceneContract>(emptyContract(props.kind))
const inputsText = ref('[]')
const eventsText = ref('[]')
const commandsText = ref('[]')
const objectsText = ref('[]')
const datapointsText = ref('')
const permissionsText = ref('')

const visibleContracts = computed(() => contracts.value.filter((item) => item.kind === props.kind))
const title = computed(() => (props.kind === '2d' ? '2D 场景公开接口' : '3D 场景公开接口'))

watch(
  () => [props.projectId, props.kind],
  () => void load(),
  { immediate: true },
)

function emptyContract(kind: SceneKind): SceneContract {
  return {
    id: '',
    kind,
    name: '',
    description: '',
    route: '',
    embedMode: 'both',
    inputs: [],
    events: [],
    commands: [],
    publicObjects: [],
    datapointRefs: [],
    permissionRefs: [],
    contractVersion: '',
  }
}

async function load(): Promise<void> {
  if (!props.projectId) return
  loading.value = true
  message.value = ''
  try {
    contracts.value = (await sceneContractApi.list(props.projectId)).contracts
    if (selectedKey.value) select(selectedKey.value)
  } catch (error) {
    message.value = getApiErrorMessage(error, '读取场景公开接口失败')
  } finally {
    loading.value = false
  }
}

function create(): void {
  selectedKey.value = ''
  form.value = emptyContract(props.kind)
  applyTexts(form.value)
  message.value = ''
}

function select(id: string): void {
  const found = visibleContracts.value.find((item) => item.id === id)
  if (!found) return
  selectedKey.value = found.id
  form.value = structuredClone(found)
  applyTexts(form.value)
  message.value = ''
}

function applyTexts(contract: SceneContract): void {
  inputsText.value = JSON.stringify(contract.inputs ?? [], null, 2)
  eventsText.value = JSON.stringify(contract.events ?? [], null, 2)
  commandsText.value = JSON.stringify(contract.commands ?? [], null, 2)
  objectsText.value = JSON.stringify(contract.publicObjects ?? [], null, 2)
  datapointsText.value = (contract.datapointRefs ?? []).join('\n')
  permissionsText.value = (contract.permissionRefs ?? []).join('\n')
}

function parseMembers(value: string, label: string): SceneContractMember[] {
  let parsed: unknown
  try {
    parsed = JSON.parse(value || '[]')
  } catch {
    throw new Error(`${label} 必须是 JSON 数组`)
  }
  if (!Array.isArray(parsed) || parsed.some((item) => !item || typeof item !== 'object' || typeof item.name !== 'string')) {
    throw new Error(`${label} 只能包含带 name 字段的对象`)
  }
  return parsed as SceneContractMember[]
}

function parseRefs(value: string): string[] {
  return [...new Set(value.split(/\r?\n/).map((item) => item.trim()).filter(Boolean))]
}

async function save(): Promise<void> {
  if (!props.projectId || saving.value) return
  saving.value = true
  message.value = ''
  try {
    const contract: SceneContract = {
      ...form.value,
      kind: props.kind,
      inputs: parseMembers(inputsText.value, '输入'),
      events: parseMembers(eventsText.value, '事件'),
      commands: parseMembers(commandsText.value, '命令'),
      publicObjects: parseMembers(objectsText.value, '公开对象'),
      datapointRefs: parseRefs(datapointsText.value),
      permissionRefs: parseRefs(permissionsText.value),
    }
    const result = await sceneContractApi.save(props.projectId, contract)
    form.value = result.contract
    selectedKey.value = result.contract.id
    await load()
    const syncMessage = result.contextSync.status === 'updated' ? '工程上下文已同步' : '场景已保存，上下文待刷新'
    message.value = syncMessage
    emit('synced', syncMessage)
  } catch (error) {
    message.value = getApiErrorMessage(error, '保存场景公开接口失败')
  } finally {
    saving.value = false
  }
}

async function remove(): Promise<void> {
  if (!props.projectId || !selectedKey.value || saving.value) return
  saving.value = true
  message.value = ''
  try {
    const result = await sceneContractApi.remove(props.projectId, props.kind, selectedKey.value)
    create()
    await load()
    const syncMessage = result.contextSync.status === 'updated' ? '场景接口已删除并同步上下文' : '场景接口已删除，上下文待刷新'
    message.value = syncMessage
    emit('synced', syncMessage)
  } catch (error) {
    message.value = getApiErrorMessage(error, '删除场景公开接口失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <aside class="scene-contract-panel">
    <header>
      <div>
        <p>SCENE CONTRACT</p>
        <h3>{{ title }}</h3>
      </div>
      <button type="button" class="text-button" @click="emit('close')">关闭</button>
    </header>

    <div class="scene-contract-list">
      <button type="button" :class="{ active: !selectedKey }" @click="create">+ 新建接口</button>
      <button
        v-for="item in visibleContracts"
        :key="item.id"
        type="button"
        :class="{ active: selectedKey === item.id }"
        @click="select(item.id)"
      >
        <strong>{{ item.name }}</strong><small>{{ item.id }}</small>
      </button>
    </div>

    <div class="scene-contract-form">
      <label>场景 ID<input v-model.trim="form.id" :disabled="Boolean(selectedKey)" placeholder="line-overview" /></label>
      <label>名称<input v-model.trim="form.name" placeholder="产线总览" /></label>
      <label>说明<textarea v-model.trim="form.description" rows="2" /></label>
      <label>路由<input v-model.trim="form.route" placeholder="/scenes/line-overview" /></label>
      <label>嵌入方式<select v-model="form.embedMode"><option value="both">独立与嵌入</option><option value="standalone">仅独立路由</option><option value="embedded">仅页面嵌入</option></select></label>
      <label>输入（JSON 数组）<textarea v-model="inputsText" rows="4" /></label>
      <label>事件（JSON 数组）<textarea v-model="eventsText" rows="4" /></label>
      <label>命令（JSON 数组）<textarea v-model="commandsText" rows="4" /></label>
      <label>公开对象（JSON 数组）<textarea v-model="objectsText" rows="4" /></label>
      <label>数据点引用（每行一个 path）<textarea v-model="datapointsText" rows="3" /></label>
      <label>权限引用（每行一个 code）<textarea v-model="permissionsText" rows="3" /></label>
    </div>

    <footer>
      <span>{{ message }}</span>
      <div><button type="button" :disabled="loading" @click="load">刷新</button><button type="button" :disabled="saving" @click="save">{{ saving ? '保存中' : '保存并同步' }}</button><button v-if="selectedKey" type="button" class="danger" :disabled="saving" @click="remove">删除</button></div>
    </footer>
  </aside>
</template>

<style scoped>
.scene-contract-panel { width: min(460px, 48vw); height: 100%; position: absolute; z-index: 5; top: 0; right: 0; display: grid; grid-template-rows: auto auto minmax(0, 1fr) auto; border-left: 1px solid #111b1e; background: #f8faf6; box-shadow: -8px 0 24px rgba(17, 27, 30, .18); }
header, footer { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 14px; border-bottom: 1px solid #c8cec6; }
header p { margin: 0; color: #66716f; font: 600 10px Bahnschrift, sans-serif; letter-spacing: .16em; } h3 { margin: 4px 0 0; font-size: 16px; } button { border: 1px solid #9fa9a3; background: #fff; color: #111b1e; cursor: pointer; } button:hover:not(:disabled), button.active { background: #d8ff36; border-color: #111b1e; } button:disabled { opacity: .5; cursor: not-allowed; }.text-button { padding: 6px 8px; }.scene-contract-list { display: flex; gap: 6px; max-height: 102px; overflow: auto; padding: 10px 14px; border-bottom: 1px solid #c8cec6; }.scene-contract-list button { min-width: 92px; padding: 7px; text-align: left; }.scene-contract-list strong,.scene-contract-list small { display:block; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }.scene-contract-list small { margin-top: 3px; color:#66716f; font: 10px Consolas, monospace; }.scene-contract-form { min-height: 0; overflow:auto; padding: 14px; }.scene-contract-form label { display:grid; gap:5px; margin-bottom:10px; color:#45514d; font-size:11px; }.scene-contract-form input,.scene-contract-form textarea,.scene-contract-form select { width:100%; box-sizing:border-box; border:1px solid #aeb6b1; padding:7px; color:#111b1e; background:#fff; font:12px Consolas, monospace; resize:vertical; }.scene-contract-form select { font-family:inherit; }footer { min-height:44px; border-top:1px solid #c8cec6; border-bottom:0; font-size:11px; }footer > span { min-width:0; color:#66716f; }footer div { display:flex; gap:6px; }footer button { padding:6px 8px; white-space:nowrap; }.danger { color:#9e2518; }.danger:hover:not(:disabled) { background:#ffd8d1 !important; }
</style>
