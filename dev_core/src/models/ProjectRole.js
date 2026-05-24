/**
 * 工程运行态角色模型
 * @description 保存工程维度的运行态角色定义，供后续授权绑定与权限聚合使用。
 */
module.exports = (sequelize, DataTypes) => {
  const ProjectRole = sequelize.define(
    'ProjectRole',
    {
      id: {
        type: DataTypes.UUID,
        defaultValue: DataTypes.UUIDV4,
        primaryKey: true,
        comment: '角色ID',
      },
      projectId: {
        type: DataTypes.UUID,
        allowNull: false,
        references: {
          model: 'projects',
          key: 'id',
        },
        comment: '所属工程ID',
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
      updatedBy: {
        type: DataTypes.UUID,
        allowNull: true,
        references: {
          model: 'users',
          key: 'id',
        },
        comment: '更新者ID',
      },
      code: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: '角色编码',
      },
      name: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: '角色名称',
      },
      description: {
        type: DataTypes.TEXT,
        allowNull: true,
        comment: '角色描述',
      },
      isSystem: {
        type: DataTypes.BOOLEAN,
        allowNull: false,
        defaultValue: false,
        comment: '是否系统内置角色',
      },
      status: {
        type: DataTypes.ENUM('active', 'inactive'),
        allowNull: false,
        defaultValue: 'active',
        comment: '角色状态',
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
      tableName: 'project_roles',
      comment: '工程运行态角色表',
      indexes: [
        {
          unique: true,
          fields: ['projectId', 'code'],
        },
        {
          fields: ['projectId', 'status'],
        },
        {
          fields: ['createdBy'],
        },
        {
          fields: ['updatedBy'],
        },
      ],
    },
  )

  return ProjectRole
}
