/**
 * 发布版本模型 - 工程发布记录与制品管理
 * @description 存储工程发布的版本信息、构建产物、发布快照
 */
module.exports = (sequelize, DataTypes) => {
  const Deployment = sequelize.define(
    'Deployment',
    {
      id: {
        type: DataTypes.CHAR(36),
        primaryKey: true,
        allowNull: false,
        comment: '发布版本ID',
      },
      projectId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: '工程ID',
      },
      tenantId: {
        type: DataTypes.CHAR(36),
        allowNull: false,
        comment: '租户ID（冗余，便于查询）',
      },
      version: {
        type: DataTypes.STRING(50),
        allowNull: false,
        comment: '版本号（语义化版本）',
      },
      name: {
        type: DataTypes.STRING(200),
        allowNull: true,
        comment: '版本名称（可选描述）',
      },
      description: {
        type: DataTypes.TEXT,
        allowNull: true,
        comment: '版本描述/发布说明',
      },
      // 部署类型
      type: {
        type: DataTypes.ENUM('development', 'staging', 'production'),
        allowNull: false,
        defaultValue: 'development',
        comment: '部署类型',
      },
      // 运行模式（DEV实时同步 / RELEASE独立运行）
      mode: {
        type: DataTypes.ENUM('DEV', 'RELEASE'),
        allowNull: false,
        defaultValue: 'RELEASE',
        comment: '运行模式：DEV=连接开发库实时同步，RELEASE=使用本地库',
      },
      // 构建状态
      status: {
        type: DataTypes.ENUM(
          'pending', // 等待构建
          'building', // 构建中
          'success', // 构建成功
          'failed', // 构建失败
        ),
        allowNull: false,
        defaultValue: 'pending',
        comment: '构建状态',
      },
      // 构建配置
      buildConfig: {
        type: DataTypes.JSON,
        allowNull: true,
        defaultValue: {},
        comment: '构建配置',
      },
      // 制品信息
      artifactUrl: {
        type: DataTypes.STRING(1000),
        allowNull: true,
        comment: '构建产物URL（IFP包）',
      },
      artifactHash: {
        type: DataTypes.STRING(64),
        allowNull: true,
        comment: '产物SHA256哈希',
      },
      artifactSize: {
        type: DataTypes.BIGINT,
        allowNull: true,
        comment: '产物大小（字节）',
      },
      // 快照信息
      snapshotUrl: {
        type: DataTypes.STRING(1000),
        allowNull: true,
        comment: '快照文件URL',
      },
      snapshotHash: {
        type: DataTypes.STRING(64),
        allowNull: true,
        comment: '快照SHA256哈希',
      },
      // 清单信息（manifest.json 的关键字段冗余）
      manifest: {
        type: DataTypes.JSON,
        allowNull: true,
        comment: '清单信息：dataRequirements, capabilities, security等',
      },
      // 统计信息
      pageCount: {
        type: DataTypes.INTEGER,
        allowNull: true,
        defaultValue: 0,
        comment: '页面数量',
      },
      componentCount: {
        type: DataTypes.INTEGER,
        allowNull: true,
        defaultValue: 0,
        comment: '组件数量',
      },
      datapointCount: {
        type: DataTypes.INTEGER,
        allowNull: true,
        defaultValue: 0,
        comment: '数据点数量',
      },
      // 构建时间线
      startedAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: '构建开始时间',
      },
      completedAt: {
        type: DataTypes.DATE,
        allowNull: true,
        comment: '构建完成时间',
      },
      // 构建日志
      buildLog: {
        type: DataTypes.JSON,
        allowNull: true,
        defaultValue: [],
        comment: '构建日志（数组）',
      },
      // 构建错误
      errorMessage: {
        type: DataTypes.TEXT,
        allowNull: true,
        comment: '错误信息',
      },
      // 发布者
      deployedBy: {
        type: DataTypes.CHAR(36),
        allowNull: true,
        comment: '发布者ID',
      },
    },
    {
      tableName: 'deployments',
      timestamps: true,
      paranoid: true, // 软删除
      indexes: [
        { fields: ['projectId'] },
        { fields: ['tenantId'] },
        { fields: ['status'] },
        { fields: ['type'] },
        { fields: ['createdAt'] },
        {
          // 工程内版本号唯一
          unique: true,
          fields: ['projectId', 'version'],
          name: 'uk_project_version',
        },
      ],
      comment: '发布版本表',
    },
  )

  return Deployment
}
