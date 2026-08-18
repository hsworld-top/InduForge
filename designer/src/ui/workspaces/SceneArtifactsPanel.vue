<script setup lang="ts">
import { computed } from 'vue'
import IconLucideBox from '~icons/lucide/box'
import IconLucideExternalLink from '~icons/lucide/external-link'
import IconLucideFileCode2 from '~icons/lucide/file-code-2'
import IconLucidePlay from '~icons/lucide/play'
import IconLucideRefreshCw from '~icons/lucide/refresh-cw'
import IconLucideTrash2 from '~icons/lucide/trash-2'
import IconLucideRoute from '~icons/lucide/route'
import type { SceneContract, SceneKind } from './scene-contract-api'

const props = defineProps<{
  kind: SceneKind
  contracts: SceneContract[]
}>()

const emit = defineEmits<{
  create: []
  openEditor: [contract: SceneContract]
  editContract: [contract: SceneContract]
  preview: [contract: SceneContract]
  remove: [contract: SceneContract]
  refresh: []
}>()

const title = computed(() => (props.kind === '2d' ? '2D 画面' : '3D 场景'))
const emptyTitle = computed(() => `暂无${title.value}产物`)

function interfaceCount(contract: SceneContract): number {
  return (
    (contract.inputs?.length ?? 0) +
    (contract.events?.length ?? 0) +
    (contract.commands?.length ?? 0) +
    (contract.publicObjects?.length ?? 0)
  )
}

function embedModeLabel(contract: SceneContract): string {
  if (contract.embedMode === 'standalone') return '独立运行'
  if (contract.embedMode === 'embedded') return '页面嵌入'
  return '独立 / 嵌入'
}

</script>

<template>
  <section class="scene-artifacts-panel">
    <header class="scene-artifacts-header">
      <div>
        <strong>{{ title }}</strong>
        <span>{{ contracts.length }} 个产物</span>
      </div>
      <div class="scene-header-actions">
      <button type="button" class="icon-button" title="刷新场景" aria-label="刷新场景" @click="emit('refresh')">
        <IconLucideRefreshCw />
      </button>
      <button type="button" class="scene-editor-button" @click="emit('create')">
        <IconLucideExternalLink />
        新建{{ title }}
      </button>
      </div>
    </header>

    <div v-if="contracts.length" class="scene-card-grid">
      <article v-for="contract in contracts" :key="contract.id" class="scene-card">
        <div class="scene-card-preview" :class="`kind-${kind}`">
          <IconLucideRoute v-if="kind === '2d'" />
          <IconLucideBox v-else />
          <span>{{ kind.toUpperCase() }}</span>
        </div>
        <div class="scene-card-body">
          <div class="scene-card-title">
            <strong>{{ contract.name }}</strong>
            <span>{{ embedModeLabel(contract) }}</span>
          </div>
          <p>{{ contract.description || '尚未填写产物说明。' }}</p>
          <dl>
            <div>
              <dt>场景 ID</dt>
              <dd>{{ contract.id }}</dd>
            </div>
            <div>
              <dt>页面路由</dt>
              <dd>{{ contract.route || '未声明' }}</dd>
            </div>
          </dl>
          <footer>
            <span>公开能力 {{ interfaceCount(contract) }}</span>
            <span>数据点 {{ contract.datapointRefs?.length ?? 0 }}</span>
            <span>r{{ contract.currentRevision ?? 0 }}</span>
          </footer>
          <div class="scene-card-actions">
            <button type="button" title="打开编辑器" aria-label="打开编辑器" @click="emit('openEditor', contract)"><IconLucideExternalLink /></button>
            <button type="button" title="编辑公开契约" aria-label="编辑公开契约" @click="emit('editContract', contract)"><IconLucideFileCode2 /></button>
            <button type="button" title="预览已提交版本" aria-label="预览已提交版本" :disabled="!contract.currentRevision" @click="emit('preview', contract)"><IconLucidePlay /></button>
            <button type="button" class="danger" title="删除场景" aria-label="删除场景" @click="emit('remove', contract)"><IconLucideTrash2 /></button>
            <span v-if="contract.draftVersion !== contract.committedDraftVersion">草稿未提交</span>
          </div>
        </div>
      </article>
    </div>

    <div v-else class="scene-empty-state">
      <div :class="`kind-${kind}`">
        <IconLucideRoute v-if="kind === '2d'" />
        <IconLucideBox v-else />
      </div>
      <strong>{{ emptyTitle }}</strong>
      <p>新建场景后即可进入编辑器，显式保存会形成不可变 revision。</p>
    </div>
  </section>
</template>

<style scoped>
.scene-artifacts-panel {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 48px minmax(0, 1fr);
  overflow: hidden;
  color: #273142;
  background: #f2f4f7;
}

.scene-artifacts-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 14px 0 16px;
  border-bottom: 1px solid #d9dee7;
  background: #fff;
}

.scene-artifacts-header > div {
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.scene-artifacts-header strong {
  font-size: 13px;
  font-weight: 650;
}

.scene-artifacts-header span {
  color: #7a8596;
  font-size: 11px;
}

.scene-editor-button {
  height: 30px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  border: 1px solid #c9d5e7;
  border-radius: 5px;
  color: #245fc2;
  background: #f5f8ff;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
}

.scene-editor-button:hover {
  border-color: #9db7df;
  background: #eaf1ff;
}

.scene-editor-button svg {
  width: 14px;
  height: 14px;
}

.scene-header-actions,
.scene-card-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.icon-button,
.scene-card-actions button {
  width: 30px;
  height: 30px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid #d1d8e2;
  border-radius: 5px;
  color: #50617a;
  background: #fff;
  cursor: pointer;
}

.icon-button svg,
.scene-card-actions svg {
  width: 14px;
  height: 14px;
}

.scene-card-actions {
  min-height: 31px;
}

.scene-card-actions .danger {
  color: #a3342b;
}

.scene-card-actions button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.scene-card-actions > span {
  margin-left: auto;
  color: #a85f18;
  font-size: 10px;
}

.scene-card-grid {
  min-height: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(270px, 1fr));
  align-content: start;
  gap: 12px;
  padding: 14px;
  overflow: auto;
}

.scene-card {
  min-width: 0;
  display: grid;
  grid-template-rows: 118px minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid #d8dee8;
  border-radius: 7px;
  background: #fff;
  box-shadow: 0 2px 8px rgba(20, 31, 51, 0.05);
}

.scene-card-preview {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-bottom: 1px solid #e0e5ec;
  color: #50617a;
  background-color: #e9edf3;
  background-image:
    linear-gradient(#dce2ea 1px, transparent 1px),
    linear-gradient(90deg, #dce2ea 1px, transparent 1px);
  background-size: 20px 20px;
}

.scene-card-preview.kind-3d {
  color: #3e665f;
  background-color: #e7eeec;
  background-image:
    linear-gradient(#d5e0dd 1px, transparent 1px),
    linear-gradient(90deg, #d5e0dd 1px, transparent 1px);
}

.scene-card-preview svg {
  width: 34px;
  height: 34px;
  stroke-width: 1.4;
}

.scene-card-preview span {
  position: absolute;
  right: 8px;
  bottom: 7px;
  padding: 2px 5px;
  border: 1px solid rgba(74, 87, 106, 0.24);
  border-radius: 3px;
  color: #596579;
  background: rgba(255, 255, 255, 0.82);
  font-size: 9px;
  font-weight: 700;
}

.scene-card-body {
  min-width: 0;
  display: grid;
  gap: 9px;
  padding: 12px;
}

.scene-card-title {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.scene-card-title strong {
  min-width: 0;
  overflow: hidden;
  color: #20293a;
  font-size: 13px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scene-card-title span {
  flex: 0 0 auto;
  padding: 2px 5px;
  border-radius: 3px;
  color: #50617a;
  background: #eef2f7;
  font-size: 9px;
}

.scene-card-body p {
  min-height: 32px;
  margin: 0;
  display: -webkit-box;
  overflow: hidden;
  color: #6b7789;
  font-size: 11px;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.scene-card-body dl {
  min-width: 0;
  margin: 0;
  display: grid;
  gap: 5px;
}

.scene-card-body dl div {
  min-width: 0;
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr);
  gap: 7px;
}

.scene-card-body dt,
.scene-card-body dd {
  margin: 0;
  overflow: hidden;
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scene-card-body dt {
  color: #8a94a4;
}

.scene-card-body dd {
  color: #465267;
  font-family: Consolas, 'SFMono-Regular', monospace;
}

.scene-card-body footer {
  display: flex;
  align-items: center;
  gap: 9px;
  padding-top: 8px;
  border-top: 1px solid #edf0f4;
  color: #7a8596;
  font-size: 9px;
}

.scene-card-body footer span:last-child {
  margin-left: auto;
  font-family: Consolas, 'SFMono-Regular', monospace;
}

.scene-empty-state {
  min-height: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px;
  color: #6b7789;
  text-align: center;
}

.scene-empty-state > div {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  margin-bottom: 12px;
  border: 1px solid #d3dae5;
  border-radius: 7px;
  color: #50617a;
  background: #e9edf3;
}

.scene-empty-state > div.kind-3d {
  color: #3e665f;
  background: #e7eeec;
}

.scene-empty-state svg {
  width: 23px;
  height: 23px;
}

.scene-empty-state strong {
  color: #2d3748;
  font-size: 13px;
}

.scene-empty-state p {
  max-width: 360px;
  margin: 7px 0 0;
  font-size: 11px;
  line-height: 1.6;
}

:global(html.dark) .scene-artifacts-panel,
:global([data-theme='dark']) .scene-artifacts-panel {
  color: #eef2f7;
  background: #171a20;
}

:global(html.dark) .scene-artifacts-header,
:global([data-theme='dark']) .scene-artifacts-header,
:global(html.dark) .scene-card,
:global([data-theme='dark']) .scene-card {
  border-color: #3c434e;
  background: #20242b;
}

@media (max-width: 700px) {
  .scene-artifacts-header {
    padding: 0 8px 0 10px;
  }

  .scene-card-grid {
    grid-template-columns: minmax(0, 1fr);
    padding: 10px;
  }
}
</style>
