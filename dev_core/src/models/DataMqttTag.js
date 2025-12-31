"use strict";
const { Model } = require("sequelize");

module.exports = (sequelize, DataTypes) => {
  class DataMqttTag extends Model {
    static associate(models) {
      // 与工程关联
      DataMqttTag.belongsTo(models.Project, {
        foreignKey: "projectId",
        as: "project",
      });

      // 与订阅关联
      DataMqttTag.belongsTo(models.DataMqttSubscription, {
        foreignKey: "subscriptionId",
        as: "subscription",
      });

      // 与分组关联
      DataMqttTag.belongsTo(models.DataMqttTagGroup, {
        foreignKey: "groupId",
        as: "group",
      });

      // 与用户关联（创建者）
      DataMqttTag.belongsTo(models.User, {
        foreignKey: "createdBy",
        as: "creator",
      });

      // 与用户关联（更新者）
      DataMqttTag.belongsTo(models.User, {
        foreignKey: "updatedBy",
        as: "updater",
      });

      // 与变量值关联
      DataMqttTag.hasOne(models.DataMqttTagValue, {
        foreignKey: "tagId",
        as: "currentValue",
      });
    }
  }

  DataMqttTag.init(
    {
      id: {
        type: DataTypes.CHAR(36),
        primaryKey: true,
        defaultValue: DataTypes.UUIDV4,
      },
      projectId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "所属工程ID",
      },
      subscriptionId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "订阅ID",
      },
      groupId: {
        type: DataTypes.CHAR(36),
        allowNull: true,
        comment: "所属分组ID",
      },
      name: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: "变量名称",
      },
      code: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: "变量标识符(用于引用)",
      },
      description: {
        type: DataTypes.TEXT,
        comment: "变量描述",
      },
      dataType: {
        type: DataTypes.ENUM("string", "number", "boolean", "object", "array"),
        allowNull: false,
        defaultValue: "string",
        comment: "数据类型",
      },
      parseType: {
        type: DataTypes.ENUM("jsonpath", "regex", "script", "fixed"),
        allowNull: false,
        defaultValue: "jsonpath",
        comment: "解析类型",
      },
      parseRule: {
        type: DataTypes.TEXT,
        allowNull: false,
        comment: "解析规则(JSONPath表达式/正则表达式/脚本代码)",
      },
      defaultValue: {
        type: DataTypes.TEXT,
        comment: "默认值",
      },
      unit: {
        type: DataTypes.STRING(50),
        comment: "单位",
      },
      transform: {
        type: DataTypes.TEXT,
        comment: "值转换函数(JavaScript代码)",
      },
      validation: {
        type: DataTypes.JSON,
        comment: "验证规则(min/max/pattern等)",
      },
      isEnabled: {
        type: DataTypes.BOOLEAN,
        allowNull: false,
        defaultValue: true,
        comment: "是否启用",
      },
      order: {
        type: DataTypes.INTEGER,
        allowNull: false,
        defaultValue: 0,
        comment: "显示顺序",
      },
      createdBy: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "创建者ID",
      },
      updatedBy: {
        type: DataTypes.CHAR(36),
        comment: "更新者ID",
      },
    },
    {
      sequelize,
      modelName: "DataMqttTag",
      tableName: "data_mqtt_tags",
      timestamps: true,
      paranoid: false,
      createdAt: "createdAt",
      updatedAt: "updatedAt",
    }
  );

  return DataMqttTag;
};
