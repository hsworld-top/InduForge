/**
 * MessageHandler - DataCenter消息处理器
 * 
 * 处理来自Designer的消息请求
 */
import dataAPI from '@/api/data.api'

class MessageHandler {
  constructor() {
    this.subscriptions = new Map()
    this.pollingTimers = new Map()
    
    // 绑定消息处理器
    this.handleMessage = this.handleMessage.bind(this)
    window.addEventListener('message', this.handleMessage)
  }
  
  /**
   * 处理接收到的消息
   * @param {MessageEvent} event - 消息事件
   */
  async handleMessage(event) {
    const data = event.data
    if (!data || typeof data !== 'object') {
      return
    }
    
    // 处理握手
    if (data.type === 'HANDSHAKE' && data.source === 'designer') {
      this.sendMessage(event.source, {
        type: 'HANDSHAKE_ACK',
        source: 'datacenter'
      })
      console.log('DataCenter: Connected to Designer')
      return
    }
    
    // 处理请求
    if (data.type === 'REQUEST' && data.requestId) {
      try {
        const result = await this.handleRequest(data.action, data.payload)
        this.sendMessage(event.source, {
          type: 'RESPONSE',
          requestId: data.requestId,
          payload: result
        })
      } catch (error) {
        this.sendMessage(event.source, {
          type: 'RESPONSE',
          requestId: data.requestId,
          error: error.message
        })
      }
    }
  }
  
  /**
   * 处理具体请求
   * @param {string} action - 动作类型
   * @param {Object} payload - 请求数据
   * @returns {Promise<any>}
   */
  async handleRequest(action, payload) {
    switch (action) {
      case 'GET_CONNECTIONS':
        return await this.getConnections(payload)
      
      case 'GET_QUERIES':
        return await this.getQueries(payload)
      
      case 'EXECUTE_QUERY':
        return await this.executeQuery(payload)
      
      case 'EXECUTE_SQL':
        return await this.executeSql(payload)
      
      case 'SUBSCRIBE':
        return await this.subscribe(payload)
      
      case 'UNSUBSCRIBE':
        return await this.unsubscribe(payload)
      
      case 'GET_CONNECTION_TABLES':
        return await this.getConnectionTables(payload)
      
      case 'GET_TABLE_DATA':
        return await this.getTableData(payload)
      
      default:
        throw new Error(`Unknown action: ${action}`)
    }
  }
  
  /**
   * 获取数据连接列表
   * @param {Object} payload - { projectId }
   * @returns {Promise<Object>}
   */
  async getConnections({ projectId }) {
    const response = await dataAPI.getConnections(projectId)
    return {
      connections: response.data?.connections || response.data || []
    }
  }
  
  /**
   * 获取数据查询列表
   * @param {Object} payload - { projectId, connectionId }
   * @returns {Promise<Object>}
   */
  async getQueries({ projectId, connectionId }) {
    const params = connectionId ? { connectionId } : {}
    const response = await dataAPI.getQueries(projectId, params)
    return {
      queries: response.data?.queries || response.data || []
    }
  }
  
  /**
   * 执行数据查询
   * @param {Object} payload - { projectId, queryId, parameters }
   * @returns {Promise<Object>}
   */
  async executeQuery({ projectId, queryId, parameters }) {
    const response = await dataAPI.executeQuery(queryId, parameters)
    return response.data || response
  }
  
  /**
   * 执行SQL查询
   * @param {Object} payload - { projectId, connectionId, sql, parameters }
   * @returns {Promise<Object>}
   */
  async executeSql({ projectId, connectionId, sql, parameters }) {
    const response = await dataAPI.executeSql(projectId, connectionId, sql, parameters)
    return response.data || response
  }
  
  /**
   * 订阅数据查询（轮询模式）
   * @param {Object} payload - { subscriptionId, projectId, queryId, parameters, interval }
   * @returns {Promise<Object>}
   */
  async subscribe({ subscriptionId, projectId, queryId, parameters, interval }) {
    // 立即执行一次
    const initialData = await this.executeQuery({ projectId, queryId, parameters })
    
    // 发送初始数据
    this.sendMessage(window.parent, {
      type: 'SUBSCRIPTION_DATA',
      subscriptionId,
      payload: initialData
    })
    
    // 设置轮询定时器
    const timer = setInterval(async () => {
      try {
        const data = await this.executeQuery({ projectId, queryId, parameters })
        this.sendMessage(window.parent, {
          type: 'SUBSCRIPTION_DATA',
          subscriptionId,
          payload: data
        })
      } catch (error) {
        this.sendMessage(window.parent, {
          type: 'SUBSCRIPTION_ERROR',
          subscriptionId,
          error: error.message
        })
      }
    }, interval)
    
    this.pollingTimers.set(subscriptionId, timer)
    this.subscriptions.set(subscriptionId, { projectId, queryId, parameters, interval })
    
    return { success: true }
  }
  
  /**
   * 取消订阅
   * @param {Object} payload - { subscriptionId }
   * @returns {Promise<Object>}
   */
  async unsubscribe({ subscriptionId }) {
    const timer = this.pollingTimers.get(subscriptionId)
    if (timer) {
      clearInterval(timer)
      this.pollingTimers.delete(subscriptionId)
    }
    
    this.subscriptions.delete(subscriptionId)
    
    return { success: true }
  }
  
  /**
   * 获取连接的表列表
   * @param {Object} payload - { projectId, connectionId }
   * @returns {Promise<Object>}
   */
  async getConnectionTables({ projectId, connectionId }) {
    const response = await dataAPI.getConnectionTables(projectId, connectionId)
    return {
      tables: response.data?.tables || response.data || []
    }
  }
  
  /**
   * 获取表数据
   * @param {Object} payload - { projectId, connectionId, tableName, params }
   * @returns {Promise<Object>}
   */
  async getTableData({ projectId, connectionId, tableName, params }) {
    const response = await dataAPI.getTableData(projectId, connectionId, tableName, params)
    return response.data || response
  }
  
  /**
   * 发送消息
   * @param {Window} target - 目标窗口
   * @param {Object} message - 消息对象
   */
  sendMessage(target, message) {
    if (target && target.postMessage) {
      target.postMessage(message, '*')
    }
  }
  
  /**
   * 销毁处理器
   */
  destroy() {
    window.removeEventListener('message', this.handleMessage)
    
    // 清除所有定时器
    this.pollingTimers.forEach(timer => clearInterval(timer))
    this.pollingTimers.clear()
    this.subscriptions.clear()
  }
}

// 创建单例
let instance = null

export function initMessageHandler() {
  if (!instance) {
    instance = new MessageHandler()
  }
  return instance
}

export function getMessageHandler() {
  return instance
}

export default MessageHandler
