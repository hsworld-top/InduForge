/**
 * 拖拽状态管理
 * 管理组件拖拽时的放置指示器状态
 */
import { reactive, readonly } from "vue";

/** 拖拽状态 */
const state = reactive({
  /** 是否正在拖拽 */
  isDragging: false,
  /** 拖拽的组件类型 */
  dragType: "",
  /** 目标容器节点 ID */
  targetContainerId: "",
  /** 插入位置索引 */
  insertIndex: -1,
  /** 指示器位置 */
  indicatorPosition: {
    x: 0,
    y: 0,
    width: 0,
    height: 0,
  },
  /** 容器布局类型 */
  layoutType: "flex",
  /** flex 方向 */
  direction: "column",
});

/**
 * 开始拖拽
 * @param {string} componentType - 组件类型
 */
export const startDrag = (componentType) => {
  state.isDragging = true;
  state.dragType = componentType;
};

/**
 * 结束拖拽
 */
export const endDrag = () => {
  state.isDragging = false;
  state.dragType = "";
  state.targetContainerId = "";
  state.insertIndex = -1;
  state.indicatorPosition = { x: 0, y: 0, width: 0, height: 0 };
};

/**
 * 更新放置目标
 * @param {Object} payload - 放置目标信息
 * @param {string} payload.containerId - 容器节点 ID
 * @param {number} payload.insertIndex - 插入位置
 * @param {Object} payload.position - 指示器位置
 * @param {string} payload.layoutType - 布局类型
 * @param {string} payload.direction - flex 方向
 */
export const updateDropTarget = (payload) => {
  state.targetContainerId = payload.containerId || "";
  state.insertIndex = payload.insertIndex ?? -1;
  state.indicatorPosition = payload.position || {
    x: 0,
    y: 0,
    width: 0,
    height: 0,
  };
  state.layoutType = payload.layoutType || "flex";
  state.direction = payload.direction || "column";
};

/**
 * 清除放置目标
 */
export const clearDropTarget = () => {
  state.targetContainerId = "";
  state.insertIndex = -1;
  state.indicatorPosition = { x: 0, y: 0, width: 0, height: 0 };
};

/**
 * 获取当前拖拽状态
 * @returns {Object} 拖拽状态
 */
export const useDragState = () => {
  return readonly(state);
};

/**
 * 获取放置信息
 * @returns {{ containerId: string, insertIndex: number }}
 */
export const getDropInfo = () => {
  return {
    containerId: state.targetContainerId,
    insertIndex: state.insertIndex,
  };
};

export default {
  state,
  startDrag,
  endDrag,
  updateDropTarget,
  clearDropTarget,
  useDragState,
  getDropInfo,
};
