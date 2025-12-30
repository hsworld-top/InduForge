"use strict";
const { Model } = require("sequelize");

module.exports = (sequelize, DataTypes) => {
  class DataMqttConfig extends Model {
    static associate(models) {
      // 与连接关联（一对一）
      DataMqttConfig.belongsTo(models.DataConnection, {
        foreignKey: "connectionId",
        as: "connection",
      });
    }
  }

  DataMqttConfig.init(
    {
      id: {
        type: DataTypes.CHAR(36),
        primaryKey: true,
        defaultValue: DataTypes.UUIDV4,
      },
      connectionId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        unique: true,
        comment: "连接ID",
      },
      brokerUrl: {
        type: DataTypes.STRING(500),
        allowNull: false,
        comment: "Broker地址",
      },
      protocol: {
        type: DataTypes.ENUM("mqtt", "mqtts", "ws", "wss"),
        allowNull: false,
        defaultValue: "mqtt",
        comment: "协议类型",
      },
      port: {
        type: DataTypes.INTEGER,
        defaultValue: 1883,
        comment: "端口号",
      },
      clientId: {
        type: DataTypes.STRING(100),
        comment: "客户端ID",
      },
      username: {
        type: DataTypes.STRING(100),
        comment: "用户名",
      },
      password: {
        type: DataTypes.STRING(255),
        comment: "密码",
      },
      keepalive: {
        type: DataTypes.INTEGER,
        defaultValue: 60,
        comment: "心跳间隔(秒)",
      },
      cleanSession: {
        type: DataTypes.BOOLEAN,
        defaultValue: true,
        comment: "清除会话",
      },
      qos: {
        type: DataTypes.TINYINT,
        defaultValue: 0,
        comment: "默认QoS(0/1/2)",
      },
      reconnectPeriod: {
        type: DataTypes.INTEGER,
        defaultValue: 5000,
        comment: "重连间隔(ms)",
      },
      connectTimeout: {
        type: DataTypes.INTEGER,
        defaultValue: 30000,
        comment: "连接超时(ms)",
      },
      will: {
        type: DataTypes.JSON,
        comment: "遗嘱消息配置",
      },
      sslConfig: {
        type: DataTypes.JSON,
        comment: "SSL/TLS配置",
      },
    },
    {
      sequelize,
      modelName: "DataMqttConfig",
      tableName: "data_mqtt_configs",
      timestamps: false,
    }
  );

  return DataMqttConfig;
};
