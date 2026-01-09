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
        <span class="toolbar-view-size">
          {{ canvasSize }}
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
        
        <!-- 图层操作按钮组 -->
        <el-button-group>
          <el-tooltip content="置顶 (Ctrl+Shift+])">
            <el-button size="small" :disabled="!canMoveLayer" @click="handleMoveToTop">
              <IconEpTop />
            </el-button>
          </el-tooltip>
          <el-tooltip content="上移 (Ctrl+])">
            <el-button size="small" :disabled="!canMoveLayer" @click="handleMoveUp">
              <IconEpArrowUp />
            </el-button>
          </el-tooltip>
          <el-tooltip content="下移 (Ctrl+[)">
            <el-button size="small" :disabled="!canMoveLayer" @click="handleMoveDown">
              <IconEpArrowDown />
            </el-button>
          </el-tooltip>
          <el-tooltip content="置底 (Ctrl+Shift+[)">
            <el-button size="small" :disabled="!canMoveLayer" @click="handleMoveToBottom">
              <IconEpBottom />
            </el-button>
          </el-tooltip>
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
      <el-button
        size="small"
        type="primary"
        @click="handleSave"
        :loading="saving"
      >
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
import IconEpTop from "~icons/ep/top";
import IconEpBottom from "~icons/ep/bottom";
import IconEpArrowUp from "~icons/ep/arrow-up";
import IconEpArrowDown from "~icons/ep/arrow-down";

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
  canMoveLayer: {
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
  "moveUp",
  "moveDown",
  "moveToTop",
  "moveToBottom",
]);

/**
 * 获取画布尺寸文本
 */
const canvasSize = computed(() => {
  const preset = props.viewPresets.find((p) => p.key === props.activeViewKey);
  return preset ? `${preset.width}×${preset.height}` : "-";
});

// const viewIcons = {
//   bigscreen: IconEpFullScreen,
//   pc: IconEpMonitor,
//   tablet: IconEpIphone,
//   phoneLandscape: IconEpCellphone,
//   phonePortrait: IconEpCellphone,
// };

/**
 * 当前视图尺寸
 */
// const activeViewSize = computed(() => {
//   const preset = props.viewPresets.find(
//     (item) => item.key === props.activeViewKey
//   );
//   if (!preset) return "-";
//   return `${preset.width}×${preset.height}`;
// });

/**
 * 切换视图
 * @param {string} key - 视图键值
 */
// const handleViewChange = (key) => {
//   emit("update:activeViewKey", key);
// };

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

/**
 * 上移图层
 */
const handleMoveUp = () => {
  emit("moveUp");
};

/**
 * 下移图层
 */
const handleMoveDown = () => {
  emit("moveDown");
};

/**
 * 置顶
 */
const handleMoveToTop = () => {
  emit("moveToTop");
};

/**
 * 置底
 */
const handleMoveToBottom = () => {
  emit("moveToBottom");
};
</script>
