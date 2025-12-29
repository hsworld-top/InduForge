/**
 * Layout Properties - Property-based Tests
 * 
 * Task 3.8: Property-based 测试
 * 
 * 使用 fast-check 进行基于属性的测试：
 * - Flexbox 布局正确性
 * - Grid 布局正确性
 * - 样式映射正确性
 */

import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import { convertDslStyleToCss } from '@/utils/styleConverter';

describe('Property 6: Flexbox 布局正确性', () => {
    it('should correctly convert flex direction', () => {
        fc.assert(
            fc.property(fc.constantFrom('row', 'row-reverse', 'column', 'column-reverse'), (flexDirection) => {
                const dslStyle = { flexDirection };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.flexDirection === flexDirection;
            })
        );
    });

    it('should correctly convert justify content', () => {
        fc.assert(
            fc.property(fc.constantFrom('flex-start', 'flex-end', 'center', 'space-between', 'space-around', 'space-evenly'), (justifyContent) => {
                const dslStyle = { justifyContent };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.justifyContent === justifyContent;
            })
        );
    });

    it('should correctly convert align items', () => {
        fc.assert(
            fc.property(fc.constantFrom('flex-start', 'flex-end', 'center', 'stretch', 'baseline'), (alignItems) => {
                const dslStyle = { alignItems };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.alignItems === alignItems;
            })
        );
    });

    it('should correctly convert flex wrap', () => {
        fc.assert(
            fc.property(fc.constantFrom('nowrap', 'wrap', 'wrap-reverse'), (flexWrap) => {
                const dslStyle = { flexWrap };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.flexWrap === flexWrap;
            })
        );
    });

    it('should correctly convert gap with arbitrary positive numbers', () => {
        fc.assert(
            fc.property(fc.nat(1000), (gap) => {
                const dslStyle = { gap };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.gap === `${gap}px`;
            })
        );
    });

    it('should correctly convert flex grow and shrink', () => {
        fc.assert(
            fc.property(fc.nat(10), fc.nat(10), (flexGrow, flexShrink) => {
                const dslStyle = { flexGrow, flexShrink };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.flexGrow === flexGrow && cssStyle.flexShrink === flexShrink;
            })
        );
    });
});

describe('Property 7: Grid 布局正确性', () => {
    it('should correctly convert grid template columns with repeat', () => {
        fc.assert(
            fc.property(fc.integer({ min: 1, max: 12 }), (columns) => {
                const gridTemplateColumns = `repeat(${columns}, 1fr)`;
                const dslStyle = { gridTemplateColumns };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.gridTemplateColumns === gridTemplateColumns;
            })
        );
    });

    it('should correctly convert grid auto flow', () => {
        fc.assert(
            fc.property(fc.constantFrom('row', 'column', 'dense', 'row dense', 'column dense'), (gridAutoFlow) => {
                const dslStyle = { gridAutoFlow };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.gridAutoFlow === gridAutoFlow;
            })
        );
    });

    it('should correctly convert grid gap with arbitrary positive numbers', () => {
        fc.assert(
            fc.property(fc.nat(100), fc.nat(100), (rowGap, columnGap) => {
                const dslStyle = { rowGap, columnGap };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.rowGap === `${rowGap}px` && cssStyle.columnGap === `${columnGap}px`;
            })
        );
    });

    it('should correctly convert grid column and row spans', () => {
        fc.assert(
            fc.property(
                fc.integer({ min: 1, max: 12 }),
                fc.integer({ min: 1, max: 12 }),
                (start, end) => {
                    const gridColumn = `${start} / ${end}`;
                    const dslStyle = { gridColumn };
                    const cssStyle = convertDslStyleToCss(dslStyle);
                    
                    return cssStyle.gridColumn === gridColumn;
                }
            )
        );
    });
});

describe('Property 8: 样式映射正确性', () => {
    it('should convert all position properties to px units', () => {
        fc.assert(
            fc.property(fc.nat(10000), fc.nat(10000), fc.nat(10000), fc.nat(10000), (left, top, right, bottom) => {
                const dslStyle = { left, top, right, bottom };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return (
                    cssStyle.left === `${left}px` &&
                    cssStyle.top === `${top}px` &&
                    cssStyle.right === `${right}px` &&
                    cssStyle.bottom === `${bottom}px`
                );
            })
        );
    });

    it('should convert all size properties to px units', () => {
        fc.assert(
            fc.property(fc.nat(5000), fc.nat(5000), (width, height) => {
                const dslStyle = { width, height };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.width === `${width}px` && cssStyle.height === `${height}px`;
            })
        );
    });

    it('should convert padding properties to px units', () => {
        fc.assert(
            fc.property(fc.nat(200), fc.nat(200), fc.nat(200), fc.nat(200), (paddingTop, paddingRight, paddingBottom, paddingLeft) => {
                const dslStyle = { paddingTop, paddingRight, paddingBottom, paddingLeft };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return (
                    cssStyle.paddingTop === `${paddingTop}px` &&
                    cssStyle.paddingRight === `${paddingRight}px` &&
                    cssStyle.paddingBottom === `${paddingBottom}px` &&
                    cssStyle.paddingLeft === `${paddingLeft}px`
                );
            })
        );
    });

    it('should preserve string values with units', () => {
        fc.assert(
            fc.property(
                fc.constantFrom('px', '%', 'em', 'rem', 'vh', 'vw'),
                fc.nat(100),
                (unit, value) => {
                    const widthValue = `${value}${unit}`;
                    const dslStyle = { width: widthValue };
                    const cssStyle = convertDslStyleToCss(dslStyle);
                    
                    return cssStyle.width === widthValue;
                }
            )
        );
    });

    it('should correctly convert rotation angles', () => {
        fc.assert(
            fc.property(fc.integer({ min: -360, max: 360 }), (rotation) => {
                const dslStyle = { rotation };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.transform === `rotate(${rotation}deg)`;
            })
        );
    });

    it('should correctly convert scale values', () => {
        fc.assert(
            fc.property(fc.double({ min: 0.1, max: 5.0 }), (scale) => {
                const dslStyle = { scale };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.transform === `scale(${scale})`;
            })
        );
    });

    it('should correctly convert opacity values', () => {
        fc.assert(
            fc.property(fc.double({ min: 0, max: 1 }), (opacity) => {
                const dslStyle = { opacity };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.opacity === opacity;
            })
        );
    });

    it('should handle multiple transform properties', () => {
        fc.assert(
            fc.property(
                fc.integer({ min: 0, max: 360 }),
                fc.double({ min: 0.5, max: 2.0 }),
                fc.integer({ min: -100, max: 100 }),
                (rotation, scale, translateX) => {
                    const dslStyle = { rotation, scale, translateX };
                    const cssStyle = convertDslStyleToCss(dslStyle);
                    
                    const expectedTransform = `rotate(${rotation}deg) scale(${scale}) translateX(${translateX}px)`;
                    return cssStyle.transform === expectedTransform;
                }
            )
        );
    });

    it('should handle border radius values', () => {
        fc.assert(
            fc.property(fc.nat(100), (borderRadius) => {
                const dslStyle = { borderRadius };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.borderRadius === `${borderRadius}px`;
            })
        );
    });

    it('should handle z-index values', () => {
        fc.assert(
            fc.property(fc.integer({ min: -100, max: 9999 }), (zIndex) => {
                const dslStyle = { zIndex };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return cssStyle.zIndex === zIndex;
            })
        );
    });
});

describe('Property 9: 复合样式转换正确性', () => {
    it('should correctly convert complex flex layouts', () => {
        fc.assert(
            fc.property(
                fc.constantFrom('row', 'column'),
                fc.constantFrom('flex-start', 'center', 'flex-end'),
                fc.nat(50),
                (flexDirection, justifyContent, gap) => {
                    const dslStyle = {
                        display: 'flex',
                        flexDirection,
                        justifyContent,
                        gap,
                    };
                    const cssStyle = convertDslStyleToCss(dslStyle);
                    
                    return (
                        cssStyle.display === 'flex' &&
                        cssStyle.flexDirection === flexDirection &&
                        cssStyle.justifyContent === justifyContent &&
                        cssStyle.gap === `${gap}px`
                    );
                }
            )
        );
    });

    it('should correctly convert complex grid layouts', () => {
        fc.assert(
            fc.property(fc.integer({ min: 2, max: 6 }), fc.nat(30), (columns, gap) => {
                const gridTemplateColumns = `repeat(${columns}, 1fr)`;
                const dslStyle = {
                    display: 'grid',
                    gridTemplateColumns,
                    gap,
                };
                const cssStyle = convertDslStyleToCss(dslStyle);
                
                return (
                    cssStyle.display === 'grid' &&
                    cssStyle.gridTemplateColumns === gridTemplateColumns &&
                    cssStyle.gap === `${gap}px`
                );
            })
        );
    });
});

