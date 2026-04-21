<template>
  <div class="mqtt-tag-list h-full flex flex-col">
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

    <div class="flex-1 overflow-auto" v-loading="loading">
      <div class="tag-tree">
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

      <div
        v-if="tags.length === 0 && !loading"
        class="empty-state text-center text-gray-400 py-16"
      >
        <IconTablerFile class="text-6xl mb-4 w-16 h-16" />
        <p class="text-lg">暂无变量</p>
        <p class="text-sm mt-2">点击“新建变量”开始创建</p>
      </div>
    </div>

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

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed, toRef, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  getMqttTags,
  deleteMqttTag,
  toggleMqttTag,
  getMqttTagGroups,
  deleteMqttTagGroup,
  getDataPoints,
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
  previewSessionId: {
    type: String,
    default: "",
  },
});

const loading = ref(false);
const tags = ref([]);
const groups = ref([]);
const searchKeyword = ref("");
const activeGroups = ref(["ungrouped"]);

const tagDialogVisible = ref(false);
const tagDialogMode = ref("create");
const currentTag = ref(null);

const groupDialogVisible = ref(false);
const groupDialogMode = ref("create");
const currentGroup = ref(null);

const projectIdRef = toRef(props, "projectId");
const previewSessionIdRef = toRef(props, "previewSessionId");

const {
  connected: socketConnected,
  disconnect,
  onMessage,
  subscribeTag,
} = useMqttSocket(projectIdRef, previewSessionIdRef);
const tagSubscriptionCleanups = new Map();

const { notify: notifyTagChange } = useMqttTagSync(props.subscriptionId);

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
      tag.code.toLowerCase().includes(keyword),
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

const getGroupStyle = (group) => {
  const baseColor = group?.color || "#3b82f6";
  return {
    "--group-color": baseColor,
    "--group-bg-color": buildGroupBackgroundColor(baseColor),
  };
};

const buildGroupBackgroundColor = (color) => {
  const normalized = String(color || "").trim();
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
    /^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i,
  );
  if (rgbMatch) {
    const r = Number(rgbMatch[1]);
    const g = Number(rgbMatch[2]);
    const b = Number(rgbMatch[3]);
    return `rgba(${r}, ${g}, ${b}, 0.12)`;
  }

  return "#f0f9ff";
};

const toggleGroup = (groupId) => {
  const index = activeGroups.value.indexOf(groupId);
  if (index > -1) {
    activeGroups.value.splice(index, 1);
  } else {
    activeGroups.value.push(groupId);
  }
};

const loadGroups = async () => {
  try {
    const response = await getMqttTagGroups(
      props.projectId,
      props.subscriptionId,
    );
    if (response.success) {
      groups.value = response.data.groups || [];
      activeGroups.value = [
        "ungrouped",
        ...groups.value.map((group) => group.id),
      ];
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
      tags.value = response.data || [];
      await loadTagDatapoints(tags.value);
    }
  } catch (error) {
    console.error("Failed to load tags:", error);
    ElMessage.error("加载变量失败");
  } finally {
    loading.value = false;
  }
};

const loadTagDatapoints = async (tagList) => {
  const ids = (tagList || []).map((tag) => tag.id).filter(Boolean);
  if (ids.length === 0) {
    tags.value.forEach((tag) => {
      tag.datapointPath = "";
      tag.datapointStatus = "";
    });
    return;
  }

  try {
    const response = await getDataPoints(props.projectId, {
      type: "mqtt.tag",
      sourceIds: ids.join(","),
      page: 1,
      pageSize: 200,
    });
    if (response.success) {
      const list = response.data?.datapoints || [];
      const map = new Map(list.map((item) => [item.sourceId, item]));
      tags.value.forEach((tag) => {
        const datapoint = map.get(tag.id);
        tag.datapointPath = datapoint?.path || "";
        tag.datapointStatus = datapoint?.status || "";
      });
    }
  } catch (error) {
    console.error("Failed to load datapoints:", error);
  }
};

const handleRefresh = async () => {
  await Promise.all([loadGroups(), loadTags()]);
  ElMessage.success("刷新成功");
};

const handleBatchExport = () => {
  ElMessage.info("批量导出功能待实现");
};

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
      `确定要删除分组“${group.name}”吗？该分组下的变量将移至未分组。`,
      "删除确认",
      {
        type: "warning",
        confirmButtonText: "删除",
        cancelButtonText: "取消",
      },
    );

    await deleteMqttTagGroup(props.projectId, group.id);
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
    groupDialogMode.value === "create" ? "创建成功" : "更新成功",
  );
};

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
    await ElMessageBox.confirm(`确定要删除变量“${tag.name}”吗？`, "删除确认", {
      type: "warning",
      confirmButtonText: "删除",
      cancelButtonText: "取消",
    });

    await deleteMqttTag(props.projectId, tag.id);
    ElMessage.success("删除成功");
    await loadTags();
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
    await toggleMqttTag(props.projectId, tag.id);
    ElMessage.success(`已${newState ? "启用" : "禁用"}`);
    await loadTags();
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

const handleTagValueUpdate = (data) => {
  const tag = tags.value.find((item) => item.id === data.tagId);
  if (!tag) {
    return;
  }

  tag.currentValue = {
    parsedValue: data.value,
    quality: data.quality,
    timestamp: data.timestamp,
    receivedAt: data.receivedAt,
    error: data.error,
  };
};

const syncTagSubscriptions = () => {
  const desiredTagIds = new Set(
    tags.value.filter((tag) => tag.isEnabled).map((tag) => tag.id),
  );

  desiredTagIds.forEach((tagId) => {
    if (!tagSubscriptionCleanups.has(tagId)) {
      tagSubscriptionCleanups.set(tagId, subscribeTag(tagId));
    }
  });

  Array.from(tagSubscriptionCleanups.entries()).forEach(([tagId, cleanup]) => {
    if (desiredTagIds.has(tagId)) {
      return;
    }

    cleanup?.();
    tagSubscriptionCleanups.delete(tagId);
  });
};

let stopSocketWatch = null;
let unsubscribeSocketMessage = null;

onMounted(async () => {
  await Promise.all([loadGroups(), loadTags()]);

  unsubscribeSocketMessage = onMessage((data) => {
    if (data?.tagId) {
      handleTagValueUpdate(data);
    }
  });

  stopSocketWatch = watch(
    () => [
      socketConnected.value,
      tags.value.map((tag) => `${tag.id}:${tag.isEnabled}`).join(","),
    ],
    () => {
      if (!socketConnected.value) {
        return;
      }
      syncTagSubscriptions();
    },
    { immediate: true },
  );
});

onBeforeUnmount(() => {
  stopSocketWatch?.();
  unsubscribeSocketMessage?.();
  Array.from(tagSubscriptionCleanups.values()).forEach((cleanup) => {
    cleanup?.();
  });
  tagSubscriptionCleanups.clear();
  disconnect();
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

.mqtt-tag-list > div:first-child {
  margin-bottom: 16px;
}
</style>
