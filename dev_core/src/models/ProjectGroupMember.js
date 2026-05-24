/**
 * 工程分组成员模型
 * @description 维护工程与分组关系，限制单工程最多归属一个分组。
 */
module.exports = (sequelize, DataTypes) => {
  const ProjectGroupMember = sequelize.define(
    'ProjectGroupMember',
    {
      id: {
        type: DataTypes.UUID,
        defaultValue: DataTypes.UUIDV4,
        primaryKey: true,
        comment: '成员关系ID',
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
      projectId: {
        type: DataTypes.UUID,
        allowNull: false,
        unique: true,
        references: {
          model: 'projects',
          key: 'id',
        },
        comment: '工程ID（唯一，保证单工程仅一个分组）',
      },
      groupId: {
        type: DataTypes.UUID,
        allowNull: false,
        references: {
          model: 'project_groups',
          key: 'id',
        },
        comment: '分组ID',
      },
      createdBy: {
        type: DataTypes.UUID,
        allowNull: true,
        references: {
          model: 'users',
          key: 'id',
        },
        comment: '创建者ID',
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
      tableName: 'project_group_members',
      comment: '工程分组成员表',
      indexes: [
        {
          unique: true,
          fields: ['projectId'],
        },
        {
          fields: ['projectId', 'tenantId'],
        },
        {
          fields: ['groupId', 'tenantId'],
        },
        {
          fields: ['tenantId', 'groupId'],
        },
        {
          fields: ['groupId'],
        },
        {
          fields: ['createdBy'],
        },
      ],
    },
  )

  return ProjectGroupMember
}
