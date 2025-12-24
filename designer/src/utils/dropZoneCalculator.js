/**
 * Drop Zone Calculator - 插入位置计算器
 * 
 * Task 4.6: 实现插入位置计算
 * 
 * 职责：
 * - 计算拖拽到容器时的插入位置
 * - 支持 Flex、Grid、Block、Row/Col 等布局模式
 * - 计算最接近的插入点
 */

/**
 * 计算 Flex 布局的插入位置
 * 
 * @param {Object} containerRect - 容器的边界矩形
 * @param {Array} childrenRects - 子组件的边界矩形数组
 * @param {Object} mousePosition - 鼠标位置 { x, y }
 * @param {string} flexDirection - flex 方向 ('row' | 'column')
 * @returns {Object} 插入信息 { index, insertLine }
 */
export function calculateFlexInsertPosition(containerRect, childrenRects, mousePosition, flexDirection = 'row', flexWrap = 'nowrap') {
    const { x, y } = mousePosition;
    
    // 如果没有子组件，插入到开始位置
    if (!childrenRects || childrenRects.length === 0) {
        return {
            index: 0,
            insertLine: getContainerStartLine(containerRect, flexDirection),
        };
    }
    
    const isHorizontal = flexDirection === 'row' || flexDirection === 'row-reverse';
    const isReverse = flexDirection.includes('reverse');

    if (flexWrap && flexWrap !== 'nowrap' && !isReverse) {
        return calculateWrappedInsertPosition(containerRect, childrenRects, mousePosition, isHorizontal);
    }
    
    // 计算每个子组件到鼠标的距离
    const distances = childrenRects.map((rect, index) => {
        const center = isHorizontal
            ? { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
            : { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
        
        const distance = isHorizontal
            ? Math.abs(x - center.x)
            : Math.abs(y - center.y);
        
        return { index, distance, rect };
    });
    
    // 找到最近的子组件
    distances.sort((a, b) => a.distance - b.distance);
    const nearest = distances[0];
    
    // 确定插入位置（前面还是后面）
    const nearestCenter = isHorizontal
        ? nearest.rect.left + nearest.rect.width / 2
        : nearest.rect.top + nearest.rect.height / 2;
    
    const mousePos = isHorizontal ? x : y;
    const insertBefore = mousePos < nearestCenter;
    
    // 计算实际插入索引（考虑 reverse）
    let insertIndex = nearest.index;
    if (insertBefore) {
        insertIndex = isReverse ? insertIndex + 1 : insertIndex;
    } else {
        insertIndex = isReverse ? insertIndex : insertIndex + 1;
    }
    
    // 计算插入线位置
    const insertLine = getFlexInsertLine(
        nearest.rect,
        insertBefore,
        flexDirection,
        containerRect
    );
    
    return { index: insertIndex, insertLine };
}

/**
 * 计算 Grid 布局的插入位置
 * 
 * @param {Object} containerRect - 容器的边界矩形
 * @param {Array} childrenRects - 子组件的边界矩形数组
 * @param {Object} mousePosition - 鼠标位置 { x, y }
 * @param {Object} gridConfig - Grid 配置 { columns, rows, gap }
 * @returns {Object} 插入信息 { index, insertLine, gridPosition }
 */
export function calculateGridInsertPosition(containerRect, childrenRects, mousePosition, gridConfig = {}) {
    const { gap = 0 } = gridConfig;
    
    // 如果没有子组件，插入到第一个网格
    if (!childrenRects || childrenRects.length === 0) {
        return {
            index: 0,
            gridPosition: { row: 1, column: 1 },
            insertLine: {
                x: containerRect.left,
                y: containerRect.top,
                width: containerRect.width,
                height: 0,
            },
        };
    }
    
    const wrappedResult = calculateWrappedInsertPosition(containerRect, childrenRects, mousePosition, true);
    if (gap && wrappedResult?.insertLine && wrappedResult.insertLine.width === 0) {
        wrappedResult.insertLine.x += gap / 2;
    }
    return wrappedResult;
}

/**
 * 计算 Block 布局的插入位置
 * 
 * @param {Object} containerRect - 容器的边界矩形
 * @param {Array} childrenRects - 子组件的边界矩形数组
 * @param {Object} mousePosition - 鼠标位置 { x, y }
 * @returns {Object} 插入信息 { index, insertLine }
 */
export function calculateBlockInsertPosition(containerRect, childrenRects, mousePosition) {
    const { y } = mousePosition;
    
    // 如果没有子组件，插入到开始位置
    if (!childrenRects || childrenRects.length === 0) {
        return {
            index: 0,
            insertLine: {
                x: containerRect.left,
                y: containerRect.top,
                width: containerRect.width,
                height: 0,
            },
        };
    }
    
    // Block 布局是垂直排列，根据 Y 坐标判断
    for (let i = 0; i < childrenRects.length; i++) {
        const rect = childrenRects[i];
        const centerY = rect.top + rect.height / 2;
        
        if (y < centerY) {
            // 插入到这个组件之前
            return {
                index: i,
                insertLine: {
                    x: containerRect.left,
                    y: rect.top,
                    width: containerRect.width,
                    height: 0,
                },
            };
        }
    }
    
    // 插入到最后
    const lastRect = childrenRects[childrenRects.length - 1];
    return {
        index: childrenRects.length,
        insertLine: {
            x: containerRect.left,
            y: lastRect.bottom,
            width: containerRect.width,
            height: 0,
        },
    };
}

/**
 * 计算 Row/Col 布局的插入位置
 * 
 * @param {Object} containerRect - 容器的边界矩形
 * @param {Array} childrenRects - 子组件的边界矩形数组
 * @param {Object} mousePosition - 鼠标位置 { x, y }
 * @param {number} gutter - 间距
 * @returns {Object} 插入信息 { index, insertLine }
 */
export function calculateRowColInsertPosition(containerRect, childrenRects, mousePosition, gutter = 0, flexWrap = 'nowrap') {
    // Row 布局类似 Flex row
    return calculateFlexInsertPosition(containerRect, childrenRects, mousePosition, 'row', flexWrap);
}

function buildAxisItems(childrenRects, isHorizontal) {
    return childrenRects.map((rect, index) => ({
        index,
        rect,
        mainStart: isHorizontal ? rect.left : rect.top,
        mainEnd: isHorizontal ? rect.right : rect.bottom,
        mainCenter: isHorizontal ? rect.left + rect.width / 2 : rect.top + rect.height / 2,
        crossStart: isHorizontal ? rect.top : rect.left,
        crossEnd: isHorizontal ? rect.bottom : rect.right,
        crossCenter: isHorizontal ? rect.top + rect.height / 2 : rect.left + rect.width / 2,
        crossSize: isHorizontal ? rect.height : rect.width,
    }));
}

function groupItemsByCrossAxis(items) {
    const sorted = [...items].sort((a, b) => {
        if (a.crossCenter === b.crossCenter) {
            return a.mainStart - b.mainStart;
        }
        return a.crossCenter - b.crossCenter;
    });

    const lines = [];
    for (const item of sorted) {
        const lastLine = lines[lines.length - 1];
        const tolerance = Math.max(6, item.crossSize * 0.6);
        if (lastLine && Math.abs(item.crossCenter - lastLine.crossCenter) <= tolerance) {
            lastLine.items.push(item);
            lastLine.crossCenter =
                (lastLine.crossCenter * (lastLine.items.length - 1) + item.crossCenter) / lastLine.items.length;
            lastLine.crossSize = Math.max(lastLine.crossSize, item.crossSize);
            lastLine.crossStart = Math.min(lastLine.crossStart, item.crossStart);
            lastLine.crossEnd = Math.max(lastLine.crossEnd, item.crossEnd);
            lastLine.mainStart = Math.min(lastLine.mainStart, item.mainStart);
            lastLine.mainEnd = Math.max(lastLine.mainEnd, item.mainEnd);
        } else {
            lines.push({
                items: [item],
                crossCenter: item.crossCenter,
                crossSize: item.crossSize,
                crossStart: item.crossStart,
                crossEnd: item.crossEnd,
                mainStart: item.mainStart,
                mainEnd: item.mainEnd,
            });
        }
    }

    lines.forEach((line) => {
        line.items.sort((a, b) => a.mainStart - b.mainStart);
    });

    return lines;
}

function calculateWrappedInsertPosition(containerRect, childrenRects, mousePosition, isHorizontal) {
    if (!childrenRects || childrenRects.length === 0) {
        return {
            index: 0,
            insertLine: getContainerStartLine(containerRect, isHorizontal ? 'row' : 'column'),
        };
    }

    const items = buildAxisItems(childrenRects, isHorizontal);
    const lines = groupItemsByCrossAxis(items);
    if (!lines.length) {
        return {
            index: childrenRects.length,
            insertLine: getContainerStartLine(containerRect, isHorizontal ? 'row' : 'column'),
        };
    }

    const crossPos = isHorizontal ? mousePosition.y : mousePosition.x;
    const mainPos = isHorizontal ? mousePosition.x : mousePosition.y;

    let lineIndex = lines.length - 1;
    for (let i = 0; i < lines.length; i += 1) {
        const line = lines[i];
        const boundary = (line.crossStart + line.crossEnd) / 2;
        if (crossPos < boundary) {
            lineIndex = i;
            break;
        }
    }

    const line = lines[lineIndex];
    let insertPosInLine = line.items.length;
    for (let i = 0; i < line.items.length; i += 1) {
        if (mainPos < line.items[i].mainCenter) {
            insertPosInLine = i;
            break;
        }
    }

    const visualOrder = [];
    for (const row of lines) {
        for (const item of row.items) {
            visualOrder.push(item.index);
        }
    }

    const prefixCount = lines.slice(0, lineIndex).reduce((sum, row) => sum + row.items.length, 0);
    const visualInsertPos = prefixCount + insertPosInLine;
    const insertIndex =
        visualInsertPos >= visualOrder.length ? childrenRects.length : visualOrder[visualInsertPos];

    let insertLine = null;
    if (line.items.length > 0) {
        if (isHorizontal) {
            const targetIndex = Math.min(insertPosInLine, line.items.length - 1);
            const targetRect = line.items[targetIndex].rect;
            const lineTop = line.crossStart;
            const lineHeight = line.crossEnd - line.crossStart;
            const lineX =
                insertPosInLine === 0
                    ? targetRect.left
                    : insertPosInLine >= line.items.length
                      ? line.items[line.items.length - 1].rect.right
                      : targetRect.left;
            insertLine = {
                x: lineX,
                y: lineTop,
                width: 0,
                height: lineHeight,
            };
        } else {
            const targetIndex = Math.min(insertPosInLine, line.items.length - 1);
            const targetRect = line.items[targetIndex].rect;
            const lineLeft = line.crossStart;
            const lineWidth = line.crossEnd - line.crossStart;
            const lineY =
                insertPosInLine === 0
                    ? targetRect.top
                    : insertPosInLine >= line.items.length
                      ? line.items[line.items.length - 1].rect.bottom
                      : targetRect.top;
            insertLine = {
                x: lineLeft,
                y: lineY,
                width: lineWidth,
                height: 0,
            };
        }
    }

    return { index: insertIndex, insertLine };
}

/**
 * 获取容器起始位置的插入线
 * 
 * @param {Object} containerRect - 容器的边界矩形
 * @param {string} flexDirection - flex 方向
 * @returns {Object} 插入线信息
 */
function getContainerStartLine(containerRect, flexDirection) {
    const isHorizontal = flexDirection === 'row' || flexDirection === 'row-reverse';
    
    if (isHorizontal) {
        return {
            x: containerRect.left,
            y: containerRect.top,
            width: 0,
            height: containerRect.height,
        };
    } else {
        return {
            x: containerRect.left,
            y: containerRect.top,
            width: containerRect.width,
            height: 0,
        };
    }
}

/**
 * 获取 Flex 布局的插入线
 * 
 * @param {Object} nearestRect - 最近的子组件矩形
 * @param {boolean} insertBefore - 是否插入到前面
 * @param {string} flexDirection - flex 方向
 * @param {Object} containerRect - 容器矩形
 * @returns {Object} 插入线信息
 */
function getFlexInsertLine(nearestRect, insertBefore, flexDirection, containerRect) {
    const isHorizontal = flexDirection === 'row' || flexDirection === 'row-reverse';
    
    if (isHorizontal) {
        const x = insertBefore ? nearestRect.left : nearestRect.right;
        return {
            x,
            y: containerRect.top,
            width: 0,
            height: containerRect.height,
        };
    } else {
        const y = insertBefore ? nearestRect.top : nearestRect.bottom;
        return {
            x: containerRect.left,
            y,
            width: containerRect.width,
            height: 0,
        };
    }
}

/**
 * 检测鼠标是否在容器上方
 * 
 * @param {Object} containerRect - 容器的边界矩形
 * @param {Object} mousePosition - 鼠标位置 { x, y }
 * @returns {boolean} 是否在容器上方
 */
export function isMouseOverContainer(containerRect, mousePosition) {
    const { x, y } = mousePosition;
    
    return (
        x >= containerRect.left &&
        x <= containerRect.right &&
        y >= containerRect.top &&
        y <= containerRect.bottom
    );
}

/**
 * 计算通用插入位置（自动检测布局类型）
 * 
 * @param {Object} container - 容器信息 { rect, layoutMode, props }
 * @param {Array} childrenRects - 子组件的边界矩形数组
 * @param {Object} mousePosition - 鼠标位置 { x, y }
 * @returns {Object} 插入信息 { index, insertLine }
 */
export function calculateInsertPosition(container, childrenRects, mousePosition) {
    const { rect, layoutMode, props = {} } = container;
    
    switch (layoutMode) {
        case 'flex':
            return calculateFlexInsertPosition(
                rect,
                childrenRects,
                mousePosition,
                props.flexDirection || 'row',
                props.flexWrap || 'nowrap'
            );
        
        case 'grid':
            return calculateGridInsertPosition(
                rect,
                childrenRects,
                mousePosition,
                {
                    columns: props.gridTemplateColumns,
                    gap: props.gap,
                }
            );
        
        case 'block':
            return calculateBlockInsertPosition(rect, childrenRects, mousePosition);
        
        case 'row':
            {
                const rowWrap =
                    typeof props.wrap === 'boolean'
                        ? (props.wrap ? 'wrap' : 'nowrap')
                        : (props.flexWrap || 'nowrap');
                return calculateRowColInsertPosition(
                    rect,
                    childrenRects,
                    mousePosition,
                    props.gutter || 0,
                    rowWrap
                );
            }
        
        default:
            // 默认使用 flex row
            return calculateFlexInsertPosition(rect, childrenRects, mousePosition, 'row');
    }
}
