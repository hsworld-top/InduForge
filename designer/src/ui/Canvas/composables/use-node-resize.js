/**
 * 节点尺寸调整 Composable
 *
 * 从 NodeRenderer 抽取的 handleResizePointerDown 逻辑。
 * 处理节点 resize 拖拽交互。
 *
 * @module ui/Canvas/composables/use-node-resize
 */

import { onBeforeUnmount } from "vue";
import { UpdateNodeCommand, createSelectableElement } from "@/editor-core";
import {
  resolveAbsoluteLayout,
  parseSizeToNumber,
  resolveElContainerMinSize,
  resolveElLayoutMinHeight,
  buildContainerSectionSizePatch,
} from "@/editor-core/utils/layoutUtils.js";

/**
 * 创建节点尺寸调整逻辑
 * @param {Object} deps - 依赖项
 * @param {import('vue').Ref} deps.node - 节点 ref
 * @param {import('vue').Ref} deps.doc - 文档 ref
 * @param {import('vue').Ref} deps.nodeRef - 节点 DOM 引用 ref
 * @param {import('vue').ComputedRef} deps.readonly - 只读状态 computed
 * @param {import('vue').ComputedRef} deps.isMovable - 是否可移动 computed
 * @param {import('vue').ComputedRef} deps.isElColInRow - 是否在 ElLayoutRow 中 computed
 * @param {import('vue').ComputedRef} deps.isChildInElCol - 是否在 ElCol 中 computed
 * @param {import('vue').Ref} deps.selection - 选择状态 ref
 * @param {import('vue').Ref} deps.canvasZoom - 画布缩放 ref
 * @param {import('vue').Ref} deps.history - 历史记录 ref
 * @param {Object} deps.editorStore - 编辑器 store
 * @param {Function} deps.isChildResizableByDescriptor - 判断子节点是否可 resize 函数
 * @param {Function} deps.getRegionResizeConfig - 获取区域 resize 配置函数
 * @returns {{ handleResizePointerDown: Function }}
 */
export function useNodeResize(deps) {
  const {
    node,
    doc,
    nodeRef,
    readonly,
    isMovable,
    isElColInRow,
    isChildInElCol,
    selection,
    canvasZoom,
    history,
    editorStore,
    isChildResizableByDescriptor,
    getRegionResizeConfig,
  } = deps;

  let activeDragHandlers = null;

  /**
   * 清理拖拽事件监听
   * @returns {void}
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
      } catch (error) {
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
   * 处理尺寸拖拽开始
   * @param {PointerEvent} event - 指针事件
   * @param {{ key: string, x: number, y: number }} handle - 方向句柄
   * @returns {void}
   */
  const handleResizePointerDown = (event, handle) => {
    if (readonly.value) return;
    if (activeDragHandlers) return;
    if (
      !node.value ||
      isChildInElCol.value ||
      (!isMovable.value &&
        !isElColInRow.value &&
        node.value.type !== "ElLayoutRow")
    )
      return;
    if (event.pointerType === "mouse" && event.button !== 0) return;
    const resizeParent = doc.value?.getParent?.(node.value.id);
    if (resizeParent && !isChildResizableByDescriptor(resizeParent.type)) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    if (selection.value) {
      // 捕获阶段避免破坏 Ctrl/Meta/Shift 多选逻辑，交由 click 阶段统一处理
      if (event.ctrlKey || event.metaKey || event.shiftKey) {
        return;
      }
      const element = createSelectableElement("node", node.value.id);
      selection.value.select(element);
    }

    const regionConfig = getRegionResizeConfig(node.value.type);
    if (regionConfig && !regionConfig.handles.includes(handle.key)) {
      return;
    }

    if (isElColInRow.value) {
      if (handle.x === 0) return;
      const parentNode = doc.value?.getParent?.(node.value.id);
      if (!parentNode) return;
      const rowChildren = Array.isArray(parentNode.children)
        ? parentNode.children
        : [];
      const colIds = rowChildren.filter((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElCol";
      });
      const currentIndex = colIds.indexOf(node.value.id);
      const leftColId = currentIndex > 0 ? colIds[currentIndex - 1] : "";
      const leftColNode = leftColId ? doc.value?.getNode?.(leftColId) : null;
      const baseLeftSpan = Math.max(
        1,
        Math.min(24, Number(leftColNode?.props?.span) || 1),
      );
      const baseSpan = Math.max(
        1,
        Math.min(24, Number(node.value.props?.span) || 1),
      );
      const totalSpan = baseLeftSpan + baseSpan;
      const rowSelector = `[data-node-id="${parentNode.id}"]`;
      const rowEl =
        document.querySelector(rowSelector) ||
        nodeRef.value?.closest?.(rowSelector);
      const rowRect = rowEl?.getBoundingClientRect?.();
      if (!rowRect || rowRect.width <= 0) return;
      const zoomValue = Number(canvasZoom?.value) || 1;
      const rowWidth = rowRect.width / zoomValue;
      const startClientX = event.clientX;
      cleanupDragHandlers();
      const originalUserSelect = document.body.style.userSelect;
      document.body.style.userSelect = "none";

      if (history.value && !history.value.isInTransaction?.()) {
        history.value.beginTransaction();
      }

      const usePointer = event.type === "pointerdown";
      const pointerTarget =
        event.target instanceof Element ? event.target : nodeRef.value?.$el;
      const pointerElement = nodeRef.value;
      const originalPointerEvents = pointerElement?.style.pointerEvents;
      if (pointerElement) {
        pointerElement.style.pointerEvents = "none";
      }
      if (
        usePointer &&
        pointerTarget?.setPointerCapture &&
        event.pointerId !== undefined
      ) {
        try {
          pointerTarget.setPointerCapture(event.pointerId);
        } catch (error) {
          // 忽略捕获失败
        }
      }

      const move = (moveEvent) => {
        if (!node.value) return;
        const deltaX = (moveEvent.clientX - startClientX) / zoomValue;
        const rawDelta = (deltaX / rowWidth) * 24;
        let deltaSpan = rawDelta > 0 ? Math.floor(rawDelta) : Math.ceil(rawDelta);
        if (handle.x === -1) {
          deltaSpan = -deltaSpan;
        }
        if (handle.x === -1 && leftColNode) {
          const nextSpan = Math.max(
            1,
            Math.min(totalSpan - 1, baseSpan + deltaSpan),
          );
          const nextLeft = Math.max(1, totalSpan - nextSpan);
          if (nextSpan === baseSpan && nextLeft === baseLeftSpan) return;
          if (history.value?.isInTransaction?.()) {
            history.value.executeInTransaction(
              new UpdateNodeCommand(leftColNode.id, {
                props: { ...(leftColNode.props || {}), span: nextLeft },
              }),
            );
            history.value.executeInTransaction(
              new UpdateNodeCommand(node.value.id, {
                props: { ...(node.value.props || {}), span: nextSpan },
              }),
            );
          } else if (history.value?.execute) {
            history.value.execute(
              new UpdateNodeCommand(leftColNode.id, {
                props: { ...(leftColNode.props || {}), span: nextLeft },
              }),
            );
            history.value.execute(
              new UpdateNodeCommand(node.value.id, {
                props: { ...(node.value.props || {}), span: nextSpan },
              }),
            );
          } else if (doc.value?._updateNode) {
            doc.value._updateNode(leftColNode.id, {
              props: { ...(leftColNode.props || {}), span: nextLeft },
            });
            doc.value._updateNode(node.value.id, {
              props: { ...(node.value.props || {}), span: nextSpan },
            });
          }
          return;
        }
        const nextSpan = Math.max(1, Math.min(24, baseSpan + deltaSpan));
        if (nextSpan === Number(node.value.props?.span || baseSpan)) return;
        editorStore.updateNode(node.value.id, {
          props: { ...(node.value.props || {}), span: nextSpan },
        });
      };

      const up = () => {
        if (history.value?.isInTransaction?.()) {
          history.value.commitTransaction("调整栅格");
        }
        cleanupDragHandlers();
      };

      activeDragHandlers = {
        move,
        up,
        userSelect: originalUserSelect,
        pointerTarget,
        pointerId: event.pointerId,
        usePointer,
        pointerEvents: originalPointerEvents,
        pointerElement,
      };

      if (usePointer) {
        document.addEventListener("pointermove", move);
        document.addEventListener("pointerup", up, { once: true });
        document.addEventListener("pointercancel", up, { once: true });
      } else {
        document.addEventListener("mousemove", move);
        document.addEventListener("mouseup", up, { once: true });
      }
      return;
    }

    if (node.value.type === "ElLayoutRow" && handle.x === 0 && handle.y !== 0) {
      const parentNode = doc.value?.getParent?.(node.value.id);
      if (parentNode?.type === "ElLayout") {
        const rowChildren = Array.isArray(parentNode.children)
          ? parentNode.children
          : [];
        const rowIds = rowChildren.filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElLayoutRow";
        });
        const currentIndex = rowIds.indexOf(node.value.id);
        const targetIndex = handle.y === -1 ? currentIndex - 1 : currentIndex + 1;
        const targetRowId = targetIndex >= 0 ? rowIds[targetIndex] : "";
        const targetRowNode = targetRowId
          ? doc.value?.getNode?.(targetRowId)
          : null;
        const currentEl = nodeRef.value;
        const targetEl = targetRowId
          ? document.querySelector(`[data-node-id="${targetRowId}"]`)
          : null;
        const zoomValue = Number(canvasZoom?.value) || 1;
        const currentRect = currentEl?.getBoundingClientRect?.();
        const targetRect = targetEl?.getBoundingClientRect?.();
        if (
          targetRowNode &&
          currentRect &&
          targetRect &&
          currentRect.height > 0 &&
          targetRect.height > 0
        ) {
          const baseCurrentHeight = currentRect.height / zoomValue;
          const baseTargetHeight = targetRect.height / zoomValue;
          const startClientY = event.clientY;
          cleanupDragHandlers();
          const originalUserSelect = document.body.style.userSelect;
          document.body.style.userSelect = "none";

          if (history.value && !history.value.isInTransaction?.()) {
            history.value.beginTransaction();
          }

          const usePointer = event.type === "pointerdown";
          const pointerTarget =
            event.target instanceof Element ? event.target : nodeRef.value?.$el;
          if (
            usePointer &&
            pointerTarget?.setPointerCapture &&
            event.pointerId !== undefined
          ) {
            try {
              pointerTarget.setPointerCapture(event.pointerId);
            } catch (error) {
              // 忽略捕获失败
            }
          }

          const minSize = 1;
          const move = (moveEvent) => {
            if (!node.value || !targetRowNode) return;
            const deltaY = (moveEvent.clientY - startClientY) / zoomValue;
            let nextCurrentHeight =
              handle.y === -1
                ? baseCurrentHeight - deltaY
                : baseCurrentHeight + deltaY;
            let nextTargetHeight =
              handle.y === -1
                ? baseTargetHeight + deltaY
                : baseTargetHeight - deltaY;

            if (nextTargetHeight < minSize) {
              nextTargetHeight = minSize;
              nextCurrentHeight = Math.max(
                minSize,
                baseCurrentHeight + (baseTargetHeight - nextTargetHeight),
              );
            }
            if (nextCurrentHeight < minSize) {
              nextCurrentHeight = minSize;
              nextTargetHeight = Math.max(
                minSize,
                baseTargetHeight + (baseCurrentHeight - nextCurrentHeight),
              );
            }

            const currentPatch = {
              style: {
                ...(node.value.style || {}),
                height: `${Math.round(nextCurrentHeight)}px`,
              },
            };
            const targetPatch = {
              style: {
                ...(targetRowNode.style || {}),
                height: `${Math.round(nextTargetHeight)}px`,
              },
            };

            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction(
                new UpdateNodeCommand(node.value.id, currentPatch),
              );
              history.value.executeInTransaction(
                new UpdateNodeCommand(targetRowNode.id, targetPatch),
              );
            } else if (history.value?.execute) {
              history.value.execute(
                new UpdateNodeCommand(node.value.id, currentPatch),
              );
              history.value.execute(
                new UpdateNodeCommand(targetRowNode.id, targetPatch),
              );
            } else if (doc.value?._updateNode) {
              doc.value._updateNode(node.value.id, currentPatch);
              doc.value._updateNode(targetRowNode.id, targetPatch);
            }
          };

          const up = () => {
            if (history.value?.isInTransaction?.()) {
              history.value.commitTransaction("调整布局行高度");
            }
            cleanupDragHandlers();
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
            document.addEventListener("pointerup", up, { once: true });
            document.addEventListener("pointercancel", up, { once: true });
          } else {
            document.addEventListener("mousemove", move);
            document.addEventListener("mouseup", up, { once: true });
          }
          return;
        }
      }
    }

    const zoomValue = Number(canvasZoom?.value) || 1;
    const baseLayout = resolveAbsoluteLayout(node.value, nodeRef.value);
    const rect = nodeRef.value?.getBoundingClientRect?.();
    const rectWidth = rect ? rect.width / zoomValue : undefined;
    const rectHeight = rect ? rect.height / zoomValue : undefined;
    const baseWidth = baseLayout.w || rectWidth || 120;
    const baseHeight = baseLayout.h || rectHeight || 40;
    const baseSectionSizes =
      node.value.type === "ElContainer"
        ? {
          headerHeight: parseSizeToNumber(node.value.props?.headerHeight) ?? 60,
          footerHeight: parseSizeToNumber(node.value.props?.footerHeight) ?? 60,
          asideWidth: parseSizeToNumber(node.value.props?.asideWidth) ?? 200,
        }
        : null;
    const startClientX = event.clientX;
    const startClientY = event.clientY;
    const minSize = ["ElLayout", "ElLayoutRow", "ElCol"].includes(
      node.value?.type,
    )
      ? 1
      : 40;
    const containerMinSize = resolveElContainerMinSize(node.value, doc.value);
    const childMinSize = (() => {
      if (
        !nodeRef.value ||
        !node.value?.children?.length ||
        ["ElLayout", "ElLayoutRow", "ElCol"].includes(node.value?.type)
      ) {
        return null;
      }
      const parentRect = nodeRef.value.getBoundingClientRect?.();
      if (!parentRect) return null;
      let minLeft = Number.POSITIVE_INFINITY;
      let minTop = Number.POSITIVE_INFINITY;
      let maxRight = Number.NEGATIVE_INFINITY;
      let maxBottom = Number.NEGATIVE_INFINITY;
      for (const childId of node.value.children) {
        const childEl = nodeRef.value.querySelector?.(
          `[data-node-id="${childId}"]`,
        );
        if (!childEl) continue;
        const childRect = childEl.getBoundingClientRect?.();
        if (!childRect) continue;
        minLeft = Math.min(minLeft, childRect.left);
        minTop = Math.min(minTop, childRect.top);
        maxRight = Math.max(maxRight, childRect.right);
        maxBottom = Math.max(maxBottom, childRect.bottom);
      }
      if (
        minLeft === Number.POSITIVE_INFINITY ||
        minTop === Number.POSITIVE_INFINITY
      ) {
        return null;
      }
      return {
        width: Math.max(0, Math.round((maxRight - minLeft) / zoomValue)),
        height: Math.max(0, Math.round((maxBottom - minTop) / zoomValue)),
      };
    })();

    const parentNode = doc.value?.getParent?.(node.value.id);
    const shouldUpdateAbsolute =
      parentNode?.type === "FreeContainer" ||
      node.value.positioning === "absolute" ||
      node.value.layoutItem?.free?.mode === "abs" ||
      node.value.absolutePos;

    cleanupDragHandlers();
    const originalUserSelect = document.body.style.userSelect;
    document.body.style.userSelect = "none";

    if (history.value && !history.value.isInTransaction?.()) {
      history.value.beginTransaction();
    }

    const usePointer = event.type === "pointerdown";
    const pointerTarget =
      event.target instanceof Element ? event.target : nodeRef.value?.$el;
    const pointerElement = nodeRef.value;
    const originalPointerEvents = pointerElement?.style.pointerEvents;
    if (pointerElement) {
      pointerElement.style.pointerEvents = "none";
    }
    if (
      usePointer &&
      pointerTarget?.setPointerCapture &&
      event.pointerId !== undefined
    ) {
      try {
        pointerTarget.setPointerCapture(event.pointerId);
      } catch (error) {
        // 忽略捕获失败
      }
    }

    const move = (moveEvent) => {
      if (!node.value) return;
      const deltaX = (moveEvent.clientX - startClientX) / zoomValue;
      const deltaY = (moveEvent.clientY - startClientY) / zoomValue;

      if (regionConfig) {
        const currentSize = regionConfig.axis === "x" ? baseWidth : baseHeight;
        const rawDelta = regionConfig.axis === "x" ? deltaX : deltaY;
        const delta = regionConfig.invert ? -rawDelta : rawDelta;
        let nextSize = currentSize + delta;
        const containerNode =
          parentNode?.type === "ElContainer"
            ? parentNode
            : doc.value?.getParent?.(parentNode?.id);
        if (containerNode) {
          const containerEl = document.querySelector(
            `[data-node-id="${containerNode.id}"]`,
          );
          const containerRect = containerEl?.getBoundingClientRect?.();
          const minBodySize = minSize;
          if (containerRect) {
            if (node.value.type === "ElAside") {
              const maxWidth = Math.max(
                minBodySize,
                Math.round(containerRect.width - minBodySize),
              );
              nextSize = Math.min(nextSize, maxWidth);
            } else if (node.value.type === "ElHeader") {
              const footerHeight = Number.parseFloat(
                containerNode.props?.footerHeight || "0",
              );
              const maxHeight = Math.max(
                minBodySize,
                Math.round(containerRect.height - footerHeight - minBodySize),
              );
              nextSize = Math.min(nextSize, maxHeight);
            } else if (node.value.type === "ElFooter") {
              const headerHeight = Number.parseFloat(
                containerNode.props?.headerHeight || "0",
              );
              const maxHeight = Math.max(
                minBodySize,
                Math.round(containerRect.height - headerHeight - minBodySize),
              );
              nextSize = Math.min(nextSize, maxHeight);
            }
          }
        }
        if (nextSize < minSize) nextSize = minSize;
        nextSize = Math.round(nextSize);
        const nextProps = {
          ...(node.value.props || {}),
          [regionConfig.prop]: `${nextSize}px`,
        };
        const parentContainer =
          parentNode?.type === "ElContainer" ? parentNode : null;
        const parentPatch =
          parentContainer && regionConfig.prop === "height"
            ? node.value.type === "ElHeader"
              ? { headerHeight: `${nextSize}px` }
              : node.value.type === "ElFooter"
                ? { footerHeight: `${nextSize}px` }
                : null
            : parentContainer && regionConfig.prop === "width"
              ? { asideWidth: `${nextSize}px` }
              : null;

        if (nodeRef.value) {
          if (regionConfig.axis === "x") {
            nodeRef.value.style.width = `${nextSize}px`;
          } else {
            nodeRef.value.style.height = `${nextSize}px`;
          }
        }

        const patch = { props: nextProps };
        if (history.value?.isInTransaction?.()) {
          history.value.executeInTransaction(
            new UpdateNodeCommand(node.value.id, patch),
          );
          if (parentContainer && parentPatch) {
            history.value.executeInTransaction(
              new UpdateNodeCommand(parentContainer.id, {
                props: { ...(parentContainer.props || {}), ...parentPatch },
              }),
            );
          }
          return;
        }
        if (history.value?.execute) {
          history.value.execute(new UpdateNodeCommand(node.value.id, patch));
          if (parentContainer && parentPatch) {
            history.value.execute(
              new UpdateNodeCommand(parentContainer.id, {
                props: { ...(parentContainer.props || {}), ...parentPatch },
              }),
            );
          }
          return;
        }
        if (doc.value?._updateNode) {
          doc.value._updateNode(node.value.id, patch);
        }
        return;
      }

      let nextWidth = baseWidth;
      let nextHeight = baseHeight;
      let nextX = baseLayout.x;
      let nextY = baseLayout.y;
      if (handle.x === 1) {
        nextWidth = baseWidth + deltaX;
      } else if (handle.x === -1) {
        nextWidth = baseWidth - deltaX;
        nextX = baseLayout.x + deltaX;
      }

      if (handle.y === 1) {
        nextHeight = baseHeight + deltaY;
      } else if (handle.y === -1) {
        nextHeight = baseHeight - deltaY;
        nextY = baseLayout.y + deltaY;
      }

      if (handle.x === -1 && nextWidth < minSize) {
        nextX = baseLayout.x + (baseWidth - minSize);
        nextWidth = minSize;
      }
      if (handle.x === 1 && nextWidth < minSize) {
        nextWidth = minSize;
      }
      if (handle.y === -1 && nextHeight < minSize) {
        nextY = baseLayout.y + (baseHeight - minSize);
        nextHeight = minSize;
      }
      if (handle.y === 1 && nextHeight < minSize) {
        nextHeight = minSize;
      }

      if (containerMinSize) {
        if (nextWidth < containerMinSize.width) {
          nextWidth = containerMinSize.width;
        }
        if (nextHeight < containerMinSize.height) {
          nextHeight = containerMinSize.height;
        }
      }
      if (childMinSize) {
        if (handle.x === -1 && nextWidth < childMinSize.width) {
          nextX = baseLayout.x + (baseWidth - childMinSize.width);
          nextWidth = childMinSize.width;
        } else if (handle.x === 1 && nextWidth < childMinSize.width) {
          nextWidth = childMinSize.width;
        }
        if (handle.y === -1 && nextHeight < childMinSize.height) {
          nextY = baseLayout.y + (baseHeight - childMinSize.height);
          nextHeight = childMinSize.height;
        } else if (handle.y === 1 && nextHeight < childMinSize.height) {
          nextHeight = childMinSize.height;
        }
      }
      if (node.value.type === "ElLayout") {
        const layoutMinHeight = resolveElLayoutMinHeight(node.value);
        if (layoutMinHeight > 0 && nextHeight < layoutMinHeight) {
          if (handle.y === -1) {
            nextY = baseLayout.y + (baseHeight - layoutMinHeight);
          }
          nextHeight = layoutMinHeight;
        }
      }

      nextWidth = Math.round(nextWidth);
      nextHeight = Math.round(nextHeight);
      nextX = Math.round(nextX);
      nextY = Math.round(nextY);

      const sectionPatch =
        node.value.type === "ElContainer"
          ? buildContainerSectionSizePatch(
            node.value,
            baseWidth,
            nextWidth,
            baseHeight,
            nextHeight,
            baseSectionSizes,
          )
          : null;

      const nextStyle = { ...(node.value.style || {}) };
      if (node.value.type === "ElLayoutRow") {
        nextStyle.height = `${nextHeight}px`;
        if (handle.x !== 0) {
          nextStyle.width = `${nextWidth}px`;
        }
      } else {
        nextStyle.width = `${nextWidth}px`;
        nextStyle.height = `${nextHeight}px`;
      }

      let patch = sectionPatch
        ? {
          style: nextStyle,
          props: { ...(node.value.props || {}), ...sectionPatch },
        }
        : { style: nextStyle };

      if (shouldUpdateAbsolute) {
        const nextAbs = {
          x: nextX,
          y: nextY,
          w: nextWidth,
          h: nextHeight,
          z: baseLayout.z,
        };

        if (nodeRef.value) {
          nodeRef.value.style.position = "absolute";
          nodeRef.value.style.left = `${nextAbs.x}px`;
          nodeRef.value.style.top = `${nextAbs.y}px`;
          nodeRef.value.style.width = `${nextAbs.w}px`;
          nodeRef.value.style.height = `${nextAbs.h}px`;
          nodeRef.value.style.zIndex = `${nextAbs.z}`;
        }

        const nextLayoutItem = {
          ...(node.value.layoutItem || {}),
          free: {
            mode: "abs",
            abs: { ...nextAbs },
          },
        };

        patch = {
          ...patch,
          positioning: "absolute",
          absolutePos: nextAbs,
          layoutItem: nextLayoutItem,
        };
      } else if (nodeRef.value) {
        if (node.value.type !== "ElLayoutRow" || handle.x !== 0) {
          nodeRef.value.style.width = `${nextWidth}px`;
        }
        nodeRef.value.style.height = `${nextHeight}px`;
      }

      if (history.value?.isInTransaction?.()) {
        history.value.executeInTransaction(
          new UpdateNodeCommand(node.value.id, patch),
        );
        if (typeof window !== "undefined") {
          window.dispatchEvent(
            new CustomEvent("designer:node-transform", {
              detail: { x: nextX, y: nextY },
            }),
          );
        }
        return;
      }
      if (history.value?.execute) {
        history.value.execute(new UpdateNodeCommand(node.value.id, patch));
        if (typeof window !== "undefined") {
          window.dispatchEvent(
            new CustomEvent("designer:node-transform", {
              detail: { x: nextX, y: nextY },
            }),
          );
        }
        return;
      }
      if (doc.value?._updateNode) {
        doc.value._updateNode(node.value.id, patch);
        if (typeof window !== "undefined") {
          window.dispatchEvent(
            new CustomEvent("designer:node-transform", {
              detail: { x: nextX, y: nextY },
            }),
          );
        }
      }
    };

    const up = () => {
      cleanupDragHandlers();
      if (history.value?.isInTransaction?.()) {
        history.value.commitTransaction("调整尺寸");
      }
      if (typeof window !== "undefined") {
        window.dispatchEvent(new CustomEvent("designer:node-transform-end"));
      }
    };

    activeDragHandlers = {
      move,
      up,
      userSelect: originalUserSelect,
      pointerTarget,
      pointerId: event.pointerId,
      usePointer,
      pointerEvents: originalPointerEvents,
      pointerElement,
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

  return {
    handleResizePointerDown,
  };
}
