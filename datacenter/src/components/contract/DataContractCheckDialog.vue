<template>
  <el-dialog
    v-model="visible"
    width="860px"
    class="data-contract-dialog"
    destroy-on-close
  >
    <template #header>
      <div class="contract-dialog__header">
        <div>
          <div class="contract-dialog__eyebrow">Release Contract Gate</div>
          <h2>{{ dataContractCheckSummary.title }}</h2>
          <p>{{ dataContractCheckSummary.description }}</p>
        </div>
        <div class="contract-dialog__score">
          <strong>{{ passedCount }}/{{ dataContractCheckItems.length }}</strong>
          <span>已满足</span>
        </div>
      </div>
    </template>

    <section class="contract-dialog__overview">
      <div>
        <span>当前项目</span>
        <strong>{{ projectId || "未选择项目" }}</strong>
      </div>
      <div>
        <span>检查阶段</span>
        <strong>发布 / 运行前</strong>
      </div>
      <div>
        <span>数据来源</span>
        <strong>前端本地模型</strong>
      </div>
    </section>

    <section class="contract-dialog__items" aria-label="数据契约检查项">
      <article
        v-for="item in dataContractCheckItems"
        :key="item.id"
        class="contract-check-card"
        :class="`contract-check-card--${item.status}`"
      >
        <div class="contract-check-card__status">
          <span>{{ dataContractCheckStatusText[item.status] }}</span>
        </div>
        <div class="contract-check-card__main">
          <div class="contract-check-card__title-row">
            <h3>{{ item.title }}</h3>
            <el-tag effect="plain" size="small">{{ item.scope }}</el-tag>
          </div>
          <p>{{ item.summary }}</p>
          <div class="contract-check-card__checkpoint">
            {{ item.checkpoint }}
          </div>
        </div>
        <div class="contract-check-card__owner">{{ item.owner }}</div>
      </article>
    </section>

    <section class="contract-dialog__note">
      <strong>扩展点</strong>
      <span>{{ dataContractCheckSummary.extensionNote }}</span>
    </section>

    <template #footer>
      <div class="contract-dialog__footer">
        <span>此入口用于发布前一致性确认，不展示运行态报警事件或实时消息。</span>
        <el-button type="primary" @click="visible = false">知道了</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  dataContractCheckItems,
  dataContractCheckStatusText,
  dataContractCheckSummary,
} from "@/components/contract/dataContractCheck";

const props = defineProps<{
  modelValue: boolean;
  projectId?: string | number;
}>();

const emit = defineEmits<{
  (event: "update:modelValue", value: boolean): void;
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});

const passedCount = computed(
  () => dataContractCheckItems.filter((item) => item.status === "passed").length,
);
</script>

<style scoped>
.contract-dialog__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding-right: 8px;
}

.contract-dialog__eyebrow {
  color: #a16207;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.2em;
  text-transform: uppercase;
}

.contract-dialog__header h2 {
  margin: 6px 0 6px;
  color: #1c1917;
  font-size: 24px;
  font-weight: 850;
  letter-spacing: -0.04em;
}

.contract-dialog__header p {
  max-width: 620px;
  margin: 0;
  color: #57534e;
  font-size: 13px;
  line-height: 1.7;
}

.contract-dialog__score {
  min-width: 98px;
  padding: 12px 14px;
  border: 1px solid #cfe1dc;
  border-radius: 8px;
  background: #e8f0ed;
  text-align: center;
}

.contract-dialog__score strong {
  display: block;
  color: #245b60;
  font-size: 22px;
  line-height: 1;
}

.contract-dialog__score span {
  color: #255b5f;
  font-size: 12px;
}

.contract-dialog__overview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}

.contract-dialog__overview > div {
  padding: 12px 14px;
  border: 1px solid #d6cebf;
  border-radius: 8px;
  background: #fbfaf4;
}

.contract-dialog__overview span,
.contract-dialog__overview strong {
  display: block;
}

.contract-dialog__overview span {
  color: #78716c;
  font-size: 12px;
}

.contract-dialog__overview strong {
  margin-top: 4px;
  color: #292524;
  font-size: 14px;
}

.contract-dialog__items {
  display: grid;
  gap: 10px;
  max-height: 430px;
  overflow: auto;
  padding-right: 4px;
}

.contract-check-card {
  display: grid;
  grid-template-columns: 74px minmax(0, 1fr) 96px;
  gap: 14px;
  align-items: stretch;
  padding: 14px;
  border: 1px solid #d6cebf;
  border-radius: 8px;
  background: #fffdf7;
}

.contract-check-card__status {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 5px;
  color: #fff;
  font-size: 12px;
  font-weight: 800;
  writing-mode: vertical-rl;
  letter-spacing: 0.08em;
}

.contract-check-card--passed .contract-check-card__status {
  background: #166534;
}

.contract-check-card--warning .contract-check-card__status {
  background: #b45309;
}

.contract-check-card--pending .contract-check-card__status {
  background: #57534e;
}

.contract-check-card__title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.contract-check-card h3 {
  margin: 0;
  color: #1c1917;
  font-size: 16px;
  font-weight: 800;
}

.contract-check-card p {
  margin: 7px 0;
  color: #44403c;
  font-size: 13px;
  line-height: 1.6;
}

.contract-check-card__checkpoint {
  padding: 9px 10px;
  border-radius: 5px;
  background: #f1eee5;
  color: #57534e;
  font-size: 12px;
  line-height: 1.6;
}

.contract-check-card__owner {
  display: flex;
  align-items: center;
  justify-content: center;
  border-left: 1px dashed rgba(120, 113, 108, 0.24);
  color: #78716c;
  font-size: 12px;
  text-align: center;
}

.contract-dialog__note {
  display: flex;
  gap: 8px;
  margin-top: 14px;
  padding: 12px 14px;
  border: 1px dashed #bdd9d2;
  border-radius: 8px;
  background: #e8f0ed;
  color: #255b5f;
  font-size: 13px;
}

.contract-dialog__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.contract-dialog__footer span {
  color: #78716c;
  font-size: 12px;
}

.contract-dialog__footer :deep(.el-button--primary) {
  --el-button-bg-color: #197278;
  --el-button-border-color: #197278;
  --el-button-hover-bg-color: #255b5f;
  --el-button-hover-border-color: #255b5f;
}

@media (max-width: 768px) {
  .contract-dialog__header,
  .contract-dialog__footer {
    flex-direction: column;
  }

  .contract-dialog__overview,
  .contract-check-card {
    grid-template-columns: 1fr;
  }

  .contract-check-card__status {
    min-height: 34px;
    writing-mode: horizontal-tb;
  }

  .contract-check-card__owner {
    justify-content: flex-start;
    border-left: 0;
    border-top: 1px dashed rgba(120, 113, 108, 0.24);
    padding-top: 10px;
  }
}
</style>
