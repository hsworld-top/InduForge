/**
 * 协议适配器基类
 * 为 MQTT、WebSocket、HTTP、OPC UA 等协议预留接口
 */
class BaseProtocol {
  constructor(config) {
    this.config = config;
  }

  /**
   * 测试连接
   * @returns {Promise<boolean>} 连接是否成功
   */
  async testConnection() {
    throw new Error('testConnection method must be implemented');
  }

  /**
   * 建立连接
   * @returns {Promise<Object>} 连接对象
   */
  async connect() {
    throw new Error('connect method must be implemented');
  }

  /**
   * 断开连接
   * @param {Object} connection - 连接对象
   */
  async disconnect(connection) {
    throw new Error('disconnect method must be implemented');
  }

  /**
   * 发送数据
   * @param {Object} connection - 连接对象
   * @param {*} data - 要发送的数据
   */
  async send(connection, data) {
    throw new Error('send method must be implemented');
  }

  /**
   * 接收数据
   * @param {Object} connection - 连接对象
   * @param {Function} callback - 数据接收回调
   */
  async receive(connection, callback) {
    throw new Error('receive method must be implemented');
  }
}

module.exports = BaseProtocol;

