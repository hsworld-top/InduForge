/**
 * useHistory - 历史记录管理 Composable
 * Task 7.2: 实现撤销/重做功能
 *
 * 功能：
 * - 维护历史记录栈（最多 100 条）
 * - 支持撤销（Undo）
 * - 支持重做（Redo）
 * - 提供快捷键支持（Ctrl+Z, Ctrl+Y）
 *
 * 设计原则：
 * - 使用快照模式存储状态
 * - 使用双栈结构（undoStack + redoStack）
 * - 支持批量操作的事务
 */

import { ref, readonly } from 'vue';

/**
 * 创建历史记录管理器
 * @param {number} maxSize - 最大历史记录数量
 * @returns {Object} 历史记录管理器实例
 */
export function useHistory(maxSize = 100) {
    // 撤销栈
    const undoStack = ref([]);
    
    // 重做栈
    const redoStack = ref([]);
    
    // 当前状态指针
    const currentIndex = ref(-1);
    
    /**
     * 是否可以撤销
     */
    const canUndo = readonly(
        ref(false)
    );
    
    /**
     * 是否可以重做
     */
    const canRedo = readonly(
        ref(false)
    );
    
    /**
     * 更新可撤销/可重做状态
     */
    function updateAvailability() {
        canUndo.value = undoStack.value.length > 0;
        canRedo.value = redoStack.value.length > 0;
    }
    
    /**
     * 推入新的历史记录
     * @param {Object} state - 状态快照
     * @param {string} description - 操作描述
     */
    function push(state, description = '') {
        // 深拷贝状态（避免引用问题）
        const snapshot = {
            state: JSON.parse(JSON.stringify(state)),
            description,
            timestamp: Date.now(),
        };
        
        // 推入撤销栈
        undoStack.value.push(snapshot);
        
        // 清空重做栈（新操作会使重做栈失效）
        redoStack.value = [];
        
        // 限制栈大小
        if (undoStack.value.length > maxSize) {
            undoStack.value.shift(); // 移除最早的记录
        }
        
        currentIndex.value = undoStack.value.length - 1;
        updateAvailability();
        
        console.log(`📝 History pushed: ${description}`, {
            undoStackSize: undoStack.value.length,
            redoStackSize: redoStack.value.length,
        });
    }
    
    /**
     * 撤销操作
     * @returns {Object|null} 上一个状态快照
     */
    function undo() {
        if (undoStack.value.length === 0) {
            console.warn('⚠️ Cannot undo: no history');
            return null;
        }
        
        // 取出当前状态，移到重做栈
        const currentSnapshot = undoStack.value.pop();
        redoStack.value.push(currentSnapshot);
        
        // 获取上一个状态
        const previousSnapshot = undoStack.value[undoStack.value.length - 1] || null;
        
        currentIndex.value = undoStack.value.length - 1;
        updateAvailability();
        
        console.log(`↩️ Undo: ${currentSnapshot.description}`, {
            undoStackSize: undoStack.value.length,
            redoStackSize: redoStack.value.length,
        });
        
        return previousSnapshot;
    }
    
    /**
     * 重做操作
     * @returns {Object|null} 下一个状态快照
     */
    function redo() {
        if (redoStack.value.length === 0) {
            console.warn('⚠️ Cannot redo: no future history');
            return null;
        }
        
        // 从重做栈取出状态，移回撤销栈
        const nextSnapshot = redoStack.value.pop();
        undoStack.value.push(nextSnapshot);
        
        currentIndex.value = undoStack.value.length - 1;
        updateAvailability();
        
        console.log(`↪️ Redo: ${nextSnapshot.description}`, {
            undoStackSize: undoStack.value.length,
            redoStackSize: redoStack.value.length,
        });
        
        return nextSnapshot;
    }
    
    /**
     * 清空历史记录
     */
    function clear() {
        undoStack.value = [];
        redoStack.value = [];
        currentIndex.value = -1;
        updateAvailability();
        
        console.log('🗑️ History cleared');
    }
    
    /**
     * 获取历史记录列表
     * @returns {Array} 历史记录快照列表
     */
    function getHistory() {
        return undoStack.value.map((snapshot, index) => ({
            index,
            description: snapshot.description,
            timestamp: snapshot.timestamp,
            isCurrent: index === currentIndex.value,
        }));
    }
    
    /**
     * 跳转到指定历史记录
     * @param {number} index - 目标索引
     * @returns {Object|null} 目标状态快照
     */
    function jumpTo(index) {
        if (index < 0 || index >= undoStack.value.length) {
            console.warn('⚠️ Invalid history index:', index);
            return null;
        }
        
        // 计算需要撤销/重做的次数
        const diff = index - currentIndex.value;
        
        if (diff > 0) {
            // 向前（重做）
            for (let i = 0; i < diff; i++) {
                redo();
            }
        } else if (diff < 0) {
            // 向后（撤销）
            for (let i = 0; i < Math.abs(diff); i++) {
                undo();
            }
        }
        
        return undoStack.value[index];
    }
    
    /**
     * 开始事务（批量操作）
     * 用于将多个操作合并为一个历史记录
     */
    let transactionStack = [];
    let inTransaction = false;
    
    function beginTransaction() {
        inTransaction = true;
        transactionStack = [];
        console.log('🔄 Transaction started');
    }
    
    function commitTransaction(description = '批量操作') {
        if (!inTransaction) {
            console.warn('⚠️ No active transaction to commit');
            return;
        }
        
        // 将事务中的所有操作合并为一个快照
        if (transactionStack.length > 0) {
            const finalState = transactionStack[transactionStack.length - 1];
            push(finalState, description);
        }
        
        inTransaction = false;
        transactionStack = [];
        console.log(`✅ Transaction committed: ${description}`);
    }
    
    function rollbackTransaction() {
        inTransaction = false;
        transactionStack = [];
        console.log('↩️ Transaction rolled back');
    }
    
    /**
     * 在事务中添加状态
     * @param {Object} state - 状态快照
     */
    function addToTransaction(state) {
        if (inTransaction) {
            transactionStack.push(JSON.parse(JSON.stringify(state)));
        } else {
            // 如果不在事务中，直接推入历史记录
            push(state);
        }
    }
    
    // 初始化可用性状态
    updateAvailability();
    
    return {
        // 状态
        canUndo,
        canRedo,
        currentIndex: readonly(currentIndex),
        
        // 方法
        push,
        undo,
        redo,
        clear,
        getHistory,
        jumpTo,
        
        // 事务支持
        beginTransaction,
        commitTransaction,
        rollbackTransaction,
        addToTransaction,
    };
}

export default useHistory;
