/**
 * 工程标签绑定模型
 * @description 维护工程与租户标签的多对多关系。
 */
module.exports = (sequelize, DataTypes) => {
  const ProjectTagBinding = sequelize.define(
    "ProjectTagBinding",
    {
      id: {
        type: DataTypes.UUID,
        defaultValue: DataTypes.UUIDV4,
        primaryKey: true,
        comment: "绑定ID",
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
      projectId: {
        type: DataTypes.UUID,
        allowNull: false,
        references: {
          model: "projects",
          key: "id",
        },
        comment: "工程ID",
      },
      tagId: {
        type: DataTypes.UUID,
        allowNull: false,
        references: {
          model: "project_tags",
          key: "id",
        },
        comment: "标签ID",
      },
      createdBy: {
        type: DataTypes.UUID,
        allowNull: true,
        references: {
          model: "users",
          key: "id",
        },
        comment: "创建者ID",
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
      tableName: "project_tag_bindings",
      comment: "工程标签绑定表",
      indexes: [
        {
          unique: true,
          fields: ["projectId", "tagId"],
        },
        {
          fields: ["projectId", "tenantId"],
        },
        {
          fields: ["tagId", "tenantId"],
        },
        {
          fields: ["tenantId", "tagId"],
        },
        {
          fields: ["projectId"],
        },
        {
          fields: ["tagId"],
        },
        {
          fields: ["createdBy"],
        },
      ],
    },
  );

  return ProjectTagBinding;
};
