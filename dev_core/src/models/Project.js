const { DataTypes } = require("sequelize");
const { sequelize } = require("../config/database");

const Project = sequelize.define(
  "Project",
  {
    id: {
      type: DataTypes.UUID,
      defaultValue: DataTypes.UUIDV4,
      primaryKey: true,
    },
    name: {
      type: DataTypes.STRING(200),
      allowNull: false,
      comment: "工程名称",
    },
    description: {
      type: DataTypes.TEXT,
      allowNull: true,
      comment: "工程描述",
    },
    projectVariables: {
      type: DataTypes.JSON,
      allowNull: true,
      comment: "工程级别全局变量",
    },
    entryConfig: {
      type: DataTypes.JSON,
      allowNull: true,
      defaultValue: {},
      comment: "入口配置：homePageId, loginPageId, logoutPageId 等",
    },
    colorTag: {
      type: DataTypes.ENUM(
        "#3b82f6",
        "#ef4444",
        "#10b981",
        "#f59e0b",
        "#8b5cf6",
        "#ec4899",
        "#6b7280"
      ),
      defaultValue: "#3b82f6",
      comment: "颜色标签",
    },
    tenantId: {
      type: DataTypes.UUID,
      allowNull: false,
      references: {
        model: "tenants",
        key: "id",
      },
      comment: "所属租户ID",
    },
    createdBy: {
      type: DataTypes.UUID,
      allowNull: false,
      references: {
        model: "users",
        key: "id",
      },
      comment: "创建者ID",
    },
    updatedBy: {
      type: DataTypes.UUID,
      allowNull: true,
      references: {
        model: "users",
        key: "id",
      },
      comment: "更新者ID",
    },
    createdAt: {
      type: DataTypes.DATE,
      allowNull: false,
    },
    updatedAt: {
      type: DataTypes.DATE,
      allowNull: false,
    },
  },
  {
    tableName: "projects",
    comment: "工程表",
    indexes: [
      {
        fields: ["tenantId"],
      },
      {
        fields: ["createdBy"],
      },
    ],
  }
);

module.exports = Project;
