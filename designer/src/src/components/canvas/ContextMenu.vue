<template>
  <Teleport to="body">
    <div
      v-if="visible"
      ref="menuRef"
      class="context-menu"
      :style="{ left: `${x}px`, top: `${y}px` }"
      @contextmenu.prevent
    >
      <template v-for="item in menuItems" :key="item.id">
        <div
          class="menu-item"
          :class="{ disabled: item.disabled }"
          @click="handleItemClick(item)"
        >
          <component :is="item.icon" class="menu-icon" />
          <span class="menu-label">{{ item.label }}</span>
          <span v-if="item.shortcut" class="menu-shortcut">{{ item.shortcut }}</span>
        </div>
        <div v-if="item.divider" class="menu-divider" />
      </template>
    </div>
  </Teleport>
</template>

<script setup>
/**
 * ContextMenu.vue - 右键菜单组件
 * Task 7.4: 实现右键菜单
 *
 * Features:
 * - 复制、粘贴、删除
 * - 层级操作（置顶、置底、上移、下移）
 * - 快捷键显示
 * - 自动定位
 */
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { useDesignStore } from '@/store/design';
import IconTablerCopy from '~icons/tabler/copy';
import IconTablerClipboard from '~icons/tabler/clipboard';
import IconTablerCopyPlus from '~icons/tabler/copy-plus';
import IconTablerTrash from '~icons/tabler/trash';
import IconTablerArrowBigUpLines from '~icons/tabler/arrow-big-up-lines';
import IconTablerArrowBigDownLines from '~icons/tabler/arrow-big-down-lines';
import IconTablerArrowUp from '~icons/tabler/arrow-up';
import IconTablerArrowDown from '~icons/tabler/arrow-down';

const props = defineProps({
  /** 组件ID（如果右键点击的是组件） */
  componentId: {
    type: String,
    default: null,
  },
});

const emit = defineEmits(['close']);

const designStore = useDesignStore();

// 菜单状态
const visible = ref(false);
const x = ref(0);
const y = ref(0);
const menuRef = ref(null);

/**
 * 菜单项配置
 */
const menuItems = computed(() => {
  const hasSelection = !!props.componentId || designStore.selectedComponentIds.length > 0;
  const hasClipboard = !!designStore.clipboard;

  return [
    {
      id: 'copy',
      label: '复制',
      icon: IconTablerCopy,
      shortcut: 'Ctrl+C',
      disabled: !hasSelection,
      action: 'copy',
    },
    {
      id: 'paste',
      label: '粘贴',
      icon: IconTablerClipboard,
      shortcut: 'Ctrl+V',
      disabled: !hasClipboard,
      action: 'paste',
    },
    {
      id: 'duplicate',
      label: '复制并粘贴',
      icon: IconTablerCopyPlus,
      shortcut: 'Ctrl+D',
      disabled: !hasSelection,
      action: 'duplicate',
    },
    {
      id: 'delete',
      label: '删除',
      icon: IconTablerTrash,
      shortcut: 'Delete',
      disabled: !hasSelection,
      action: 'delete',
      divider: true,
    },
    {
      id: 'bring-to-front',
      label: '置于顶层',
      icon: IconTablerArrowBigUpLines,
      shortcut: '',
      disabled: !props.componentId,
      action: 'bringToFront',
    },
    {
      id: 'send-to-back',
      label: '置于底层',
      icon: IconTablerArrowBigDownLines,
      shortcut: '',
      disabled: !props.componentId,
      action: 'sendToBack',
    },
    {
      id: 'move-up',
      label: '上移一层',
      icon: IconTablerArrowUp,
      shortcut: '',
      disabled: !props.componentId,
      action: 'moveUp',
    },
    {
      id: 'move-down',
      label: '下移一层',
      icon: IconTablerArrowDown,
      shortcut: '',
      disabled: !props.componentId,
      action: 'moveDown',
    },
  ];
});

/**
 * 显示菜单
 */
function show(event) {
  x.value = event.clientX;
  y.value = event.clientY;
  visible.value = true;

  // 下一帧调整位置，防止超出屏幕
  requestAnimationFrame(() => {
    if (menuRef.value) {
      const rect = menuRef.value.getBoundingClientRect();
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      if (rect.right > viewportWidth) {
        x.value = viewportWidth - rect.width - 10;
      }
      if (rect.bottom > viewportHeight) {
        y.value = viewportHeight - rect.height - 10;
      }
    }
  });
}

/**
 * 隐藏菜单
 */
function hide() {
  visible.value = false;
  emit('close');
}

/**
 * 处理菜单项点击
 */
function handleItemClick(item) {
  if (item.disabled) return;

  const componentId = props.componentId || designStore.selectedComponentIds[0];

  switch (item.action) {
    case 'copy':
      designStore.copyComponents();
      break;
    case 'paste':
      designStore.pasteComponents();
      break;
    case 'duplicate':
      designStore.duplicateComponents();
      break;
    case 'delete':
      if (designStore.selectedComponentIds.length > 0) {
        designStore.batchDeleteComponents(designStore.selectedComponentIds);
      }
      break;
    case 'bringToFront':
      designStore.bringToFront(componentId);
      break;
    case 'sendToBack':
      designStore.sendToBack(componentId);
      break;
    case 'moveUp':
      designStore.moveUp(componentId);
      break;
    case 'moveDown':
      designStore.moveDown(componentId);
      break;
  }

  hide();
}

/**
 * 点击外部关闭菜单
 */
function handleClickOutside(event) {
  if (visible.value && menuRef.value && !menuRef.value.contains(event.target)) {
    hide();
  }
}

/**
 * ESC 键关闭菜单
 */
function handleKeyDown(event) {
  if (event.key === 'Escape' && visible.value) {
    hide();
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside);
  document.addEventListener('keydown', handleKeyDown);
});

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside);
  document.removeEventListener('keydown', handleKeyDown);
});

defineExpose({
  show,
  hide,
});
</script>

<style scoped>
.context-menu {
  position: fixed;
  z-index: 10000;
  background: white;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 6px 0;
  min-width: 180px;
  font-size: 14px;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.menu-item:hover:not(.disabled) {
  background-color: #f5f5f5;
}

.menu-item.disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.menu-icon {
  margin-right: 10px;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.menu-label {
  flex: 1;
}

.menu-shortcut {
  margin-left: 24px;
  font-size: 12px;
  color: #999;
}

.menu-divider {
  height: 1px;
  background-color: #e0e0e0;
  margin: 6px 0;
}
</style>
