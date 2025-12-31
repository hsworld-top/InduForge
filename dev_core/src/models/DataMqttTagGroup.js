const { DataTypes } = require('sequelize');

module.exports = (sequelize) => {
  const DataMqttTagGroup = sequelize.define(
    'DataMqttTagGroup',
    {
      id: {
        type: DataTypes.UUID,
        defaultValue: DataTypes.UUIDV4,
        primaryKey: true,
        comment: '变量组ID',
      },
      projectId: {
        type: DataTypes.UUID,
        allowNull: false,
        comment: '所属工程ID',
      },
      subscriptionId: {
        type: DataTypes.UUID,
        allowNull: false,
        comment: '订阅ID',
      },
      name: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: '分组名称',
      },
      code: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: '分组标识符',
      },
      description: {
        type: DataTypes.TEXT,
        comment: '分组描述',
      },
      color: {
        type: DataTypes.STRING(20),
        comment: '分组颜色(用于UI显示)',
      },
      icon: {
        type: DataTypes.STRING(50),
        comment: '分组图标',
      },
      order: {
        type: DataTypes.INTEGER,
        defaultValue: 0,
        comment: '显示顺序',
      },
      createdBy: {
        type: DataTypes.UUID,
        allowNull: false,
        comment: '创建者ID',
      },
      updatedBy: {
        type: DataTypes.UUID,
        comment: '更新者ID',
      },
    },
    {
      tableName: 'data_mqtt_tag_groups',
      timestamps: true,
      indexes: [
        {
          unique: true,
          fields: ['subscriptionId', 'code'],
          name: 'mqtt_tag_groups_subscription_code_uq',
        },
        {
          fields: ['projectId'],
          name: 'mqtt_tag_groups_project_idx',
        },
        {
          fields: ['subscriptionId'],
          name: 'mqtt_tag_groups_subscription_idx',
        },
      ],
    }
  );

  DataMqttTagGroup.associate = (models) => {
    DataMqttTagGroup.belongsTo(models.Project, {
      foreignKey: 'projectId',
      as: 'project',
    });
    DataMqttTagGroup.belongsTo(models.DataMqttSubscription, {
      foreignKey: 'subscriptionId',
      as: 'subscription',
    });
    DataMqttTagGroup.belongsTo(models.User, {
      foreignKey: 'createdBy',
      as: 'creator',
    });
    DataMqttTagGroup.belongsTo(models.User, {
      foreignKey: 'updatedBy',
      as: 'updater',
    });
    DataMqttTagGroup.hasMany(models.DataMqttTag, {
      foreignKey: 'groupId',
      as: 'tags',
    });
  };

  return DataMqttTagGroup;
};

