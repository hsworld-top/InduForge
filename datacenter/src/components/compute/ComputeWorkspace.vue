<template>
  <div class="compute-workspace">
    <aside class="compute-workspace__side">
      <div class="compute-workspace__section-title">任务模式</div>
      <button
        v-for="mode in computeTaskModes"
        :key="mode.id"
        type="button"
        class="compute-workspace__mode"
        :class="{ 'is-active': mode.id === activeMode }"
        @click="activeMode = mode.id"
      >
        <span>
          <strong>{{ mode.label }}</strong>
          <small>{{ mode.summary }}</small>
        </span>
        <em>{{ mode.badge }}</em>
      </button>

      <div class="compute-workspace__section-title is-spaced">触发方式</div>
      <button
        v-for="trigger in computeTriggerModes"
        :key="trigger.id"
        type="button"
        class="compute-workspace__trigger"
        :class="{ 'is-active': trigger.id === activeTrigger }"
        @click="activeTrigger = trigger.id"
      >
        <span>{{ trigger.label }}</span>
        <small>{{ trigger.badge }}</small>
      </button>
    </aside>

    <section class="compute-workspace__main">
      <div class="compute-workspace__hero">
        <div>
          <span class="compute-workspace__eyebrow">计算单元 / 脚本任务</span>
          <h2>计算不等于必须输出数据点</h2>
          <p>
            计算可以不输出数据点，可以在脚本内写库、发布 MQTT/Kafka、发
            HTTP 请求；也可以输出 calc.* 数据点，或两者混合。
          </p>
        </div>
        <div class="compute-workspace__hero-badge">
          {{ selectedMode?.label }}
        </div>
      </div>

      <div class="compute-workspace__mode-strip">
        <article
          v-for="mode in computeTaskModes"
          :key="mode.id"
          class="compute-workspace__mode-card"
          :class="{ 'is-active': mode.id === activeMode }"
        >
          <div>
            <strong>{{ mode.label }}</strong>
            <span>{{ mode.badge }}</span>
          </div>
          <p>{{ mode.summary }}</p>
        </article>
      </div>

      <div class="compute-workspace__script-card">
        <div class="compute-workspace__script-head">
          <div>
            <strong>脚本任务区</strong>
            <p>下方保留现有创建、运行、调试能力，不引入新的必需后端路径。</p>
          </div>
          <span>{{ selectedTrigger?.label }}</span>
        </div>
        <ComputeUnitPanel :project-id="projectId" />
      </div>
    </section>

    <aside class="compute-workspace__detail">
      <div class="compute-workspace__section-title">能力面板 / API</div>
      <div class="compute-workspace__capability-list">
        <article
          v-for="capability in computeCapabilities"
          :key="`${capability.category}-${capability.signature}`"
          class="compute-workspace__capability"
        >
          <div>
            <strong>{{ capability.title }}</strong>
            <code>{{ capability.signature }}</code>
          </div>
          <p>{{ capability.summary }}</p>
        </article>
      </div>

      <div class="compute-workspace__panel">
        <div class="compute-workspace__section-title">安全约束</div>
        <ul>
          <li
            v-for="constraint in computeSafetyConstraints"
            :key="constraint"
          >
            {{ constraint }}
          </li>
        </ul>
      </div>

      <div class="compute-workspace__panel">
        <div class="compute-workspace__section-title">调试提示</div>
        <ul>
          <li v-for="hint in computeDebugHints" :key="hint">{{ hint }}</li>
        </ul>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import ComputeUnitPanel from "@/components/compute/ComputeUnitPanel.vue";
import {
  computeCapabilities,
  computeDebugHints,
  computeSafetyConstraints,
  computeTaskModes,
  computeTriggerModes,
  type ComputeTaskModeId,
  type ComputeTriggerModeId,
} from "@/components/compute/computeCapabilities";

defineProps<{
  projectId: string;
}>();

const activeMode = ref<ComputeTaskModeId>("side-effect");
const activeTrigger = ref<ComputeTriggerModeId>("manual-debug");

const selectedMode = computed(() =>
  computeTaskModes.find((mode) => mode.id === activeMode.value),
);

const selectedTrigger = computed(() =>
  computeTriggerModes.find((trigger) => trigger.id === activeTrigger.value),
);
</script>

<style scoped>
.compute-workspace {
  --compute-paper: var(--dc-surface-raised);
  --compute-canvas: var(--dc-surface-subtle);
  --compute-line: var(--dc-border);
  --compute-ink: var(--dc-text);
  --compute-muted: var(--dc-text-secondary);
  --compute-accent: var(--dc-primary);
  --compute-soft: var(--dc-primary-soft);

  height: 100%;
  display: grid;
  grid-template-columns: minmax(220px, 280px) minmax(0, 1fr) minmax(300px, 360px);
  gap: 12px;
  color: var(--compute-ink);
}

.compute-workspace__side,
.compute-workspace__main,
.compute-workspace__detail {
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.compute-workspace__side,
.compute-workspace__detail {
  padding: 12px;
  overflow-y: auto;
}

.compute-workspace__main {
  display: flex;
  flex-direction: column;
  background: var(--compute-canvas);
}

.compute-workspace__section-title {
  margin-bottom: 12px;
  color: var(--compute-ink);
  font-size: 14px;
  font-weight: 700;
}

.compute-workspace__section-title.is-spaced {
  margin-top: 20px;
}

.compute-workspace__mode,
.compute-workspace__trigger {
  width: 100%;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  text-align: left;
}

.compute-workspace__mode {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 10px;
  margin-bottom: 8px;
  padding: 10px;
  border-radius: var(--dc-radius-sm);
}

.compute-workspace__mode strong,
.compute-workspace__mode small {
  display: block;
}

.compute-workspace__mode small {
  margin-top: 5px;
  color: var(--compute-muted);
  font-size: 12px;
  line-height: 1.5;
}

.compute-workspace__mode em,
.compute-workspace__trigger small,
.compute-workspace__hero-badge,
.compute-workspace__script-head span,
.compute-workspace__mode-card span {
  border-radius: 4px;
  background: var(--compute-soft);
  color: var(--dc-primary);
  font-size: 11px;
  font-style: normal;
  font-weight: 800;
  white-space: nowrap;
}

.compute-workspace__mode em {
  align-self: start;
  padding: 4px 8px;
}

.compute-workspace__mode.is-active,
.compute-workspace__trigger.is-active,
.compute-workspace__mode-card.is-active {
  border-color: var(--compute-accent);
  background: var(--dc-primary-soft);
  font-weight: 700;
}

.compute-workspace__trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
  padding: 9px 10px;
  border-radius: var(--dc-radius-sm);
  font-size: 13px;
  font-weight: 700;
}

.compute-workspace__trigger small {
  padding: 3px 7px;
}

.compute-workspace__hero {
  display: flex;
  justify-content: space-between;
  gap: 18px;
  padding: 16px 18px 14px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--compute-paper);
}

.compute-workspace__eyebrow {
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 700;
}

.compute-workspace__hero h2 {
  margin: 6px 0 8px;
  font-size: 22px;
  line-height: 1.2;
}

.compute-workspace__hero p {
  max-width: 780px;
  margin: 0;
  color: var(--compute-muted);
  font-size: 13px;
  line-height: 1.7;
}

.compute-workspace__hero-badge {
  align-self: start;
  padding: 7px 11px;
}

.compute-workspace__mode-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  padding: 12px;
}

.compute-workspace__mode-card {
  min-height: 94px;
  padding: 12px;
  border: 1px solid var(--compute-line);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
}

.compute-workspace__mode-card div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.compute-workspace__mode-card span {
  padding: 4px 8px;
}

.compute-workspace__mode-card p {
  margin: 8px 0 0;
  color: var(--compute-muted);
  font-size: 12px;
  line-height: 1.6;
}

.compute-workspace__script-card {
  min-height: 0;
  margin: 0 12px 12px;
  overflow: hidden;
  display: flex;
  flex: 1;
  flex-direction: column;
  border: 1px solid var(--compute-line);
  border-radius: var(--dc-radius-md);
  background: var(--compute-paper);
}

.compute-workspace__script-head {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.compute-workspace__script-head strong {
  font-size: 14px;
}

.compute-workspace__script-head p {
  margin: 4px 0 0;
  color: var(--compute-muted);
  font-size: 12px;
}

.compute-workspace__script-head span {
  align-self: start;
  padding: 5px 9px;
}

.compute-workspace__script-card :deep(.compute-panel) {
  padding: 16px;
}

.compute-workspace__capability-list {
  display: grid;
  gap: 8px;
}

.compute-workspace__capability,
.compute-workspace__panel {
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-subtle);
}

.compute-workspace__capability {
  padding: 10px;
}

.compute-workspace__capability strong,
.compute-workspace__capability code {
  display: block;
}

.compute-workspace__capability strong {
  font-size: 13px;
}

.compute-workspace__capability code {
  margin-top: 4px;
  color: var(--dc-primary);
  font-size: 11px;
  white-space: normal;
  word-break: break-all;
}

.compute-workspace__capability p,
.compute-workspace__panel li {
  color: var(--compute-muted);
  font-size: 12px;
  line-height: 1.55;
}

.compute-workspace__capability p {
  margin: 6px 0 0;
}

.compute-workspace__panel {
  margin-top: 12px;
  padding: 12px;
}

.compute-workspace__panel ul {
  margin: 0;
  padding-left: 18px;
}

.compute-workspace__panel li + li {
  margin-top: 7px;
}

@media (max-width: 1260px) {
  .compute-workspace {
    grid-template-columns: 230px minmax(0, 1fr);
  }

  .compute-workspace__detail {
    display: none;
  }
}

@media (max-width: 820px) {
  .compute-workspace {
    grid-template-columns: 1fr;
    overflow-y: auto;
  }

  .compute-workspace__side,
  .compute-workspace__main {
    min-height: 360px;
  }

  .compute-workspace__mode-strip {
    grid-template-columns: 1fr;
  }

  .compute-workspace__hero {
    display: block;
  }

  .compute-workspace__hero-badge {
    display: inline-flex;
    margin-top: 12px;
  }
}
</style>
