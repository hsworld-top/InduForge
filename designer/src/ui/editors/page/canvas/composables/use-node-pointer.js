/**
 * 节点指针拖拽 Composable
 *
 * 从 NodeRenderer 抽取的 handlePointerDown 逻辑（含 activeDragHandlers / cleanupDragHandlers）。
 *
 * @module ui/Canvas/composables/use-node-pointer
 */

import { onBeforeUnmount } from "vue";
import {
  createSelectableElement,
  UpdateNodeCommand,
  MoveNodeCommand,
} from "@/editor-core";
import {
  getDescriptor,
  getDefaultSize,
  isRegionType,
  isContainerType,
} from "@/components/descriptors/registry";
import {
  resolveAbsoluteLayout,
  buildFlowResetStyle,
  resolveElContainerMinSize,
  clampElContainerPropsBySize,
  resolveElContainerMain,
} from "@/editor-core/utils/layout-utils";

/** 与 use-node-drop 保持一致的插入线边缘阈值（px） */
const rowInsertEdgeThreshold = 8;
const colInsertEdgeThreshold = 8;

/**
 * 创建节点指针按下 / 拖拽移动逻辑
 * @param {Object} deps - 依赖项
 * @returns {{ handlePointerDown: Function }}
 */
export function useNodePointer(deps) {
  const {
    node,
    doc,
    nodeRef,
    readonly,
    isMovable,
    isContainer,
    selection,
    canvasZoom,
    history,
    editorStore,
    currentPage,
    startDrag,
    endDrag,
    updateDropTarget,
    clearDropTarget,
    dragDropManager,
    canAcceptChild,
    showInsertLine,
    insertLineStyle,
    rowInsertInfo,
    layoutInsertInfo,
    activeTabName,
    tabsList,
    resolveFlexDirection,
  } = deps;

  let activeDragHandlers = null;

  /**
   * 清理拖拽事件监听
   * @returns {void}
   * @throws {Error} 无
   */
  const cleanupDragHandlers = () => {
    if (!activeDragHandlers) return;
    const {
      move,
      up,
      userSelect,
      pointerTarget,
      pointerId,
      usePointer,
      pointerEvents,
      pointerElement,
    } = activeDragHandlers;
    if (usePointer) {
      document.removeEventListener("pointermove", move);
      document.removeEventListener("pointerup", up);
      document.removeEventListener("pointercancel", up);
    } else {
      document.removeEventListener("mousemove", move);
      document.removeEventListener("mouseup", up);
    }
    if (pointerTarget?.releasePointerCapture && pointerId !== undefined) {
      try {
        pointerTarget.releasePointerCapture(pointerId);
      } catch {
        // 忽略释放失败
      }
    }
    if (pointerElement && pointerEvents !== undefined) {
      pointerElement.style.pointerEvents = pointerEvents;
    }
    document.body.style.userSelect = userSelect ?? "";
    activeDragHandlers = null;
  };

  onBeforeUnmount(() => {
    cleanupDragHandlers();
  });

  /**
   * 处理节点指针按下事件
   * @param {PointerEvent} event - 指针事件
   */
  const handlePointerDown = (event) => {
    if (readonly.value) return;
    if (activeDragHandlers) return;
    if (!node.value || !isMovable.value) return;
    if (event.pointerType === "mouse" && event.button !== 0) return;
    if (event.target?.closest?.(".resize-handle")) return;
    const targetNodeEl = event.target?.closest?.("[data-node-id]");
    const targetNodeId = targetNodeEl?.getAttribute?.("data-node-id");
    if (
      node.value.type === "Tabs" &&
      targetNodeId &&
      targetNodeId !== node.value.id
    ) {
      return;
    }
    if (targetNodeId && targetNodeId !== node.value.id) {
      if (event.altKey) return;
      if (!isContainer.value) return;
      // descriptor 架构流式容器：让子节点自行处理选中和拖拽
      const desc = getDescriptor(node.value.type);
      if (desc?.childPositioning === "flow") return;
    }

    event.preventDefault();
    event.stopPropagation();

    if (selection.value) {
      // 捕获阶段避免破坏 Ctrl/Meta/Shift 多选逻辑，交由 click 阶段统一处理
      if (event.ctrlKey || event.metaKey || event.shiftKey) {
        return;
      }
      const element = createSelectableElement("node", node.value.id);
      // 如果节点已在选中集合中，不覆盖多选，保持多选状态以便后续拖拽
      if (!selection.value.isSelected(node.value.id)) {
        selection.value.select(element);
      }
    }

    const resetFlowStyle = () => {
      if (!node.value) return;
      const parentNode = doc.value?.getParent?.(node.value.id);
      const isFlow =
        parentNode?.type &&
        parentNode.type !== "FreeContainer" &&
        node.value.positioning !== "absolute";
      if (!isFlow) return;
      const patch = { ...(node.value.style || {}) };
      delete patch.width;
      delete patch.height;
      if (Object.keys(patch).length === 0) {
        return;
      }
      if (history.value?.execute) {
        history.value.execute(
          new UpdateNodeCommand(node.value.id, { style: patch }),
        );
        return;
      }
      if (doc.value?._updateNode) {
        doc.value._updateNode(node.value.id, { style: patch });
      }
    };

    const originParent = doc.value?.getParent?.(node.value.id);
    const isRegionNode =
      node.value.type === "ElHeader" ||
      node.value.type === "ElAside" ||
      node.value.type === "ElMain" ||
      node.value.type === "ElFooter";
    const isRegionParent =
      originParent?.type === "ElHeader" ||
      originParent?.type === "ElAside" ||
      originParent?.type === "ElMain" ||
      originParent?.type === "ElFooter";
    const originContainer = isRegionParent
      ? doc.value?.getParent?.(originParent?.id)
      : isRegionNode
        ? originParent
        : null;
    const allowRegionMoveOut = isRegionNode;
    const rootNodeId = currentPage.value?.rootNodeId || "";
    const isInsideElContainer = (targetNode) => {
      if (!targetNode || !doc.value) return false;
      if (targetNode.type === "ElContainer") return true;
      let current = doc.value.getParent?.(targetNode.id);
      while (current) {
        if (current.type === "ElContainer") return true;
        current = doc.value.getParent?.(current.id);
      }
      return false;
    };
    /**
     * 获取最近的 ElContainer 祖先
     * @param {string} nodeId - 节点 ID
     * @returns {import('@/editor-core').ComponentNode | null}
     */
    const resolveAncestorContainer = (nodeId) => {
      if (!nodeId || !doc.value) return null;
      let current = doc.value.getParent?.(nodeId);
      while (current) {
        if (current.type === "ElContainer") return current;
        current = doc.value.getParent?.(current.id);
      }
      return null;
    };
    const dragContainer = resolveAncestorContainer(node.value?.id);
    const restrictToContainer =
      Boolean(dragContainer) && node.value?.type !== "ElContainer";

    const zoomValue = Number(canvasZoom?.value) || 1;
    const baseLayout = resolveAbsoluteLayout(node.value, nodeRef.value);
    const selectedNodeIds =
      selection.value
        ?.getSelectedElements?.()
        ?.filter((el) => el.kind === "node")
        ?.map((el) => el.id) || [];
    const dragNodeIds =
      selection.value?.isSelected?.(node.value.id) && selectedNodeIds.length
        ? selectedNodeIds
        : [node.value.id];
    const isMultiDrag = dragNodeIds.length > 1;
    /**
     * 解析节点基础绝对布局（不依赖 DOM，适用于批量拖拽）
     * @param {import('@/editor-core').ComponentNode | null | undefined} targetNode - 目标节点
     * @returns {{ x: number, y: number, w: number, h: number, z: number } | null}
     */
    const resolveBaseLayoutFromNode = (targetNode) => {
      if (!targetNode) return null;
      if (targetNode.positioning === "flow") return null;
      const abs = targetNode.absolutePos;
      const legacyAbs = targetNode.layoutItem?.free?.abs;
      if (
        abs &&
        (targetNode.positioning === "absolute" ||
          Number.isFinite(abs.x) ||
          Number.isFinite(abs.y))
      ) {
        return {
          x: Number.isFinite(abs.x) ? abs.x : 0,
          y: Number.isFinite(abs.y) ? abs.y : 0,
          w: Number.isFinite(abs.w) ? abs.w : 100,
          h: Number.isFinite(abs.h) ? abs.h : 100,
          z: Number.isFinite(abs.z) ? abs.z : 1,
        };
      }
      if (legacyAbs) {
        return {
          x: Number.isFinite(legacyAbs.x) ? legacyAbs.x : 0,
          y: Number.isFinite(legacyAbs.y) ? legacyAbs.y : 0,
          w: Number.isFinite(legacyAbs.w) ? legacyAbs.w : 100,
          h: Number.isFinite(legacyAbs.h) ? legacyAbs.h : 100,
          z: Number.isFinite(legacyAbs.z) ? legacyAbs.z : 1,
        };
      }
      return {
        x: Number.isFinite(targetNode.style?.left) ? targetNode.style.left : 0,
        y: Number.isFinite(targetNode.style?.top) ? targetNode.style.top : 0,
        w: Number.isFinite(targetNode.style?.width)
          ? targetNode.style.width
          : 100,
        h: Number.isFinite(targetNode.style?.height)
          ? targetNode.style.height
          : 100,
        z: Number.isFinite(targetNode.style?.zIndex)
          ? targetNode.style.zIndex
          : 1,
      };
    };
    const baseLayoutsById = new Map();
    for (const dragNodeId of dragNodeIds) {
      const dragNode = doc.value?.getNode?.(dragNodeId);
      const base = resolveBaseLayoutFromNode(dragNode);
      if (base) {
        baseLayoutsById.set(dragNodeId, base);
      }
    }
    const startClientX = event.clientX;
    const startClientY = event.clientY;
    let hasMoved = false;
    let startedDragFromMove = false;

    // 判断当前节点是否为 descriptor 架构流式容器的直接子项
    const isFlowChildInDescContainer = (() => {
      if (!node.value || node.value.positioning !== "flow") return false;
      if (!originParent) return false;
      const parentDesc = getDescriptor(originParent.type);
      return parentDesc?.childPositioning === "flow";
    })();
    // 流式子项拖拽阈值：超过自身宽/高一半才真正移出容器
    let flowDragThresholdW = 0;
    let flowDragThresholdH = 0;
    let flowDragExceeded = false;
    if (isFlowChildInDescContainer) {
      // nodeRef 可能在 pointerdown 时尚未就绪，降级通过 data-node-id 查询 DOM
      const rect =
        nodeRef.value?.getBoundingClientRect?.() ??
        document
          .querySelector(`[data-node-id="${node.value?.id}"]`)
          ?.getBoundingClientRect?.();
      if (rect && rect.width > 0 && rect.height > 0) {
        flowDragThresholdW = rect.width / 2;
        flowDragThresholdH = rect.height / 2;
      } else {
        // 最终降级：默认 20px
        flowDragThresholdW = 20;
        flowDragThresholdH = 20;
      }
    }

    const resolveDropRegion = (upEvent) => {
      if (!upEvent) return null;
      const hitList = document.elementsFromPoint(
        upEvent.clientX,
        upEvent.clientY,
      );
      const childType = node.value?.type;
      let containerNode = null;
      /**
       * 判断是否为区域容器类型
       * @param {string} type - 组件类型
       * @returns {boolean}
       */
      for (const hit of hitList) {
        const nodeElement = hit.closest?.("[data-node-id]");
        const nodeId = nodeElement?.getAttribute?.("data-node-id");
        if (!nodeId) continue;
        if (isSelfOrDescendant(nodeId)) continue;
        const targetNode = doc.value?.getNode?.(nodeId);
        if (!targetNode) continue;
        if (restrictToContainer && !isInsideElContainer(targetNode)) {
          continue;
        }
        if (isRegionType(targetNode.type)) {
          if (childType && !canAcceptChild(targetNode, childType)) continue;
          return targetNode;
        }
        if (targetNode.type === "ElContainer") {
          if (!containerNode) {
            containerNode =
              resolveElContainerMain(doc.value, targetNode) || targetNode;
          }
          continue;
        }
        if (isContainerType(targetNode.type)) {
          if (childType && !canAcceptChild(targetNode, childType)) continue;
          return targetNode;
        }
      }
      if (
        containerNode &&
        (!childType || canAcceptChild(containerNode, childType))
      ) {
        return containerNode;
      }
      return null;
    };
    const isSelfOrDescendant = (targetId) => {
      if (!targetId || !node.value?.id || !doc.value) return false;
      if (targetId === node.value.id) return true;
      let current = doc.value.getParent?.(targetId);
      while (current) {
        if (current.id === node.value.id) return true;
        current = doc.value.getParent?.(current.id);
      }
      return false;
    };

    cleanupDragHandlers();
    const originalUserSelect = document.body.style.userSelect;
    document.body.style.userSelect = "none";

    if (history.value && !history.value.isInTransaction?.()) {
      history.value.beginTransaction();
    }

    const usePointer = event.type === "pointerdown";
    const pointerTarget =
      event.target instanceof window.Element
        ? event.target
        : nodeRef.value?.$el;
    if (
      usePointer &&
      pointerTarget?.setPointerCapture &&
      event.pointerId !== undefined
    ) {
      try {
        pointerTarget.setPointerCapture(event.pointerId);
      } catch {
        // 忽略捕获失败
      }
    }

    const resolveRowInsertFromPoint = (pointEvent) => {
      const hitList = document.elementsFromPoint(
        pointEvent.clientX,
        pointEvent.clientY,
      );
      for (const hit of hitList) {
        const nodeElement = hit.closest?.("[data-node-id][data-node-type]");
        if (!nodeElement) continue;
        const nodeType = nodeElement.getAttribute("data-node-type");
        if (nodeType !== "ElCol") continue;
        const nodeId = nodeElement.getAttribute("data-node-id");
        if (isSelfOrDescendant(nodeId)) continue;
        const colNode = nodeId ? doc.value?.getNode?.(nodeId) : null;
        if (!colNode) continue;
        const parentNode = doc.value?.getParent?.(colNode.id);
        if (parentNode?.type !== "ElLayoutRow") continue;
        if (isSelfOrDescendant(parentNode.id)) continue;
        const colRect = nodeElement.getBoundingClientRect?.();
        const edgeThreshold = colInsertEdgeThreshold;
        const nearEdge = colRect
          ? pointEvent.clientX - colRect.left <= edgeThreshold ||
            colRect.right - pointEvent.clientX <= edgeThreshold
          : false;
        const rowSelector = `[data-node-id="${parentNode.id}"]`;
        const rowElement =
          document.querySelector(rowSelector) ||
          nodeElement.closest?.(rowSelector);
        if (!rowElement) continue;
        return { rowElement, parentNode, nearEdge };
      }
      return null;
    };

    const resolveLayoutInsertFromPoint = (pointEvent) => {
      const hitList = document.elementsFromPoint(
        pointEvent.clientX,
        pointEvent.clientY,
      );
      for (const hit of hitList) {
        const layoutElement = hit.closest?.(
          '[data-node-type="ElLayout"][data-node-id]',
        );
        if (!layoutElement) continue;
        const layoutId = layoutElement.getAttribute("data-node-id");
        if (isSelfOrDescendant(layoutId)) continue;
        const layoutNode = layoutId ? doc.value?.getNode?.(layoutId) : null;
        if (!layoutNode || layoutNode.type !== "ElLayout") continue;
        const rowIds = (layoutNode.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElLayoutRow";
        });
        if (rowIds.length === 0) return null;
        const pointY = pointEvent.clientY;
        const threshold = rowInsertEdgeThreshold;
        for (let i = 0; i < rowIds.length; i += 1) {
          const rowId = rowIds[i];
          const rowElement = layoutElement.querySelector(
            `[data-node-id="${rowId}"]`,
          );
          if (!rowElement) continue;
          const rect = rowElement.getBoundingClientRect?.();
          if (!rect) continue;
          if (Math.abs(pointY - rect.top) <= threshold) {
            return { layoutNode, layoutElement, index: i, lineY: rect.top };
          }
          if (Math.abs(pointY - rect.bottom) <= threshold) {
            return {
              layoutNode,
              layoutElement,
              index: i + 1,
              lineY: rect.bottom,
            };
          }
        }
      }
      return null;
    };

    const updateRowInsertLineFromPoint = (pointEvent) => {
      if (!node.value || node.value.type === "ElCol") return false;
      const resolvedRow = resolveRowInsertFromPoint(pointEvent);
      if (!resolvedRow || !resolvedRow.nearEdge) {
        showInsertLine.value = false;
        insertLineStyle.value = null;
        rowInsertInfo.value = null;
        return false;
      }
      const { rowElement, parentNode } = resolvedRow;
      const rowRect = rowElement.getBoundingClientRect?.();
      if (!rowRect) return false;
      const direction = resolveFlexDirection("ElLayoutRow", rowElement);
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        rowElement,
        pointEvent,
        direction,
      );
      const insertLine = insertInfo.insertLine || {
        orientation: "vertical",
        offset: rowRect.right - rowRect.left,
      };
      const adjustedLine = {
        ...insertLine,
        offset: Math.max(0, insertLine.offset),
      };
      showInsertLine.value = true;
      insertLineStyle.value = adjustedLine;
      rowInsertInfo.value = {
        rowId: parentNode.id,
        index: insertInfo.index,
        lineBox: {
          left: rowRect.left,
          top: rowRect.top,
          width: rowRect.width,
          height: rowRect.height,
        },
      };
      return true;
    };

    const updateLayoutInsertLineFromPoint = (pointEvent) => {
      if (!node.value || node.value.type === "ElLayoutRow") return false;
      const resolvedLayout = resolveLayoutInsertFromPoint(pointEvent);
      if (!resolvedLayout) {
        showInsertLine.value = false;
        insertLineStyle.value = null;
        layoutInsertInfo.value = null;
        return false;
      }
      const { layoutElement, layoutNode, index, lineY } = resolvedLayout;
      const layoutRect = layoutElement.getBoundingClientRect?.();
      if (!layoutRect) return false;
      showInsertLine.value = true;
      insertLineStyle.value = {
        orientation: "horizontal",
        offset: Math.max(0, lineY - layoutRect.top),
      };
      rowInsertInfo.value = null;
      layoutInsertInfo.value = {
        layoutId: layoutNode.id,
        index,
        lineBox: {
          left: layoutRect.left,
          top: layoutRect.top,
          width: layoutRect.width,
          height: layoutRect.height,
        },
      };
      return true;
    };

    const move = (moveEvent) => {
      if (!node.value) return;
      const deltaX = (moveEvent.clientX - startClientX) / zoomValue;
      const deltaY = (moveEvent.clientY - startClientY) / zoomValue;
      // 流式容器子项：未超过自身宽/高一半时保持不动
      if (isFlowChildInDescContainer && !flowDragExceeded) {
        if (
          Math.abs(deltaX) <= flowDragThresholdW &&
          Math.abs(deltaY) <= flowDragThresholdH
        ) {
          return;
        }
        flowDragExceeded = true;
      }
      if (Math.abs(deltaX) > 1 || Math.abs(deltaY) > 1) {
        hasMoved = true;
        if (!startedDragFromMove) {
          startDrag(node.value.type);
          startedDragFromMove = true;
          resetFlowStyle();
        }
        if (!isMultiDrag) {
          const dropRegion = resolveDropRegion(moveEvent);
          if (dropRegion?.id && !isSelfOrDescendant(dropRegion.id)) {
            const rect = document
              .querySelector(`[data-node-id="${dropRegion.id}"]`)
              ?.getBoundingClientRect?.();
            updateDropTarget({
              containerId: dropRegion.id,
              insertIndex: (dropRegion.children || []).length,
              position: rect
                ? {
                    x: rect.left,
                    y: rect.top,
                    width: rect.width,
                    height: rect.height,
                  }
                : { x: 0, y: 0, width: 0, height: 0 },
              layoutType: "flex",
              direction: "column",
            });
          } else {
            clearDropTarget();
          }
          if (!updateLayoutInsertLineFromPoint(moveEvent)) {
            updateRowInsertLineFromPoint(moveEvent);
          }
        }
        if (restrictToContainer && dragContainer) {
          const containerEl = document.querySelector(
            `[data-node-id="${dragContainer.id}"]`,
          );
          const containerRect = containerEl?.getBoundingClientRect?.();
          const isInsideContainer = containerRect
            ? moveEvent.clientX >= containerRect.left &&
              moveEvent.clientX <= containerRect.right &&
              moveEvent.clientY >= containerRect.top &&
              moveEvent.clientY <= containerRect.bottom
            : true;
          if (isInsideContainer) {
            return;
          }
        }
      }
      const executePatch = (nodeId, patch) => {
        if (history.value?.isInTransaction?.()) {
          history.value.executeInTransaction(
            new UpdateNodeCommand(nodeId, patch),
          );
          return;
        }
        if (history.value?.execute) {
          history.value.execute(new UpdateNodeCommand(nodeId, patch));
          return;
        }
        if (doc.value?._updateNode) {
          doc.value._updateNode(nodeId, patch);
        }
      };

      const primaryBase = baseLayoutsById.get(node.value.id) || baseLayout;
      const primaryNextAbs = {
        x: Math.round(primaryBase.x + deltaX),
        y: Math.round(primaryBase.y + deltaY),
        w: primaryBase.w,
        h: primaryBase.h,
        z: primaryBase.z,
      };
      if (node.value.type === "ElLayout") {
        const layoutMin = getDefaultSize("ElLayout");
        if (layoutMin?.width) {
          primaryNextAbs.w = Math.max(primaryNextAbs.w, layoutMin.width);
        }
      }
      if (nodeRef.value) {
        nodeRef.value.style.position = "absolute";
        nodeRef.value.style.left = `${primaryNextAbs.x}px`;
        nodeRef.value.style.top = `${primaryNextAbs.y}px`;
        nodeRef.value.style.width = `${primaryNextAbs.w}px`;
        nodeRef.value.style.height = `${primaryNextAbs.h}px`;
        nodeRef.value.style.zIndex = `${primaryNextAbs.z}`;
      }

      for (const [dragNodeId, dragBase] of baseLayoutsById.entries()) {
        const dragNode = doc.value?.getNode?.(dragNodeId);
        if (!dragNode) continue;
        const nextAbs = {
          x: Math.round(dragBase.x + deltaX),
          y: Math.round(dragBase.y + deltaY),
          w: dragBase.w,
          h: dragBase.h,
          z: dragBase.z,
        };
        const nextLayoutItem = {
          ...(dragNode.layoutItem || {}),
          free: {
            mode: "abs",
            abs: { ...nextAbs },
          },
        };
        const regionClampPatch =
          dragNode.type === "ElContainer"
            ? clampElContainerPropsBySize(dragNode, nextAbs.w, nextAbs.h)
            : null;
        const patch = {
          positioning: "absolute",
          absolutePos: nextAbs,
          layoutItem: nextLayoutItem,
          ...(regionClampPatch
            ? { props: { ...(dragNode.props || {}), ...regionClampPatch } }
            : {}),
        };
        executePatch(dragNodeId, patch);
      }

      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new window.CustomEvent("designer:node-transform", {
            detail: { x: primaryNextAbs.x, y: primaryNextAbs.y },
          }),
        );
      }
    };

    const up = (upEvent) => {
      // 流式容器子项：未超出阈值时回滚事务，保持原位
      if (isFlowChildInDescContainer && !flowDragExceeded) {
        cleanupDragHandlers();
        if (history.value?.isInTransaction?.()) {
          history.value.rollbackTransaction();
        }
        if (typeof window !== "undefined") {
          window.dispatchEvent(
            new window.CustomEvent("designer:node-transform-end"),
          );
        }
        return;
      }
      if (startedDragFromMove) {
        endDrag();
        clearDropTarget();
      }
      const rowInsertSnapshot = rowInsertInfo.value;
      const layoutInsertSnapshot = layoutInsertInfo.value;
      showInsertLine.value = false;
      insertLineStyle.value = null;
      rowInsertInfo.value = null;
      layoutInsertInfo.value = null;
      if (hasMoved && !isRegionNode && upEvent && !isMultiDrag) {
        // 流式容器子项超出阈值：移出原容器，放入新容器或画布根
        if (isFlowChildInDescContainer && flowDragExceeded && originParent) {
          if (startedDragFromMove) {
            endDrag();
            clearDropTarget();
          }
          showInsertLine.value = false;
          insertLineStyle.value = null;

          const dropRegion = resolveDropRegion(upEvent);
          const hasValidDrop =
            dropRegion &&
            dropRegion.id !== originParent.id &&
            canAcceptChild(dropRegion, node.value.type) &&
            !isSelfOrDescendant(dropRegion.id);

          if (hasValidDrop) {
            // 拖入其他容器
            const targetId = dropRegion.id;
            const insertIdx = (dropRegion.children || []).length;
            const moveCmd = new MoveNodeCommand(
              node.value.id,
              targetId,
              insertIdx,
            );
            const parentDesc = getDescriptor(dropRegion.type);
            const isFlowTarget = parentDesc?.childPositioning === "flow";
            // 从流式容器拖出后落入绝对定位容器时，baseLayout 是原父坐标系，不能直接用；用鼠标释放位置相对目标容器计算落点
            let absPosForTarget = baseLayout;
            if (!isFlowTarget && isFlowChildInDescContainer) {
              const targetEl = document.querySelector(
                `[data-node-id="${targetId}"]`,
              );
              const targetRect = targetEl?.getBoundingClientRect?.();
              const nextX = targetRect
                ? (upEvent.clientX - targetRect.left) / zoomValue
                : 0;
              const nextY = targetRect
                ? (upEvent.clientY - targetRect.top) / zoomValue
                : 0;
              const nodeElRect = document
                .querySelector(`[data-node-id="${node.value.id}"]`)
                ?.getBoundingClientRect?.();
              const nodeW = nodeElRect
                ? nodeElRect.width / zoomValue
                : (baseLayout.w ?? 100);
              const nodeH = nodeElRect
                ? nodeElRect.height / zoomValue
                : (baseLayout.h ?? 40);
              absPosForTarget = {
                x: Math.round(nextX - nodeW / 2),
                y: Math.round(nextY - nodeH / 2),
                w: Math.round(nodeW),
                h: Math.round(nodeH),
                z: baseLayout.z ?? 1,
              };
            }
            const updatePatch = isFlowTarget
              ? {
                  positioning: "flow",
                  absolutePos: undefined,
                  flowLayout: parentDesc.childFlowLayout
                    ? { ...parentDesc.childFlowLayout }
                    : undefined,
                  layoutItem: undefined,
                  style: {
                    ...(buildFlowResetStyle(node.value.style) || {}),
                    ...(parentDesc.childStyle
                      ? parentDesc.childStyle(dropRegion.type)
                      : {}),
                  },
                }
              : {
                  positioning: "absolute",
                  absolutePos: { ...absPosForTarget },
                  flowLayout: undefined,
                  layoutItem: {
                    ...(node.value.layoutItem || {}),
                    free: { mode: "abs", abs: { ...absPosForTarget } },
                  },
                };
            const updateCmd = new UpdateNodeCommand(node.value.id, updatePatch);
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction(moveCmd);
              history.value.executeInTransaction(updateCmd);
            } else if (history.value?.execute) {
              history.value.execute(moveCmd);
              history.value.execute(updateCmd);
            }
          } else {
            // 无合适容器：移到画布根，绝对定位，节点中心对准鼠标释放位置
            // 优先用根节点 DOM 的 rect；找不到时用 .design-canvas 作为坐标系，避免 nextX/nextY 为 0 导致节点跑到左上角
            const rootEl =
              document.querySelector(`[data-node-id="${rootNodeId}"]`) ??
              document.querySelector(".design-canvas");
            const rootRect = rootEl?.getBoundingClientRect?.();
            const nextX = rootRect
              ? (upEvent.clientX - rootRect.left) / zoomValue
              : 0;
            const nextY = rootRect
              ? (upEvent.clientY - rootRect.top) / zoomValue
              : 0;
            // 从 DOM 读取节点实际宽高，使节点中心对准鼠标释放点
            const nodeElRect = document
              .querySelector(`[data-node-id="${node.value.id}"]`)
              ?.getBoundingClientRect?.();
            const nodeW = nodeElRect
              ? nodeElRect.width / zoomValue
              : (baseLayout.w ?? 100);
            const nodeH = nodeElRect
              ? nodeElRect.height / zoomValue
              : (baseLayout.h ?? 40);
            const nextAbs = {
              x: Math.round(nextX - nodeW / 2),
              y: Math.round(nextY - nodeH / 2),
              w: Math.round(nodeW),
              h: Math.round(nodeH),
              z: baseLayout.z ?? 1,
            };
            const rootNode = doc.value?.getNode?.(rootNodeId);
            const insertIdx = rootNode?.children?.length ?? 0;
            const moveCmd = new MoveNodeCommand(
              node.value.id,
              rootNodeId,
              insertIdx,
            );
            const updateCmd = new UpdateNodeCommand(node.value.id, {
              positioning: "absolute",
              absolutePos: nextAbs,
              flowLayout: undefined,
              layoutItem: {
                ...(node.value.layoutItem || {}),
                free: { mode: "abs", abs: { ...nextAbs } },
              },
              style: buildFlowResetStyle(node.value.style),
            });
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction(moveCmd);
              history.value.executeInTransaction(updateCmd);
            } else if (history.value?.execute) {
              history.value.execute(moveCmd);
              history.value.execute(updateCmd);
            }
          }
          cleanupDragHandlers();
          if (history.value?.isInTransaction?.()) {
            history.value.commitTransaction("移出布局容器");
          }
          if (typeof window !== "undefined") {
            window.dispatchEvent(
              new window.CustomEvent("designer:node-transform-end"),
            );
          }
          return;
        }
        if (
          layoutInsertSnapshot?.layoutId &&
          !isSelfOrDescendant(layoutInsertSnapshot.layoutId) &&
          node.value?.type !== "ElLayoutRow" &&
          node.value?.type !== "ElCol"
        ) {
          const layoutNode = doc.value?.getNode?.(
            layoutInsertSnapshot.layoutId,
          );
          if (layoutNode) {
            const rowNode = editorStore.insertNode(
              "ElLayoutRow",
              layoutNode.id,
              layoutInsertSnapshot.index,
            );
            if (rowNode) {
              const latestLayout = doc.value?.getNode?.(layoutNode.id);
              const rowCount = (latestLayout?.children || []).filter(
                (childId) => {
                  const childNode = doc.value?.getNode?.(childId);
                  return childNode?.type === "ElLayoutRow";
                },
              ).length;
              editorStore.updateNode(layoutNode.id, {
                props: {
                  ...(latestLayout?.props || layoutNode.props || {}),
                  rows: Math.max(1, rowCount),
                },
              });
              editorStore.updateNode(rowNode.id, {
                props: { ...(rowNode.props || {}), columns: 1 },
              });
              const latestRow = doc.value?.getNode?.(rowNode.id);
              const colIds = (latestRow?.children || []).filter((childId) => {
                const childNode = doc.value?.getNode?.(childId);
                return childNode?.type === "ElCol";
              });
              let colId = colIds[0];
              if (!colId) {
                const colNode = editorStore.insertNode("ElCol", rowNode.id, 0);
                if (!colNode) return;
                colId = colNode.id;
              }
              const moveCommand = new MoveNodeCommand(
                node.value.id,
                colId,
                (latestRow?.children || []).length,
              );
              const updatePatch = {
                positioning: "flow",
                absolutePos: undefined,
                flowLayout: undefined,
                layoutItem: undefined,
                style: {
                  ...(buildFlowResetStyle(node.value.style) || {}),
                  width: "100%",
                  height: isContainer.value ? "100%" : "auto",
                },
              };
              const updateCommand = new UpdateNodeCommand(
                node.value.id,
                updatePatch,
              );
              if (history.value?.isInTransaction?.()) {
                history.value.executeInTransaction(moveCommand);
                history.value.executeInTransaction(updateCommand);
              } else if (history.value?.execute) {
                history.value.execute(moveCommand);
                history.value.execute(updateCommand);
              } else if (doc.value?._moveNode && doc.value?._updateNode) {
                doc.value._moveNode(
                  node.value.id,
                  colId,
                  (latestRow?.children || []).length,
                );
                doc.value._updateNode(node.value.id, updatePatch);
              }
            }
          }
          return;
        }
        if (
          rowInsertSnapshot?.rowId &&
          !isSelfOrDescendant(rowInsertSnapshot.rowId) &&
          node.value?.type !== "ElCol"
        ) {
          const rowNode = doc.value?.getNode?.(rowInsertSnapshot.rowId);
          if (rowNode) {
            const colNode = editorStore.insertNode(
              "ElCol",
              rowNode.id,
              rowInsertSnapshot.index,
            );
            if (colNode) {
              const latestRow = doc.value?.getNode?.(rowNode.id);
              const colCount = (latestRow?.children || []).filter((childId) => {
                const childNode = doc.value?.getNode?.(childId);
                return childNode?.type === "ElCol";
              }).length;
              editorStore.updateNode(rowNode.id, {
                props: {
                  ...(latestRow?.props || rowNode.props || {}),
                  columns: colCount,
                },
              });
              const moveCommand = new MoveNodeCommand(
                node.value.id,
                colNode.id,
                (colNode.children || []).length,
              );
              const shouldResetSize =
                node.value.positioning === "absolute" ||
                node.value.layoutItem?.free;
              const updatePatch = {
                positioning: "flow",
                absolutePos: undefined,
                flowLayout: undefined,
                layoutItem: undefined,
              };
              // 放入 ElCol 时默认铺满容器
              updatePatch.style = {
                ...(buildFlowResetStyle(node.value.style) || {}),
                width: "100%",
                height: isContainer.value ? "100%" : "auto",
              };
              if (shouldResetSize) {
                updatePatch.style = {
                  ...(buildFlowResetStyle(node.value.style) || {}),
                  width: "100%",
                  height: isContainer.value ? "100%" : "auto",
                };
              }
              const updateCommand = new UpdateNodeCommand(
                node.value.id,
                updatePatch,
              );
              if (history.value?.isInTransaction?.()) {
                history.value.executeInTransaction(moveCommand);
                history.value.executeInTransaction(updateCommand);
              } else if (history.value?.execute) {
                history.value.execute(moveCommand);
                history.value.execute(updateCommand);
              } else if (doc.value?._moveNode && doc.value?._updateNode) {
                doc.value._moveNode(
                  node.value.id,
                  colNode.id,
                  (colNode.children || []).length,
                );
                doc.value._updateNode(node.value.id, updatePatch);
              }
            }
          }
          return;
        }
        const dropRegion = resolveDropRegion(upEvent);
        if (
          restrictToContainer &&
          dropRegion &&
          isInsideElContainer(dropRegion)
        ) {
          clearDropTarget();
          return;
        }
        if (
          dropRegion &&
          dropRegion.id &&
          dropRegion.id !== originParent?.id &&
          canAcceptChild(dropRegion, node.value.type) &&
          !isSelfOrDescendant(dropRegion.id)
        ) {
          const insertIndex = (dropRegion.children || []).length;
          const moveCommand = new MoveNodeCommand(
            node.value.id,
            dropRegion.id,
            insertIndex,
          );
          const shouldResetSize =
            node.value.positioning === "absolute" ||
            node.value.layoutItem?.free;
          if (dropRegion?.type === "FreeContainer") {
            const containerEl = document.querySelector(
              `[data-node-id="${dropRegion.id}"]`,
            );
            const containerRect = containerEl?.getBoundingClientRect?.();
            const nextX = containerRect
              ? (upEvent.clientX - containerRect.left) / zoomValue
              : 0;
            const nextY = containerRect
              ? (upEvent.clientY - containerRect.top) / zoomValue
              : 0;
            const nextAbs = {
              x: Math.round(nextX),
              y: Math.round(nextY),
              w: baseLayout.w,
              h: baseLayout.h,
              z: baseLayout.z,
            };
            const nextLayoutItem = {
              ...(node.value.layoutItem || {}),
              free: { mode: "abs", abs: { ...nextAbs } },
            };
            const nextProps = { ...(node.value.props || {}) };
            if ("tabKey" in nextProps) delete nextProps.tabKey;
            const updateCommand = new UpdateNodeCommand(node.value.id, {
              positioning: "absolute",
              absolutePos: nextAbs,
              flowLayout: undefined,
              layoutItem: nextLayoutItem,
              props: nextProps,
            });
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction(moveCommand);
              history.value.executeInTransaction(updateCommand);
            } else if (history.value?.execute) {
              history.value.execute(moveCommand);
              history.value.execute(updateCommand);
            } else if (doc.value?._moveNode && doc.value?._updateNode) {
              doc.value._moveNode(node.value.id, dropRegion.id, insertIndex);
              doc.value._updateNode(node.value.id, {
                positioning: "absolute",
                absolutePos: nextAbs,
                flowLayout: undefined,
                layoutItem: nextLayoutItem,
                props: nextProps,
              });
            }
            return;
          }
          const updatePatch = {
            positioning: "flow",
            absolutePos: undefined,
            flowLayout: undefined,
            layoutItem: undefined,
          };
          if (dropRegion?.type === "ElCol") {
            updatePatch.style = {
              ...(buildFlowResetStyle(node.value.style) || {}),
              width: "100%",
              height: isContainer.value ? "100%" : "auto",
            };
          } else if (dropRegion?.type === "Tabs") {
            updatePatch.style = {
              ...(buildFlowResetStyle(node.value.style) || {}),
              width: "100%",
              height: "100%",
            };
            const tabKey =
              activeTabName.value ||
              tabsList.value?.[0]?.name ||
              tabsList.value?.[0]?.label ||
              "";
            if (tabKey) {
              updatePatch.props = {
                ...(node.value.props || {}),
                tabKey: String(tabKey),
              };
            }
          } else if (shouldResetSize) {
            updatePatch.style = buildFlowResetStyle(node.value.style);
          }
          const updateCommand = new UpdateNodeCommand(
            node.value.id,
            updatePatch,
          );
          if (history.value?.isInTransaction?.()) {
            history.value.executeInTransaction(moveCommand);
            history.value.executeInTransaction(updateCommand);
          } else if (history.value?.execute) {
            history.value.execute(moveCommand);
            history.value.execute(updateCommand);
          } else if (doc.value?._moveNode && doc.value?._updateNode) {
            doc.value._moveNode(node.value.id, dropRegion.id, insertIndex);
            doc.value._updateNode(node.value.id, {
              positioning: "flow",
              absolutePos: undefined,
              flowLayout: undefined,
              layoutItem: undefined,
            });
          }
        }
        if (restrictToContainer && dragContainer && rootNodeId) {
          const containerEl = document.querySelector(
            `[data-node-id="${dragContainer.id}"]`,
          );
          const containerRect = containerEl?.getBoundingClientRect?.();
          const isInsideContainer = containerRect
            ? upEvent.clientX >= containerRect.left &&
              upEvent.clientX <= containerRect.right &&
              upEvent.clientY >= containerRect.top &&
              upEvent.clientY <= containerRect.bottom
            : false;
          if (!isInsideContainer) {
            const rootEl = document.querySelector(
              `[data-node-id="${rootNodeId}"]`,
            );
            const rootRect = rootEl?.getBoundingClientRect?.();
            const nextX = rootRect
              ? (upEvent.clientX - rootRect.left) / zoomValue
              : 0;
            const nextY = rootRect
              ? (upEvent.clientY - rootRect.top) / zoomValue
              : 0;
            const nextAbs = {
              x: Math.round(nextX),
              y: Math.round(nextY),
              w: baseLayout.w,
              h: baseLayout.h,
              z: baseLayout.z,
            };
            if (node.value.type === "ElLayout") {
              const layoutMin = getDefaultSize("ElLayout");
              if (layoutMin?.width) {
                nextAbs.w = Math.max(nextAbs.w, layoutMin.width);
              }
            }
            const nextLayoutItem = {
              ...(node.value.layoutItem || {}),
              free: {
                mode: "abs",
                abs: { ...nextAbs },
              },
            };
            const rootNode = doc.value?.getNode?.(rootNodeId);
            const insertIndex = rootNode?.children?.length ?? 0;
            const moveCommand = new MoveNodeCommand(
              node.value.id,
              rootNodeId,
              insertIndex,
            );
            const updateCommand = new UpdateNodeCommand(node.value.id, {
              positioning: "absolute",
              absolutePos: nextAbs,
              layoutItem: nextLayoutItem,
            });
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction(moveCommand);
              history.value.executeInTransaction(updateCommand);
            } else if (history.value?.execute) {
              history.value.execute(moveCommand);
              history.value.execute(updateCommand);
            } else if (doc.value?._moveNode && doc.value?._updateNode) {
              doc.value._moveNode(node.value.id, rootNodeId, insertIndex);
              doc.value._updateNode(node.value.id, {
                positioning: "absolute",
                absolutePos: nextAbs,
                layoutItem: nextLayoutItem,
              });
            }
          }
        }
      }
      if (
        hasMoved &&
        (isRegionParent || allowRegionMoveOut) &&
        originContainer?.type === "ElContainer" &&
        rootNodeId &&
        upEvent &&
        !restrictToContainer
      ) {
        const containerEl = document.querySelector(
          `[data-node-id="${originContainer.id}"]`,
        );
        const containerRect = containerEl?.getBoundingClientRect?.();
        const isInsideContainer = containerRect
          ? upEvent.clientX >= containerRect.left &&
            upEvent.clientX <= containerRect.right &&
            upEvent.clientY >= containerRect.top &&
            upEvent.clientY <= containerRect.bottom
          : (() => {
              const hit = document.elementFromPoint(
                upEvent.clientX,
                upEvent.clientY,
              );
              return Boolean(containerEl && hit && containerEl.contains(hit));
            })();
        if (!isInsideContainer) {
          const rootEl = document.querySelector(
            `[data-node-id="${rootNodeId}"]`,
          );
          const rootRect = rootEl?.getBoundingClientRect?.();
          const nextX = rootRect
            ? (upEvent.clientX - rootRect.left) / zoomValue
            : 0;
          const nextY = rootRect
            ? (upEvent.clientY - rootRect.top) / zoomValue
            : 0;
          const nextAbs = {
            x: Math.round(nextX),
            y: Math.round(nextY),
            w: baseLayout.w,
            h: baseLayout.h,
            z: baseLayout.z,
          };
          if (node.value.type === "ElLayout") {
            const layoutMin = getDefaultSize("ElLayout");
            if (layoutMin?.width) {
              nextAbs.w = Math.max(nextAbs.w, layoutMin.width);
            }
          }
          if (node.value.type === "ElContainer") {
            const minSize = resolveElContainerMinSize(node.value, doc.value);
            if (minSize) {
              if (Number.isFinite(minSize.width) && nextAbs.w < minSize.width) {
                nextAbs.w = minSize.width;
              }
              if (
                Number.isFinite(minSize.height) &&
                nextAbs.h < minSize.height
              ) {
                nextAbs.h = minSize.height;
              }
            }
          }
          const nextLayoutItem = {
            ...(node.value.layoutItem || {}),
            free: {
              mode: "abs",
              abs: { ...nextAbs },
            },
          };
          const rootNode = doc.value?.getNode?.(rootNodeId);
          const insertIndex = rootNode?.children?.length ?? 0;
          const moveCommand = new MoveNodeCommand(
            node.value.id,
            rootNodeId,
            insertIndex,
          );
          const updateCommand = new UpdateNodeCommand(node.value.id, {
            positioning: "absolute",
            absolutePos: nextAbs,
            layoutItem: nextLayoutItem,
          });
          if (history.value?.isInTransaction?.()) {
            history.value.executeInTransaction(moveCommand);
            history.value.executeInTransaction(updateCommand);
          } else if (history.value?.execute) {
            history.value.execute(moveCommand);
            history.value.execute(updateCommand);
          } else if (doc.value?._moveNode && doc.value?._updateNode) {
            doc.value._moveNode(node.value.id, rootNodeId, insertIndex);
            doc.value._updateNode(node.value.id, {
              positioning: "absolute",
              absolutePos: nextAbs,
              layoutItem: nextLayoutItem,
            });
          }
        }
      }
      cleanupDragHandlers();
      if (history.value?.isInTransaction?.()) {
        history.value.commitTransaction("移动组件");
      }
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new window.CustomEvent("designer:node-transform-end"),
        );
      }
    };

    activeDragHandlers = {
      move,
      up,
      userSelect: originalUserSelect,
      pointerTarget,
      pointerId: event.pointerId,
      usePointer,
    };

    if (usePointer) {
      document.addEventListener("pointermove", move);
      document.addEventListener("pointerup", up);
      document.addEventListener("pointercancel", up);
    } else {
      document.addEventListener("mousemove", move);
      document.addEventListener("mouseup", up);
    }
  };

  return { handlePointerDown };
}
