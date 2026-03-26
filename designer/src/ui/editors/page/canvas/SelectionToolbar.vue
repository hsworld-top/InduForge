<!--
  SelectionToolbar - 选中工具栏
  多选时显示：对齐、分布、等大小、图层操作
-->
<script setup lang="ts">
import type { CSSProperties } from "vue";
import { storeToRefs } from "pinia";
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import IconAlignCenterH from "~icons/lucide/align-horizontal-justify-center";

import IconAlignRight from "~icons/lucide/align-horizontal-justify-end";
import IconAlignLeft from "~icons/lucide/align-horizontal-justify-start";
import IconDistributeH from "~icons/lucide/align-horizontal-space-around";
import IconAlignCenterV from "~icons/lucide/align-vertical-justify-center";
import IconAlignBottom from "~icons/lucide/align-vertical-justify-end";
import IconAlignTop from "~icons/lucide/align-vertical-justify-start";
import IconDistributeV from "~icons/lucide/align-vertical-space-around";
import IconMoveDown from "~icons/lucide/chevron-down";
import IconMoveUp from "~icons/lucide/chevron-up";
import IconMoveToBottom from "~icons/lucide/chevrons-down";
import IconMoveToTop from "~icons/lucide/chevrons-up";
import IconEqualBoth from "~icons/lucide/maximize-2";
import IconEqualWidth from "~icons/lucide/unfold-horizontal";
import IconEqualHeight from "~icons/lucide/unfold-vertical";
import { useEditorStore } from "@/stores/editor-store";

interface SelectionElementLike {
  id: string;
}

interface SelectionLike {
  getSelectionCount?: () => number;
  getSelectedElements?: () => SelectionElementLike[];
  getPrimaryElement?: () => SelectionElementLike | null;
}

type SelectionToolbarAlignType = "left" | "centerH" | "right" | "top" | "centerV" | "bottom";
type SelectionToolbarDistributeType = "horizontal" | "vertical";
type SelectionToolbarMatchSizeType = "width" | "height" | "both";

const editorStore = useEditorStore();
const { selection, selectionVersion, currentPage } = storeToRefs(editorStore);

const TOOLBAR_GAP = 8;

const pos = ref<{ x: number; y: number }>({ x: 0, y: 0 });
const visible = ref(false);

const selectionCount = computed(() => {
  void selectionVersion.value;
  return (selection.value as SelectionLike | null)?.getSelectionCount?.() || 0;
});

/**
 * 合并选中元素的 DOM 包围盒
 * @returns {{ left: number, top: number, right: number, bottom: number } | null}
 */
function computeSelectionRect() {
  const selectionState = selection.value as SelectionLike | null;
  if (!selectionState) return null;
  const elements = selectionState.getSelectedElements?.() || [];
  if (!elements.length) return null;

  let minL = Infinity;
  let minT = Infinity;
  let maxR = -Infinity;
  let maxB = -Infinity;
  for (const el of elements) {
    const dom = document.querySelector(`[data-node-id="${el.id}"]`);
    if (!dom) continue;
    const rect = dom.getBoundingClientRect();
    minL = Math.min(minL, rect.left);
    minT = Math.min(minT, rect.top);
    maxR = Math.max(maxR, rect.right);
    maxB = Math.max(maxB, rect.bottom);
  }
  if (minL === Infinity) return null;
  return { left: minL, top: minT, right: maxR, bottom: maxB };
}

function updatePosition() {
  if (selectionCount.value < 1) {
    visible.value = false;
    return;
  }
  const primaryEl = (selection.value as SelectionLike | null)?.getPrimaryElement?.();
  if (primaryEl && primaryEl.id === currentPage.value?.rootNodeId) {
    visible.value = false;
    return;
  }
  const rect = computeSelectionRect();
  if (!rect) {
    visible.value = false;
    return;
  }

  const toolbarWidth = selectionCount.value >= 2 ? 480 : 140;
  const toolbarHeight = 36;
  const centerX = (rect.left + rect.right) / 2;

  let x = centerX - toolbarWidth / 2;
  let y = rect.top - toolbarHeight - TOOLBAR_GAP;

  if (y < 50) {
    y = rect.bottom + TOOLBAR_GAP;
  }

  x = Math.max(4, Math.min(x, window.innerWidth - toolbarWidth - 4));
  y = Math.max(4, Math.min(y, window.innerHeight - toolbarHeight - 4));

  pos.value = { x, y };
  visible.value = true;
}

const toolbarStyle = computed<CSSProperties>(() => ({
  position: "fixed",
  left: `${pos.value.x}px`,
  top: `${pos.value.y}px`,
  zIndex: 9999,
}));

let rafId: number | null = null;
function scheduleUpdate() {
  if (rafId) return;
  rafId = requestAnimationFrame(() => {
    rafId = null;
    updatePosition();
  });
}

watch(selectionVersion, () => nextTick(scheduleUpdate));

let scrollTarget: HTMLElement | null = null;

onMounted(() => {
  scrollTarget = document.querySelector(".canvas-wrapper");
  if (scrollTarget) {
    scrollTarget.addEventListener("scroll", scheduleUpdate, { passive: true });
  }
  window.addEventListener("resize", scheduleUpdate, { passive: true });
  nextTick(scheduleUpdate);
});

onBeforeUnmount(() => {
  if (scrollTarget) {
    scrollTarget.removeEventListener("scroll", scheduleUpdate);
  }
  window.removeEventListener("resize", scheduleUpdate);
  if (rafId) cancelAnimationFrame(rafId);
});

const handleAlign = (type: SelectionToolbarAlignType) => editorStore.alignElements(type);
const handleDistribute = (dir: SelectionToolbarDistributeType) =>
  editorStore.distributeElements(dir);
const handleMatchSize = (mode: SelectionToolbarMatchSizeType) => editorStore.matchElementSize(mode);
const handleMoveUp = () => editorStore.moveNodeUp();
const handleMoveDown = () => editorStore.moveNodeDown();
const handleMoveToTop = () => editorStore.moveNodeToTop();
const handleMoveToBottom = () => editorStore.moveNodeToBottom();
</script>

<template>
  <teleport to="body">
    <transition name="sel-toolbar-fade">
      <div
        v-if="visible"
        class="selection-toolbar"
        :style="toolbarStyle"
        @pointerdown.stop
        @click.stop
      >
        <!-- 对齐组：选中 >= 2 -->
        <template v-if="selectionCount >= 2">
          <div class="sel-toolbar__group">
            <el-tooltip content="左对齐" placement="top">
              <button class="sel-toolbar__btn" @click="handleAlign('left')">
                <IconAlignLeft />
              </button>
            </el-tooltip>
            <el-tooltip content="水平居中" placement="top">
              <button class="sel-toolbar__btn" @click="handleAlign('centerH')">
                <IconAlignCenterH />
              </button>
            </el-tooltip>
            <el-tooltip content="右对齐" placement="top">
              <button class="sel-toolbar__btn" @click="handleAlign('right')">
                <IconAlignRight />
              </button>
            </el-tooltip>
            <el-tooltip content="顶对齐" placement="top">
              <button class="sel-toolbar__btn" @click="handleAlign('top')">
                <IconAlignTop />
              </button>
            </el-tooltip>
            <el-tooltip content="垂直居中" placement="top">
              <button class="sel-toolbar__btn" @click="handleAlign('centerV')">
                <IconAlignCenterV />
              </button>
            </el-tooltip>
            <el-tooltip content="底对齐" placement="top">
              <button class="sel-toolbar__btn" @click="handleAlign('bottom')">
                <IconAlignBottom />
              </button>
            </el-tooltip>
          </div>

          <!-- 分布组：选中 >= 3 -->
          <div v-if="selectionCount >= 3" class="sel-toolbar__group">
            <el-tooltip content="水平等距分布" placement="top">
              <button class="sel-toolbar__btn" @click="handleDistribute('horizontal')">
                <IconDistributeH />
              </button>
            </el-tooltip>
            <el-tooltip content="垂直等距分布" placement="top">
              <button class="sel-toolbar__btn" @click="handleDistribute('vertical')">
                <IconDistributeV />
              </button>
            </el-tooltip>
          </div>

          <!-- 等大小组：选中 >= 2 -->
          <div class="sel-toolbar__group">
            <el-tooltip content="等宽" placement="top">
              <button class="sel-toolbar__btn" @click="handleMatchSize('width')">
                <IconEqualWidth />
              </button>
            </el-tooltip>
            <el-tooltip content="等高" placement="top">
              <button class="sel-toolbar__btn" @click="handleMatchSize('height')">
                <IconEqualHeight />
              </button>
            </el-tooltip>
            <el-tooltip content="等大小" placement="top">
              <button class="sel-toolbar__btn" @click="handleMatchSize('both')">
                <IconEqualBoth />
              </button>
            </el-tooltip>
          </div>
        </template>

        <!-- 排序组：选中 >= 1 -->
        <div class="sel-toolbar__group">
          <el-tooltip content="上移一层 (Ctrl+])" placement="top">
            <button class="sel-toolbar__btn" @click="handleMoveUp">
              <IconMoveUp />
            </button>
          </el-tooltip>
          <el-tooltip content="下移一层 (Ctrl+[)" placement="top">
            <button class="sel-toolbar__btn" @click="handleMoveDown">
              <IconMoveDown />
            </button>
          </el-tooltip>
          <el-tooltip content="置于顶层" placement="top">
            <button class="sel-toolbar__btn" @click="handleMoveToTop">
              <IconMoveToTop />
            </button>
          </el-tooltip>
          <el-tooltip content="置于底层" placement="top">
            <button class="sel-toolbar__btn" @click="handleMoveToBottom">
              <IconMoveToBottom />
            </button>
          </el-tooltip>
        </div>
      </div>
    </transition>
  </teleport>
</template>
