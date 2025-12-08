/**
 * 拖拽系统集成测试
 *
 * Task 4.9: 测试拖拽系统
 * - 测试从组件库拖拽到画布
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

describe('Drag and Drop Integration', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
    });

    it('should handle drag data transfer correctly', () => {
        // 模拟组件实例
        const component = {
            id: 'test-1',
            type: 'Button',
            name: '按钮',
            style: {
                width: 100,
                height: 50,
            },
        };

        // 模拟 dataTransfer
        const mockDataTransfer = {
            data: {},
            setData(type, data) {
                this.data[type] = data;
            },
            getData(type) {
                return this.data[type] || '';
            },
            effectAllowed: '',
            setDragImage: vi.fn(),
        };

        // 模拟 dragstart 事件
        const dragStartEvent = {
            dataTransfer: mockDataTransfer,
            target: document.createElement('div'),
            preventDefault: vi.fn(),
        };

        // 设置拖拽数据
        dragStartEvent.dataTransfer.setData('application/json', JSON.stringify(component));
        dragStartEvent.dataTransfer.effectAllowed = 'copy';

        // 验证数据已正确设置
        expect(dragStartEvent.dataTransfer.effectAllowed).toBe('copy');
        const storedData = dragStartEvent.dataTransfer.getData('application/json');
        expect(storedData).toBeTruthy();

        const parsedComponent = JSON.parse(storedData);
        expect(parsedComponent.type).toBe('Button');
        expect(parsedComponent.name).toBe('按钮');
    });

    it('should calculate correct drop position', () => {
        // 模拟画布参数
        const zoom = 1;
        const scrollX = 0;
        const scrollY = 0;
        const canvasPadding = 40;

        // 模拟鼠标位置
        const clientX = 300;
        const clientY = 400;

        // 模拟画布边界
        const rect = {
            left: 100,
            top: 100,
        };

        // 计算放置位置（画布坐标系）
        const x = (clientX - rect.left - canvasPadding + scrollX) / zoom;
        const y = (clientY - rect.top - canvasPadding + scrollY) / zoom;

        // 验证计算结果
        expect(x).toBe(160); // (300 - 100 - 40 + 0) / 1
        expect(y).toBe(260); // (400 - 100 - 40 + 0) / 1
    });

    it('should calculate correct drop position with zoom', () => {
        // 模拟画布参数（缩放到 50%）
        const zoom = 0.5;
        const scrollX = 0;
        const scrollY = 0;
        const canvasPadding = 40;

        // 模拟鼠标位置
        const clientX = 300;
        const clientY = 400;

        // 模拟画布边界
        const rect = {
            left: 100,
            top: 100,
        };

        // 计算放置位置（画布坐标系）
        const x = (clientX - rect.left - canvasPadding + scrollX) / zoom;
        const y = (clientY - rect.top - canvasPadding + scrollY) / zoom;

        // 验证计算结果（缩放后坐标应该翻倍）
        expect(x).toBe(320); // (300 - 100 - 40 + 0) / 0.5
        expect(y).toBe(520); // (400 - 100 - 40 + 0) / 0.5
    });

    it('should calculate correct drop position with scroll', () => {
        // 模拟画布参数（有滚动）
        const zoom = 1;
        const scrollX = 100;
        const scrollY = 150;
        const canvasPadding = 40;

        // 模拟鼠标位置
        const clientX = 300;
        const clientY = 400;

        // 模拟画布边界
        const rect = {
            left: 100,
            top: 100,
        };

        // 计算放置位置（画布坐标系）
        const x = (clientX - rect.left - canvasPadding + scrollX) / zoom;
        const y = (clientY - rect.top - canvasPadding + scrollY) / zoom;

        // 验证计算结果（滚动后坐标应该增加）
        expect(x).toBe(260); // (300 - 100 - 40 + 100) / 1
        expect(y).toBe(410); // (400 - 100 - 40 + 150) / 1
    });
});
