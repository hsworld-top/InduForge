<template>
  <header class="designer-toolbar">
    <div class="toolbar-section toolbar-section--left">
      <span class="toolbar-page-name">
        {{ pageName }}
        <span v-if="isDirty" class="toolbar-dirty">●</span>
      </span>
      <el-button size="small" text @click="handleToggleLock">
        <IconEpLock v-if="isLocked" />
        <IconEpUnlock v-else />
        <span class="ml-1">{{ isLocked ? "已锁定" : "未锁定" }}</span>
      </el-button>
    </div>

    <div class="toolbar-section toolbar-section--center">
      <div class="toolbar-view-group">
        <el-button-group>
          <el-button
            v-for="preset in viewPresets"
            :key="preset.key"
            size="small"
            :type="activeViewKey === preset.key ? 'primary' : ''"
            @click="handleViewChange(preset.key)"
          >
            <component
              v-if="viewIcons[preset.key]"
              :is="viewIcons[preset.key]"
              class="mr-1"
            />
            {{ preset.label }}
          </el-button>
        </el-button-group>
        <span class="toolbar-view-size">
          {{ activeViewSize }}
        </span>
        <span class="toolbar-zoom">{{ Math.round(zoom * 100) }}%</span>
      </div>
      <div class="toolbar-action-group">
        <el-button-group>
          <el-button size="small" :disabled="!canUndo" @click="handleUndo">
            <IconEpBack />
          </el-button>
          <el-button size="small" :disabled="!canRedo" @click="handleRedo">
            <IconEpRight />
          </el-button>
        </el-button-group>
        <el-button size="small" @click="handlePreview">
          <IconEpView />
          预览
        </el-button>
      </div>
    </div>

    <div class="toolbar-section toolbar-section--right">
      <el-button size="small" @click="handleExport">
        <IconEpDocument />
        导出页面
      </el-button>
      <el-button size="small" type="primary" @click="handleSave" :loading="saving">
        <IconEpUpload />
        保存
      </el-button>
    </div>
  </header>
</template>

<script setup>
import { computed } from "vue";
import IconEpBack from "~icons/ep/back";
import IconEpRight from "~icons/ep/right";
import IconEpView from "~icons/ep/view";
import IconEpUpload from "~icons/ep/upload";
import IconEpLock from "~icons/ep/lock";
import IconEpUnlock from "~icons/ep/unlock";
import IconEpDocument from "~icons/ep/document";
import IconEpMonitor from "~icons/ep/monitor";
import IconEpFullScreen from "~icons/ep/full-screen";
import IconEpIphone from "~icons/ep/iphone";
import IconEpCellphone from "~icons/ep/cellphone";

const props = defineProps({
  pageName: {
    type: String,
    default: "未命名页面",
  },
  isLocked: {
    type: Boolean,
    default: false,
  },
  isDirty: {
    type: Boolean,
    default: false,
  },
  viewPresets: {
    type: Array,
    default: () => [],
  },
  activeViewKey: {
    type: String,
    default: "pc",
  },
  canUndo: {
    type: Boolean,
    default: false,
  },
  canRedo: {
    type: Boolean,
    default: false,
  },
  zoom: {
    type: Number,
    default: 1,
  },
  saving: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits([
  "update:activeViewKey",
  "toggleLock",
  "export",
  "undo",
  "redo",
  "preview",
  "save",
]);

const viewIcons = {
  bigscreen: IconEpFullScreen,
  pc: IconEpMonitor,
  tablet: IconEpIphone,
  phoneLandscape: IconEpCellphone,
  phonePortrait: IconEpCellphone,
};

/**
 * 当前视图尺寸
 */
const activeViewSize = computed(() => {
  const preset = props.viewPresets.find(
    (item) => item.key === props.activeViewKey
  );
  if (!preset) return "-";
  return `${preset.width}×${preset.height}`;
});

/**
 * 切换视图
 * @param {string} key - 视图键值
 */
const handleViewChange = (key) => {
  emit("update:activeViewKey", key);
};

/**
 * 切换锁定状态
 */
const handleToggleLock = () => {
  emit("toggleLock");
};

/**
 * 导出页面
 */
const handleExport = () => {
  emit("export");
};

/**
 * 触发撤销
 */
const handleUndo = () => {
  emit("undo");
};

/**
 * 触发重做
 */
const handleRedo = () => {
  emit("redo");
};

/**
 * 触发预览
 */
const handlePreview = () => {
  emit("preview");
};

/**
 * 触发保存
 */
const handleSave = () => {
  emit("save");
};
</script>
