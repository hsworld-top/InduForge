<template>
  <div class="tag-item">
    <div class="tag-item-content">
      <!-- 左侧信息 -->
      <div class="tag-info">
        <div class="tag-header">
          <span class="tag-code">{{ tag.code }}</span>
          <span class="tag-name">{{ tag.name }}</span>
          <el-tag size="small" :type="getDataTypeColor(tag.dataType)">
            {{ getDataTypeLabel(tag.dataType) }}
          </el-tag>
          <el-tag size="small" type="info">
            {{ getParseTypeLabel(tag.parseType) }}
          </el-tag>
        </div>
        <div class="tag-rule">
          <span class="text-xs text-gray-500">解析规则:</span>
          <span class="font-mono text-xs text-gray-700 ml-1">
            {{ tag.parseRule }}
          </span>
        </div>
        <div class="tag-datapoint">
          <span class="text-xs text-gray-500">数据点:</span>
          <span
            class="text-xs ml-1 truncate"
            :class="
              tag.datapointStatus === 'invalid'
                ? 'text-gray-400'
                : 'text-gray-700'
            "
          >
            {{ tag.datapointPath || "-" }}
          </span>
          <el-tag
            v-if="tag.datapointPath"
            size="small"
            :type="tag.datapointStatus === 'invalid' ? 'info' : 'success'"
          >
            {{ tag.datapointStatus === "invalid" ? "失效" : "活跃" }}
          </el-tag>
        </div>
      </div>

      <!-- 右侧操作 -->
      <div class="tag-actions">
        <!-- 启用开关 -->
        <el-tooltip
          :content="tag.isEnabled ? '点击禁用' : '点击启用'"
          placement="top"
        >
          <el-switch
            :model-value="tag.isEnabled"
            @change="$emit('toggle', tag)"
            active-color="#10b981"
            inactive-color="#ef4444"
            size="default"
          />
        </el-tooltip>

        <!-- 操作按钮 -->
        <div class="action-buttons">
          <el-tooltip content="查看" placement="top">
            <button type="button" class="tag-action-btn" @click="$emit('view', tag)">
              <IconTablerEye />
            </button>
          </el-tooltip>
          <el-tooltip content="编辑" placement="top">
            <button type="button" class="tag-action-btn" @click="$emit('edit', tag)">
              <IconTablerEdit />
            </button>
          </el-tooltip>
          <el-tooltip content="删除" placement="top">
            <button
              type="button"
              class="tag-action-btn is-danger"
              @click="$emit('delete', tag)"
            >
              <IconTablerTrash />
            </button>
          </el-tooltip>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import IconTablerEdit from "~icons/tabler/edit";
import IconTablerEye from "~icons/tabler/eye";
import IconTablerTrash from "~icons/tabler/trash";

defineProps({
  tag: {
    type: Object,
    required: true,
  },
});

defineEmits(["edit", "delete", "toggle", "view"]);

// 数据类型映射
const getDataTypeLabel = (type) => {
  const map = {
    string: "字符串",
    number: "数字",
    boolean: "布尔",
    object: "对象",
    array: "数组",
  };
  return map[type] || type;
};

const getDataTypeColor = (type) => {
  const map = {
    string: "",
    number: "success",
    boolean: "warning",
    object: "info",
    array: "info",
  };
  return map[type] || "";
};

// 解析类型映射
const getParseTypeLabel = (type) => {
  const map = {
    jsonpath: "JSONPath",
    regex: "正则",
    script: "脚本",
    fixed: "固定值",
  };
  return map[type] || type;
};
</script>

<style scoped>
.tag-item {
  position: relative;
  background: var(--dc-surface-raised, #fff);
  border: 1px solid var(--dc-border, #e4e7ed);
  border-radius: var(--dc-radius-sm, 4px);
  padding: 10px 12px;
  transition: all 0.3s;
}

.tag-item:hover {
  border-color: color-mix(in oklch, var(--dc-primary, #409eff) 34%, var(--dc-border, #e4e7ed));
  box-shadow: var(--dc-shadow-surface, 0 2px 8px rgba(64, 158, 255, 0.1));
}

.tag-item-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.tag-info {
  flex: 1;
  min-width: 0;
}

.tag-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.tag-code {
  font-family: monospace;
  font-weight: 600;
  font-size: 13px;
  color: var(--dc-text, #303133);
}

.tag-name {
  font-size: 12px;
  color: var(--dc-text-secondary, #606266);
}

.tag-rule {
  display: flex;
  align-items: center;
  overflow: hidden;
}

.tag-datapoint {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
  overflow: hidden;
}

.tag-rule span:last-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.3s;
}

.tag-item:hover .action-buttons {
  opacity: 1;
}

.tag-action-btn {
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm, 4px);
  background: transparent;
  color: var(--dc-text-muted, #6b7280);
}

.tag-action-btn:hover {
  border-color: var(--dc-border, #e4e7ed);
  background: var(--dc-surface-subtle, #f8fafc);
  color: var(--dc-primary, #409eff);
}

.tag-action-btn.is-danger:hover {
  color: #b91c1c;
}

.tag-action-btn svg {
  width: 14px;
  height: 14px;
}

/* 启用状态指示器 */
.tag-item::before {
  content: "";
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 4px;
  height: 60%;
  border-radius: 0 4px 4px 0;
  background: currentColor;
  opacity: 0;
  transition: opacity 0.3s;
}

.tag-item:hover::before {
  opacity: 0.3;
}
</style>
