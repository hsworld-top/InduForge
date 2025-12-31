"use strict";
const { Model } = require("sequelize");

module.exports = (sequelize, DataTypes) => {
  class DataMqttTagValue extends Model {
    static associate(models) {
      // 与变量关联
      DataMqttTagValue.belongsTo(models.DataMqttTag, {
        foreignKey: "tagId",
        as: "tag",
      });
    }
  }

  DataMqttTagValue.init(
    {
      id: {
        type: DataTypes.CHAR(36),
        primaryKey: true,
        defaultValue: DataTypes.UUIDV4,
      },
      tagId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        unique: true,
        comment: "变量ID",
      },
      rawValue: {
        type: DataTypes.TEXT,
        comment: "原始消息内容",
      },
      parsedValue: {
        type: DataTypes.TEXT,
        comment: "解析后的值",
      },
      quality: {
        type: DataTypes.ENUM("good", "bad", "uncertain"),
        allowNull: false,
        defaultValue: "good",
        comment: "数据质量",
      },
      timestamp: {
        type: DataTypes.DATE(6),
        allowNull: false,
        comment: "数据时间戳",
      },
      receivedAt: {
        type: DataTypes.DATE(6),
        allowNull: false,
        comment: "接收时间",
      },
      error: {
        type: DataTypes.TEXT,
        comment: "解析错误信息",
      },
    },
    {
      sequelize,
      modelName: "DataMqttTagValue",
      tableName: "data_mqtt_tag_values",
      timestamps: false, // 不需要createdAt/updatedAt，使用timestamp/receivedAt
    }
  );

  return DataMqttTagValue;
};
