<template>
  <div class="datapoint-workspace">
    <aside class="datapoint-workspace__filters">
      <div class="datapoint-workspace__panel-title">筛选数据点</div>
      <el-input
        v-model="search"
        clearable
        placeholder="搜索路径、名称或来源"
      />

      <div class="datapoint-workspace__filter-block">
        <div class="datapoint-workspace__filter-label">来源类型</div>
        <button
          v-for="item in sourceTypeOptions"
          :key="item.value"
          type="button"
          class="datapoint-workspace__filter-item"
          :class="{ 'is-active': sourceType === item.value }"
          @click="sourceType = item.value"
        >
          <span>{{ item.label }}</span>
          <small>{{ item.hint }}</small>
        </button>
      </div>

      <div class="datapoint-workspace__filter-block">
        <div class="datapoint-workspace__filter-label">状态</div>
        <el-segmented v-model="status" :options="statusOptions" block />
      </div>
    </aside>

    <section class="datapoint-workspace__list">
      <div class="datapoint-workspace__list-head">
        <div>
          <div class="datapoint-workspace__panel-title">数据点目录</div>
          <p>按来源分组展示，点击一行可在右侧查看关键字段。</p>
        </div>
      </div>
      <DataPointList
        :project-id="projectId"
        :source-type="sourceType || undefined"
        :status="status || undefined"
        :search="search || undefined"
        :show-toolbar="false"
        @select="selectedDatapoint = $event"
      />
    </section>

    <aside class="datapoint-workspace__detail">
      <template v-if="selectedDatapoint">
        <div class="datapoint-workspace__panel-title">详情</div>
        <div class="datapoint-workspace__detail-card">
          <div class="datapoint-workspace__detail-name">
            {{ selectedDatapoint.name || "-" }}
          </div>
          <div class="datapoint-workspace__detail-path">
            {{ selectedDatapoint.path || "-" }}
          </div>
        </div>
        <dl class="datapoint-workspace__detail-list">
          <div>
            <dt>来源</dt>
            <dd>{{ selectedDatapoint.sourceType || "-" }}</dd>
          </div>
          <div>
            <dt>类型</dt>
            <dd>{{ selectedDatapoint.dataType || "-" }}</dd>
          </div>
          <div>
            <dt>状态</dt>
            <dd>
              <el-tag
                size="small"
                :type="
                  selectedDatapoint.status === 'invalid' ? 'info' : 'success'
                "
              >
                {{ selectedDatapoint.status === "invalid" ? "失效" : "有效" }}
              </el-tag>
            </dd>
          </div>
        </dl>
      </template>
      <div v-else class="datapoint-workspace__empty-detail">
        <div class="datapoint-workspace__empty-orb" />
        <strong>选择一个数据点</strong>
        <span>右侧面板会展示来源、路径、类型和状态。</span>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import DataPointList from "@/components/datapoint/DataPointList.vue";

defineProps<{
  projectId: string;
}>();

type DataPointRow = Record<string, unknown>;

const sourceType = ref("");
const status = ref("");
const search = ref("");
const selectedDatapoint = ref<DataPointRow | null>(null);

const sourceTypeOptions = [
  { label: "全部", value: "", hint: "所有来源" },
  { label: "数据库查询", value: "db.query", hint: "SQL 输出" },
  { label: "MQTT 变量", value: "mqtt.tag", hint: "订阅标签" },
  { label: "MQTT 订阅", value: "mqtt.subscription", hint: "消息入口" },
  { label: "计算输出", value: "calc.output", hint: "脚本结果" },
];

const statusOptions = [
  { label: "全部", value: "" },
  { label: "有效", value: "active" },
  { label: "失效", value: "invalid" },
];
</script>

<style scoped>
.datapoint-workspace {
  height: 100%;
  display: grid;
  grid-template-columns: minmax(210px, 260px) minmax(0, 1fr) minmax(240px, 300px);
  gap: 12px;
}

.datapoint-workspace__filters,
.datapoint-workspace__list,
.datapoint-workspace__detail {
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--dc-line, #d6cebf);
  border-radius: 8px;
  background: rgba(255, 253, 247, 0.94);
  box-shadow: var(--dc-shadow, 0 14px 34px rgba(48, 42, 32, 0.11));
}

.datapoint-workspace__filters,
.datapoint-workspace__detail {
  padding: 12px;
  background: #fbfaf4;
}

.datapoint-workspace__list {
  display: flex;
  flex-direction: column;
}

.datapoint-workspace__list-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 13px 14px 11px;
  border-bottom: 1px solid var(--dc-line, #d6cebf);
  background: var(--dc-paper, #fffdf7);
}

.datapoint-workspace__list-head p {
  margin: 2px 0 0;
  color: var(--dc-muted, #687066);
  font-size: 12px;
}

.datapoint-workspace__panel-title {
  margin-bottom: 12px;
  color: var(--dc-ink, #20231f);
  font-size: 14px;
  font-weight: 800;
}

.datapoint-workspace__filter-block {
  margin-top: 18px;
}

.datapoint-workspace__filter-label {
  margin-bottom: 8px;
  color: var(--dc-muted, #687066);
  font-size: 12px;
  font-weight: 700;
}

.datapoint-workspace__filter-item {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
  padding: 10px 12px;
  border: 1px solid #ded5c6;
  border-radius: 7px;
  background: #f1eee5;
  color: #465047;
  text-align: left;
}

.datapoint-workspace__filter-item small {
  color: #7c8078;
  font-size: 11px;
}

.datapoint-workspace__filter-item.is-active {
  border-color: #cfe1dc;
  background: var(--dc-soft, #e8f0ed);
  color: #245b60;
  font-weight: 900;
}

.datapoint-workspace__detail {
  display: flex;
  flex-direction: column;
}

.datapoint-workspace__detail-card {
  padding: 14px;
  border-radius: 8px;
  background: #26322e;
  color: #fffdf7;
}

.datapoint-workspace__detail-name {
  font-size: 16px;
  font-weight: 800;
}

.datapoint-workspace__detail-path {
  margin-top: 8px;
  color: rgba(255, 253, 247, 0.72);
  font-size: 12px;
  word-break: break-all;
}

.datapoint-workspace__detail-list {
  margin: 14px 0 0;
}

.datapoint-workspace__detail-list div {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  padding: 12px 0;
  border-bottom: 1px solid #e5ddcf;
}

.datapoint-workspace__detail-list dt {
  color: var(--dc-muted, #687066);
  font-size: 12px;
}

.datapoint-workspace__detail-list dd {
  margin: 0;
  color: var(--dc-ink, #20231f);
  font-size: 12px;
  font-weight: 700;
  text-align: right;
}

.datapoint-workspace__empty-detail {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--dc-muted, #687066);
  text-align: center;
}

.datapoint-workspace__empty-detail strong {
  color: var(--dc-ink, #20231f);
}

.datapoint-workspace__empty-detail span {
  max-width: 190px;
  font-size: 12px;
  line-height: 1.6;
}

.datapoint-workspace__empty-orb {
  width: 58px;
  height: 58px;
  border-radius: 8px;
  background:
    linear-gradient(135deg, rgba(25, 114, 120, 0.18), transparent 48%),
    #26322e;
}

@media (max-width: 1100px) {
  .datapoint-workspace {
    grid-template-columns: 220px minmax(0, 1fr);
  }

  .datapoint-workspace__detail {
    display: none;
  }
}

@media (max-width: 760px) {
  .datapoint-workspace {
    grid-template-columns: 1fr;
    overflow-y: auto;
  }

  .datapoint-workspace__filters,
  .datapoint-workspace__list {
    min-height: 320px;
  }
}
</style>
