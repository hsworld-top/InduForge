<script setup lang="ts">
import { ref, watch } from 'vue'
import { getApiErrorMessage } from '@/utils/request'
import SceneSchemaEditor from './SceneSchemaEditor.vue'
import {
  sceneContractApi,
  type SceneContract,
  type SceneContractCommand,
  type SceneContractMember,
  type SceneContractParameter,
  type SceneKind,
} from './scene-contract-api'

const props = defineProps<{ projectId: string; kind: SceneKind; sceneId?: string }>()
const emit = defineEmits<{
  close: []
  synced: [message: string]
  committed: [event: { projectId: string; target: SceneKind; sceneId: string; revision: number }]
}>()
const loading = ref(false)
const saving = ref(false)
const message = ref('')
const form = ref<SceneContract>(emptyContract(props.kind))

watch(() => [props.projectId, props.kind, props.sceneId], () => void load(), { immediate: true })

function emptyContract(kind: SceneKind): SceneContract {
  return { id: '', kind, name: '', description: '', parameters: [], events: [], commands: [], managedEvents: [], managedCommands: [], datapointRefs: [], contractVersion: '' }
}
function objectSchema(): Record<string, unknown> { return { type: 'object', properties: {}, required: [], additionalProperties: false } }

async function load(): Promise<void> {
  if (!props.projectId || !props.sceneId) return
  loading.value = true
  try {
    const contracts = (await sceneContractApi.list(props.projectId)).contracts
    const found = contracts.find((item) => item.id === props.sceneId && item.kind === props.kind)
    if (!found) throw new Error('场景不存在或尚未提交')
    form.value = structuredClone(found)
    message.value = ''
  } catch (error) { message.value = getApiErrorMessage(error, '读取场景设置失败') }
  finally { loading.value = false }
}

function addParameter(): void {
  const item: SceneContractParameter = { name: `parameter${(form.value.parameters?.length || 0) + 1}`, description: '', required: false, schema: { type: 'string' } }
  form.value.parameters = [...(form.value.parameters || []), item]
}
function addEvent(): void {
  const item: SceneContractMember = { name: `event${(form.value.events?.length || 0) + 1}`, description: '', schema: objectSchema() }
  form.value.events = [...(form.value.events || []), item]
}
function addCommand(): void {
  const item: SceneContractCommand = { name: `command${(form.value.commands?.length || 0) + 1}`, description: '', inputSchema: objectSchema(), outputSchema: objectSchema() }
  form.value.commands = [...(form.value.commands || []), item]
}

async function save(): Promise<void> {
  if (!props.projectId || saving.value || form.value.draftVersion == null) return
  saving.value = true
  message.value = ''
  try {
    const updated = await sceneContractApi.update(props.projectId, { ...form.value, kind: props.kind })
    form.value = structuredClone(updated.contract)
    const committed = await sceneContractApi.commit(props.projectId, props.kind, updated.contract.id, updated.contract.draftVersion!)
    await load()
    message.value = '场景设置已保存并应用'
    emit('committed', { projectId: props.projectId, target: props.kind, sceneId: updated.contract.id, revision: committed.revision })
    emit('synced', message.value)
  } catch (error) { message.value = getApiErrorMessage(error, '保存场景设置失败；未提交内容已保留为草稿') }
  finally { saving.value = false }
}
</script>

<template>
  <aside class="scene-contract-panel">
    <header><div><p>SCENE SETTINGS</p><h3>{{ kind === '2d' ? '2D' : '3D' }} 场景设置</h3></div><button type="button" @click="emit('close')">关闭</button></header>
    <div class="form-body">
      <label>名称<input v-model.trim="form.name" placeholder="产线总览" /></label>
      <label>说明<textarea v-model.trim="form.description" rows="2" /></label>
      <section><div class="section-title"><strong>参数</strong><button type="button" @click="addParameter">添加</button></div>
        <article v-for="(item, index) in form.parameters" :key="index"><div class="member-head"><input v-model.trim="item.name" placeholder="参数名称" /><label><input v-model="item.required" type="checkbox" />必填</label><button type="button" @click="form.parameters!.splice(index, 1)">删除</button></div><input v-model.trim="item.description" placeholder="说明" /><SceneSchemaEditor v-model="item.schema" /></article>
        <p v-if="!form.parameters?.length">未声明页面传入参数</p></section>
      <section><div class="section-title"><strong>事件</strong><button type="button" @click="addEvent">添加</button></div>
        <article v-for="(item, index) in form.events" :key="index"><div class="member-head"><input v-model.trim="item.name" placeholder="事件名称" /><span></span><button type="button" @click="form.events!.splice(index, 1)">删除</button></div><input v-model.trim="item.description" placeholder="说明" /><SceneSchemaEditor v-model="item.schema" /></article>
        <p v-if="!form.events?.length">未声明场景事件</p></section>
      <section><div class="section-title"><strong>命令</strong><button type="button" @click="addCommand">添加</button></div>
        <article v-for="(item, index) in form.commands" :key="index"><div class="member-head"><input v-model.trim="item.name" placeholder="命令名称" /><span></span><button type="button" @click="form.commands!.splice(index, 1)">删除</button></div><input v-model.trim="item.description" placeholder="说明" /><small>输入</small><SceneSchemaEditor v-model="item.inputSchema" /><small>输出</small><SceneSchemaEditor v-model="item.outputSchema" /></article>
        <p v-if="!form.commands?.length">未声明页面可调用命令</p></section>
      <section><div class="section-title"><strong>关联数据点</strong><span>{{ form.datapointRefs?.length || 0 }}</span></div><ul><li v-for="item in form.datapointRefs" :key="item">{{ item }}</li></ul><p v-if="!form.datapointRefs?.length">由场景保存时自动提取</p></section>
      <section><div class="section-title"><strong>场景生成的交互</strong><span>{{ (form.managedEvents?.length || 0) + (form.managedCommands?.length || 0) }}</span></div>
        <ul><li v-for="item in form.managedEvents" :key="`event-${item.name}`">事件 · {{ item.name }}</li><li v-for="item in form.managedCommands" :key="`command-${item.name}`">命令 · {{ item.name }}</li></ul>
        <p v-if="!form.managedEvents?.length && !form.managedCommands?.length">在 2D/3D 编辑器的“交互”面板配置后自动生成，此处只读</p>
      </section>
    </div>
    <footer><span>{{ message }}</span><div><button type="button" :disabled="loading" @click="load">刷新</button><button type="button" :disabled="saving || !form.id" @click="save">{{ saving ? '保存中' : '保存并应用' }}</button></div></footer>
  </aside>
</template>

<style scoped>
.scene-contract-panel{position:absolute;z-index:5;top:0;right:0;display:grid;grid-template-rows:auto minmax(0,1fr) auto;width:min(520px,58vw);height:100%;border-left:1px solid #b9c1bc;background:#f8faf8;box-shadow:-8px 0 24px rgb(17 27 30 / 18%)}header,footer,.section-title,.member-head{display:flex;align-items:center;justify-content:space-between;gap:8px}header,footer{padding:12px 14px;border-bottom:1px solid #d1d7d3}footer{border-top:1px solid #d1d7d3;border-bottom:0}header p{margin:0;color:#66716f;font-size:10px}h3{margin:3px 0 0;font-size:16px}.form-body{overflow:auto;padding:14px}.form-body>label{display:grid;gap:5px;margin-bottom:10px;font-size:11px}section{display:grid;gap:8px;padding:12px 0;border-top:1px solid #dce1dd}article{display:grid;gap:7px;padding:9px;border:1px solid #ccd3ce;background:#fff}.member-head{display:grid;grid-template-columns:minmax(0,1fr) auto auto}.member-head label{display:flex;align-items:center;font-size:11px}.member-head label input{width:auto}input,textarea{box-sizing:border-box;width:100%;min-width:0;border:1px solid #aeb6b1;padding:7px;background:#fff;color:#15201c}button{min-height:28px;border:1px solid #9fa9a3;background:#fff;cursor:pointer}button:hover:not(:disabled){border-color:#19241f;background:#e8f3e9}button:disabled{opacity:.5;cursor:not-allowed}section p{margin:0;color:#7a8581;font-size:11px}small{color:#64706b}ul{margin:0;padding-left:18px;font:11px Consolas,monospace}footer>span{overflow:hidden;color:#59645f;font-size:11px;text-overflow:ellipsis;white-space:nowrap}footer>div{display:flex;gap:6px;flex:none}@media(max-width:700px){.scene-contract-panel{width:100%}}
</style>
