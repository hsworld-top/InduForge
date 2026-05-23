<template>
  <form class="alarm-policy-form" @submit.prevent>
    <section class="alarm-policy-form__section">
      <header>
        <div>
          <strong>{{ draft.mode === "derived" ? "输入点" : "目标点" }}</strong>
          <button
            type="button"
            class="alarm-policy-form__hint"
            :title="pointSectionTip"
            :aria-label="draft.mode === 'derived' ? '输入点说明' : '目标点说明'"
          >
            ?
          </button>
        </div>
        <span>{{ pointCountText }}</span>
      </header>
      <div
        v-if="draft.mode === 'per_target' && coverageSummary.count > 0"
        class="alarm-policy-form__coverage"
      >
        <span>其中 {{ coverageSummary.count }} 个点位已被其他策略引用</span>
        <button
          type="button"
          class="alarm-policy-form__hint"
          :title="coverageSummary.tip"
          aria-label="目标点引用说明"
        >
          ?
        </button>
      </div>
      <div
        v-else-if="draft.mode === 'per_target' && coverageLoading"
        class="alarm-policy-form__coverage is-loading"
      >
        正在检查目标点引用
      </div>
      <AlarmPointPicker
        v-if="draft.mode === 'derived'"
        :project-id="projectId"
        :items="draft.inputs"
        with-key
        @update="updatePoints"
      />
      <div
        v-else-if="draft.targets.length"
        class="alarm-policy-form__target-chips"
      >
        <el-tag
          v-for="target in draft.targets"
          :key="target.datapointId"
          class="alarm-policy-form__target-chip"
          type="primary"
          effect="light"
          closable
          disable-transitions
          :title="`${target.name || target.path} · ${target.path} · ${target.dataType || '-'}`"
          @close="removeTarget(target.datapointId)"
        >
          <strong>{{ target.name || target.path }}</strong>
          <em
            v-if="targetCoverageCount(target) > 0"
            class="alarm-policy-form__target-warning"
            :title="targetCoverageTip(target)"
          >
            {{ targetCoverageCount(target) }}
          </em>
        </el-tag>
      </div>
      <div v-else class="alarm-policy-form__empty">
        暂未选择目标点，请从顶部“数据点变量”选择
      </div>
    </section>

    <section v-if="draft.mode === 'derived'" class="alarm-policy-form__section">
      <header>
        <strong>计算表达式</strong>
        <span title="条件集判断的是这个计算表达式的结果">
          结果用于条件判断
        </span>
      </header>
      <textarea
        :value="draft.derivedExpression"
        class="alarm-policy-form__code"
        rows="4"
        spellcheck="false"
        placeholder="例如：(tempA + tempB) / 2"
        @input="updateField('derivedExpression', inputValue($event))"
      />
    </section>

    <section class="alarm-policy-form__condition-scroll" aria-label="报警配置区域">
      <AlarmConditionMatrix
        :conditions="draft.conditions"
        :mode="draft.mode"
        :targets="draft.targets"
        @update="updateField('conditions', $event)"
      />
    </section>
  </form>
</template>

<script setup lang="ts">
import type {
  AlarmInputRef,
  AlarmPolicyCoverage,
  AlarmTargetRef,
} from "@/api/schemas/alarm.schema";
import { computed } from "vue";
import type { AlarmPolicyDraft } from "@/components/alarm/alarmPolicyModel";
import AlarmConditionMatrix from "./AlarmConditionMatrix.vue";
import AlarmPointPicker from "./AlarmPointPicker.vue";

const props = defineProps<{
  projectId: string;
  draft: AlarmPolicyDraft;
  coverages?: Record<string, AlarmPolicyCoverage>;
  coverageLoading?: boolean;
}>();

const emit = defineEmits<{
  update: [patch: Partial<AlarmPolicyDraft>];
}>();

const inputValue = (event: Event) =>
  (event.target as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement).value;

const pointCountText = computed(() =>
  props.draft.mode === "derived"
    ? `${props.draft.inputs.length} 个输入点`
    : `${props.draft.targets.length} 个点位`,
);

const pointSectionTip = computed(() =>
  props.draft.mode === "derived"
    ? "计算结果报警：这些输入点用于上方计算表达式，条件集判断计算结果。"
    : "统一模板报警：同一套条件会分别应用到这些目标点。若点位也被其他策略引用，保存后会同时生效。",
);

const coverageKey = (target: AlarmTargetRef) =>
  String(target.datapointId || target.path);

const targetCoveragePolicies = (target: AlarmTargetRef) =>
  props.coverages?.[coverageKey(target)]?.policies || [];

const targetCoverageCount = (target: AlarmTargetRef) =>
  targetCoveragePolicies(target).length;

const targetCoverageTip = (target: AlarmTargetRef) => {
  const policies = targetCoveragePolicies(target);
  if (!policies.length) {
    return "";
  }
  return policies
    .map((policy) => {
      const mode =
        policy.mode === "derived" ? "计算结果报警" : "统一模板报警";
      const status = policy.effectiveEnabled ? "生效中" : "未生效";
      return `${target.name || target.path} 已被「${policy.name}」引用（${mode}，${status}），保存后会同时生效`;
    })
    .join("\n");
};

const coverageSummary = computed(() => {
  const targets = props.draft.mode === "per_target" ? props.draft.targets : [];
  const covered = targets.filter((target) => targetCoverageCount(target) > 0);
  return {
    count: covered.length,
    tip: covered.map(targetCoverageTip).filter(Boolean).join("\n"),
  };
});

const updateField = <K extends keyof AlarmPolicyDraft>(
  field: K,
  value: AlarmPolicyDraft[K],
) => {
  emit("update", { [field]: value, dirty: true } as Partial<AlarmPolicyDraft>);
};

const updatePoints = (items: Array<AlarmInputRef | AlarmTargetRef>) => {
  if (props.draft.mode === "derived") {
    updateField("inputs", items as AlarmInputRef[]);
    return;
  }
  updateField("targets", items as AlarmTargetRef[]);
};

const removeTarget = (datapointId: string) => {
  updateField(
    "targets",
    props.draft.targets.filter((target) => target.datapointId !== datapointId),
  );
};
</script>

<style scoped>
.alarm-policy-form {
  min-height: 0;
  height: 100%;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 14px;
  overflow: hidden;
}

.alarm-policy-form:has(.alarm-policy-form__section + .alarm-policy-form__section) {
  grid-template-rows: auto auto minmax(0, 1fr);
}

.alarm-policy-form__section {
  min-height: 0;
  display: grid;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.alarm-policy-form__condition-scroll {
  min-height: 0;
  overflow: auto;
  padding-right: 2px;
  scrollbar-gutter: stable;
}

.alarm-policy-form__section header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.alarm-policy-form__section header > div {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.alarm-policy-form__section strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-policy-form__section header span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-policy-form__hint {
  width: 18px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  cursor: help;
  font-family: inherit;
  font-size: 11px;
  font-weight: 800;
  padding: 0;
}

.alarm-policy-form__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.alarm-policy-form label {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.alarm-policy-form label.is-wide {
  grid-column: 1 / -1;
}

.alarm-policy-form label.is-switch {
  grid-template-columns: auto 1fr;
  align-items: center;
  align-content: end;
  gap: 8px;
  min-height: 34px;
}

.alarm-policy-form label > span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.alarm-policy-form input,
.alarm-policy-form select,
.alarm-policy-form textarea {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font-family: inherit;
  font-size: 13px;
  outline: none;
}

.alarm-policy-form input:focus,
.alarm-policy-form select:focus,
.alarm-policy-form textarea:focus {
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.1);
}

.alarm-policy-form input,
.alarm-policy-form select {
  height: 34px;
  padding: 0 9px;
}

.alarm-policy-form input[type="checkbox"] {
  width: 15px;
  height: 15px;
  padding: 0;
  accent-color: var(--dc-primary);
}

.alarm-policy-form textarea {
  padding: 9px;
  line-height: 1.5;
  resize: vertical;
}

.alarm-policy-form__code {
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

.alarm-policy-form__target-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.alarm-policy-form__coverage {
  min-height: 28px;
  display: inline-flex;
  align-items: center;
  justify-self: start;
  gap: 6px;
  padding: 0 8px;
  border: 1px solid rgba(245, 158, 11, 0.28);
  border-radius: var(--dc-radius-sm);
  background: rgba(245, 158, 11, 0.08);
  color: #92400e;
  font-size: 12px;
  font-weight: 700;
}

.alarm-policy-form__coverage.is-loading {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-weight: 600;
}

.alarm-policy-form__target-chip :deep(.el-tag__content) {
  max-width: min(220px, 100%);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-policy-form__target-chip strong {
  font-size: 13px;
  font-weight: 500;
}

.alarm-policy-form__target-warning {
  min-width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-left: 6px;
  border-radius: 999px;
  background: rgba(245, 158, 11, 0.16);
  color: #92400e;
  font-size: 11px;
  font-style: normal;
  font-weight: 800;
  line-height: 1;
}

.alarm-policy-form__target-chip.el-tag {
  --el-tag-bg-color: rgba(64, 158, 255, 0.1);
  --el-tag-border-color: rgba(64, 158, 255, 0.22);
  --el-tag-text-color: var(--el-color-primary);
  height: 30px;
  border-radius: 4px;
  padding: 0 8px 0 10px;
  font-size: 13px;
}

.alarm-policy-form__target-chip :deep(.el-tag__close) {
  margin-left: 7px;
}

.alarm-policy-form__empty {
  padding: 12px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 13px;
}

@media (max-width: 720px) {
  .alarm-policy-form__grid {
    grid-template-columns: 1fr;
  }

  .alarm-policy-form__target-chip :deep(.el-tag__content) {
    max-width: 160px;
  }
}
</style>
