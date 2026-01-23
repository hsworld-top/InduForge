/**
 * 节点模型 - 运行时节点注册与状态管理
 * @description 存储 NodeAgent 注册信息、心跳状态、运行指标
 */
module.exports = (sequelize, DataTypes) => {
  const Node = sequelize.define(
    "Node",
    {
      id: {
        type: DataTypes.CHAR(36),
        primaryKey: true,
        allowNull: false,
        comment: "节点ID",
      },
      tenantId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "所属租户ID",
      },
      name: {
        type: DataTypes.STRING(100),
        allowNull: false,
        comment: "节点名称",
      },
      description: {
        type: DataTypes.STRING(500),
        allowNull: true,
        comment: "节点描述",
      },
      agentVersion: {
        type: DataTypes.STRING(20),
        allowNull: true,
        comment: "NodeAgent版本",
      },
      status: {
        type: DataTypes.ENUM("online", "offline", "error"),
        allowNull: false,
        defaultValue: "offline",
        comment: "节点状态",
      },
      // 当前运行的工程信息
      currentProjectId: {
        type: DataTypes.CHAR(36),
        allowNull: true,
        comment: "当前运行的工程ID",
      },
      currentVersion: {
        type: DataTypes.STRING(50),
        allowNull: true,
        comment: "当前运行的版本号",
      },
      currentDeploymentId: {
        type: DataTypes.CHAR(36),
        allowNull: true,
        comment: "当前部署记录ID",
      },
      // 网络信息
      ipAddress: {
        type: DataTypes.STRING(45),
        allowNull: true,
        comment: "节点IP地址",
      },
      port: {
        type: DataTypes.INTEGER,
        allowNull: true,
        defaultValue: 8080,
        comment: "RuntimeEngine运行端口",
      },
      // 心跳与健康
      lastHeartbeatAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "最后心跳时间",
      },
      lastErrorMessage: {
        type: DataTypes.TEXT,
        allowNull: true,
        comment: "最后错误信息",
      },
      lastErrorAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "最后错误时间",
      },
      // 运行指标（JSON存储，便于扩展）
      metrics: {
        type: DataTypes.JSON,
        allowNull: true,
        defaultValue: {},
        comment: "运行指标：cpu, memory, disk, network, uptime等",
      },
      // 节点配置
      config: {
        type: DataTypes.JSON,
        allowNull: true,
        defaultValue: {},
        comment: "节点配置：标签、分组等",
      },
      // 注册密钥（用于节点认证）
      registrationToken: {
        type: DataTypes.STRING(64),
        allowNull: true,
        comment: "注册令牌（首次注册后生成）",
      },
      approvalStatus: {
        type: DataTypes.ENUM("pending", "approved", "rejected"),
        allowNull: false,
        defaultValue: "pending",
        comment: "审批状态",
      },
      approvedAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "审批时间",
      },
      approvedBy: {
        type: DataTypes.CHAR(36),
        allowNull: true,
        comment: "审批人ID",
      },
      registeredBy: {
        type: DataTypes.CHAR(36),
        allowNull: true,
        comment: "注册申请人ID",
      },
      mode: {
        type: DataTypes.ENUM("online", "offline"),
        allowNull: false,
        defaultValue: "online",
        comment: "节点模式(在线/离线)",
      },
      createdBy: {
        type: DataTypes.CHAR(36),
        allowNull: true,
        comment: "创建者ID",
      },
      updatedBy: {
        type: DataTypes.CHAR(36),
        allowNull: true,
        comment: "更新者ID",
      },
    },
    {
      tableName: "nodes",
      timestamps: true,
      paranoid: true, // 软删除
      indexes: [
        { fields: ["tenantId"] },
        { fields: ["status"] },
        { fields: ["currentProjectId"] },
        { fields: ["lastHeartbeatAt"] },
        {
          unique: true,
          fields: ["tenantId", "name"],
          name: "uk_tenant_node_name",
        },
      ],
      comment: "运行时节点表",
    }
  );

  return Node;
};
