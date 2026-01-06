/**
 * Layout Components Tests
 * 
 * Task 3.8: 测试布局组件
 * 
 * 测试内容：
 * - 所有布局容器渲染
 * - CSS 布局属性应用
 * - 容器嵌套
 */

import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import Container from '../Container.vue';
import Flex from '../Flex.vue';
import Grid from '../Grid.vue';
import Row from '../Row.vue';
import Col from '../Col.vue';
import CenterLayout from '../CenterLayout.vue';

describe('Layout Components', () => {
    describe('Container Component', () => {
        it('should render with flex layout mode', () => {
            const wrapper = mount(Container, {
                props: {
                    layoutMode: 'flex',
                    flexDirection: 'row',
                    justifyContent: 'center',
                    alignItems: 'center',
                },
            });

            expect(wrapper.exists()).toBe(true);
            expect(wrapper.classes()).toContain('layout-container');
            expect(wrapper.classes()).toContain('layout-mode-flex');
            
            const style = wrapper.element.style;
            expect(style.display).toBe('flex');
            expect(style.flexDirection).toBe('row');
            expect(style.justifyContent).toBe('center');
            expect(style.alignItems).toBe('center');
        });

        it('should render with grid layout mode', () => {
            const wrapper = mount(Container, {
                props: {
                    layoutMode: 'grid',
                    gridTemplateColumns: 'repeat(3, 1fr)',
                    gridTemplateRows: 'auto',
                    gap: 20,
                },
            });

            expect(wrapper.classes()).toContain('layout-mode-grid');
            
            const style = wrapper.element.style;
            expect(style.display).toBe('block');
            expect(style.getPropertyValue('--grid-columns')).toBe('3');
            expect(style.getPropertyValue('--grid-gap')).toBe('20px');
        });

        it('should render with block layout mode', () => {
            const wrapper = mount(Container, {
                props: {
                    layoutMode: 'block',
                },
            });

            expect(wrapper.classes()).toContain('layout-mode-block');
            
            const style = wrapper.element.style;
            expect(style.display).toBe('block');
        });

        it('should show drop zone indicator', () => {
            const wrapper = mount(Container, {
                props: {
                    isDropZone: true,
                },
            });

            expect(wrapper.classes()).toContain('is-drop-zone');
        });

        it('should show empty hint when isEmpty is true', () => {
            const wrapper = mount(Container, {
                props: {
                    isEmpty: true,
                },
            });

            const hint = wrapper.find('.empty-hint');
            expect(hint.exists()).toBe(true);
            expect(hint.text()).toBe('拖拽组件到此处');
        });

        it('should render children in slot', () => {
            const wrapper = mount(Container, {
                slots: {
                    default: '<div class="test-child">Child Content</div>',
                },
            });

            const child = wrapper.find('.test-child');
            expect(child.exists()).toBe(true);
            expect(child.text()).toBe('Child Content');
        });
    });

    describe('Flex Component', () => {
        it('should render with default flex properties', () => {
            const wrapper = mount(Flex);

            expect(wrapper.exists()).toBe(true);
            expect(wrapper.classes()).toContain('layout-flex');
            
            const style = wrapper.element.style;
            expect(style.display).toBe('flex');
            expect(style.flexDirection).toBe('row');
            expect(style.justifyContent).toBe('flex-start');
            expect(style.alignItems).toBe('flex-start');
        });

        it('should apply custom flex properties', () => {
            const wrapper = mount(Flex, {
                props: {
                    flexDirection: 'column',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    flexWrap: 'wrap',
                    gap: 20,
                },
            });

            const style = wrapper.element.style;
            expect(style.flexDirection).toBe('column');
            expect(style.justifyContent).toBe('space-between');
            expect(style.alignItems).toBe('center');
            expect(style.flexWrap).toBe('wrap');
            expect(style.gap).toBe('20px');
        });

        it('should validate flexDirection prop', () => {
            const wrapper = mount(Flex, {
                props: {
                    flexDirection: 'row-reverse',
                },
            });

            const style = wrapper.element.style;
            expect(style.flexDirection).toBe('row-reverse');
        });

        it('should render children in slot', () => {
            const wrapper = mount(Flex, {
                slots: {
                    default: '<div class="flex-child">Flex Item</div>',
                },
            });

            const child = wrapper.find('.flex-child');
            expect(child.exists()).toBe(true);
        });
    });

    describe('Grid Component', () => {
        it('should render with default grid properties', () => {
            const wrapper = mount(Grid);

            expect(wrapper.exists()).toBe(true);
            expect(wrapper.classes()).toContain('layout-grid');
            
            const style = wrapper.element.style;
            expect(style.display).toBe('grid');
            expect(style.gridTemplateColumns).toBe('repeat(3, 1fr)');
        });

        it('should apply custom grid properties', () => {
            const wrapper = mount(Grid, {
                props: {
                    gridTemplateColumns: 'repeat(4, 1fr)',
                    gridTemplateRows: '100px 200px',
                    gap: 15,
                    gridAutoFlow: 'column',
                },
            });

            const style = wrapper.element.style;
            expect(style.gridTemplateColumns).toBe('repeat(4, 1fr)');
            expect(style.gridTemplateRows).toBe('100px 200px');
            expect(style.gap).toBe('15px');
            expect(style.gridAutoFlow).toBe('column');
        });

        it('should render children in slot', () => {
            const wrapper = mount(Grid, {
                slots: {
                    default: '<div class="grid-child">Grid Item</div>',
                },
            });

            const child = wrapper.find('.grid-child');
            expect(child.exists()).toBe(true);
        });
    });

    describe('Row Component', () => {
        it('should render with default row properties', () => {
            const wrapper = mount(Row);

            expect(wrapper.exists()).toBe(true);
            expect(wrapper.classes()).toContain('layout-row');
            
            expect(wrapper.props('justify')).toBe('start');
            expect(wrapper.props('align')).toBe('top');
        });

        it('should apply gutter spacing', () => {
            const wrapper = mount(Row, {
                props: {
                    gutter: 20,
                },
            });

            expect(wrapper.props('gutter')).toBe(20);
        });

        it('should apply justify and align properties', () => {
            const wrapper = mount(Row, {
                props: {
                    justify: 'center',
                    align: 'middle',
                },
            });

            expect(wrapper.props('justify')).toBe('center');
            expect(wrapper.props('align')).toBe('middle');
        });

        it('should render children in slot', () => {
            const wrapper = mount(Row, {
                slots: {
                    default: '<div class="row-child">Row Item</div>',
                },
            });

            const child = wrapper.find('.row-child');
            expect(child.exists()).toBe(true);
        });
    });

    describe('Col Component', () => {
        it('should render with default col properties', () => {
            const wrapper = mount(Col);

            expect(wrapper.exists()).toBe(true);
            expect(wrapper.classes()).toContain('layout-col');
            
            expect(wrapper.props('span')).toBe(24);
        });

        it('should apply span property', () => {
            const wrapper = mount(Col, {
                props: {
                    span: 12,
                },
            });

            expect(wrapper.props('span')).toBe(12);
        });

        it('should apply offset property', () => {
            const wrapper = mount(Col, {
                props: {
                    span: 12,
                    offset: 6,
                },
            });

            expect(wrapper.props('offset')).toBe(6);
        });

        it('should render children in slot', () => {
            const wrapper = mount(Col, {
                slots: {
                    default: '<div class="col-child">Col Item</div>',
                },
            });

            const child = wrapper.find('.col-child');
            expect(child.exists()).toBe(true);
        });
    });

    describe('CenterLayout Component', () => {
        it('should render with both horizontal and vertical centering', () => {
            const wrapper = mount(CenterLayout);

            expect(wrapper.exists()).toBe(true);
            expect(wrapper.classes()).toContain('layout-center');
            
            const style = wrapper.element.style;
            expect(style.display).toBe('flex');
            expect(style.justifyContent).toBe('center');
            expect(style.alignItems).toBe('center');
        });

        it('should render with horizontal centering only', () => {
            const wrapper = mount(CenterLayout, {
                props: {
                    horizontal: true,
                    vertical: false,
                },
            });

            const style = wrapper.element.style;
            expect(style.justifyContent).toBe('center');
            expect(style.alignItems).not.toBe('center');
        });

        it('should render with vertical centering only', () => {
            const wrapper = mount(CenterLayout, {
                props: {
                    horizontal: false,
                    vertical: true,
                },
            });

            const style = wrapper.element.style;
            expect(style.justifyContent).not.toBe('center');
            expect(style.alignItems).toBe('center');
        });

        it('should render children in slot', () => {
            const wrapper = mount(CenterLayout, {
                slots: {
                    default: '<div class="center-child">Centered Content</div>',
                },
            });

            const child = wrapper.find('.center-child');
            expect(child.exists()).toBe(true);
        });
    });

    describe('Container Nesting', () => {
        it('should support nesting containers', () => {
            const wrapper = mount(Container, {
                props: {
                    layoutMode: 'flex',
                },
                slots: {
                    default: `
                        <Container layoutMode="grid">
                            <div class="nested-child">Nested Content</div>
                        </Container>
                    `,
                },
                global: {
                    components: {
                        Container,
                    },
                },
            });

            expect(wrapper.exists()).toBe(true);
            const nestedContainer = wrapper.findComponent(Container);
            expect(nestedContainer.exists()).toBe(true);
        });

        it('should support Row/Col nesting', () => {
            const wrapper = mount(Row, {
                slots: {
                    default: `
                        <Col :span="12">
                            <div class="col-content">Column 1</div>
                        </Col>
                        <Col :span="12">
                            <div class="col-content">Column 2</div>
                        </Col>
                    `,
                },
                global: {
                    components: {
                        Col,
                    },
                },
            });

            expect(wrapper.exists()).toBe(true);
        });
    });
});

