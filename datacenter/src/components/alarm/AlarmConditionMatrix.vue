<template>
  <section class="alarm-condition-matrix">
    <header class="alarm-condition-matrix__header">
      <div>
        <strong>条件集</strong>
        <span>{{ summaryText }}</span>
        <button
          type="button"
          class="alarm-condition-matrix__hint"
          :title="conditionScopeTip"
          aria-label="条件集说明"
        >
          ?
        </button>
      </div>
      <button type="button" @click="openCreateDialog">新增条件</button>
    </header>

    <div v-if="!conditions.length" class="alarm-condition-matrix__empty">
      尚未配置条件，点击“新增条件”选择报警类型。
    </div>

    <template v-else>
      <section v-if="showThresholdGroup" class="alarm-condition-matrix__group">
        <header>
          <div>
            <strong>阈值报警</strong>
            <span title="同一策略中高高限、高限、低限、低低限每种最多配置一个">
              每种限值最多一个
            </span>
          </div>
          <button
            v-if="availableThresholdTypes.length"
            type="button"
            class="is-ghost"
            @click="openCreateDialog('threshold')"
          >
            添加阈值
          </button>
        </header>

        <div class="alarm-condition-matrix__thresholds">
          <article
            v-for="slot in thresholdSlots"
            :key="slot.value"
            class="alarm-condition-matrix__threshold"
            :class="{ 'is-empty': !slot.condition, 'is-disabled': slot.condition && !slot.condition.isEnabled }"
          >
            <div>
              <strong>{{ slot.label }}</strong>
              <span v-if="slot.condition">{{ describeCondition(slot.condition) }}</span>
              <span v-else>未配置</span>
            </div>
            <template v-if="slot.condition">
              <em>{{ severityLabel(slot.condition.severity) }}</em>
              <div class="alarm-condition-matrix__actions">
                <button
                  type="button"
                  :title="slot.condition.isEnabled ? '停用' : '启用'"
                  @click="patchCondition(slot.condition.id, { isEnabled: !slot.condition.isEnabled })"
                >
                  {{ slot.condition.isEnabled ? "停用" : "启用" }}
                </button>
                <button type="button" title="编辑" @click="openEditDialog(slot.condition)">
                  编辑
                </button>
                <button
                  type="button"
                  class="is-danger"
                  title="删除"
                  @click="removeCondition(slot.condition.id)"
                >
                  删除
                </button>
              </div>
            </template>
            <button
              v-else
              type="button"
              class="is-ghost"
              @click="openCreateDialog('threshold', slot.value)"
            >
              配置
            </button>
          </article>
        </div>
      </section>

      <section
        v-for="group in conditionGroups"
        :key="group.kind"
        class="alarm-condition-matrix__group"
      >
        <header>
          <div>
            <strong>{{ group.title }}</strong>
            <span>{{ group.description }}</span>
          </div>
        </header>

        <div class="alarm-condition-matrix__cards">
          <article
            v-for="condition in group.conditions"
            :key="condition.id"
            class="alarm-condition-matrix__card"
            :class="{ 'is-disabled': !condition.isEnabled }"
          >
            <div class="alarm-condition-matrix__card-main">
              <strong>{{ condition.name || conditionLabel(condition.type) }}</strong>
              <span>{{ describeCondition(condition) }}</span>
            </div>
            <em>{{ severityLabel(condition.severity) }}</em>
            <div class="alarm-condition-matrix__actions">
              <button
                type="button"
                :title="condition.isEnabled ? '停用' : '启用'"
                @click="patchCondition(condition.id, { isEnabled: !condition.isEnabled })"
              >
                {{ condition.isEnabled ? "停用" : "启用" }}
              </button>
              <button type="button" title="编辑" @click="openEditDialog(condition)">
                编辑
              </button>
              <button
                type="button"
                class="is-danger"
                title="删除"
                @click="removeCondition(condition.id)"
              >
                删除
              </button>
            </div>
          </article>
        </div>
      </section>
    </template>

    <DcDialog
      v-model="dialogVisible"
      :title="editingId ? '编辑条件' : '新增条件'"
      width="720px"
      body-max-height="calc(100vh - 220px)"
      @close="closeDialog"
    >
      <div v-if="draftCondition" class="alarm-condition-dialog">
        <section v-if="!editingId" class="alarm-condition-dialog__types">
          <button
            v-for="option in addOptions"
            :key="option.kind"
            type="button"
            :class="{ 'is-active': selectedKind === option.kind }"
            @click="selectKind(option.kind)"
          >
            <strong>{{ option.title }}</strong>
            <span>{{ option.description }}</span>
          </button>
        </section>

        <div class="alarm-condition-dialog__form">
          <label>
            <span>报警类型</span>
            <select
              v-if="selectedKind === 'threshold'"
              :value="draftCondition.type"
              @change="changeDialogType(inputValue($event) as AlarmConditionType)"
            >
              <option
                v-for="item in thresholdTypeOptions"
                :key="item.value"
                :value="item.value"
              >
                {{ item.label }}
              </option>
            </select>
            <input v-else :value="conditionLabel(draftCondition.type)" disabled />
          </label>

          <label>
            <span>名称</span>
            <input
              :value="draftCondition.name"
              type="text"
              @input="patchDraft({ name: inputValue($event) })"
            />
          </label>

          <label>
            <span>报警级别</span>
            <select
              :value="draftCondition.severity"
              @change="patchDraft({ severity: inputValue($event) as AlarmSeverity })"
            >
              <option v-for="item in severityOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </option>
            </select>
          </label>

          <label class="is-switch">
            <input
              type="checkbox"
              :checked="draftCondition.isEnabled"
              @change="patchDraft({ isEnabled: checked($event) })"
            />
            <span>开启条件</span>
          </label>

          <template v-if="isThresholdCondition(draftCondition.type)">
            <label>
              <span>触发值</span>
              <input
                :value="numberParam(draftCondition, 'limit')"
                type="number"
                step="any"
                @input="patchNumberDraftParam('limit', inputValue($event))"
              />
            </label>
            <label>
              <span>{{ recoveryLabel(draftCondition.type) }}</span>
              <input
                :value="numberParam(draftCondition, 'hysteresis')"
                type="number"
                step="any"
                @input="patchOptionalNumberDraftParam('hysteresis', inputValue($event))"
              />
            </label>
            <label>
              <span>持续多久触发 ms</span>
              <input
                :value="numberParam(draftCondition, 'durationMs')"
                type="number"
                min="0"
                step="1"
                @input="patchOptionalNumberDraftParam('durationMs', inputValue($event))"
              />
            </label>
          </template>

          <template v-else-if="draftCondition.type === 'rate_of_change'">
            <label>
              <span>判断方向</span>
              <select
                :value="textParam(draftCondition, 'direction') || 'up'"
                @change="patchDraftParams({ direction: inputValue($event) })"
              >
                <option value="up">上升过快</option>
                <option value="down">下降过快</option>
              </select>
            </label>
            <label>
              <span>变化超过</span>
              <input
                :value="numberParam(draftCondition, 'limit')"
                type="number"
                step="any"
                @input="patchNumberDraftParam('limit', inputValue($event))"
              />
            </label>
            <label>
              <span>统计时长 ms</span>
              <input
                :value="numberParam(draftCondition, 'windowMs')"
                type="number"
                min="0"
                step="1"
                @input="patchNumberDraftParam('windowMs', inputValue($event))"
              />
            </label>
          </template>

          <template v-else-if="draftCondition.type === 'deviation_high' || draftCondition.type === 'deviation_low'">
            <label>
              <span>偏差超过</span>
              <input
                :value="numberParam(draftCondition, 'limit')"
                type="number"
                step="any"
                @input="patchNumberDraftParam('limit', inputValue($event))"
              />
            </label>
          </template>

          <template v-else-if="draftCondition.type === 'bool_equal'">
            <label>
              <span>目标状态</span>
              <select
                :value="String(boolParam(draftCondition, 'expected'))"
                @change="patchDraftParams({ expected: inputValue($event) === 'true' })"
              >
                <option value="true">为 true 时报警</option>
                <option value="false">为 false 时报警</option>
              </select>
            </label>
          </template>

          <template v-else-if="isStringCondition(draftCondition.type)">
            <label class="is-wide">
              <span>{{ draftCondition.type === "string_regex" ? "正则表达式" : "匹配文本" }}</span>
              <input
                :value="draftCondition.type === 'string_regex' ? textParam(draftCondition, 'pattern') : textParam(draftCondition, 'expected')"
                type="text"
                @input="patchStringDraftParam(inputValue($event))"
              />
            </label>
          </template>

          <template v-else-if="draftCondition.type === 'cel'">
            <label class="is-wide">
              <span>判断表达式</span>
              <textarea
                :value="textParam(draftCondition, 'expression')"
                rows="4"
                spellcheck="false"
                placeholder="例如：value > 80 && quality == &quot;good&quot;"
                @input="patchDraftParams({ expression: inputValue($event) })"
              />
            </label>
          </template>
        </div>

        <p v-if="dialogError" class="alarm-condition-dialog__error">
          {{ dialogError }}
        </p>
      </div>

      <template #footer>
        <div class="alarm-condition-dialog__footer">
          <button type="button" class="is-ghost" @click="closeDialog">取消</button>
          <button type="button" @click="saveDialogCondition">确定</button>
        </div>
      </template>
    </DcDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type {
  AlarmCondition,
  AlarmConditionType,
  AlarmPolicyMode,
  AlarmSeverity,
  AlarmTargetRef,
} from "@/api/schemas/alarm.schema";
import DcDialog from "@/components/shared/DcDialog.vue";
import { createThresholdCondition } from "@/components/alarm/alarmPolicyModel";

type AddKind = "threshold" | "rate" | "deviation_high" | "deviation_low" | "bool" | "string_equal" | "string_not_equal" | "string_contains" | "string_regex" | "custom";

const props = withDefaults(
  defineProps<{
    conditions: AlarmCondition[];
    mode?: AlarmPolicyMode;
    targets?: AlarmTargetRef[];
  }>(),
  {
    mode: "per_target",
    targets: () => [],
  },
);

const emit = defineEmits<{
  update: [conditions: AlarmCondition[]];
}>();

const thresholdTypes = ["HH", "H", "L", "LL"] as const;
const thresholdLabels: Record<(typeof thresholdTypes)[number], string> = {
  HH: "高高限",
  H: "高限",
  L: "低限",
  LL: "低低限",
};

const conditionLabels: Record<AlarmConditionType, string> = {
  HH: "高高限",
  H: "高限",
  L: "低限",
  LL: "低低限",
  deviation_high: "高偏差",
  deviation_low: "低偏差",
  rate_of_change: "变化率",
  bool_equal: "状态判断",
  string_equal: "文本等于",
  string_not_equal: "文本不等于",
  string_contains: "文本包含",
  string_regex: "正则匹配",
  cel: "自定义表达式",
};

const severityOptions: Array<{ value: AlarmSeverity; label: string }> = [
  { value: "info", label: "提示" },
  { value: "warning", label: "警告" },
  { value: "major", label: "重要" },
  { value: "critical", label: "紧急" },
];

const dialogVisible = ref(false);
const editingId = ref("");
const selectedKind = ref<AddKind>("custom");
const draftCondition = ref<AlarmCondition | null>(null);
const dialogError = ref("");

const enabledCount = computed(() =>
  props.conditions.filter((condition) => condition.isEnabled).length,
);

const summaryText = computed(
  () => `${props.conditions.length} 项，${enabledCount.value} 项开启`,
);

const targetKind = computed(() => {
  if (props.mode === "derived") {
    return "number";
  }
  if (!props.targets.length) {
    return "unknown";
  }
  if (props.targets.every((target) => isNumericType(target.dataType))) {
    return "number";
  }
  if (props.targets.every((target) => isBooleanType(target.dataType))) {
    return "boolean";
  }
  if (props.targets.every((target) => isStringType(target.dataType))) {
    return "string";
  }
  if (props.targets.every((target) => isObjectType(target.dataType))) {
    return "object";
  }
  return "mixed";
});

const conditionScopeTip = computed(() => {
  if (props.mode === "derived") {
    return "计算结果报警：条件判断的是上方计算表达式的结果。";
  }
  if (!props.targets.length) {
    return "统一模板报警：选择目标点后，会按数据类型显示可用条件。";
  }
  if (targetKind.value === "number") {
    return "统一模板报警：同一套条件会分别应用到上方目标点。数值点支持阈值、变化率、偏差和判断表达式。";
  }
  if (targetKind.value === "boolean") {
    return "统一模板报警：同一套条件会分别应用到上方目标点。布尔点支持状态判断和判断表达式。";
  }
  if (targetKind.value === "string") {
    return "统一模板报警：同一套条件会分别应用到上方目标点。字符串点支持文本匹配、正则匹配和判断表达式。";
  }
  if (targetKind.value === "object") {
    return "统一模板报警：对象类型目标点仅支持判断表达式。";
  }
  return "统一模板报警：目标点类型不一致，仅建议使用判断表达式，或改用计算结果报警。";
});

const addOptions = computed<Array<{ kind: AddKind; title: string; description: string }>>(() => {
  if (targetKind.value === "number") {
    const options: Array<{ kind: AddKind; title: string; description: string }> = [];
    if (availableThresholdTypes.value.length) {
      options.push({ kind: "threshold", title: "阈值报警", description: "配置高高限、高限、低限、低低限" });
    }
    options.push(
      { kind: "rate", title: "变化率", description: "判断一段时间内上升或下降过快" },
      { kind: "deviation_high", title: "高偏差", description: "高于基准值一定幅度后报警" },
      { kind: "deviation_low", title: "低偏差", description: "低于基准值一定幅度后报警" },
      { kind: "custom", title: "判断表达式", description: "使用 value、quality 等变量表达复杂判断" },
    );
    return options;
  }
  if (targetKind.value === "boolean") {
    return [
      { kind: "bool", title: "状态判断", description: "为 true 或 false 时报警" },
      { kind: "custom", title: "判断表达式", description: "使用表达式处理特殊状态" },
    ];
  }
  if (targetKind.value === "string") {
    return [
      { kind: "string_equal", title: "文本等于", description: "完全等于指定文本时报警" },
      { kind: "string_not_equal", title: "文本不等于", description: "不等于指定文本时报警" },
      { kind: "string_contains", title: "文本包含", description: "包含指定片段时报警" },
      { kind: "string_regex", title: "正则匹配", description: "匹配正则表达式时报警" },
      { kind: "custom", title: "判断表达式", description: "使用表达式处理复杂文本判断" },
    ];
  }
  return [{ kind: "custom", title: "判断表达式", description: "适用于混合类型或对象类型目标点" }];
});

const thresholdConditions = computed(() =>
  props.conditions.filter((condition) => isThresholdCondition(condition.type)),
);

const showThresholdGroup = computed(() => thresholdConditions.value.length > 0);

const availableThresholdTypes = computed(() =>
  thresholdTypes.filter(
    (type) => !props.conditions.some((condition) => condition.type === type),
  ),
);

const thresholdTypeOptions = computed(() =>
  thresholdTypes
    .filter(
      (type) =>
        type === draftCondition.value?.type ||
        !props.conditions.some((condition) => condition.type === type),
    )
    .map((value) => ({ value, label: thresholdLabels[value] })),
);

const thresholdSlots = computed(() =>
  thresholdTypes.map((value) => ({
    value,
    label: thresholdLabels[value],
    condition: props.conditions.find((condition) => condition.type === value),
  })),
);

const conditionGroups = computed(() => {
  const nonThreshold = props.conditions.filter(
    (condition) => !isThresholdCondition(condition.type),
  );
  const groups = [
    {
      kind: "trend",
      title: "变化与偏差",
      description: "根据变化速度或基准偏差判断",
      conditions: nonThreshold.filter((condition) =>
        ["rate_of_change", "deviation_high", "deviation_low"].includes(condition.type),
      ),
    },
    {
      kind: "state",
      title: "状态与文本",
      description: "根据布尔状态或文本内容判断",
      conditions: nonThreshold.filter((condition) =>
        ["bool_equal", "string_equal", "string_not_equal", "string_contains", "string_regex"].includes(condition.type),
      ),
    },
    {
      kind: "custom",
      title: "判断表达式",
      description: "适合复杂逻辑或对象类型目标点",
      conditions: nonThreshold.filter((condition) => condition.type === "cel"),
    },
  ];
  return groups.filter((group) => group.conditions.length);
});

const update = (conditions: AlarmCondition[]) => emit("update", conditions);

const inputValue = (event: Event) =>
  (event.target as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement).value;

const checked = (event: Event) => (event.target as HTMLInputElement).checked;

const openCreateDialog = (kind?: AddKind, preferredThreshold?: AlarmConditionType) => {
  dialogError.value = "";
  editingId.value = "";
  const firstKind = kind ?? addOptions.value[0]?.kind ?? "custom";
  selectedKind.value = firstKind;
  draftCondition.value = createConditionForKind(firstKind, preferredThreshold);
  dialogVisible.value = true;
};

const openEditDialog = (condition: AlarmCondition) => {
  dialogError.value = "";
  editingId.value = condition.id;
  selectedKind.value = kindFromCondition(condition.type);
  draftCondition.value = cloneCondition(condition);
  dialogVisible.value = true;
};

const closeDialog = () => {
  dialogVisible.value = false;
  editingId.value = "";
  dialogError.value = "";
  draftCondition.value = null;
};

const selectKind = (kind: AddKind) => {
  selectedKind.value = kind;
  draftCondition.value = createConditionForKind(kind);
  dialogError.value = "";
};

const changeDialogType = (type: AlarmConditionType) => {
  const next = createThresholdCondition(type);
  draftCondition.value = {
    ...next,
    id: draftCondition.value?.id ?? next.id,
    isEnabled: draftCondition.value?.isEnabled ?? true,
    severity: draftCondition.value?.severity ?? next.severity,
  };
};

const saveDialogCondition = () => {
  if (!draftCondition.value) {
    return;
  }
  const condition = cloneCondition(draftCondition.value);
  const error = validateDraftCondition(condition);
  if (error) {
    dialogError.value = error;
    return;
  }
  if (editingId.value) {
    update(
      props.conditions.map((item) =>
        item.id === editingId.value ? condition : item,
      ),
    );
  } else {
    update([...props.conditions, condition]);
  }
  closeDialog();
};

const patchCondition = (id: string, patch: Partial<AlarmCondition>) => {
  update(
    props.conditions.map((condition) =>
      condition.id === id ? { ...condition, ...patch } : condition,
    ),
  );
};

const removeCondition = (id: string) => {
  update(props.conditions.filter((condition) => condition.id !== id));
};

const patchDraft = (patch: Partial<AlarmCondition>) => {
  if (!draftCondition.value) {
    return;
  }
  draftCondition.value = { ...draftCondition.value, ...patch };
};

const patchDraftParams = (patch: Record<string, unknown>) => {
  if (!draftCondition.value) {
    return;
  }
  draftCondition.value = {
    ...draftCondition.value,
    params: { ...draftCondition.value.params, ...patch },
  };
};

const patchNumberDraftParam = (field: string, raw: string) => {
  patchDraftParams({ [field]: raw === "" ? undefined : Number(raw) });
};

const patchOptionalNumberDraftParam = (field: string, raw: string) => {
  if (!draftCondition.value) {
    return;
  }
  const params = { ...draftCondition.value.params };
  if (raw === "") {
    delete params[field];
  } else {
    params[field] = Number(raw);
  }
  patchDraft({ params });
};

const patchStringDraftParam = (value: string) => {
  if (draftCondition.value?.type === "string_regex") {
    patchDraftParams({ pattern: value });
    return;
  }
  patchDraftParams({ expected: value });
};

const createConditionForKind = (
  kind: AddKind,
  preferredThreshold?: AlarmConditionType,
): AlarmCondition => {
  const thresholdType =
    preferredThreshold && isThresholdCondition(preferredThreshold)
      ? preferredThreshold
      : availableThresholdTypes.value[0] ?? "H";
  const typeMap: Record<AddKind, AlarmConditionType> = {
    threshold: thresholdType,
    rate: "rate_of_change",
    deviation_high: "deviation_high",
    deviation_low: "deviation_low",
    bool: "bool_equal",
    string_equal: "string_equal",
    string_not_equal: "string_not_equal",
    string_contains: "string_contains",
    string_regex: "string_regex",
    custom: "cel",
  };
  const fallbackKind = addOptions.value.some((option) => option.kind === kind)
    ? kind
    : "custom";
  return createThresholdCondition(typeMap[fallbackKind]);
};

const kindFromCondition = (type: AlarmConditionType): AddKind => {
  if (isThresholdCondition(type)) {
    return "threshold";
  }
  if (type === "rate_of_change") {
    return "rate";
  }
  if (type === "deviation_high" || type === "deviation_low") {
    return type;
  }
  if (type === "bool_equal") {
    return "bool";
  }
  if (isStringCondition(type)) {
    return type;
  }
  return "custom";
};

const validateDraftCondition = (condition: AlarmCondition) => {
  if (!condition.name.trim()) {
    return "请填写条件名称";
  }
  if (isThresholdCondition(condition.type) || condition.type === "rate_of_change" || condition.type === "deviation_high" || condition.type === "deviation_low") {
    if (!Number.isFinite(condition.params.limit)) {
      return "请填写有效的数值";
    }
  }
  if (condition.type === "rate_of_change" && !Number.isFinite(condition.params.windowMs)) {
    return "请填写统计时长";
  }
  if (condition.type === "cel" && !textParam(condition, "expression").trim()) {
    return "请填写判断表达式";
  }
  if (isStringCondition(condition.type)) {
    const value = condition.type === "string_regex"
      ? textParam(condition, "pattern")
      : textParam(condition, "expected");
    if (!value.trim()) {
      return condition.type === "string_regex" ? "请填写正则表达式" : "请填写匹配文本";
    }
  }
  if (isThresholdCondition(condition.type)) {
    const duplicate = props.conditions.some(
      (item) => item.id !== editingId.value && item.type === condition.type,
    );
    if (duplicate) {
      return `${conditionLabel(condition.type)} 已存在`;
    }
  }
  return "";
};

const describeCondition = (condition: AlarmCondition) => {
  if (condition.type === "HH" || condition.type === "H") {
    return `达到或超过 ${numberParam(condition, "limit") || "-"} 报警，${recoveryLabel(condition.type)} ${numberParam(condition, "hysteresis") || "0"}`;
  }
  if (condition.type === "L" || condition.type === "LL") {
    return `达到或低于 ${numberParam(condition, "limit") || "-"} 报警，${recoveryLabel(condition.type)} ${numberParam(condition, "hysteresis") || "0"}`;
  }
  if (condition.type === "rate_of_change") {
    const direction = textParam(condition, "direction") === "down" ? "下降过快" : "上升过快";
    return `${direction}，${numberParam(condition, "windowMs") || "-"} ms 内变化超过 ${numberParam(condition, "limit") || "-"}`;
  }
  if (condition.type === "deviation_high") {
    return `高于基准值超过 ${numberParam(condition, "limit") || "-"}`;
  }
  if (condition.type === "deviation_low") {
    return `低于基准值超过 ${numberParam(condition, "limit") || "-"}`;
  }
  if (condition.type === "bool_equal") {
    return boolParam(condition, "expected") ? "值为 true 时报警" : "值为 false 时报警";
  }
  if (condition.type === "string_equal") {
    return `等于 “${textParam(condition, "expected")}”`;
  }
  if (condition.type === "string_not_equal") {
    return `不等于 “${textParam(condition, "expected")}”`;
  }
  if (condition.type === "string_contains") {
    return `包含 “${textParam(condition, "expected")}”`;
  }
  if (condition.type === "string_regex") {
    return `匹配 /${textParam(condition, "pattern")}/`;
  }
  return textParam(condition, "expression") || "判断表达式";
};

const numberParam = (condition: AlarmCondition, field: string) => {
  const param = condition.params[field];
  return typeof param === "number" && Number.isFinite(param) ? String(param) : "";
};

const textParam = (condition: AlarmCondition, field: string) => {
  const param = condition.params[field];
  return typeof param === "string" ? param : "";
};

const boolParam = (condition: AlarmCondition, field: string) =>
  condition.params[field] === true;

const recoveryLabel = (type: AlarmConditionType) =>
  type === "H" || type === "HH" ? "回落多少后恢复" : "回升多少后恢复";

const severityLabel = (value: AlarmSeverity) =>
  severityOptions.find((item) => item.value === value)?.label ?? value;

const conditionLabel = (type: AlarmConditionType) => conditionLabels[type] ?? type;

const cloneCondition = (condition: AlarmCondition): AlarmCondition => ({
  ...condition,
  params: { ...condition.params },
});

const isThresholdCondition = (type: AlarmConditionType) =>
  (thresholdTypes as readonly string[]).includes(type);

const isStringCondition = (type: AlarmConditionType) =>
  ["string_equal", "string_not_equal", "string_contains", "string_regex"].includes(type);

const isNumericType = (dataType: string) =>
  ["number", "integer", "int", "float", "double", "decimal"].includes(
    dataType.trim().toLowerCase(),
  );

const isBooleanType = (dataType: string) =>
  ["boolean", "bool"].includes(dataType.trim().toLowerCase());

const isStringType = (dataType: string) =>
  ["string", "text"].includes(dataType.trim().toLowerCase());

const isObjectType = (dataType: string) =>
  ["object", "json"].includes(dataType.trim().toLowerCase());
</script>

<style scoped>
.alarm-condition-matrix {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.alarm-condition-matrix__header,
.alarm-condition-matrix__group header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.alarm-condition-matrix__header div,
.alarm-condition-matrix__group header div {
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.alarm-condition-matrix strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-condition-matrix span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-condition-matrix button,
.alarm-condition-dialog button,
.alarm-condition-dialog input,
.alarm-condition-dialog select,
.alarm-condition-dialog textarea {
  font-family: inherit;
}

.alarm-condition-matrix__header > button,
.alarm-condition-matrix__group header button,
.alarm-condition-matrix__actions button,
.alarm-condition-matrix__threshold > button,
.alarm-condition-dialog__footer button {
  height: 30px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
  padding: 0 10px;
}

.alarm-condition-matrix__header > button,
.alarm-condition-dialog__footer button:not(.is-ghost) {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: #fff;
}

.alarm-condition-matrix__empty {
  display: grid;
  gap: 4px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.alarm-condition-matrix__empty {
  border-style: dashed;
  color: var(--dc-text-muted);
  font-size: 13px;
}

.alarm-condition-matrix__hint {
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
  font-size: 11px;
  font-weight: 800;
  padding: 0;
}

.alarm-condition-matrix__group {
  display: grid;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.alarm-condition-matrix__thresholds,
.alarm-condition-matrix__cards {
  display: grid;
  gap: 8px;
}

.alarm-condition-matrix__threshold,
.alarm-condition-matrix__card {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 10px;
  padding: 9px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.alarm-condition-matrix__threshold.is-empty {
  background: transparent;
  border-style: dashed;
}

.alarm-condition-matrix__threshold.is-disabled,
.alarm-condition-matrix__card.is-disabled {
  opacity: 0.62;
}

.alarm-condition-matrix__threshold > div,
.alarm-condition-matrix__card-main {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.alarm-condition-matrix__threshold em,
.alarm-condition-matrix__card em {
  padding: 2px 7px;
  border-radius: 999px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 11px;
  font-style: normal;
  white-space: nowrap;
}

.alarm-condition-matrix__actions {
  display: flex;
  gap: 6px;
}

.alarm-condition-matrix__actions .is-danger {
  color: var(--dc-danger, #b91c1c);
}

.alarm-condition-dialog {
  display: grid;
  gap: 14px;
}

.alarm-condition-dialog__types {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.alarm-condition-dialog__types button {
  display: grid;
  gap: 5px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  cursor: pointer;
  text-align: left;
}

.alarm-condition-dialog__types button.is-active {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
}

.alarm-condition-dialog__form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.alarm-condition-dialog label {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.alarm-condition-dialog label.is-wide {
  grid-column: 1 / -1;
}

.alarm-condition-dialog label.is-switch {
  grid-template-columns: auto 1fr;
  align-items: center;
  align-content: end;
  gap: 8px;
}

.alarm-condition-dialog label > span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.alarm-condition-dialog input,
.alarm-condition-dialog select,
.alarm-condition-dialog textarea {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font-size: 13px;
  outline: none;
}

.alarm-condition-dialog input,
.alarm-condition-dialog select {
  height: 34px;
  padding: 0 9px;
}

.alarm-condition-dialog input[type="checkbox"] {
  width: 15px;
  height: 15px;
  padding: 0;
  accent-color: var(--dc-primary);
}

.alarm-condition-dialog textarea {
  padding: 9px;
  resize: vertical;
}

.alarm-condition-dialog__error {
  margin: 0;
  padding: 9px 10px;
  border: 1px solid rgba(220, 38, 38, 0.28);
  border-radius: var(--dc-radius-sm);
  background: rgba(220, 38, 38, 0.06);
  color: var(--dc-danger, #b91c1c);
  font-size: 13px;
}

.alarm-condition-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.alarm-condition-dialog__footer .is-ghost {
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

@media (max-width: 900px) {
  .alarm-condition-matrix__threshold,
  .alarm-condition-matrix__card,
  .alarm-condition-dialog__types,
  .alarm-condition-dialog__form {
    grid-template-columns: 1fr;
  }

  .alarm-condition-matrix__actions {
    flex-wrap: wrap;
  }
}
</style>
