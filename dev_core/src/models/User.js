const { DataTypes } = require('sequelize')
const { sequelize } = require('../config/database')
const bcrypt = require('bcryptjs')

const User = sequelize.define(
  'User',
  {
    id: {
      type: DataTypes.UUID,
      defaultValue: DataTypes.UUIDV4,
      primaryKey: true,
    },
    username: {
      type: DataTypes.STRING(50),
      allowNull: false,
      comment: '用户名',
    },
    password: {
      type: DataTypes.STRING(255),
      allowNull: false,
      comment: '密码哈希',
    },
    email: {
      type: DataTypes.STRING(255),
      allowNull: true,
      comment: '邮箱',
    },
    phone: {
      type: DataTypes.STRING(20),
      allowNull: true,
      comment: '手机号',
    },
    fullName: {
      type: DataTypes.STRING(100),
      allowNull: true,
      comment: '真实姓名',
    },
    role: {
      type: DataTypes.ENUM(
        'SUPER_ADMIN',
        'SYSTEM_ADMIN',
        'PROJECT_ADMIN',
        'OPS_ADMIN',
        'USER_ADMIN',
      ),
      allowNull: false,
      comment: '角色：系统管理员、工程管理员、运维管理员、用户管理员',
    },
    status: {
      type: DataTypes.ENUM('active', 'inactive', 'suspended'),
      defaultValue: 'active',
      comment: '用户状态',
    },
    tenantId: {
      type: DataTypes.UUID,
      allowNull: false,
      references: {
        model: 'tenants',
        key: 'id',
      },
      comment: '所属租户ID',
    },
    lastLoginAt: {
      type: DataTypes.DATE,
      allowNull: true,
      comment: '最后登录时间',
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
    tableName: 'users',
    comment: '用户表',
    indexes: [
      {
        fields: ['username', 'tenantId'],
        unique: true,
      },
      {
        fields: ['email'],
      },
      {
        fields: ['tenantId', 'role'],
      },
      {
        fields: ['status'],
      },
    ],
  },
)

// 密码加密钩子
User.beforeCreate(async (user) => {
  if (user.password) {
    user.password = await bcrypt.hash(user.password, 12)
  }
})

User.beforeUpdate(async (user) => {
  if (user.changed('password')) {
    user.password = await bcrypt.hash(user.password, 12)
  }
})

module.exports = User
