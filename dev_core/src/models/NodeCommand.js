/**
 * 节点命令模型 - 记录中心下发到节点的命令
 * @description 用于命令下发、重试、超时与死信治理
 */
module.exports = (sequelize, DataTypes) => {
  const NodeCommand = sequelize.define(
    "NodeCommand",
    {
      id: {
        type: DataTypes.CHAR(36),
        primaryKey: true,
        allowNull: false,
        comment: "命令ID",
      },
      tenantId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "租户ID",
      },
      nodeId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "节点ID",
      },
      deploymentId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "节点部署记录ID",
      },
      projectId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "工程ID",
      },
      type: {
        type: DataTypes.ENUM("deploy", "start", "stop", "restart"),
        allowNull: false,
        comment: "命令类型",
      },
      status: {
        type: DataTypes.ENUM(
          "pending",
          "issued",
          "acknowledged",
          "completed",
          "failed",
          "dead_letter"
        ),
        allowNull: false,
        defaultValue: "pending",
        comment: "命令状态",
      },
      payload: {
        type: DataTypes.JSON,
        allowNull: true,
        defaultValue: {},
        comment: "命令负载",
      },
      attempts: {
        type: DataTypes.INTEGER,
        allowNull: false,
        defaultValue: 0,
        comment: "已重试次数",
      },
      maxAttempts: {
        type: DataTypes.INTEGER,
        allowNull: false,
        defaultValue: 3,
        comment: "最大重试次数",
      },
      timeoutSeconds: {
        type: DataTypes.INTEGER,
        allowNull: false,
        defaultValue: 30,
        comment: "命令超时秒数",
      },
      requestedAt: {
        type: DataTypes.DATE,
        allowNull: false,
        defaultValue: DataTypes.NOW,
        comment: "请求时间",
      },
      issuedAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "下发时间",
      },
      acknowledgedAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "节点确认时间",
      },
      completedAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "完成时间",
      },
      lastError: {
        type: DataTypes.TEXT,
        allowNull: true,
        comment: "最近错误信息",
      },
    },
    {
      tableName: "node_commands",
      timestamps: true,
      paranoid: true,
      indexes: [
        { fields: ["tenantId"] },
        { fields: ["nodeId"] },
        { fields: ["deploymentId"] },
        { fields: ["projectId"] },
        { fields: ["type"] },
        { fields: ["status"] },
        { fields: ["requestedAt"] },
      ],
      comment: "节点命令表",
    }
  );

  return NodeCommand;
};

