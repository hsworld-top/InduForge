const { DataTypes } = require('sequelize')
const { sequelize } = require('../config/database')

const Tenant = sequelize.define(
  'Tenant',
  {
    id: {
      type: DataTypes.UUID,
      defaultValue: DataTypes.UUIDV4,
      primaryKey: true,
    },
    name: {
      type: DataTypes.STRING(100),
      allowNull: false,
      comment: '租户名称',
    },
    code: {
      type: DataTypes.STRING(50),
      allowNull: false,
      unique: true,
      comment: '租户代码',
    },
    description: {
      type: DataTypes.TEXT,
      allowNull: true,
      comment: '租户描述',
    },
    status: {
      type: DataTypes.ENUM('active', 'inactive', 'suspended'),
      defaultValue: 'active',
      comment: '租户状态',
    },
    contactEmail: {
      type: DataTypes.STRING(255),
      allowNull: true,
      comment: '联系邮箱',
    },
    contactPhone: {
      type: DataTypes.STRING(20),
      allowNull: true,
      comment: '联系电话',
    },
    maxUsers: {
      type: DataTypes.INTEGER,
      defaultValue: 100,
      comment: '最大用户数',
    },
    maxProjects: {
      type: DataTypes.INTEGER,
      defaultValue: 50,
      comment: '最大工程数',
    },
    logoUrl: {
      type: DataTypes.TEXT,
      allowNull: true,
      comment: 'Logo base64数据',
    },
    loginBackgroundUrl: {
      type: DataTypes.TEXT,
      allowNull: true,
      comment: '登录页面背景图路径',
    },
    companyName: {
      type: DataTypes.STRING(200),
      allowNull: true,
      comment: '公司名称',
    },
    companyAddress: {
      type: DataTypes.TEXT,
      allowNull: true,
      comment: '公司地址',
    },
    companyPhone: {
      type: DataTypes.STRING(20),
      allowNull: true,
      comment: '公司电话',
    },
    companyWebsite: {
      type: DataTypes.STRING(500),
      allowNull: true,
      comment: '公司网站',
    },
    settings: {
      type: DataTypes.JSON,
      allowNull: true,
      comment: '租户配置（含登录页展示配置）',
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
    tableName: 'tenants',
    comment: '租户表',
    indexes: [
      {
        fields: ['code'],
        unique: true,
      },
      {
        fields: ['status'],
      },
    ],
  },
)

module.exports = Tenant
