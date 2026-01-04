<template>
  <div class="mqtt-tag-list h-full flex flex-col">
    <!-- 工具栏 -->
    <div class="p-3 border-b space-y-2">
      <div class="flex items-center gap-2">
        <el-button type="primary" size="small" @click="handleCreateTag">
          <IconTablerPlus class="mr-1 w-4 h-4" />
          新建变量
        </el-button>
        <el-button type="success" size="small" @click="handleCreateGroup">
          <IconTablerFolderAdd class="mr-1 w-4 h-4" />
          新建分组
        </el-button>
        <el-button size="small" @click="handleBatchCreate">
          <IconTablerDocumentAdd class="mr-1 w-4 h-4" />
          批量导入
        </el-button>
        <el-button size="small" @click="handleBatchExport">
          <IconTablerDownload class="mr-1 w-4 h-4" />
          批量导出
        </el-button>
      </div>
      <div class="flex items-center gap-2">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索变量名称或标识符"
          clearable
          style="width: 220px"
          @input="handleSearch"
        >
          <template #prefix>
            <IconTablerSearch class="w-4 h-4" />
          </template>
        </el-input>
        <el-button size="small" @click="handleRefresh">
          <IconTablerRefresh class="w-4 h-4" />
          刷新
        </el-button>
      </div>
    </div>

    <!-- 变量树形结构 -->
    <div class="flex-1 overflow-auto" v-loading="loading">
      <div class="tag-tree">
        <!-- 未分组的变量 -->
        <div v-if="ungroupedTags.length > 0" class="group-node">
          <div
            class="group-header"
            :class="{ expanded: activeGroups.includes('ungrouped') }"
            @click="toggleGroup('ungrouped')"
          >
            <div
              class="group-icon-wrapper"
              style="--group-color: #6c757d; --group-bg-color: #f8f9fa"
            >
              <IconTablerFolderOpened class="w-4 h-4" />
            </div>
            <div class="group-info">
              <div class="group-name">未分组</div>
              <div class="group-stats">{{ ungroupedTags.length }} 个变量</div>
            </div>
            <div class="group-actions">
              <IconTablerChevronRight
                v-if="!activeGroups.includes('ungrouped')"
                class="group-expand-icon w-4 h-4"
              />
              <IconTablerChevronDown v-else class="group-expand-icon w-4 h-4" />
            </div>
          </div>
          <div v-if="activeGroups.includes('ungrouped')" class="group-content">
            <div class="tag-items">
              <TagItem
                v-for="tag in ungroupedTags"
                :key="tag.id"
                :tag="tag"
                @edit="handleEditTag"
                @delete="handleDeleteTag"
                @toggle="handleToggleTag"
                @view="handleViewTag"
              />
            </div>
          </div>
        </div>

        <!-- 各个变量组 -->
        <div
          v-for="group in sortedGroups"
          :key="group.id"
          class="group-node"
          :style="getGroupStyle(group)"
        >
          <div
            class="group-header"
            :class="{ expanded: activeGroups.includes(group.id) }"
            @click="toggleGroup(group.id)"
          >
            <div class="group-icon-wrapper">
              <IconTablerFolder class="w-4 h-4" />
            </div>
            <div class="group-info">
              <div class="group-name">{{ group.name }}</div>
              <div class="group-stats">
                {{ getGroupTagCount(group.id) }} 个变量
              </div>
            </div>
            <div class="group-actions">
              <el-button
                type="text"
                size="small"
                @click.stop="handleEditGroup(group)"
              >
                <IconTablerEdit class="w-4 h-4" />
              </el-button>
              <el-button
                type="text"
                size="small"
                class="text-red-500"
                @click.stop="handleDeleteGroup(group)"
              >
                <IconTablerTrash class="w-4 h-4" />
              </el-button>
              <IconTablerChevronRight
                v-if="!activeGroups.includes(group.id)"
                class="group-expand-icon w-4 h-4"
              />
              <IconTablerChevronDown v-else class="group-expand-icon w-4 h-4" />
            </div>
          </div>
          <div v-if="activeGroups.includes(group.id)" class="group-content">
            <div class="tag-items">
              <TagItem
                v-for="tag in getGroupTags(group.id)"
                :key="tag.id"
                :tag="tag"
                @edit="handleEditTag"
                @delete="handleDeleteTag"
                @toggle="handleToggleTag"
                @view="handleViewTag"
              />
              <div
                v-if="getGroupTags(group.id).length === 0"
                class="empty-group"
              >
                <IconTablerFolderOpened class="empty-icon w-6 h-6" />
                <p class="empty-text">该分组暂无变量</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div
        v-if="tags.length === 0 && !loading"
        class="empty-state text-center text-gray-400 py-16"
      >
        <IconTablerFile class="text-6xl mb-4 w-16 h-16" />
        <p class="text-lg">暂无变量</p>
        <p class="text-sm mt-2">点击"新建变量"开始创建</p>
      </div>
    </div>

    <!-- Tag对话框 -->
    <MqttTagDialog
      v-if="tagDialogVisible"
      :visible="tagDialogVisible"
      :tag="currentTag"
      :project-id="projectId"
      :subscription-id="subscriptionId"
      :groups="groups"
      :mode="tagDialogMode"
      @close="tagDialogVisible = false"
      @success="handleTagDialogSuccess"
    />

    <!-- 变量组对话框 -->
    <MqttTagGroupDialog
      v-if="groupDialogVisible"
      :visible="groupDialogVisible"
      :group="currentGroup"
      :project-id="projectId"
      :subscription-id="subscriptionId"
      :mode="groupDialogMode"
      @close="groupDialogVisible = false"
      @success="handleGroupDialogSuccess"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, computed } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  getMqttTags,
  deleteMqttTag,
  toggleMqttTag,
  getMqttTagGroups,
  deleteMqttTagGroup,
} from "@/api/data.api";
import { useMqttSocket } from "@/composables/useMqttSocket";
import { useMqttTagSync } from "@/composables/useMqttTagSync";
import MqttTagDialog from "./MqttTagDialog.vue";
import MqttTagGroupDialog from "./MqttTagGroupDialog.vue";
import TagItem from "./TagItem.vue";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerFolderAdd from "~icons/tabler/folder-plus";
import IconTablerDocumentAdd from "~icons/tabler/file-plus";
import IconTablerRefresh from "~icons/tabler/refresh";
import IconTablerSearch from "~icons/tabler/search";
import IconTablerFolderOpened from "~icons/tabler/folder-open";
import IconTablerFolder from "~icons/tabler/folder";
import IconTablerEdit from "~icons/tabler/edit";
import IconTablerTrash from "~icons/tabler/trash";
import IconTablerFile from "~icons/tabler/file";
import IconTablerChevronRight from "~icons/tabler/chevron-right";
import IconTablerChevronDown from "~icons/tabler/chevron-down";
import IconTablerDownload from "~icons/tabler/download";

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
  subscriptionId: {
    type: String,
    required: true,
  },
});

// 状态
const loading = ref(false);
const tags = ref([]);
const groups = ref([]);
const searchKeyword = ref("");
const activeGroups = ref(["ungrouped"]); // 默认展开未分组

// Tag 对话框
const tagDialogVisible = ref(false);
const tagDialogMode = ref("create");
const currentTag = ref(null);

// 变量组对话框
const groupDialogVisible = ref(false);
const groupDialogMode = ref("create");
const currentGroup = ref(null);

// Socket.IO 连接
const { socket, connect, disconnect } = useMqttSocket(props.projectId);
const socketConnected = ref(false);

// Tag 同步管理（用于左右联动）
const { notify: notifyTagChange } = useMqttTagSync(props.subscriptionId);

// 计算属性
const sortedGroups = computed(() => {
  return [...groups.value].sort((a, b) => a.order - b.order);
});

const filteredTags = computed(() => {
  if (!searchKeyword.value) {
    return tags.value;
  }
  const keyword = searchKeyword.value.toLowerCase();
  return tags.value.filter(
    (tag) =>
      tag.name.toLowerCase().includes(keyword) ||
      tag.code.toLowerCase().includes(keyword)
  );
});

const ungroupedTags = computed(() => {
  return filteredTags.value.filter((tag) => !tag.groupId);
});

const getGroupTags = (groupId) => {
  return filteredTags.value.filter((tag) => tag.groupId === groupId);
};

const getGroupTagCount = (groupId) => {
  return tags.value.filter((tag) => tag.groupId === groupId).length;
};

/**
 * 生成分组样式变量
 * @param {object} group - 分组数据
 * @returns {object} CSS 变量对象
 */
const getGroupStyle = (group) => {
  const baseColor = group?.color || "#3b82f6";
  return {
    "--group-color": baseColor,
    "--group-bg-color": buildGroupBackgroundColor(baseColor),
  };
};

/**
 * 构建浅色背景
 * @param {string} color - 基础颜色
 * @returns {string} 背景色
 */
const buildGroupBackgroundColor = (color) => {
  const normalized = String(color || "").trim();
  // 颜色格式不合法时回退默认值
  if (!normalized) {
    return "#f0f9ff";
  }

  const hexMatch = normalized.match(/^#([0-9a-fA-F]{6})([0-9a-fA-F]{2})?$/);
  if (hexMatch) {
    const hex = hexMatch[1];
    const r = parseInt(hex.slice(0, 2), 16);
    const g = parseInt(hex.slice(2, 4), 16);
    const b = parseInt(hex.slice(4, 6), 16);
    return `rgba(${r}, ${g}, ${b}, 0.12)`;
  }

  const rgbMatch = normalized.match(
    /^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i
  );
  if (rgbMatch) {
    const r = Number(rgbMatch[1]);
    const g = Number(rgbMatch[2]);
    const b = Number(rgbMatch[3]);
    return `rgba(${r}, ${g}, ${b}, 0.12)`;
  }

  return "#f0f9ff";
};

// 切换分组展开状态
const toggleGroup = (groupId) => {
  const index = activeGroups.value.indexOf(groupId);
  if (index > -1) {
    activeGroups.value.splice(index, 1);
  } else {
    activeGroups.value.push(groupId);
  }
};

// 加载数据
const loadGroups = async () => {
  try {
    const response = await getMqttTagGroups(
      props.projectId,
      props.subscriptionId
    );
    if (response.success) {
      groups.value = response.data.groups || [];
      // 默认展开所有组
      activeGroups.value = ["ungrouped", ...groups.value.map((g) => g.id)];
    }
  } catch (error) {
    console.error("Failed to load tag groups:", error);
    ElMessage.error("加载变量组失败");
  }
};

const loadTags = async () => {
  try {
    loading.value = true;
    const response = await getMqttTags(props.projectId, props.subscriptionId);
    if (response.success) {
      // 后端返回: { success: true, data: tags数组, pagination }
      tags.value = response.data || [];
    }
  } catch (error) {
    console.error("Failed to load tags:", error);
    ElMessage.error("加载变量失败");
  } finally {
    loading.value = false;
  }
};

const handleRefresh = async () => {
  await Promise.all([loadGroups(), loadTags()]);
  ElMessage.success("刷新成功");
};

/**
 * 处理批量导出占位
 * @returns {void}
 */
const handleBatchExport = () => {
  ElMessage.info("批量导出功能待实现");
};

// 变量组操作
const handleCreateGroup = () => {
  currentGroup.value = null;
  groupDialogMode.value = "create";
  groupDialogVisible.value = true;
};

const handleEditGroup = (group) => {
  currentGroup.value = { ...group };
  groupDialogMode.value = "edit";
  groupDialogVisible.value = true;
};

const handleDeleteGroup = async (group) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除分组"${group.name}"吗？该分组下的变量将移至未分组。`,
      "删除确认",
      {
        type: "warning",
        confirmButtonText: "删除",
        cancelButtonText: "取消",
      }
    );

    await deleteMqttTagGroup(group.id);
    ElMessage.success("删除成功");
    await handleRefresh();
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to delete group:", error);
      ElMessage.error("删除失败");
    }
  }
};

const handleGroupDialogSuccess = async () => {
  groupDialogVisible.value = false;
  await loadGroups();
  ElMessage.success(
    groupDialogMode.value === "create" ? "创建成功" : "更新成功"
  );
};

// Tag 操作
const handleCreateTag = () => {
  currentTag.value = null;
  tagDialogMode.value = "create";
  tagDialogVisible.value = true;
};

const handleEditTag = (tag) => {
  currentTag.value = { ...tag };
  tagDialogMode.value = "edit";
  tagDialogVisible.value = true;
};

const handleViewTag = (tag) => {
  currentTag.value = { ...tag };
  tagDialogMode.value = "view";
  tagDialogVisible.value = true;
};

const handleDeleteTag = async (tag) => {
  try {
    await ElMessageBox.confirm(`确定要删除变量"${tag.name}"吗？`, "删除确认", {
      type: "warning",
      confirmButtonText: "删除",
      cancelButtonText: "取消",
    });

    await deleteMqttTag(tag.id);
    ElMessage.success("删除成功");
    await loadTags();

    // 通知右侧监控面板
    notifyTagChange("deleted", { tagId: tag.id });
  } catch (error) {
    if (error !== "cancel") {
      console.error("Failed to delete tag:", error);
      ElMessage.error("删除失败");
    }
  }
};

const handleToggleTag = async (tag) => {
  try {
    const newState = !tag.isEnabled;
    await toggleMqttTag(tag.id);
    ElMessage.success(`已${newState ? "启用" : "禁用"}`);
    await loadTags();

    // 通知右侧监控面板
    notifyTagChange("toggled", { tagId: tag.id, isEnabled: newState });
  } catch (error) {
    console.error("Failed to toggle tag:", error);
    ElMessage.error("操作失败");
  }
};

const handleTagDialogSuccess = async () => {
  tagDialogVisible.value = false;
  await loadTags();
  const isCreate = tagDialogMode.value === "create";
  ElMessage.success(isCreate ? "创建成功" : "更新成功");

  // 通知右侧监控面板
  notifyTagChange(isCreate ? "created" : "updated", {
    tagId: currentTag.value?.id,
  });
};

const handleBatchCreate = () => {
  ElMessage.info("批量导入功能开发中...");
};

const handleSearch = () => {
  // 搜索逻辑由 computed 自动处理
};

// 处理Tag值更新
const handleTagValueUpdate = (data) => {
  const tag = tags.value.find((t) => t.id === data.tagId);
  if (tag) {
    tag.currentValue = {
      parsedValue: data.value,
      quality: data.quality,
      timestamp: data.timestamp,
      receivedAt: data.receivedAt,
      error: data.error,
    };
  }
};

// 初始化Socket.IO
const setupSocketIO = () => {
  if (!socket.value) return;

  socket.value.on("connect", () => {
    socketConnected.value = true;
    console.log("[MqttTagList] Socket connected");

    // 订阅所有Tag的值更新
    tags.value.forEach((tag) => {
      if (tag.isEnabled) {
        socket.value.emit("mqtt:tag:subscribe", { tagId: tag.id });
      }
    });
  });

  socket.value.on("disconnect", () => {
    socketConnected.value = false;
    console.log("[MqttTagList] Socket disconnected");
  });

  // 监听Tag值更新
  socket.value.on("mqtt:tag:value", handleTagValueUpdate);
};

// 清理Socket.IO
const cleanupSocketIO = () => {
  if (!socket.value || !socketConnected.value) return;

  // 取消订阅所有Tag
  tags.value.forEach((tag) => {
    socket.value.emit("mqtt:tag:unsubscribe", { tagId: tag.id });
  });

  disconnect();
};

// 初始化
onMounted(async () => {
  await Promise.all([loadGroups(), loadTags()]);

  // 连接Socket.IO并订阅Tag值更新
  connect(props.projectId);
  setupSocketIO();
});

// 清理
onBeforeUnmount(() => {
  cleanupSocketIO();
});
</script>

<style scoped>
.mqtt-tag-list {
  background: #fff;
  padding: 16px;
}

.tag-tree {
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #fafafa;
  overflow: hidden;
}

.group-node {
  border-bottom: 1px solid #e4e7ed;
}

.group-node:last-child {
  border-bottom: none;
}

.group-header {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: white;
  border-radius: 6px;
  margin: 2px 6px;
  position: relative;
}

.group-header:hover {
  background: #f5f7fa;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.group-header.expanded {
  background: var(--group-bg-color, #f0f9ff);
  border-left: 4px solid var(--group-color, #3b82f6);
  margin-left: 4px;
  margin-right: 4px;
  padding-left: 12px;
}

.group-icon-wrapper {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
  background: var(--group-color, #3b82f6);
  color: white;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.group-header:hover .group-icon-wrapper {
  transform: scale(1.05);
}

.group-info {
  flex: 1;
  min-width: 0;
}

.group-name {
  font-size: 13px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 2px;
}

.group-stats {
  font-size: 11px;
  color: #6b7280;
}

.group-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.group-header:hover .group-actions {
  opacity: 1;
}

.group-actions .el-button {
  padding: 4px 6px;
  border-radius: 4px;
}

.group-expand-icon {
  color: #9ca3af;
  transition: transform 0.2s ease;
  margin-left: 8px;
}

.group-header.expanded .group-expand-icon {
  transform: rotate(90deg);
}

.group-content {
  background: var(--group-bg-color, #f0f9ff);
  border-top: 1px solid rgba(0, 0, 0, 0.05);
}

.tag-items {
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.empty-group {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  color: #9ca3af;
}

.empty-icon {
  font-size: 24px;
  margin-bottom: 8px;
  opacity: 0.6;
}

.empty-text {
  font-size: 12px;
  margin: 0;
}

/* TagItem样式调整 */
.tag-items :deep(.tag-item) {
  margin: 0;
  border-radius: 4px;
  background: white;
  border: 1px solid #e4e7ed;
  transition: all 0.2s ease;
}

.tag-items :deep(.tag-item:hover) {
  border-color: var(--group-color, #3b82f6);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

/* 工具栏样式 */
.mqtt-tag-list > div:first-child {
  margin-bottom: 16px;
}
</style>
