/**
 * DataCenterBridge - DataCenter通信桥接
 *
 * 支持两种通信模式：
 * 1. iframe模式：通过postMessage与iframe中的DataCenter通信
 * 2. 标签页模式：通过BroadcastChannel与独立标签页的DataCenter通信
 */

export class DataCenterBridge {
    constructor() {
        this.mode = null; // 'iframe' | 'tab'
        this.iframe = null;
        this.channel = null;
        this.messageHandlers = new Map();
        this.requestId = 0;
        this.pendingRequests = new Map();
        this.subscriptions = new Map();
        this.isReady = false;
        this.readyCallbacks = [];

        // 绑定消息处理器
        this.handleMessage = this.handleMessage.bind(this);
        this.handleChannelMessage = this.handleChannelMessage.bind(this);
    }

    /**
     * 初始化通信
     * @param {string} iframeId - iframe元素ID（可选，用于iframe模式）
     */
    init(iframeId = 'datacenter-iframe') {
        // 尝试iframe模式
        const iframe = document.getElementById(iframeId);
        if (iframe) {
            this.mode = 'iframe';
            this.iframe = iframe;
            window.addEventListener('message', this.handleMessage);

            // 等待iframe加载完成
            if (this.iframe.contentWindow) {
                this.sendHandshake();
            } else {
                this.iframe.addEventListener('load', () => {
                    this.sendHandshake();
                });
            }

            console.log('DataCenterBridge: Using iframe mode');
            return true;
        }

        // 使用标签页模式（BroadcastChannel）
        if (typeof BroadcastChannel !== 'undefined') {
            this.mode = 'tab';
            this.channel = new BroadcastChannel('induforge_datacenter');
            this.channel.addEventListener('message', this.handleChannelMessage);

            // 发送握手
            this.sendHandshake();

            // 设置超时，如果5秒内没有收到响应，标记为未就绪
            setTimeout(() => {
                if (!this.isReady) {
                    console.warn('DataCenterBridge: DataCenter not responding, but will continue in tab mode');
                    // 即使没有响应也标记为就绪，允许继续使用
                    this.isReady = true;
                    this.readyCallbacks.forEach((cb) => cb());
                    this.readyCallbacks = [];
                }
            }, 5000);

            console.log('DataCenterBridge: Using tab mode (BroadcastChannel)');
            return true;
        }

        console.error('DataCenterBridge: No communication method available');
        return false;
    }

    /**
     * 发送握手消息
     */
    sendHandshake() {
        this.postMessage({
            type: 'HANDSHAKE',
            source: 'designer',
            timestamp: Date.now(),
        });
    }

    /**
     * 等待DataCenter就绪
     * @returns {Promise<void>}
     */
    waitForReady() {
        if (this.isReady) {
            return Promise.resolve();
        }

        return new Promise((resolve) => {
            this.readyCallbacks.push(resolve);
        });
    }

    /**
     * 处理接收到的消息
     * @param {MessageEvent} event - 消息事件
     */
    handleMessage(event) {
        // 验证消息来源（生产环境应该检查origin）
        const data = event.data;
        if (!data || typeof data !== 'object') {
            return;
        }

        // 处理握手响应
        if (data.type === 'HANDSHAKE_ACK' && data.source === 'datacenter') {
            this.isReady = true;
            this.readyCallbacks.forEach((cb) => cb());
            this.readyCallbacks = [];
            console.log('DataCenterBridge: Connected to DataCenter');
            return;
        }

        // 处理请求响应
        if (data.type === 'RESPONSE' && data.requestId) {
            const resolver = this.pendingRequests.get(data.requestId);
            if (resolver) {
                this.pendingRequests.delete(data.requestId);
                if (data.error) {
                    resolver.reject(new Error(data.error));
                } else {
                    resolver.resolve(data.payload);
                }
            }
            return;
        }

        // 处理订阅数据推送
        if (data.type === 'SUBSCRIPTION_DATA' && data.subscriptionId) {
            const handler = this.subscriptions.get(data.subscriptionId);
            if (handler) {
                handler(data.payload);
            }
            return;
        }

        // 处理订阅错误
        if (data.type === 'SUBSCRIPTION_ERROR' && data.subscriptionId) {
            const handler = this.subscriptions.get(data.subscriptionId);
            if (handler && handler.onError) {
                handler.onError(new Error(data.error));
            }
            return;
        }
    }

    /**
     * 发送消息到DataCenter
     * @param {Object} message - 消息对象
     */
    postMessage(message) {
        if (!this.iframe || !this.iframe.contentWindow) {
            console.warn('DataCenterBridge: iframe not available');
            return;
        }

        this.iframe.contentWindow.postMessage(message, '*');
    }

    /**
     * 发送请求并等待响应
     * @param {string} action - 动作类型
     * @param {Object} payload - 请求数据
     * @returns {Promise<any>}
     */
    async request(action, payload = {}) {
        await this.waitForReady();

        const requestId = ++this.requestId;

        return new Promise((resolve, reject) => {
            // 设置超时
            const timeout = setTimeout(() => {
                this.pendingRequests.delete(requestId);
                reject(new Error('Request timeout'));
            }, 30000);

            this.pendingRequests.set(requestId, {
                resolve: (data) => {
                    clearTimeout(timeout);
                    resolve(data);
                },
                reject: (error) => {
                    clearTimeout(timeout);
                    reject(error);
                },
            });

            this.postMessage({
                type: 'REQUEST',
                requestId,
                action,
                payload,
            });
        });
    }

    /**
     * 获取数据连接列表
     * @param {string} projectId - 项目ID
     * @returns {Promise<Array>}
     */
    async getConnections(projectId) {
        const result = await this.request('GET_CONNECTIONS', { projectId });
        return result.connections || [];
    }

    /**
     * 获取数据查询列表
     * @param {string} projectId - 项目ID
     * @param {string} connectionId - 连接ID（可选）
     * @returns {Promise<Array>}
     */
    async getQueries(projectId, connectionId = null) {
        const result = await this.request('GET_QUERIES', { projectId, connectionId });
        return result.queries || [];
    }

    /**
     * 执行数据查询
     * @param {string} projectId - 项目ID
     * @param {string} queryId - 查询ID
     * @param {Object} parameters - 查询参数
     * @returns {Promise<Object>}
     */
    async executeQuery(projectId, queryId, parameters = {}) {
        return await this.request('EXECUTE_QUERY', { projectId, queryId, parameters });
    }

    /**
     * 执行SQL查询
     * @param {string} projectId - 项目ID
     * @param {string} connectionId - 连接ID
     * @param {string} sql - SQL语句
     * @param {Array} parameters - 参数数组
     * @returns {Promise<Object>}
     */
    async executeSql(projectId, connectionId, sql, parameters = []) {
        return await this.request('EXECUTE_SQL', { projectId, connectionId, sql, parameters });
    }

    /**
     * 订阅数据查询（实时更新）
     * @param {string} projectId - 项目ID
     * @param {string} queryId - 查询ID
     * @param {Object} parameters - 查询参数
     * @param {Function} onData - 数据回调
     * @param {Function} onError - 错误回调
     * @param {number} interval - 轮询间隔（毫秒）
     * @returns {Promise<string>} 订阅ID
     */
    async subscribe(projectId, queryId, parameters = {}, onData, onError = null, interval = 5000) {
        const subscriptionId = `sub_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

        // 注册回调
        this.subscriptions.set(subscriptionId, onData);
        if (onError) {
            this.subscriptions.get(subscriptionId).onError = onError;
        }

        // 发送订阅请求
        await this.request('SUBSCRIBE', {
            subscriptionId,
            projectId,
            queryId,
            parameters,
            interval,
        });

        return subscriptionId;
    }

    /**
     * 取消订阅
     * @param {string} subscriptionId - 订阅ID
     */
    async unsubscribe(subscriptionId) {
        this.subscriptions.delete(subscriptionId);
        await this.request('UNSUBSCRIBE', { subscriptionId });
    }

    /**
     * 获取连接的表列表
     * @param {string} projectId - 项目ID
     * @param {string} connectionId - 连接ID
     * @returns {Promise<Array>}
     */
    async getConnectionTables(projectId, connectionId) {
        const result = await this.request('GET_CONNECTION_TABLES', { projectId, connectionId });
        return result.tables || [];
    }

    /**
     * 获取表数据
     * @param {string} projectId - 项目ID
     * @param {string} connectionId - 连接ID
     * @param {string} tableName - 表名
     * @param {Object} params - 查询参数（page, limit等）
     * @returns {Promise<Object>}
     */
    async getTableData(projectId, connectionId, tableName, params = {}) {
        return await this.request('GET_TABLE_DATA', { projectId, connectionId, tableName, params });
    }

    /**
     * 销毁桥接
     */
    destroy() {
        window.removeEventListener('message', this.handleMessage);

        // 取消所有订阅
        this.subscriptions.forEach((_, subscriptionId) => {
            this.unsubscribe(subscriptionId).catch(() => {});
        });

        this.subscriptions.clear();
        this.pendingRequests.clear();
        this.messageHandlers.clear();
        this.iframe = null;
    }
}

// 单例模式
let instance = null;

export function getDataCenterBridge() {
    if (!instance) {
        instance = new DataCenterBridge();
    }
    return instance;
}

export default DataCenterBridge;
