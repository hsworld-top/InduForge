const { DataTypes } = require('sequelize')
const { sequelize } = require('../config/database')

const Project = sequelize.define(
  'Project',
  {
    id: {
      type: DataTypes.UUID,
      defaultValue: DataTypes.UUIDV4,
      primaryKey: true,
    },
    name: {
      type: DataTypes.STRING(200),
      allowNull: false,
      comment: '工程名称',
    },
    code: {
      type: DataTypes.STRING(50),
      allowNull: true,
      comment: '工程代码',
    },
    description: {
      type: DataTypes.TEXT,
      allowNull: true,
      comment: '工程描述',
    },
    projectVariables: {
      type: DataTypes.JSON,
      allowNull: true,
      comment: '工程级别全局变量',
    },
    entryConfig: {
      type: DataTypes.JSON,
      allowNull: true,
      defaultValue: {},
      comment: '入口配置：homePageId, loginPageId, logoutPageId 等',
    },
    icon: {
      type: DataTypes.STRING(100),
      allowNull: true,
      comment: '工程图标',
    },
    status: {
      type: DataTypes.ENUM('active', 'archived', 'deleted'),
      allowNull: false,
      defaultValue: 'active',
      comment: '工程状态',
    },
    visibility: {
      type: DataTypes.ENUM('private', 'internal', 'public'),
      allowNull: false,
      defaultValue: 'private',
      comment: '可见性',
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
    createdBy: {
      type: DataTypes.UUID,
      allowNull: false,
      references: {
        model: 'users',
        key: 'id',
      },
      comment: '创建者ID',
    },
    updatedBy: {
      type: DataTypes.UUID,
      allowNull: true,
      references: {
        model: 'users',
        key: 'id',
      },
      comment: '更新者ID',
    },
    archivedAt: {
      type: DataTypes.DATE,
      allowNull: true,
      comment: '归档时间',
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
    tableName: 'projects',
    comment: '工程表',
    indexes: [
      {
        fields: ['tenantId'],
      },
      {
        fields: ['createdBy'],
      },
      {
        fields: ['status'],
      },
      {
        unique: true,
        fields: ['tenantId', 'code'],
      },
    ],
  },
)

/**
 * 仅用于工程总览/列表场景的局部序列化。
 * 说明：
 * 1. 不覆写全局 toJSON，避免影响既有接口响应。
 * 2. 优先使用显式 tags/group；缺失时再从绑定关系推导。
 * 3. 对外始终输出稳定的 tags/group 字段。
 */
Project.prototype.toOverviewPayload = function toOverviewPayload() {
  const values = { ...this.get({ plain: true }) }
  const tagBindings = Array.isArray(values.tagBindings) ? values.tagBindings : []

  let tags = []
  if (Array.isArray(values.tags)) {
    tags = values.tags
  } else if (tagBindings.length > 0) {
    tags = tagBindings.map((binding) => binding?.tag).filter(Boolean)
  }

  let group = null
  if (typeof values.group !== 'undefined') {
    group = values.group
  } else if (values.groupMember?.group) {
    group = values.groupMember.group
  } else if (values.groupMember) {
    group = values.groupMember
  }

  const { tagBindings: _ignoredTagBindings, groupMember: _ignoredGroupMember, ...rest } = values

  return {
    ...rest,
    tags,
    group: group ?? null,
  }
}

module.exports = Project
