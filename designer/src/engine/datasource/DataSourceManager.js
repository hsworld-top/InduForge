/**
 * DataSourceManager - 数据源管理器
 *
 * 管理所有数据源的生命周期、数据获取和订阅
 * 支持两种模式：
 * 1. API模式（推荐）：直接调用后端API
 * 2. Bridge模式（可选）：通过iframe postMessage与DataCenter通信
 */
import { reactive, watch } from 'vue';
import { getDataCenterBridge } from './DataCenterBridge';
import expressionEngine from '../binding/ExpressionEngine';
import request from '@/utils/request';

export class DataSourceManager {
    constructor(store, options = {}) {
        this.store = store;
        this.mode = options.mode || 'api'; // 'api' | 'bridge'
        this.bridge = this.mode === 'bridge' ? getDataCenterBridge() : null;
        this.dataSources = new Map();
        this.subscriptions = new Map();
        this.pollingTimers = new Map();
        this.dataCache = reactive({});
    }

    /**
     * 初始化数据源管理器
     * @param {string} iframeId - DataCenter iframe ID（仅bridge模式需要）
     */
    async init(iframeId = 'datacenter-iframe') {
        if (this.mode === 'bridge') {
            const success = this.bridge.init(iframeId);
            if (success) {
                await this.bridge.waitForReady();
                console.log('DataSourceManager: Initialized in bridge mode');
            }
            return success;
        } else {
            console.log('DataSourceManager: Initialized in API mode');
            return true;
        }
    }

    /**
     * 注册数据源
     * @param {Object} dataSourceConfig - 数据源配置
     */
    register(dataSourceConfig) {
        const { id, type, config, mode = 'request', interval = 5000, transformer, errorHandler } = dataSourceConfig;

        if (this.dataSources.has(id)) {
            console.warn(`DataSourceManager: DataSource ${id} already registered`);
            return;
        }

        const dataSource = {
            id,
            type,
            config,
            mode,
            interval,
            transformer: transformer ? new Function('data', `return (${transformer})(data)`) : null,
            errorHandler: errorHandler ? new Function('error', `return (${errorHandler})(error)`) : null,
            status: 'idle', // idle, loading, ready, error
            error: null,
            lastUpdate: null,
        };

        this.dataSources.set(id, dataSource);

        // 初始化数据缓存
        this.dataCache[id] = null;

        // 如果是自动启动，立即加载
        if (dataSourceConfig.options?.autoStart !== false) {
            this.start(id);
        }
    }

    /**
     * 启动数据源
     * @param {string} dataSourceId - 数据源ID
     */
    async start(dataSourceId) {
        const dataSource = this.dataSources.get(dataSourceId);
        if (!dataSource) {
            console.warn(`DataSourceManager: DataSource ${dataSourceId} not found`);
            return;
        }

        if (dataSource.type === 'dataCenter') {
            await this.startDataCenterSource(dataSource);
        } else if (dataSource.type === 'http') {
            await this.startHttpSource(dataSource);
        } else if (dataSource.type === 'static') {
            this.startStaticSource(dataSource);
        } else if (dataSource.type === 'computed') {
            this.startComputedSource(dataSource);
        }
    }

    /**
     * 启动DataCenter数据源
     * @param {Object} dataSource - 数据源对象
     */
    async startDataCenterSource(dataSource) {
        const { id, config, mode, interval } = dataSource;
        const projectId = this.store.projectId;

        if (!projectId) {
            console.error('DataSourceManager: projectId not set');
            return;
        }

        try {
            dataSource.status = 'loading';

            if (config.sourceType === 'query') {
                // 查询模式
                if (mode === 'subscription' || mode === 'poll') {
                    // 订阅或轮询模式
                    await this.subscribeQuery(dataSource, projectId);
                } else {
                    // 单次请求模式
                    await this.fetchQuery(dataSource, projectId);
                }
            } else if (config.sourceType === 'tags') {
                // 点位订阅模式（工业场景）
                await this.subscribeTags(dataSource, projectId);
            }

            dataSource.status = 'ready';
        } catch (error) {
            dataSource.status = 'error';
            dataSource.error = error.message;
            this.handleError(dataSource, error);
        }
    }

    /**
     * 订阅查询
     * @param {Object} dataSource - 数据源对象
     * @param {string} projectId - 项目ID
     */
    async subscribeQuery(dataSource, projectId) {
        const { id, config, interval } = dataSource;
        const { queryId, parameters = {} } = config;

        // 解析参数表达式
        const resolvedParams = this.resolveParameters(parameters);

        if (dataSource.mode === 'subscription') {
            if (this.mode === 'bridge') {
                // Bridge模式：WebSocket订阅
                const subscriptionId = await this.bridge.subscribe(
                    projectId,
                    queryId,
                    resolvedParams,
                    (data) => this.updateData(dataSource, data),
                    (error) => this.handleError(dataSource, error),
                    interval,
                );
                this.subscriptions.set(id, subscriptionId);
            } else {
                // API模式：轮询实现订阅
                await this.pollQuery(dataSource, projectId, queryId, resolvedParams, interval);
            }
        } else {
            // 轮询模式
            await this.pollQuery(dataSource, projectId, queryId, resolvedParams, interval);
        }
    }

    /**
     * 轮询查询
     * @param {Object} dataSource - 数据源对象
     * @param {string} projectId - 项目ID
     * @param {string} queryId - 查询ID
     * @param {Object} parameters - 参数
     * @param {number} interval - 间隔
     */
    async pollQuery(dataSource, projectId, queryId, parameters, interval) {
        const poll = async () => {
            try {
                const data = await this.executeQueryAPI(projectId, queryId, parameters);
                this.updateData(dataSource, data);
            } catch (error) {
                this.handleError(dataSource, error);
            }
        };

        // 立即执行一次
        await poll();

        // 设置定时器
        const timer = setInterval(poll, interval);
        this.pollingTimers.set(dataSource.id, timer);
    }

    /**
     * 获取查询数据（单次）
     * @param {Object} dataSource - 数据源对象
     * @param {string} projectId - 项目ID
     */
    async fetchQuery(dataSource, projectId) {
        const { config } = dataSource;
        const { queryId, parameters = {} } = config;

        const resolvedParams = this.resolveParameters(parameters);
        const data = await this.executeQueryAPI(projectId, queryId, resolvedParams);
        this.updateData(dataSource, data);
    }

    /**
     * 订阅点位数据（工业场景）
     * @param {Object} dataSource - 数据源对象
     * @param {string} projectId - 项目ID
     */
    async subscribeTags(dataSource, projectId) {
        const { id, config, interval } = dataSource;
        const { connectionId, tags } = config;

        // 构建SQL查询订阅点位表
        const tagsStr = tags.map((t) => `'${t}'`).join(',');
        const sql = `SELECT tag_name, tag_value, update_time FROM tags WHERE tag_name IN (${tagsStr})`;

        // 轮询模式获取点位数据
        const poll = async () => {
            try {
                const result = await this.executeSqlAPI(projectId, connectionId, sql, []);

                // 转换为键值对格式
                const tagData = {};
                if (result.rows) {
                    result.rows.forEach((row) => {
                        const tagName = row[0];
                        const tagValue = row[1];
                        tagData[tagName] = tagValue;
                    });
                }

                this.updateData(dataSource, tagData);
            } catch (error) {
                this.handleError(dataSource, error);
            }
        };

        // 立即执行一次
        await poll();

        // 设置定时器
        const timer = setInterval(poll, interval);
        this.pollingTimers.set(id, timer);
    }

    /**
     * 执行查询（API模式）
     * @param {string} projectId - 项目ID
     * @param {string} queryId - 查询ID
     * @param {Object} parameters - 参数
     * @returns {Promise<any>}
     */
    async executeQueryAPI(projectId, queryId, parameters = {}) {
        if (this.mode === 'bridge') {
            return await this.bridge.executeQuery(projectId, queryId, parameters);
        } else {
            const response = await request({
                url: `/data/queries/${queryId}/execute`,
                method: 'post',
                data: { parameters },
            });
            return response.data || response;
        }
    }

    /**
     * 执行SQL（API模式）
     * @param {string} projectId - 项目ID
     * @param {string} connectionId - 连接ID
     * @param {string} sql - SQL语句
     * @param {Array} parameters - 参数数组
     * @returns {Promise<any>}
     */
    async executeSqlAPI(projectId, connectionId, sql, parameters = []) {
        if (this.mode === 'bridge') {
            return await this.bridge.executeSql(projectId, connectionId, sql, parameters);
        } else {
            const response = await request({
                url: `/data/projects/${projectId}/connections/${connectionId}/execute-sql`,
                method: 'post',
                data: { sql, parameters },
            });
            return response.data || response;
        }
    }

    /**
     * 获取连接列表（API模式）
     * @param {string} projectId - 项目ID
     * @returns {Promise<Array>}
     */
    async getConnectionsAPI(projectId) {
        if (this.mode === 'bridge') {
            return await this.bridge.getConnections(projectId);
        } else {
            const response = await request({
                url: `/data/projects/${projectId}/connections`,
                method: 'get',
            });
            return response.data?.connections || response.data || [];
        }
    }

    /**
     * 获取查询列表（API模式）
     * @param {string} projectId - 项目ID
     * @returns {Promise<Array>}
     */
    async getQueriesAPI(projectId) {
        if (this.mode === 'bridge') {
            return await this.bridge.getQueries(projectId);
        } else {
            const response = await request({
                url: `/data/projects/${projectId}/queries`,
                method: 'get',
            });
            return response.data?.queries || response.data || [];
        }
    }

    /**
     * 启动HTTP数据源
     * @param {Object} dataSource - 数据源对象
     */
    async startHttpSource(dataSource) {
        const { id, config, mode, interval } = dataSource;

        const fetchData = async () => {
            try {
                dataSource.status = 'loading';

                const { url, method = 'GET', headers = {}, params = {}, data: body, timeout = 10000 } = config;

                // 解析URL和参数中的表达式
                const resolvedUrl = this.resolveExpression(url);
                const resolvedParams = this.resolveParameters(params);
                const resolvedHeaders = this.resolveParameters(headers);
                const resolvedBody = body ? this.resolveParameters(body) : undefined;

                // 构建完整URL
                const urlObj = new URL(resolvedUrl);
                Object.entries(resolvedParams).forEach(([key, value]) => {
                    urlObj.searchParams.append(key, value);
                });

                // 发送请求
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), timeout);

                const response = await fetch(urlObj.toString(), {
                    method,
                    headers: resolvedHeaders,
                    body: resolvedBody ? JSON.stringify(resolvedBody) : undefined,
                    signal: controller.signal,
                });

                clearTimeout(timeoutId);

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }

                const data = await response.json();
                this.updateData(dataSource, data);
                dataSource.status = 'ready';
            } catch (error) {
                dataSource.status = 'error';
                dataSource.error = error.message;
                this.handleError(dataSource, error);
            }
        };

        // 立即执行一次
        await fetchData();

        // 如果是轮询模式，设置定时器
        if (mode === 'poll') {
            const timer = setInterval(fetchData, interval);
            this.pollingTimers.set(id, timer);
        }
    }

    /**
     * 启动静态数据源
     * @param {Object} dataSource - 数据源对象
     */
    startStaticSource(dataSource) {
        const { config } = dataSource;
        this.updateData(dataSource, config.data);
        dataSource.status = 'ready';
    }

    /**
     * 启动计算数据源
     * @param {Object} dataSource - 数据源对象
     */
    startComputedSource(dataSource) {
        const { id, config } = dataSource;
        const { dependencies = [], compute } = config;

        if (!compute) {
            console.warn(`DataSourceManager: Computed dataSource ${id} has no compute function`);
            return;
        }

        // 创建计算函数
        const computeFn = new Function('sources', `return (${compute})(sources)`);

        // 监听依赖的数据源变化
        const unwatch = watch(
            () => {
                const sources = {};
                dependencies.forEach((depId) => {
                    sources[depId] = this.dataCache[depId];
                });
                return sources;
            },
            (sources) => {
                try {
                    const result = computeFn(sources);
                    this.updateData(dataSource, result);
                    dataSource.status = 'ready';
                } catch (error) {
                    dataSource.status = 'error';
                    dataSource.error = error.message;
                    this.handleError(dataSource, error);
                }
            },
            { immediate: true, deep: true },
        );

        // 保存取消监听函数
        dataSource.unwatch = unwatch;
    }

    /**
     * 更新数据
     * @param {Object} dataSource - 数据源对象
     * @param {any} rawData - 原始数据
     */
    updateData(dataSource, rawData) {
        const { id, transformer } = dataSource;

        try {
            // 应用转换器
            const data = transformer ? transformer(rawData) : rawData;

            // 更新缓存
            this.dataCache[id] = data;
            dataSource.lastUpdate = Date.now();

            // 更新store
            if (this.store.dataSources) {
                this.store.dataSources[id] = data;
            }
        } catch (error) {
            console.error(`DataSourceManager: Transform error for ${id}:`, error);
            this.handleError(dataSource, error);
        }
    }

    /**
     * 处理错误
     * @param {Object} dataSource - 数据源对象
     * @param {Error} error - 错误对象
     */
    handleError(dataSource, error) {
        const { id, errorHandler } = dataSource;

        console.error(`DataSourceManager: Error in dataSource ${id}:`, error);

        if (errorHandler) {
            try {
                const fallbackData = errorHandler(error);
                this.dataCache[id] = fallbackData;
                if (this.store.dataSources) {
                    this.store.dataSources[id] = fallbackData;
                }
            } catch (handlerError) {
                console.error(`DataSourceManager: Error handler failed for ${id}:`, handlerError);
            }
        }
    }

    /**
     * 解析参数对象中的表达式
     * @param {Object} params - 参数对象
     * @returns {Object} 解析后的参数
     */
    resolveParameters(params) {
        const resolved = {};
        Object.entries(params).forEach(([key, value]) => {
            resolved[key] = this.resolveExpression(value);
        });
        return resolved;
    }

    /**
     * 解析单个表达式
     * @param {any} value - 值（可能包含表达式）
     * @returns {any} 解析后的值
     */
    resolveExpression(value) {
        if (typeof value !== 'string') {
            return value;
        }

        // 检查是否是表达式
        if (value.includes('{{') && value.includes('}}')) {
            const context = this.buildContext();
            expressionEngine.setContext(context);
            return expressionEngine.evaluate(value);
        }

        return value;
    }

    /**
     * 构建表达式上下文
     * @returns {Object} 上下文对象
     */
    buildContext() {
        const currentPage = this.store.currentPage;

        return {
            vars: currentPage?.variables || {},
            data: this.dataCache,
            $user: this.store.user || {},
            $route: {},
            $env: {
                API_BASE: import.meta.env.VITE_API_BASE || '',
                MODE: import.meta.env.MODE,
            },
            $global: {},
        };
    }

    /**
     * 刷新数据源
     * @param {string} dataSourceId - 数据源ID
     */
    async refresh(dataSourceId) {
        const dataSource = this.dataSources.get(dataSourceId);
        if (!dataSource) {
            console.warn(`DataSourceManager: DataSource ${dataSourceId} not found`);
            return;
        }

        // 停止当前的订阅/轮询
        this.stop(dataSourceId);

        // 重新启动
        await this.start(dataSourceId);
    }

    /**
     * 停止数据源
     * @param {string} dataSourceId - 数据源ID
     */
    stop(dataSourceId) {
        const dataSource = this.dataSources.get(dataSourceId);
        if (!dataSource) {
            return;
        }

        // 取消订阅
        const subscriptionId = this.subscriptions.get(dataSourceId);
        if (subscriptionId) {
            this.bridge.unsubscribe(subscriptionId).catch(() => {});
            this.subscriptions.delete(dataSourceId);
        }

        // 清除轮询定时器
        const timer = this.pollingTimers.get(dataSourceId);
        if (timer) {
            clearInterval(timer);
            this.pollingTimers.delete(dataSourceId);
        }

        // 取消computed监听
        if (dataSource.unwatch) {
            dataSource.unwatch();
            delete dataSource.unwatch;
        }

        dataSource.status = 'idle';
    }

    /**
     * 注销数据源
     * @param {string} dataSourceId - 数据源ID
     */
    unregister(dataSourceId) {
        this.stop(dataSourceId);
        this.dataSources.delete(dataSourceId);
        delete this.dataCache[dataSourceId];
        if (this.store.dataSources) {
            delete this.store.dataSources[dataSourceId];
        }
    }

    /**
     * 获取数据源数据
     * @param {string} dataSourceId - 数据源ID
     * @returns {any} 数据
     */
    getData(dataSourceId) {
        return this.dataCache[dataSourceId];
    }

    /**
     * 获取数据源状态
     * @param {string} dataSourceId - 数据源ID
     * @returns {Object} 状态对象
     */
    getStatus(dataSourceId) {
        const dataSource = this.dataSources.get(dataSourceId);
        if (!dataSource) {
            return null;
        }

        return {
            status: dataSource.status,
            error: dataSource.error,
            lastUpdate: dataSource.lastUpdate,
        };
    }

    /**
     * 清理所有数据源
     */
    clear() {
        this.dataSources.forEach((_, id) => {
            this.stop(id);
        });

        this.dataSources.clear();
        this.subscriptions.clear();
        this.pollingTimers.clear();
        Object.keys(this.dataCache).forEach((key) => {
            delete this.dataCache[key];
        });
    }

    /**
     * 销毁管理器
     */
    destroy() {
        this.clear();
        // 不销毁bridge，因为它是单例
    }
}

export default DataSourceManager;
