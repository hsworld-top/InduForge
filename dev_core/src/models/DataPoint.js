"use strict";
const { Model } = require("sequelize");

module.exports = (sequelize, DataTypes) => {
  class DataPoint extends Model {
    static associate(models) {
      DataPoint.belongsTo(models.Project, {
        foreignKey: "projectId",
        as: "project",
      });

      DataPoint.belongsTo(models.User, {
        foreignKey: "createdBy",
        as: "creator",
      });

      DataPoint.belongsTo(models.User, {
        foreignKey: "updatedBy",
        as: "updater",
      });
    }
  }

  DataPoint.init(
    {
      id: {
        type: DataTypes.CHAR(36),
        primaryKey: true,
        defaultValue: DataTypes.UUIDV4,
      },
      projectId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        field: "project_id",
        comment: "所属工程ID",
      },
      path: {
        type: DataTypes.STRING(255),
        allowNull: false,
        comment: "数据点路径",
      },
      name: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: "数据点名称",
      },
      description: {
        type: DataTypes.TEXT,
        comment: "描述",
      },
      sourceType: {
        type: DataTypes.STRING(50),
        allowNull: false,
        field: "source_type",
        comment: "来源类型",
      },
      sourceId: {
        type: DataTypes.CHAR(36),
        field: "source_id",
        comment: "来源ID",
      },
      sourceConfig: {
        type: DataTypes.JSON,
        field: "source_config",
        comment: "来源配置",
      },
      dataType: {
        type: DataTypes.STRING(20),
        allowNull: false,
        field: "data_type",
        comment: "数据类型",
      },
      unit: {
        type: DataTypes.STRING(20),
        comment: "单位",
      },
      precisionNum: {
        type: DataTypes.INTEGER,
        field: "precision_num",
        comment: "精度",
      },
      defaultValue: {
        type: DataTypes.TEXT,
        field: "default_value",
        comment: "默认值",
      },
      minValue: {
        type: DataTypes.DECIMAL(20, 6),
        field: "min_value",
        comment: "最小值",
      },
      maxValue: {
        type: DataTypes.DECIMAL(20, 6),
        field: "max_value",
        comment: "最大值",
      },
      alarmLow: {
        type: DataTypes.DECIMAL(20, 6),
        field: "alarm_low",
        comment: "低报警阈值",
      },
      alarmHigh: {
        type: DataTypes.DECIMAL(20, 6),
        field: "alarm_high",
        comment: "高报警阈值",
      },
      tags: {
        type: DataTypes.JSON,
        comment: "标签",
      },
      refreshMode: {
        type: DataTypes.STRING(20),
        field: "refresh_mode",
        defaultValue: "auto",
        comment: "刷新模式",
      },
      refreshInterval: {
        type: DataTypes.INTEGER,
        field: "refresh_interval",
        comment: "刷新间隔",
      },
      status: {
        type: DataTypes.STRING(20),
        defaultValue: "active",
        comment: "状态",
      },
      createdBy: {
        type: DataTypes.CHAR(36),
        field: "created_by",
        comment: "创建者ID",
      },
      updatedBy: {
        type: DataTypes.CHAR(36),
        field: "updated_by",
        comment: "更新者ID",
      },
    },
    {
      sequelize,
      modelName: "DataPoint",
      tableName: "data_points",
      timestamps: true,
      paranoid: false,
      createdAt: "created_at",
      updatedAt: "updated_at",
      indexes: [
        {
          unique: true,
          fields: ["projectId", "path"],
          name: "data_points_project_path_uq",
        },
        {
          fields: ["projectId"],
          name: "data_points_project_idx",
        },
        {
          fields: ["sourceType"],
          name: "data_points_source_type_idx",
        },
        {
          fields: ["status"],
          name: "data_points_status_idx",
        },
      ],
    }
  );

  return DataPoint;
};
