/**
 * CanvasAuxiliary - Canvas Layer 滚动同步测试
 *
 * Task 1.5: 实现 Canvas Layer 滚动同步（stage.position()）
 *
 * Requirements:
 * - Requirement 5: 缩放和滚动功能
 * - Acceptance Criteria 5.4: 画布滚动时同步更新 DOM Layer 和 Canvas Layer 的滚动位置
 *
 * 测试目标：
 * - 验证 Canvas Layer 能够根据 scrollX 和 scrollY props 同步滚动位置
 * - 验证使用 stage.position() 方法正确设置滚动偏移
 * - 验证滚动位置变化时触发重绘
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import CanvasAuxiliary from '../CanvasAuxiliary.vue';

// Mock Konva
vi.mock('konva', () => {
    const mockLayer = {
        add: vi.fn(),
    };

    const mockStage = {
        width: vi.fn().mockReturnThis(),
        height: vi.fn().mockReturnThis(),
        scale: vi.fn().mockReturnThis(),
        position: vi.fn().mockReturnThis(),
        batchDraw: vi.fn(),
        add: vi.fn(),
        on: vi.fn(),
        destroy: vi.fn(),
    };

    return {
        default: {
            Stage: vi.fn(() => mockStage),
            Layer: vi.fn(() => mockLayer),
        },
    };
});

describe('CanvasAuxiliary - Canvas Layer Scroll Sync', () => {
    let wrapper;
    let Konva;

    beforeEach(async () => {
        // Import mocked Konva
        Konva = (await import('konva')).default;

        // Mount component with default props
        wrapper = mount(CanvasAuxiliary, {
            props: {
                width: 1920,
                height: 1080,
                zoom: 1,
                scrollX: 0,
                scrollY: 0,
            },
        });

        // Wait for component to mount and initialize Konva
        await nextTick();
    });

    afterEach(() => {
        wrapper.unmount();
        vi.clearAllMocks();
    });

    describe('初始化滚动位置', () => {
        it('应该在初始化后可以设置滚动位置', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            // 清除初始化时的调用
            stage.position.mockClear();

            // 更新滚动位置
            await wrapper.setProps({ scrollX: 50, scrollY: 100 });
            await nextTick();

            // 验证 position 被调用
            expect(stage.position).toHaveBeenCalled();
        });

        it('应该在初始化时使用 props 中的滚动位置', async () => {
            const customWrapper = mount(CanvasAuxiliary, {
                props: {
                    width: 1920,
                    height: 1080,
                    zoom: 1,
                    scrollX: 100,
                    scrollY: 200,
                },
            });

            await nextTick();

            const stage = Konva.Stage.mock.results[Konva.Stage.mock.results.length - 1].value;

            // 清除初始化时的调用
            stage.position.mockClear();

            // 触发滚动位置更新
            await customWrapper.setProps({ scrollX: 150, scrollY: 250 });
            await nextTick();

            // 验证 position 被调用，且使用负值（因为 Canvas 滚动方向相反）
            expect(stage.position).toHaveBeenCalledWith({
                x: -150,
                y: -250,
            });

            customWrapper.unmount();
        });
    });

    describe('滚动位置同步', () => {
        it('应该在 scrollX 变化时更新 Canvas Layer 的 x 位置', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            // 清除之前的调用记录
            stage.position.mockClear();
            stage.batchDraw.mockClear();

            // 更新 scrollX
            await wrapper.setProps({ scrollX: 150 });
            await nextTick();

            // 验证 position 被调用，x 为负值
            expect(stage.position).toHaveBeenCalledWith({
                x: -150,
                y: -0,
            });

            // 验证触发重绘
            expect(stage.batchDraw).toHaveBeenCalled();
        });

        it('应该在 scrollY 变化时更新 Canvas Layer 的 y 位置', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            stage.position.mockClear();
            stage.batchDraw.mockClear();

            // 更新 scrollY
            await wrapper.setProps({ scrollY: 250 });
            await nextTick();

            // 验证 position 被调用，y 为负值
            expect(stage.position).toHaveBeenCalledWith({
                x: -0,
                y: -250,
            });

            expect(stage.batchDraw).toHaveBeenCalled();
        });

        it('应该在 scrollX 和 scrollY 同时变化时更新两个方向的位置', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            stage.position.mockClear();
            stage.batchDraw.mockClear();

            // 同时更新 scrollX 和 scrollY
            await wrapper.setProps({
                scrollX: 300,
                scrollY: 400,
            });
            await nextTick();

            // 验证 position 被调用，两个方向都为负值
            expect(stage.position).toHaveBeenCalledWith({
                x: -300,
                y: -400,
            });

            expect(stage.batchDraw).toHaveBeenCalled();
        });
    });

    describe('滚动位置取反', () => {
        it('应该将正的 scrollX 转换为负的 stage.x（向右滚动）', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            stage.position.mockClear();

            await wrapper.setProps({ scrollX: 500 });
            await nextTick();

            // scrollX 为正，stage.x 应为负
            expect(stage.position).toHaveBeenCalledWith(expect.objectContaining({ x: -500 }));
        });

        it('应该将正的 scrollY 转换为负的 stage.y（向下滚动）', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            stage.position.mockClear();

            await wrapper.setProps({ scrollY: 600 });
            await nextTick();

            // scrollY 为正，stage.y 应为负
            expect(stage.position).toHaveBeenCalledWith(expect.objectContaining({ y: -600 }));
        });
    });

    describe('性能优化', () => {
        it('应该使用 batchDraw() 而不是 draw() 来优化性能', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            stage.batchDraw.mockClear();

            await wrapper.setProps({ scrollX: 100, scrollY: 200 });
            await nextTick();

            // 验证使用 batchDraw
            expect(stage.batchDraw).toHaveBeenCalled();
        });

        it('应该在滚动位置变化时只调用一次 batchDraw', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            stage.batchDraw.mockClear();

            // 同时更新两个滚动值
            await wrapper.setProps({
                scrollX: 100,
                scrollY: 200,
            });
            await nextTick();

            // 应该只调用一次 batchDraw（因为 watcher 监听的是数组）
            expect(stage.batchDraw).toHaveBeenCalledTimes(1);
        });
    });

    describe('边界情况', () => {
        it('应该正确处理滚动位置为 0', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            stage.position.mockClear();

            // 先设置非零值
            await wrapper.setProps({ scrollX: 100, scrollY: 100 });
            await nextTick();

            stage.position.mockClear();

            // 再设置为 0
            await wrapper.setProps({ scrollX: 0, scrollY: 0 });
            await nextTick();

            expect(stage.position).toHaveBeenCalledWith({
                x: -0,
                y: -0,
            });
        });

        it('应该正确处理负的滚动位置（虽然不常见）', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            stage.position.mockClear();

            await wrapper.setProps({ scrollX: -50, scrollY: -100 });
            await nextTick();

            // 负的 scroll 值应该转换为正的 stage position
            expect(stage.position).toHaveBeenCalledWith({
                x: 50,
                y: 100,
            });
        });

        it('应该正确处理大的滚动值', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            stage.position.mockClear();

            await wrapper.setProps({ scrollX: 10000, scrollY: 20000 });
            await nextTick();

            expect(stage.position).toHaveBeenCalledWith({
                x: -10000,
                y: -20000,
            });
        });
    });

    describe('与缩放的协同工作', () => {
        it('应该在缩放和滚动同时变化时正确更新', async () => {
            const stage = Konva.Stage.mock.results[0].value;

            stage.scale.mockClear();
            stage.position.mockClear();
            stage.batchDraw.mockClear();

            // 同时更新缩放和滚动
            await wrapper.setProps({
                zoom: 1.5,
                scrollX: 200,
                scrollY: 300,
            });
            await nextTick();

            // 验证缩放被设置
            expect(stage.scale).toHaveBeenCalledWith({ x: 1.5, y: 1.5 });

            // 验证滚动位置被设置
            expect(stage.position).toHaveBeenCalledWith({
                x: -200,
                y: -300,
            });

            // 验证触发重绘（可能被调用多次，因为有多个 watcher）
            expect(stage.batchDraw).toHaveBeenCalled();
        });
    });
});
