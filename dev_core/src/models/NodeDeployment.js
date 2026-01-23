/**
 * 节点部署关系模型 - 记录工程部署到节点的关系
 * @description 一个工程可部署到多个节点，一个节点也可以运行多个工程
 */
module.exports = (sequelize, DataTypes) => {
  const NodeDeployment = sequelize.define(
    "NodeDeployment",
    {
      id: {
        type: DataTypes.CHAR(36),
        primaryKey: true,
        allowNull: false,
        comment: "部署记录ID",
      },
      nodeId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "节点ID",
      },
      deploymentId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "发布版本ID（关联 deployments 表）",
      },
      projectId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: "工程ID（冗余，便于查询）",
      },
      version: {
        type: DataTypes.STRING(50),
        allowNull: false,
        comment: "版本号（冗余，便于查询）",
      },
      // 部署状态
      status: {
        type: DataTypes.ENUM(
          "pending",     // 等待部署
          "deploying",   // 部署中
          "running",     // 运行中
          "stopped",     // 已停止
          "error",       // 错误
          "rollback"     // 已回滚
        ),
        allowNull: false,
        defaultValue: "pending",
        comment: "部署状态",
      },
      // 运行模式
      mode: {
        type: DataTypes.ENUM("DEV", "RELEASE"),
        allowNull: false,
        defaultValue: "RELEASE",
        comment: "运行模式：DEV=连接开发库，RELEASE=使用本地库",
      },
      // 运行时配置（部署时指定）
      runtimeConfig: {
        type: DataTypes.JSON,
        allowNull: true,
        defaultValue: {},
        comment: "运行时配置：port, uiTarget, defaultLocale, defaultTheme",
      },
      // 部署时间线
      deployedAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "部署完成时间",
      },
      startedAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "启动时间",
      },
      stoppedAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: "停止时间",
      },
      // 部署操作者
      deployedBy: {
        type: DataTypes.CHAR(36),
        allowNull: true,
        comment: "部署操作者ID",
      },
      // 错误信息
      errorMessage: {
        type: DataTypes.TEXT,
        allowNull: true,
        comment: "错误信息",
      },
      errorStack: {
        type: DataTypes.TEXT,
        allowNull: true,
        comment: "错误堆栈",
      },
      // 部署日志（最近的关键日志）
      deployLog: {
        type: DataTypes.JSON,
        allowNull: true,
        defaultValue: [],
        comment: "部署日志（数组，最近N条）",
      },
      // 运行时指标
      runtimeMetrics: {
        type: DataTypes.JSON,
        allowNull: true,
        defaultValue: {},
        comment: "运行时指标：onlineUsers, concurrentUsers, requestsPerSecond等",
      },
    },
    {
      tableName: "node_deployments",
      timestamps: true,
      paranoid: true, // 软删除
      indexes: [
        { fields: ["nodeId"] },
        { fields: ["deploymentId"] },
        { fields: ["projectId"] },
        { fields: ["status"] },
        {
          // 同一节点同一工程同一时间只能有一个活跃部署
          unique: true,
          fields: ["nodeId", "projectId"],
          where: {
            status: ["pending", "deploying", "running"],
            deletedAt: null,
          },
          name: "uk_node_project_active_deployment",
        },
      ],
      comment: "节点部署关系表",
    }
  );

  return NodeDeployment;
};
