const { DataTypes } = require('sequelize');

module.exports = (sequelize) => {
  const DataMqttSubscription = sequelize.define(
    'DataMqttSubscription',
    {
      id: {
        type: DataTypes.UUID,
        defaultValue: DataTypes.UUIDV4,
        primaryKey: true,
        comment: '订阅ID'
      },
      projectId: {
        type: DataTypes.UUID,
        allowNull: false,
        comment: '所属工程ID'
      },
      connectionId: {
        type: DataTypes.UUID,
        allowNull: false,
        comment: 'MQTT连接ID'
      },
      name: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: '订阅名称'
      },
      topic: {
        type: DataTypes.STRING(500),
        allowNull: false,
        comment: 'MQTT主题(支持通配符)'
      },
      qos: {
        type: DataTypes.TINYINT,
        defaultValue: 0,
        validate: {
          min: 0,
          max: 2
        },
        comment: 'QoS等级(0/1/2)'
      },
      description: {
        type: DataTypes.TEXT,
        allowNull: true,
        comment: '订阅描述'
      },
      isEnabled: {
        type: DataTypes.BOOLEAN,
        defaultValue: true,
        comment: '是否启用'
      },
      messageRetention: {
        type: DataTypes.INTEGER,
        defaultValue: 100,
        comment: '消息保留数量'
      },
      createdBy: {
        type: DataTypes.UUID,
        allowNull: false,
        comment: '创建者ID'
      },
      updatedBy: {
        type: DataTypes.UUID,
        allowNull: true,
        comment: '更新者ID'
      }
    },
    {
      tableName: 'data_mqtt_subscriptions',
      timestamps: true,
      comment: 'MQTT主题订阅表',
      indexes: [
        {
          unique: true,
          fields: ['projectId', 'name'],
          name: 'mqtt_subs_project_name_uq'
        },
        {
          fields: ['connectionId'],
          name: 'mqtt_subs_conn_idx'
        },
        {
          fields: ['connectionId', 'isEnabled'],
          name: 'mqtt_subs_enabled_idx'
        }
      ]
    }
  );

  // 定义关联
  DataMqttSubscription.associate = (models) => {
    // 属于某个工程
    DataMqttSubscription.belongsTo(models.Project, {
      foreignKey: 'projectId',
      as: 'project'
    });

    // 属于某个连接
    DataMqttSubscription.belongsTo(models.DataConnection, {
      foreignKey: 'connectionId',
      as: 'connection'
    });

    // 创建者
    DataMqttSubscription.belongsTo(models.User, {
      foreignKey: 'createdBy',
      as: 'creator'
    });

    // 更新者
    DataMqttSubscription.belongsTo(models.User, {
      foreignKey: 'updatedBy',
      as: 'updater'
    });
  };

  return DataMqttSubscription;
};
