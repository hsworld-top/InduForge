<template>
  <div class="alarm-workspace">
    <aside class="alarm-workspace__side">
      <div class="alarm-workspace__section-title">规则分组</div>
      <button
        v-for="group in ruleGroups"
        :key="group.name"
        type="button"
        class="alarm-workspace__group"
        :class="{ 'is-active': group.name === activeGroup }"
        @click="activeGroup = group.name"
      >
        <span>{{ group.name }}</span>
        <em>{{ group.count }}</em>
      </button>

      <div class="alarm-workspace__section-title is-spaced">规则类型</div>
      <button
        v-for="type in alarmRuleTypes"
        :key="type.id"
        type="button"
        class="alarm-workspace__type"
        :class="{ 'is-active': type.id === draft.ruleType }"
        @click="selectType(type.id)"
      >
        <span>
          <strong>{{ type.label }}</strong>
          <small>{{ type.summary }}</small>
        </span>
        <em>{{ type.badge }}</em>
      </button>
    </aside>

    <section class="alarm-workspace__main">
      <div class="alarm-workspace__hero">
        <div>
          <span class="alarm-workspace__eyebrow">报警单元 / 规则构建</span>
          <h2>只构建规则，不管理运行态事件</h2>
          <p>
            数据中心负责目标数据点、质量条件、阈值表达式、抑制恢复策略和消息模板；
            节点侧运行态负责执行规则、产生事件和处理确认流程。
          </p>
        </div>
        <span class="alarm-workspace__hero-badge">契约预览</span>
      </div>

      <div class="alarm-workspace__toolbar">
        <div>
          <strong>当前规则：{{ selectedRule.name }}</strong>
          <span>{{ selectedRule.targetPath }}</span>
        </div>
        <div class="alarm-workspace__toolbar-actions">
          <button type="button">保存草稿</button>
          <button type="button">样本试算</button>
          <button type="button" class="is-primary">校验契约</button>
        </div>
      </div>

      <div class="alarm-workspace__builder">
        <article class="alarm-workspace__form-card">
          <div class="alarm-workspace__card-title">
            <span>规则构建器</span>
            <em>{{ selectedRuleType?.badge }}</em>
          </div>

          <div class="alarm-workspace__form-grid">
            <label class="alarm-workspace__field">
              <span>目标数据点</span>
              <input v-model="draft.targetPath" type="text" />
            </label>
            <label class="alarm-workspace__field">
              <span>质量条件</span>
              <input v-model="draft.qualityCondition" type="text" />
            </label>
            <label class="alarm-workspace__field">
              <span>在线条件</span>
              <input v-model="draft.onlineCondition" type="text" />
            </label>
            <label class="alarm-workspace__field">
              <span>规则类型</span>
              <select v-model="draft.ruleType">
                <option
                  v-for="type in alarmRuleTypes"
                  :key="type.id"
                  :value="type.id"
                >
                  {{ type.label }}
                </option>
              </select>
            </label>
            <label class="alarm-workspace__field">
              <span>阈值或表达式</span>
              <input v-model="draft.trigger" type="text" />
            </label>
            <label class="alarm-workspace__field">
              <span>严重级别</span>
              <select v-model="draft.severity">
                <option
                  v-for="severity in alarmSeverities"
                  :key="severity.id"
                  :value="severity.id"
                >
                  {{ severity.label }}
                </option>
              </select>
            </label>
            <label class="alarm-workspace__field">
              <span>抑制策略</span>
              <input v-model="draft.suppression" type="text" />
            </label>
            <label class="alarm-workspace__field">
              <span>恢复策略</span>
              <input v-model="draft.recovery" type="text" />
            </label>
            <label class="alarm-workspace__field is-wide">
              <span>消息模板</span>
              <textarea v-model="draft.messageTemplate" rows="3" />
            </label>
            <label class="alarm-workspace__field is-wide">
              <span>运行态节点契约</span>
              <input v-model="draft.runtimeNodeContract" type="text" />
            </label>
          </div>
        </article>

        <article class="alarm-workspace__rule-list">
          <div class="alarm-workspace__card-title">
            <span>规则清单</span>
            <em>{{ filteredRules.length }} 条</em>
          </div>
          <button
            v-for="rule in filteredRules"
            :key="rule.id"
            type="button"
            class="alarm-workspace__rule-row"
            :class="{ 'is-active': rule.id === selectedRuleId }"
            @click="selectedRuleId = rule.id"
          >
            <span>
              <strong>{{ rule.name }}</strong>
              <small>{{ rule.targetPath }}</small>
            </span>
            <em :class="`is-${rule.contractStatus}`">{{ rule.contractStatus }}</em>
          </button>
        </article>
      </div>
    </section>

    <aside class="alarm-workspace__detail">
      <div class="alarm-workspace__section-title">运行态契约预览</div>
      <pre class="alarm-workspace__contract">{{ contractPreview }}</pre>

      <div class="alarm-workspace__panel">
        <div class="alarm-workspace__section-title">校验提示</div>
        <div
          v-for="field in alarmContractFields"
          :key="field.key"
          class="alarm-workspace__contract-row"
        >
          <strong>{{ field.label }}</strong>
          <span>{{ field.required ? "必填" : "可选" }}</span>
          <p>{{ field.summary }}</p>
        </div>
      </div>

      <div class="alarm-workspace__panel is-boundary">
        <div class="alarm-workspace__section-title">职责边界</div>
        <p v-for="note in alarmRuntimeBoundaryNotes" :key="note">
          {{ note }}
        </p>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import {
  alarmContractFields,
  alarmRuntimeBoundaryNotes,
  alarmRuleSamples,
  alarmRuleTypes,
  alarmSeverities,
  type AlarmRuleSample,
  type AlarmRuleTypeId,
} from "@/components/alarm/alarmRuleModel";

const props = defineProps<{
  projectId: string;
}>();

const selectedRuleId = ref(alarmRuleSamples[0].id);
const activeGroup = ref(alarmRuleSamples[0].group);

const selectedRule = computed(
  () =>
    alarmRuleSamples.find((rule) => rule.id === selectedRuleId.value) ??
    alarmRuleSamples[0],
);

const draft = reactive<AlarmRuleSample>({ ...selectedRule.value });

const ruleGroups = computed(() =>
  alarmRuleSamples.reduce<Array<{ name: string; count: number }>>((groups, rule) => {
    const group = groups.find((item) => item.name === rule.group);
    if (group) {
      group.count += 1;
    } else {
      groups.push({ name: rule.group, count: 1 });
    }
    return groups;
  }, []),
);

const filteredRules = computed(() =>
  alarmRuleSamples.filter((rule) => rule.group === activeGroup.value),
);

const selectedRuleType = computed(() =>
  alarmRuleTypes.find((type) => type.id === draft.ruleType),
);

const selectedSeverity = computed(() =>
  alarmSeverities.find((severity) => severity.id === draft.severity),
);

const contractPreview = computed(() =>
  JSON.stringify(
    {
      projectId: props.projectId,
      ruleId: draft.id,
      targetPath: draft.targetPath,
      conditions: {
        quality: draft.qualityCondition,
        online: draft.onlineCondition,
        trigger: draft.trigger,
        recovery: draft.recovery,
      },
      ruleType: draft.ruleType,
      severity: selectedSeverity.value?.level,
      suppression: draft.suppression,
      messageTemplate: draft.messageTemplate,
      runtimeNodeContract: draft.runtimeNodeContract,
      executionOwner: "runtime-node",
    },
    null,
    2,
  ),
);

const selectType = (typeId: AlarmRuleTypeId) => {
  draft.ruleType = typeId;
};

watch(selectedRule, (rule) => {
  Object.assign(draft, rule);
});

watch(activeGroup, (group) => {
  const firstRule = alarmRuleSamples.find((rule) => rule.group === group);
  if (firstRule) {
    selectedRuleId.value = firstRule.id;
  }
});
</script>

<style scoped>
.alarm-workspace {
  --alarm-paper: var(--dc-surface-raised);
  --alarm-canvas: var(--dc-surface-subtle);
  --alarm-line: var(--dc-border);
  --alarm-ink: var(--dc-text);
  --alarm-muted: var(--dc-text-secondary);
  --alarm-accent: var(--dc-primary);
  --alarm-accent-dark: var(--dc-primary);
  --alarm-soft: var(--dc-primary-soft);

  height: 100%;
  display: grid;
  grid-template-columns: minmax(220px, 280px) minmax(0, 1fr) minmax(310px, 380px);
  gap: 12px;
  color: var(--alarm-ink);
}

.alarm-workspace__side,
.alarm-workspace__main,
.alarm-workspace__detail {
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.alarm-workspace__side,
.alarm-workspace__detail {
  padding: 12px;
  overflow-y: auto;
}

.alarm-workspace__main {
  display: flex;
  flex-direction: column;
  background: var(--alarm-canvas);
}

.alarm-workspace__section-title {
  margin-bottom: 12px;
  color: var(--alarm-ink);
  font-size: 14px;
  font-weight: 700;
}

.alarm-workspace__section-title.is-spaced {
  margin-top: 20px;
}

.alarm-workspace__group,
.alarm-workspace__type,
.alarm-workspace__rule-row,
.alarm-workspace__toolbar-actions button {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
}

.alarm-workspace__group,
.alarm-workspace__type {
  width: 100%;
  margin-bottom: 8px;
  border-radius: var(--dc-radius-sm);
  text-align: left;
}

.alarm-workspace__group {
  display: flex;
  justify-content: space-between;
  padding: 10px 12px;
  font-size: 13px;
  font-weight: 800;
}

.alarm-workspace__group em,
.alarm-workspace__type em,
.alarm-workspace__hero-badge,
.alarm-workspace__card-title em,
.alarm-workspace__rule-row em {
  border-radius: 4px;
  background: var(--alarm-soft);
  color: var(--alarm-accent-dark);
  font-size: 11px;
  font-style: normal;
  font-weight: 700;
}

.alarm-workspace__group em,
.alarm-workspace__type em,
.alarm-workspace__hero-badge,
.alarm-workspace__card-title em {
  padding: 4px 8px;
}

.alarm-workspace__type {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 10px;
  padding: 12px;
}

.alarm-workspace__type strong,
.alarm-workspace__type small,
.alarm-workspace__rule-row strong,
.alarm-workspace__rule-row small {
  display: block;
}

.alarm-workspace__type small,
.alarm-workspace__rule-row small,
.alarm-workspace__hero p,
.alarm-workspace__toolbar span,
.alarm-workspace__contract-row p,
.alarm-workspace__panel p {
  color: var(--alarm-muted);
  font-size: 12px;
  line-height: 1.55;
}

.alarm-workspace__type small,
.alarm-workspace__rule-row small {
  margin-top: 5px;
}

.alarm-workspace__group.is-active,
.alarm-workspace__type.is-active,
.alarm-workspace__rule-row.is-active {
  border-color: var(--alarm-accent);
  background: var(--alarm-soft);
  font-weight: 700;
}

.alarm-workspace__hero {
  display: flex;
  justify-content: space-between;
  gap: 18px;
  padding: 16px 18px 14px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--alarm-paper);
}

.alarm-workspace__eyebrow {
  color: var(--alarm-accent-dark);
  font-size: 12px;
  font-weight: 900;
}

.alarm-workspace__hero h2 {
  margin: 6px 0 8px;
  font-size: 22px;
  line-height: 1.2;
}

.alarm-workspace__hero p {
  max-width: 760px;
  margin: 0;
  font-size: 13px;
  line-height: 1.7;
}

.alarm-workspace__hero-badge {
  align-self: start;
}

.alarm-workspace__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 12px;
}

.alarm-workspace__toolbar strong,
.alarm-workspace__toolbar span {
  display: block;
}

.alarm-workspace__toolbar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}

.alarm-workspace__toolbar-actions button {
  padding: 8px 12px;
  border-radius: var(--dc-radius-sm);
  font-size: 12px;
  font-weight: 700;
}

.alarm-workspace__toolbar-actions button.is-primary {
  border-color: var(--alarm-accent);
  background: var(--alarm-accent);
  color: var(--dc-surface-raised);
}

.alarm-workspace__builder {
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(240px, 320px);
  gap: 12px;
  padding: 0 12px 12px;
  flex: 1;
}

.alarm-workspace__form-card,
.alarm-workspace__rule-list,
.alarm-workspace__panel {
  border: 1px solid var(--alarm-line);
  border-radius: var(--dc-radius-md);
  background: var(--alarm-paper);
}

.alarm-workspace__form-card,
.alarm-workspace__rule-list {
  min-height: 0;
  overflow: hidden;
}

.alarm-workspace__card-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
  font-size: 14px;
  font-weight: 700;
}

.alarm-workspace__form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  padding: 14px;
  overflow-y: auto;
}

.alarm-workspace__field {
  display: grid;
  gap: 6px;
}

.alarm-workspace__field.is-wide {
  grid-column: 1 / -1;
}

.alarm-workspace__field span {
  color: var(--alarm-muted);
  font-size: 12px;
  font-weight: 900;
}

.alarm-workspace__field input,
.alarm-workspace__field select,
.alarm-workspace__field textarea {
  width: 100%;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--alarm-ink);
  font-size: 13px;
  outline: none;
}

.alarm-workspace__field input,
.alarm-workspace__field select {
  height: 38px;
  padding: 0 12px;
}

.alarm-workspace__field textarea {
  resize: vertical;
  padding: 10px 12px;
}

.alarm-workspace__rule-list {
  overflow-y: auto;
}

.alarm-workspace__rule-row {
  width: calc(100% - 24px);
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 10px;
  margin: 10px 12px 0;
  padding: 12px;
  border-radius: 8px;
  text-align: left;
}

.alarm-workspace__rule-row em {
  align-self: start;
  padding: 4px 8px;
}

.alarm-workspace__rule-row em.is-valid {
  background: #dceee7;
  color: #27664c;
}

.alarm-workspace__rule-row em.is-pending {
  background: #faecd0;
  color: #8a5b17;
}

.alarm-workspace__rule-row em.is-broken {
  background: #f6d8d1;
  color: #8f3322;
}

.alarm-workspace__contract {
  margin: 0 0 12px;
  padding: 12px;
  overflow-x: auto;
  border: 1px solid var(--alarm-line);
  border-radius: 8px;
  background: #111827;
  color: #e5e7eb;
  font-size: 11px;
  line-height: 1.55;
}

.alarm-workspace__panel {
  margin-top: 12px;
  padding: 12px;
}

.alarm-workspace__contract-row {
  padding: 10px 0;
  border-top: 1px solid var(--dc-border);
}

.alarm-workspace__contract-row:first-of-type {
  border-top: 0;
}

.alarm-workspace__contract-row strong {
  font-size: 13px;
}

.alarm-workspace__contract-row span {
  float: right;
  color: var(--alarm-accent-dark);
  font-size: 11px;
  font-weight: 900;
}

.alarm-workspace__contract-row p,
.alarm-workspace__panel p {
  margin: 5px 0 0;
}

.alarm-workspace__panel.is-boundary {
  background: var(--dc-accent-soft);
}

@media (max-width: 1280px) {
  .alarm-workspace {
    grid-template-columns: 230px minmax(0, 1fr);
  }

  .alarm-workspace__detail {
    display: none;
  }
}

@media (max-width: 920px) {
  .alarm-workspace,
  .alarm-workspace__builder {
    grid-template-columns: 1fr;
    overflow-y: auto;
  }

  .alarm-workspace__hero,
  .alarm-workspace__toolbar {
    display: block;
  }

  .alarm-workspace__toolbar-actions {
    justify-content: flex-start;
    margin-top: 10px;
  }

  .alarm-workspace__form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
