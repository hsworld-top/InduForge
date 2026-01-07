/**
 * EventEmitter - 简单的事件发射器
 * 用于文档模型的变更事件通知
 */

/**
 * 事件发射器类
 */
export class EventEmitter {
    constructor() {
        /** @type {Map<string, Set<Function>>} */
        this._listeners = new Map();
    }

    /**
     * 订阅事件
     * @param {string} event - 事件名称
     * @param {Function} handler - 事件处理函数
     * @returns {() => void} 取消订阅函数
     */
    on(event, handler) {
        if (!this._listeners.has(event)) {
            this._listeners.set(event, new Set());
        }
        this._listeners.get(event).add(handler);

        // 返回取消订阅函数
        return () => this.off(event, handler);
    }

    /**
     * 订阅一次性事件
     * @param {string} event - 事件名称
     * @param {Function} handler - 事件处理函数
     * @returns {() => void} 取消订阅函数
     */
    once(event, handler) {
        const wrapper = (...args) => {
            this.off(event, wrapper);
            handler.apply(this, args);
        };
        return this.on(event, wrapper);
    }

    /**
     * 取消订阅事件
     * @param {string} event - 事件名称
     * @param {Function} handler - 事件处理函数
     */
    off(event, handler) {
        const handlers = this._listeners.get(event);
        if (handlers) {
            handlers.delete(handler);
            if (handlers.size === 0) {
                this._listeners.delete(event);
            }
        }
    }

    /**
     * 触发事件
     * @param {string} event - 事件名称
     * @param {...*} args - 事件参数
     */
    emit(event, ...args) {
        const handlers = this._listeners.get(event);
        if (handlers) {
            for (const handler of handlers) {
                try {
                    handler.apply(this, args);
                } catch (error) {
                    console.error(`EventEmitter: Error in handler for "${event}"`, error);
                }
            }
        }
    }

    /**
     * 移除所有事件监听
     * @param {string} [event] - 可选的事件名称，不传则移除所有
     */
    removeAllListeners(event) {
        if (event) {
            this._listeners.delete(event);
        } else {
            this._listeners.clear();
        }
    }

    /**
     * 获取事件监听器数量
     * @param {string} event - 事件名称
     * @returns {number}
     */
    listenerCount(event) {
        const handlers = this._listeners.get(event);
        return handlers ? handlers.size : 0;
    }
}

export default EventEmitter;

