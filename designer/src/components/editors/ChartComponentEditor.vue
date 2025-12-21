<template>
  <div class="chart-component-editor">
    <div class="editor-section">
      <div class="section-title">图表属性</div>
      <!-- 图表类型 -->
      <div class="form-item">
        <label>图表类型</label>
        <el-select
          :model-value="chartType"
          @update:model-value="handleChartTypeChange"
          placeholder="选择图表类型"
        >
          <el-option label="折线图" value="line" />
          <el-option label="柱状图" value="bar" />
          <el-option label="饼图" value="pie" />
          <el-option label="散点图" value="scatter" />
          <el-option label="雷达图" value="radar" />
        </el-select>
      </div>

      <!-- 图表标题 -->
      <div class="form-item">
        <label>标题</label>
        <el-input
          :model-value="chartTitle"
          @update:model-value="handleTitleChange"
          placeholder="输入图表标题"
        />
      </div>

      <!-- 数据源类型 -->
      <div class="form-item">
        <label>数据源</label>
        <el-radio-group
          :model-value="dataSourceType"
          @update:model-value="handleDataSourceTypeChange"
        >
          <el-radio label="static">静态数据</el-radio>
          <el-radio label="api">API接口</el-radio>
        </el-radio-group>
      </div>

      <!-- 静态数据编辑 -->
      <div v-if="dataSourceType === 'static'" class="form-item">
        <label>静态数据</label>
        <el-input
          type="textarea"
          :model-value="staticData"
          @update:model-value="handleStaticDataChange"
          :rows="6"
          placeholder='{"labels": [], "datasets": []}'
        />
      </div>

      <!-- API接口配置 -->
      <div v-if="dataSourceType === 'api'" class="form-item">
        <label>API地址</label>
        <el-input
          :model-value="apiUrl"
          @update:model-value="handleApiUrlChange"
          placeholder="输入API接口地址"
        />
      </div>

      <!-- 刷新间隔 -->
      <div v-if="dataSourceType === 'api'" class="form-item">
        <label>刷新间隔 (秒)</label>
        <el-input-number
          :model-value="refreshInterval"
          @update:model-value="handleRefreshIntervalChange"
          :min="0"
          :max="3600"
          :step="5"
        />
        <span class="hint">0 表示不自动刷新</span>
      </div>

      <!-- 显示图例 -->
      <div class="form-item">
        <el-checkbox
          :model-value="showLegend"
          @update:model-value="handleShowLegendChange"
        >
          显示图例
        </el-checkbox>
      </div>

      <!-- 显示工具栏 -->
      <div class="form-item">
        <el-checkbox
          :model-value="showToolbox"
          @update:model-value="handleShowToolboxChange"
        >
          显示工具栏
        </el-checkbox>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * ChartComponentEditor.vue - 图表组件属性编辑器
 * Task 6.3: 实现组件专有属性编辑器
 */
import { computed } from "vue";

const props = defineProps({
  component: {
    type: Object,
    required: true,
  },
});

const emit = defineEmits(["update"]);

/**
 * 组件 props
 */
const componentProps = computed(() => props.component?.props || {});

/**
 * 图表类型
 */
const chartType = computed(() => {
  const type = props.component?.type || "";
  if (type.includes("LineChart")) return "line";
  if (type.includes("BarChart")) return "bar";
  if (type.includes("PieChart")) return "pie";
  if (type.includes("ScatterChart")) return "scatter";
  if (type.includes("RadarChart")) return "radar";
  return "line";
});

/**
 * 图表标题
 */
const chartTitle = computed(
  () => componentProps.value.option?.title?.text || ""
);

/**
 * 数据源类型
 */
const dataSourceType = computed(
  () => componentProps.value.dataSource?.type || "static"
);

/**
 * 静态数据
 */
const staticData = computed(() => {
  const data = componentProps.value.dataSource?.data;
  return data ? JSON.stringify(data, null, 2) : "";
});

/**
 * API URL
 */
const apiUrl = computed(() => componentProps.value.dataSource?.url || "");

/**
 * 刷新间隔
 */
const refreshInterval = computed(
  () => componentProps.value.dataSource?.refreshInterval || 0
);

/**
 * 显示图例
 */
const showLegend = computed(
  () => componentProps.value.option?.legend?.show !== false
);

/**
 * 显示工具栏
 */
const showToolbox = computed(
  () => componentProps.value.option?.toolbox?.show === true
);

/**
 * 处理图表类型变化
 */
function handleChartTypeChange(type) {
  const typeMap = {
    line: "LineChart",
    bar: "BarChart",
    pie: "PieChart",
    scatter: "ScatterChart",
    radar: "RadarChart",
  };

  emit("update", {
    type: typeMap[type],
  });
}

/**
 * 处理标题变化
 */
function handleTitleChange(value) {
  const option = { ...componentProps.value.option };
  if (!option.title) option.title = {};
  option.title.text = value;

  emit("update", {
    props: { option },
  });
}

/**
 * 处理数据源类型变化
 */
function handleDataSourceTypeChange(type) {
  const dataSource = { ...componentProps.value.dataSource };
  dataSource.type = type;

  emit("update", {
    props: { dataSource },
  });
}

/**
 * 处理静态数据变化
 */
function handleStaticDataChange(value) {
  try {
    const data = JSON.parse(value);
    const dataSource = { ...componentProps.value.dataSource };
    dataSource.data = data;

    emit("update", {
      props: { dataSource },
    });
  } catch (error) {
    console.error("Invalid JSON:", error);
  }
}

/**
 * 处理API地址变化
 */
function handleApiUrlChange(value) {
  const dataSource = { ...componentProps.value.dataSource };
  dataSource.url = value;

  emit("update", {
    props: { dataSource },
  });
}

/**
 * 处理刷新间隔变化
 */
function handleRefreshIntervalChange(value) {
  const dataSource = { ...componentProps.value.dataSource };
  dataSource.refreshInterval = value;

  emit("update", {
    props: { dataSource },
  });
}

/**
 * 处理显示图例变化
 */
function handleShowLegendChange(value) {
  const option = { ...componentProps.value.option };
  if (!option.legend) option.legend = {};
  option.legend.show = value;

  emit("update", {
    props: { option },
  });
}

/**
 * 处理显示工具栏变化
 */
function handleShowToolboxChange(value) {
  const option = { ...componentProps.value.option };
  if (!option.toolbox) option.toolbox = {};
  option.toolbox.show = value;

  emit("update", {
    props: { option },
  });
}
</script>

<style scoped>
.chart-component-editor {
  padding: 12px 0;
}

.editor-section {
  margin-bottom: 16px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e4e7ed;
}

.form-item {
  margin-bottom: 16px;
}

.form-item label {
  display: block;
  font-size: 13px;
  color: #606266;
  margin-bottom: 8px;
}

.el-input,
.el-select {
  width: 100%;
}

.hint {
  display: block;
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
