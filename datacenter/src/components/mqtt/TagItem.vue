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
          <el-button type="text" size="small" @click="$emit('view', tag)">
            查看
          </el-button>
          <el-button type="text" size="small" @click="$emit('edit', tag)">
            编辑
          </el-button>
          <el-button
            type="text"
            size="small"
            class="text-red-500"
            @click="$emit('delete', tag)"
          >
            删除
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
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
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 10px 12px;
  transition: all 0.3s;
}

.tag-item:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.1);
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
  color: #303133;
}

.tag-name {
  font-size: 12px;
  color: #606266;
}

.tag-rule {
  display: flex;
  align-items: center;
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
