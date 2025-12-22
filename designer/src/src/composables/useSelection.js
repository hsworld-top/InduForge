/**
 * useSelection Composable - 选择逻辑
 * 处理组件选择和取消选择
 * Requirements: 3.1, 3.2
 */
import { computed } from 'vue';
import { useDesignStore } from '@/store/design';

/**
 * useSelection Composable
 * 提供组件选择状态和操作方法
 */
export function useSelection() {
    const designStore = useDesignStore();

    /**
     * 当前选中的组件 ID
     */
    const selectedId = computed(() => designStore.selectedComponentId);

    /**
     * 当前选中的组件对象
     */
    const selectedComponent = computed(() => designStore.selectedComponent);

    /**
     * 选择组件
     * Requirements: 3.1 - 点击组件时选中并高亮
     * @param {string} componentId - 组件ID
     */
    function select(componentId) {
        if (componentId) {
            designStore.selectComponent(componentId);
        }
    }

    /**
     * 取消选择
     * Requirements: 3.2 - 点击空白区域时取消选择
     */
    function deselect() {
        designStore.selectComponent(null);
    }

    /**
     * 检查组件是否被选中
     * @param {string} componentId - 组件ID
     * @returns {boolean} 是否选中
     */
    function isSelected(componentId) {
        return designStore.selectedComponentId === componentId;
    }

    /**
     * 切换选择状态
     * @param {string} componentId - 组件ID
     */
    function toggleSelect(componentId) {
        if (isSelected(componentId)) {
            deselect();
        } else {
            select(componentId);
        }
    }

    return {
        // 状态
        selectedId,
        selectedComponent,

        // 方法
        select,
        deselect,
        isSelected,
        toggleSelect,
    };
}

export default useSelection;
