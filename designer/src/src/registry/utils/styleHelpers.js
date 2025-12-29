/**
 * Strip layout styles handled by the wrapper and keep size fill.
 * @param {Object} attrsAttrs - attrs from useAttrs()
 */
export function extractLayoutFreeStyle(attrs = {}) {
    const styleAttr = attrs.style || {};
    const { left, top, right, bottom, position, zIndex, ...rest } = styleAttr;
    return {
        width: '100%',
        height: '100%',
        ...rest,
    };
}
