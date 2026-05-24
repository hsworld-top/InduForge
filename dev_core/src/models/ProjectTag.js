/**
 * 工程标签模型
 * @description 租户级标签主数据，供工程进行多标签绑定。
 */
module.exports = (sequelize, DataTypes) => {
  const ProjectTag = sequelize.define(
    'ProjectTag',
    {
      id: {
        type: DataTypes.UUID,
        defaultValue: DataTypes.UUIDV4,
        primaryKey: true,
        comment: '标签ID',
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
      name: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: '标签名称',
      },
      description: {
        type: DataTypes.TEXT,
        allowNull: true,
        comment: '标签描述',
      },
      sortOrder: {
        type: DataTypes.INTEGER,
        allowNull: false,
        defaultValue: 0,
        comment: '排序顺序',
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
      tableName: 'project_tags',
      comment: '工程标签表',
      indexes: [
        {
          unique: true,
          fields: ['id', 'tenantId'],
        },
        {
          unique: true,
          fields: ['tenantId', 'name'],
        },
        {
          fields: ['tenantId', 'sortOrder'],
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

  return ProjectTag
}
