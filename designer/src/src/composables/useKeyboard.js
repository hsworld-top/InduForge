/**
 * useKeyboard - 快捷键管理 Composable
 * Task 7.5: 实现完整的快捷键系统
 *
 * Features:
 * - 方向键微调位置
 * - 复制、粘贴、删除
 * - 撤销、重做
 * - 全选、取消选择
 * - 缩放操作
 * - 层级操作
 *
 * @returns {Object} 快捷键功能
 */
import { onMounted, onUnmounted } from 'vue';

/**
 * 快捷键配置映射
 */
export const KEYBOARD_SHORTCUTS = {
  // 选择操作
  SELECT_ALL: { key: 'a', ctrl: true, description: '全选' },
  DESELECT_ALL: { key: 'Escape', description: '取消选择' },

  // 编辑操作
  COPY: { key: 'c', ctrl: true, description: '复制' },
  PASTE: { key: 'v', ctrl: true, description: '粘贴' },
  DUPLICATE: { key: 'd', ctrl: true, description: '复制并粘贴' },
  DELETE: { key: 'Delete', description: '删除' },
  DELETE_ALT: { key: 'Backspace', description: '删除（备用）' },

  // 历史操作
  UNDO: { key: 'z', ctrl: true, description: '撤销' },
  REDO: { key: 'y', ctrl: true, description: '重做' },
  REDO_ALT: { key: 'z', ctrl: true, shift: true, description: '重做（Mac风格）' },

  // 缩放操作
  ZOOM_IN: { key: '+', ctrl: true, description: '放大' },
  ZOOM_IN_ALT: { key: '=', ctrl: true, description: '放大（备用）' },
  ZOOM_OUT: { key: '-', ctrl: true, description: '缩小' },
  ZOOM_RESET: { key: '0', ctrl: true, description: '重置缩放' },

  // 微调位置（方向键）
  MOVE_UP: { key: 'ArrowUp', description: '上移1px' },
  MOVE_DOWN: { key: 'ArrowDown', description: '下移1px' },
  MOVE_LEFT: { key: 'ArrowLeft', description: '左移1px' },
  MOVE_RIGHT: { key: 'ArrowRight', description: '右移1px' },

  // 微调位置（快速移动，Shift+方向键）
  MOVE_UP_FAST: { key: 'ArrowUp', shift: true, description: '上移10px' },
  MOVE_DOWN_FAST: { key: 'ArrowDown', shift: true, description: '下移10px' },
  MOVE_LEFT_FAST: { key: 'ArrowLeft', shift: true, description: '左移10px' },
  MOVE_RIGHT_FAST: { key: 'ArrowRight', shift: true, description: '右移10px' },

  // 层级操作
  BRING_TO_FRONT: { key: ']', ctrl: true, description: '置于顶层' },
  SEND_TO_BACK: { key: '[', ctrl: true, description: '置于底层' },
  MOVE_UP_LAYER: { key: ']', description: '上移一层' },
  MOVE_DOWN_LAYER: { key: '[', description: '下移一层' },
};

/**
 * useKeyboard Composable
 * @param {Object} handlers - 快捷键处理函数映射
 * @param {Object} options - 选项
 * @returns {Object} 快捷键功能
 */
export function useKeyboard(handlers = {}, options = {}) {
  const {
    enabled = true,
    preventDefault = true,
    stopPropagation = false,
  } = options;

  /**
   * 检查快捷键是否匹配
   */
  function matchesShortcut(event, shortcut) {
    // 检查基本键
    if (event.key !== shortcut.key) return false;

    // 检查修饰键
    const ctrl = event.ctrlKey || event.metaKey;
    const shift = event.shiftKey;
    const alt = event.altKey;

    if (!!shortcut.ctrl !== ctrl) return false;
    if (!!shortcut.shift !== shift) return false;
    if (!!shortcut.alt !== alt) return false;

    return true;
  }

  /**
   * 处理键盘事件
   */
  function handleKeyDown(event) {
    if (!enabled) return;

    // 如果焦点在输入框中，不处理快捷键（除了 ESC）
    const target = event.target;
    const isInput =
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.isContentEditable;

    if (isInput && event.key !== 'Escape') {
      return;
    }

    // 遍历快捷键配置，查找匹配项
    for (const [name, shortcut] of Object.entries(KEYBOARD_SHORTCUTS)) {
      if (matchesShortcut(event, shortcut)) {
        // 查找对应的处理函数
        const handlerName = name
          .toLowerCase()
          .replace(/_/g, '-')
          .replace(/-([a-z])/g, (g) => g[1].toUpperCase());

        const handler = handlers[handlerName] || handlers[name.toLowerCase()];

        if (handler && typeof handler === 'function') {
          if (preventDefault) event.preventDefault();
          if (stopPropagation) event.stopPropagation();

          handler(event);
          return;
        }
      }
    }
  }

  /**
   * 注册键盘事件监听
   */
  function register() {
    document.addEventListener('keydown', handleKeyDown);
  }

  /**
   * 注销键盘事件监听
   */
  function unregister() {
    document.removeEventListener('keydown', handleKeyDown);
  }

  // 自动注册和注销
  if (enabled) {
    onMounted(register);
    onUnmounted(unregister);
  }

  return {
    register,
    unregister,
    shortcuts: KEYBOARD_SHORTCUTS,
  };
}

/**
 * 格式化快捷键显示
 * @param {Object} shortcut - 快捷键配置
 * @returns {string} 格式化的快捷键字符串
 */
export function formatShortcut(shortcut) {
  const parts = [];

  if (shortcut.ctrl) {
    parts.push(navigator.platform.includes('Mac') ? 'Cmd' : 'Ctrl');
  }
  if (shortcut.shift) parts.push('Shift');
  if (shortcut.alt) parts.push('Alt');

  // 格式化键名
  let keyName = shortcut.key;
  if (keyName === ' ') keyName = 'Space';
  else if (keyName.startsWith('Arrow')) keyName = keyName.replace('Arrow', '');
  
  parts.push(keyName);

  return parts.join('+');
}

/**
 * 获取所有快捷键列表（用于显示帮助）
 * @returns {Array} 快捷键列表
 */
export function getAllShortcuts() {
  return Object.entries(KEYBOARD_SHORTCUTS).map(([name, shortcut]) => ({
    name,
    shortcut: formatShortcut(shortcut),
    description: shortcut.description,
  }));
}
