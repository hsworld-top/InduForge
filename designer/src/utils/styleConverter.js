/**
 * Style Converter - DSL 样式转换为 CSS 样式
 *
 * 职责：
 * - 将 DSL Schema 中的样式属性转换为 CSS 样式对象
 * - 处理位置、尺寸、背景、边框、间距、变换等属性
 * - 处理 Flexbox 和 Grid 布局属性
 *
 * Requirements:
 * - Requirement 9: 样式属性映射
 *
 * Task: 1.2 - 创建 DomRenderer.vue (依赖工具)
 */

/**
 * 转换 DSL 样式为 CSS 样式
 *
 * @param {Object} dslStyle - DSL 样式对象
 * @returns {Object} CSS 样式对象
 *
 * @example
 * const dslStyle = {
 *   left: 100,
 *   top: 100,
 *   width: 200,
 *   height: 150,
 *   backgroundColor: '#409EFF',
 *   rotation: 45
 * }
 *
 * const cssStyle = convertDslStyleToCss(dslStyle)
 * // {
 * //   position: 'absolute',
 * //   left: '100px',
 * //   top: '100px',
 * //   width: '200px',
 * //   height: '150px',
 * //   backgroundColor: '#409EFF',
 * //   transform: 'rotate(45deg)'
 * // }
 */
export function convertDslStyleToCss(dslStyle) {
    if (!dslStyle || typeof dslStyle !== 'object') {
        return {};
    }

    const cssStyle = {};

    // ========== 位置和尺寸 ==========

    // 位置
    if (dslStyle.left !== undefined) {
        cssStyle.left = convertToPixels(dslStyle.left);
    }
    if (dslStyle.top !== undefined) {
        cssStyle.top = convertToPixels(dslStyle.top);
    }
    if (dslStyle.right !== undefined) {
        cssStyle.right = convertToPixels(dslStyle.right);
    }
    if (dslStyle.bottom !== undefined) {
        cssStyle.bottom = convertToPixels(dslStyle.bottom);
    }

    // 尺寸
    if (dslStyle.width !== undefined) {
        cssStyle.width = convertToPixels(dslStyle.width);
    }
    if (dslStyle.height !== undefined) {
        cssStyle.height = convertToPixels(dslStyle.height);
    }
    if (dslStyle.minWidth !== undefined) {
        cssStyle.minWidth = convertToPixels(dslStyle.minWidth);
    }
    if (dslStyle.minHeight !== undefined) {
        cssStyle.minHeight = convertToPixels(dslStyle.minHeight);
    }
    if (dslStyle.maxWidth !== undefined) {
        cssStyle.maxWidth = convertToPixels(dslStyle.maxWidth);
    }
    if (dslStyle.maxHeight !== undefined) {
        cssStyle.maxHeight = convertToPixels(dslStyle.maxHeight);
    }

    // 定位方式
    if (dslStyle.position) {
        cssStyle.position = dslStyle.position;
    } else if (dslStyle.left !== undefined || dslStyle.top !== undefined) {
        // 如果有 left/top 但没有指定 position，默认使用 absolute
        cssStyle.position = 'absolute';
    }

    // ========== 背景 ==========

    if (dslStyle.backgroundColor) {
        cssStyle.backgroundColor = dslStyle.backgroundColor;
    }
    if (dslStyle.backgroundImage) {
        cssStyle.backgroundImage = dslStyle.backgroundImage;
    }
    if (dslStyle.backgroundSize) {
        cssStyle.backgroundSize = dslStyle.backgroundSize;
    }
    if (dslStyle.backgroundPosition) {
        cssStyle.backgroundPosition = dslStyle.backgroundPosition;
    }
    if (dslStyle.backgroundRepeat) {
        cssStyle.backgroundRepeat = dslStyle.backgroundRepeat;
    }

    // ========== 边框 ==========

    if (dslStyle.border) {
        cssStyle.border = dslStyle.border;
    }
    if (dslStyle.borderTop) {
        cssStyle.borderTop = dslStyle.borderTop;
    }
    if (dslStyle.borderRight) {
        cssStyle.borderRight = dslStyle.borderRight;
    }
    if (dslStyle.borderBottom) {
        cssStyle.borderBottom = dslStyle.borderBottom;
    }
    if (dslStyle.borderLeft) {
        cssStyle.borderLeft = dslStyle.borderLeft;
    }
    if (dslStyle.borderRadius !== undefined) {
        cssStyle.borderRadius = convertToPixels(dslStyle.borderRadius);
    }
    if (dslStyle.borderColor) {
        cssStyle.borderColor = dslStyle.borderColor;
    }
    if (dslStyle.borderWidth !== undefined) {
        cssStyle.borderWidth = convertToPixels(dslStyle.borderWidth);
    }
    if (dslStyle.borderStyle) {
        cssStyle.borderStyle = dslStyle.borderStyle;
    }

    // ========== 间距 ==========

    // Padding
    if (dslStyle.padding !== undefined) {
        cssStyle.padding = convertToPixels(dslStyle.padding);
    }
    if (dslStyle.paddingTop !== undefined) {
        cssStyle.paddingTop = convertToPixels(dslStyle.paddingTop);
    }
    if (dslStyle.paddingRight !== undefined) {
        cssStyle.paddingRight = convertToPixels(dslStyle.paddingRight);
    }
    if (dslStyle.paddingBottom !== undefined) {
        cssStyle.paddingBottom = convertToPixels(dslStyle.paddingBottom);
    }
    if (dslStyle.paddingLeft !== undefined) {
        cssStyle.paddingLeft = convertToPixels(dslStyle.paddingLeft);
    }

    // Margin
    if (dslStyle.margin !== undefined) {
        cssStyle.margin = convertToPixels(dslStyle.margin);
    }
    if (dslStyle.marginTop !== undefined) {
        cssStyle.marginTop = convertToPixels(dslStyle.marginTop);
    }
    if (dslStyle.marginRight !== undefined) {
        cssStyle.marginRight = convertToPixels(dslStyle.marginRight);
    }
    if (dslStyle.marginBottom !== undefined) {
        cssStyle.marginBottom = convertToPixels(dslStyle.marginBottom);
    }
    if (dslStyle.marginLeft !== undefined) {
        cssStyle.marginLeft = convertToPixels(dslStyle.marginLeft);
    }

    // ========== 阴影和透明度 ==========

    if (dslStyle.boxShadow) {
        cssStyle.boxShadow = dslStyle.boxShadow;
    }
    if (dslStyle.textShadow) {
        cssStyle.textShadow = dslStyle.textShadow;
    }
    if (dslStyle.opacity !== undefined) {
        cssStyle.opacity = dslStyle.opacity;
    }

    // ========== 变换 ==========

    const transforms = [];

    if (dslStyle.rotation !== undefined) {
        transforms.push(`rotate(${dslStyle.rotation}deg)`);
    }
    if (dslStyle.scale !== undefined) {
        transforms.push(`scale(${dslStyle.scale})`);
    }
    if (dslStyle.scaleX !== undefined) {
        transforms.push(`scaleX(${dslStyle.scaleX})`);
    }
    if (dslStyle.scaleY !== undefined) {
        transforms.push(`scaleY(${dslStyle.scaleY})`);
    }
    if (dslStyle.translateX !== undefined) {
        transforms.push(`translateX(${convertToPixels(dslStyle.translateX)})`);
    }
    if (dslStyle.translateY !== undefined) {
        transforms.push(`translateY(${convertToPixels(dslStyle.translateY)})`);
    }
    if (dslStyle.skewX !== undefined) {
        transforms.push(`skewX(${dslStyle.skewX}deg)`);
    }
    if (dslStyle.skewY !== undefined) {
        transforms.push(`skewY(${dslStyle.skewY}deg)`);
    }

    if (transforms.length > 0) {
        cssStyle.transform = transforms.join(' ');
    }

    if (dslStyle.transformOrigin) {
        cssStyle.transformOrigin = dslStyle.transformOrigin;
    }

    // ========== 显示和布局 ==========

    if (dslStyle.display) {
        cssStyle.display = dslStyle.display;
    }
    if (dslStyle.overflow) {
        cssStyle.overflow = dslStyle.overflow;
    }
    if (dslStyle.overflowX) {
        cssStyle.overflowX = dslStyle.overflowX;
    }
    if (dslStyle.overflowY) {
        cssStyle.overflowY = dslStyle.overflowY;
    }
    if (dslStyle.visibility) {
        cssStyle.visibility = dslStyle.visibility;
    }
    if (dslStyle.zIndex !== undefined) {
        cssStyle.zIndex = dslStyle.zIndex;
    }

    // ========== Flexbox 容器属性 ==========

    if (dslStyle.flexDirection) {
        cssStyle.flexDirection = dslStyle.flexDirection;
    }
    if (dslStyle.justifyContent) {
        cssStyle.justifyContent = dslStyle.justifyContent;
    }
    if (dslStyle.alignItems) {
        cssStyle.alignItems = dslStyle.alignItems;
    }
    if (dslStyle.alignContent) {
        cssStyle.alignContent = dslStyle.alignContent;
    }
    if (dslStyle.flexWrap) {
        cssStyle.flexWrap = dslStyle.flexWrap;
    }
    if (dslStyle.gap !== undefined) {
        cssStyle.gap = convertToPixels(dslStyle.gap);
    }
    if (dslStyle.rowGap !== undefined) {
        cssStyle.rowGap = convertToPixels(dslStyle.rowGap);
    }
    if (dslStyle.columnGap !== undefined) {
        cssStyle.columnGap = convertToPixels(dslStyle.columnGap);
    }

    // ========== Flexbox 子项属性 ==========

    if (dslStyle.flex) {
        cssStyle.flex = dslStyle.flex;
    }
    if (dslStyle.flexGrow !== undefined) {
        cssStyle.flexGrow = dslStyle.flexGrow;
    }
    if (dslStyle.flexShrink !== undefined) {
        cssStyle.flexShrink = dslStyle.flexShrink;
    }
    if (dslStyle.flexBasis) {
        cssStyle.flexBasis = dslStyle.flexBasis;
    }
    if (dslStyle.alignSelf) {
        cssStyle.alignSelf = dslStyle.alignSelf;
    }
    if (dslStyle.order !== undefined) {
        cssStyle.order = dslStyle.order;
    }

    // ========== Grid 容器属性 ==========

    if (dslStyle.gridTemplateColumns) {
        cssStyle.gridTemplateColumns = dslStyle.gridTemplateColumns;
    }
    if (dslStyle.gridTemplateRows) {
        cssStyle.gridTemplateRows = dslStyle.gridTemplateRows;
    }
    if (dslStyle.gridTemplateAreas) {
        cssStyle.gridTemplateAreas = dslStyle.gridTemplateAreas;
    }
    if (dslStyle.gridAutoColumns) {
        cssStyle.gridAutoColumns = dslStyle.gridAutoColumns;
    }
    if (dslStyle.gridAutoRows) {
        cssStyle.gridAutoRows = dslStyle.gridAutoRows;
    }
    if (dslStyle.gridAutoFlow) {
        cssStyle.gridAutoFlow = dslStyle.gridAutoFlow;
    }

    // ========== Grid 子项属性 ==========

    if (dslStyle.gridColumn) {
        cssStyle.gridColumn = dslStyle.gridColumn;
    }
    if (dslStyle.gridRow) {
        cssStyle.gridRow = dslStyle.gridRow;
    }
    if (dslStyle.gridArea) {
        cssStyle.gridArea = dslStyle.gridArea;
    }
    if (dslStyle.gridColumnStart) {
        cssStyle.gridColumnStart = dslStyle.gridColumnStart;
    }
    if (dslStyle.gridColumnEnd) {
        cssStyle.gridColumnEnd = dslStyle.gridColumnEnd;
    }
    if (dslStyle.gridRowStart) {
        cssStyle.gridRowStart = dslStyle.gridRowStart;
    }
    if (dslStyle.gridRowEnd) {
        cssStyle.gridRowEnd = dslStyle.gridRowEnd;
    }

    // ========== 文本样式 ==========

    if (dslStyle.color) {
        cssStyle.color = dslStyle.color;
    }
    if (dslStyle.fontSize !== undefined) {
        cssStyle.fontSize = convertToPixels(dslStyle.fontSize);
    }
    if (dslStyle.fontFamily) {
        cssStyle.fontFamily = dslStyle.fontFamily;
    }
    if (dslStyle.fontWeight) {
        cssStyle.fontWeight = dslStyle.fontWeight;
    }
    if (dslStyle.fontStyle) {
        cssStyle.fontStyle = dslStyle.fontStyle;
    }
    if (dslStyle.lineHeight !== undefined) {
        cssStyle.lineHeight = typeof dslStyle.lineHeight === 'number' ? dslStyle.lineHeight : convertToPixels(dslStyle.lineHeight);
    }
    if (dslStyle.textAlign) {
        cssStyle.textAlign = dslStyle.textAlign;
    }
    if (dslStyle.textDecoration) {
        cssStyle.textDecoration = dslStyle.textDecoration;
    }
    if (dslStyle.textTransform) {
        cssStyle.textTransform = dslStyle.textTransform;
    }
    if (dslStyle.letterSpacing !== undefined) {
        cssStyle.letterSpacing = convertToPixels(dslStyle.letterSpacing);
    }
    if (dslStyle.wordSpacing !== undefined) {
        cssStyle.wordSpacing = convertToPixels(dslStyle.wordSpacing);
    }
    if (dslStyle.whiteSpace) {
        cssStyle.whiteSpace = dslStyle.whiteSpace;
    }

    // ========== 其他属性 ==========

    if (dslStyle.cursor) {
        cssStyle.cursor = dslStyle.cursor;
    }
    if (dslStyle.pointerEvents) {
        cssStyle.pointerEvents = dslStyle.pointerEvents;
    }
    if (dslStyle.userSelect) {
        cssStyle.userSelect = dslStyle.userSelect;
    }

    return cssStyle;
}

/**
 * 转换数值为像素字符串
 *
 * @param {number|string} value - 数值或字符串
 * @returns {string} 像素字符串
 *
 * @example
 * convertToPixels(100) // '100px'
 * convertToPixels('100px') // '100px'
 * convertToPixels('50%') // '50%'
 */
function convertToPixels(value) {
    if (typeof value === 'number') {
        return `${value}px`;
    }
    if (typeof value === 'string') {
        // 如果已经包含单位，直接返回
        if (value.includes('px') || value.includes('%') || value.includes('em') || value.includes('rem') || value.includes('vh') || value.includes('vw') || value.includes('auto')) {
            return value;
        }
        // 如果是纯数字字符串，添加 px
        if (!isNaN(value)) {
            return `${value}px`;
        }
    }
    return value;
}
