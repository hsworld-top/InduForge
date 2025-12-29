/**
 * Style Converter Tests
 * 
 * Task 3.8: 测试样式转换器
 * 
 * 测试内容：
 * - 样式映射正确性
 * - 单位转换
 * - Flexbox 属性
 * - Grid 属性
 * - 变换属性
 */

import { describe, it, expect } from 'vitest';
import { convertDslStyleToCss } from '../styleConverter';

describe('Style Converter', () => {
    describe('Position and Size', () => {
        it('should convert position properties', () => {
            const dslStyle = {
                left: 100,
                top: 50,
                right: 20,
                bottom: 30,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.left).toBe('100px');
            expect(cssStyle.top).toBe('50px');
            expect(cssStyle.right).toBe('20px');
            expect(cssStyle.bottom).toBe('30px');
            expect(cssStyle.position).toBe('absolute');
        });

        it('should convert size properties', () => {
            const dslStyle = {
                width: 200,
                height: 150,
                minWidth: 100,
                maxWidth: 300,
                minHeight: 80,
                maxHeight: 250,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.width).toBe('200px');
            expect(cssStyle.height).toBe('150px');
            expect(cssStyle.minWidth).toBe('100px');
            expect(cssStyle.maxWidth).toBe('300px');
            expect(cssStyle.minHeight).toBe('80px');
            expect(cssStyle.maxHeight).toBe('250px');
        });

        it('should handle string values with units', () => {
            const dslStyle = {
                width: '50%',
                height: '100vh',
                left: '2em',
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.width).toBe('50%');
            expect(cssStyle.height).toBe('100vh');
            expect(cssStyle.left).toBe('2em');
        });
    });

    describe('Background and Border', () => {
        it('should convert background properties', () => {
            const dslStyle = {
                backgroundColor: '#409EFF',
                backgroundImage: 'url(image.png)',
                backgroundSize: 'cover',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat',
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.backgroundColor).toBe('#409EFF');
            expect(cssStyle.backgroundImage).toBe('url(image.png)');
            expect(cssStyle.backgroundSize).toBe('cover');
            expect(cssStyle.backgroundPosition).toBe('center');
            expect(cssStyle.backgroundRepeat).toBe('no-repeat');
        });

        it('should convert border properties', () => {
            const dslStyle = {
                border: '1px solid #ccc',
                borderRadius: 4,
                borderColor: '#409EFF',
                borderWidth: 2,
                borderStyle: 'dashed',
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.border).toBe('1px solid #ccc');
            expect(cssStyle.borderRadius).toBe('4px');
            expect(cssStyle.borderColor).toBe('#409EFF');
            expect(cssStyle.borderWidth).toBe('2px');
            expect(cssStyle.borderStyle).toBe('dashed');
        });
    });

    describe('Spacing', () => {
        it('should convert padding properties', () => {
            const dslStyle = {
                padding: 10,
                paddingTop: 5,
                paddingRight: 15,
                paddingBottom: 8,
                paddingLeft: 12,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.padding).toBe('10px');
            expect(cssStyle.paddingTop).toBe('5px');
            expect(cssStyle.paddingRight).toBe('15px');
            expect(cssStyle.paddingBottom).toBe('8px');
            expect(cssStyle.paddingLeft).toBe('12px');
        });

        it('should convert margin properties', () => {
            const dslStyle = {
                margin: 20,
                marginTop: 10,
                marginRight: 25,
                marginBottom: 15,
                marginLeft: 18,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.margin).toBe('20px');
            expect(cssStyle.marginTop).toBe('10px');
            expect(cssStyle.marginRight).toBe('25px');
            expect(cssStyle.marginBottom).toBe('15px');
            expect(cssStyle.marginLeft).toBe('18px');
        });
    });

    describe('Transform', () => {
        it('should convert rotation', () => {
            const dslStyle = {
                rotation: 45,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.transform).toBe('rotate(45deg)');
        });

        it('should convert scale', () => {
            const dslStyle = {
                scale: 1.5,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.transform).toBe('scale(1.5)');
        });

        it('should convert multiple transforms', () => {
            const dslStyle = {
                rotation: 30,
                scale: 1.2,
                translateX: 50,
                translateY: 100,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.transform).toBe('rotate(30deg) scale(1.2) translateX(50px) translateY(100px)');
        });

        it('should convert transformOrigin', () => {
            const dslStyle = {
                rotation: 45,
                transformOrigin: 'center center',
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.transformOrigin).toBe('center center');
        });
    });

    describe('Flexbox Properties', () => {
        it('should convert flex container properties', () => {
            const dslStyle = {
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center',
                alignItems: 'flex-start',
                flexWrap: 'wrap',
                gap: 10,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.display).toBe('flex');
            expect(cssStyle.flexDirection).toBe('column');
            expect(cssStyle.justifyContent).toBe('center');
            expect(cssStyle.alignItems).toBe('flex-start');
            expect(cssStyle.flexWrap).toBe('wrap');
            expect(cssStyle.gap).toBe('10px');
        });

        it('should convert flex item properties', () => {
            const dslStyle = {
                flex: '1 1 auto',
                flexGrow: 1,
                flexShrink: 0,
                flexBasis: '200px',
                alignSelf: 'center',
                order: 2,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.flex).toBe('1 1 auto');
            expect(cssStyle.flexGrow).toBe(1);
            expect(cssStyle.flexShrink).toBe(0);
            expect(cssStyle.flexBasis).toBe('200px');
            expect(cssStyle.alignSelf).toBe('center');
            expect(cssStyle.order).toBe(2);
        });

        it('should convert gap properties', () => {
            const dslStyle = {
                gap: 20,
                rowGap: 15,
                columnGap: 25,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.gap).toBe('20px');
            expect(cssStyle.rowGap).toBe('15px');
            expect(cssStyle.columnGap).toBe('25px');
        });
    });

    describe('Grid Properties', () => {
        it('should convert grid container properties', () => {
            const dslStyle = {
                display: 'grid',
                gridTemplateColumns: 'repeat(3, 1fr)',
                gridTemplateRows: '100px auto',
                gridAutoFlow: 'dense',
                gap: 10,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.display).toBe('grid');
            expect(cssStyle.gridTemplateColumns).toBe('repeat(3, 1fr)');
            expect(cssStyle.gridTemplateRows).toBe('100px auto');
            expect(cssStyle.gridAutoFlow).toBe('dense');
            expect(cssStyle.gap).toBe('10px');
        });

        it('should convert grid item properties', () => {
            const dslStyle = {
                gridColumn: '1 / 3',
                gridRow: '2 / 4',
                gridArea: 'header',
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.gridColumn).toBe('1 / 3');
            expect(cssStyle.gridRow).toBe('2 / 4');
            expect(cssStyle.gridArea).toBe('header');
        });

        it('should convert grid template areas', () => {
            const dslStyle = {
                gridTemplateAreas: '"header header" "sidebar content" "footer footer"',
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.gridTemplateAreas).toBe('"header header" "sidebar content" "footer footer"');
        });
    });

    describe('Text Properties', () => {
        it('should convert text properties', () => {
            const dslStyle = {
                color: '#333',
                fontSize: 16,
                fontFamily: 'Arial, sans-serif',
                fontWeight: 'bold',
                lineHeight: 1.5,
                textAlign: 'center',
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.color).toBe('#333');
            expect(cssStyle.fontSize).toBe('16px');
            expect(cssStyle.fontFamily).toBe('Arial, sans-serif');
            expect(cssStyle.fontWeight).toBe('bold');
            expect(cssStyle.lineHeight).toBe(1.5);
            expect(cssStyle.textAlign).toBe('center');
        });
    });

    describe('Other Properties', () => {
        it('should convert opacity and z-index', () => {
            const dslStyle = {
                opacity: 0.8,
                zIndex: 100,
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.opacity).toBe(0.8);
            expect(cssStyle.zIndex).toBe(100);
        });

        it('should convert display and overflow', () => {
            const dslStyle = {
                display: 'block',
                overflow: 'hidden',
                visibility: 'visible',
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.display).toBe('block');
            expect(cssStyle.overflow).toBe('hidden');
            expect(cssStyle.visibility).toBe('visible');
        });

        it('should handle empty or null input', () => {
            expect(convertDslStyleToCss(null)).toEqual({});
            expect(convertDslStyleToCss(undefined)).toEqual({});
            expect(convertDslStyleToCss({})).toEqual({});
        });
    });

    describe('Complex Styles', () => {
        it('should convert complex layout styles', () => {
            const dslStyle = {
                position: 'absolute',
                left: 100,
                top: 50,
                width: 400,
                height: 300,
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'space-between',
                alignItems: 'center',
                padding: 20,
                backgroundColor: '#f5f5f5',
                borderRadius: 8,
                boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
            };

            const cssStyle = convertDslStyleToCss(dslStyle);

            expect(cssStyle.position).toBe('absolute');
            expect(cssStyle.left).toBe('100px');
            expect(cssStyle.top).toBe('50px');
            expect(cssStyle.width).toBe('400px');
            expect(cssStyle.height).toBe('300px');
            expect(cssStyle.display).toBe('flex');
            expect(cssStyle.flexDirection).toBe('column');
            expect(cssStyle.justifyContent).toBe('space-between');
            expect(cssStyle.alignItems).toBe('center');
            expect(cssStyle.padding).toBe('20px');
            expect(cssStyle.backgroundColor).toBe('#f5f5f5');
            expect(cssStyle.borderRadius).toBe('8px');
            expect(cssStyle.boxShadow).toBe('0 2px 8px rgba(0,0,0,0.1)');
        });
    });
});

