/**
 * EventEmitter - 简单的事件发射器（文档模型变更通知等）
 */

export type EventHandler = (...args: unknown[]) => void;

export class EventEmitter {
  private _listeners = new Map<string, Set<EventHandler>>();

  on(event: string, handler: EventHandler): () => void {
    if (!this._listeners.has(event)) {
      this._listeners.set(event, new Set());
    }
    const set = this._listeners.get(event);
    if (set) {
      set.add(handler);
    }
    return () => this.off(event, handler);
  }

  once(event: string, handler: EventHandler): () => void {
    const wrapper: EventHandler = (...args: unknown[]) => {
      this.off(event, wrapper);
      handler.apply(this, args);
    };
    return this.on(event, wrapper);
  }

  off(event: string, handler: EventHandler): void {
    const handlers = this._listeners.get(event);
    if (handlers) {
      handlers.delete(handler);
      if (handlers.size === 0) {
        this._listeners.delete(event);
      }
    }
  }

  emit(event: string, ...args: unknown[]): void {
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

  removeAllListeners(event?: string): void {
    if (event !== undefined) {
      this._listeners.delete(event);
    } else {
      this._listeners.clear();
    }
  }

  listenerCount(event: string): number {
    const handlers = this._listeners.get(event);
    return handlers ? handlers.size : 0;
  }
}

export default EventEmitter;
