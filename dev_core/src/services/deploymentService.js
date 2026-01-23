/**
 * 部署服务 - 部署管理、启停、回滚
 * @description 处理工程部署到节点的相关操作，支持DEV/RELEASE模式
 */
const crypto = require("crypto");
const {
  Deployment,
  NodeDeployment,
  Node,
  Project,
  User,
} = require("../models");
const { Op } = require("sequelize");

/**
 * 部署服务类
 */
class DeploymentService {
  /**
   * 创建发布版本（发布流水线调用）
   * @param {Object} data - 发布数据
   * @returns {Promise<Object>} 发布版本记录
   */
  async createDeployment(data) {
    const {
      projectId,
      tenantId,
      version,
      name,
      description,
      type = "development",
      mode = "RELEASE",
      deployedBy,
    } = data;

    // RELEASE模式检查版本号是否重复
    if (mode === "RELEASE") {
      const existing = await Deployment.findOne({
        where: { projectId, version, deletedAt: null },
      });
      if (existing) {
        throw new Error(`版本 ${version} 已存在`);
      }
    }

    const id = crypto.randomUUID();
    const deployment = await Deployment.create({
      id,
      projectId,
      tenantId,
      version,
      name: name || `v${version}`,
      description,
      type,
      mode,
      status: "pending",
      deployedBy,
      startedAt: new Date(),
    });

    return deployment;
  }

  /**
   * 更新发布状态（发布流水线调用）
   * @param {string} deploymentId - 发布版本ID
   * @param {Object} data - 更新数据
   */
  async updateDeploymentStatus(deploymentId, data) {
    const deployment = await Deployment.findByPk(deploymentId);
    if (!deployment) {
      throw new Error(`发布版本 ${deploymentId} 不存在`);
    }

    const {
      status,
      artifactUrl,
      artifactHash,
      artifactSize,
      snapshotUrl,
      snapshotHash,
      manifest,
      pageCount,
      componentCount,
      datapointCount,
      errorMessage,
      buildLog,
    } = data;

    const updateData = {};
    if (status) updateData.status = status;
    if (artifactUrl) updateData.artifactUrl = artifactUrl;
    if (artifactHash) updateData.artifactHash = artifactHash;
    if (artifactSize) updateData.artifactSize = artifactSize;
    if (snapshotUrl) updateData.snapshotUrl = snapshotUrl;
    if (snapshotHash) updateData.snapshotHash = snapshotHash;
    if (manifest) updateData.manifest = manifest;
    if (pageCount !== undefined) updateData.pageCount = pageCount;
    if (componentCount !== undefined)
      updateData.componentCount = componentCount;
    if (datapointCount !== undefined)
      updateData.datapointCount = datapointCount;
    if (errorMessage) updateData.errorMessage = errorMessage;
    if (buildLog) {
      updateData.buildLog = [...(deployment.buildLog || []), ...buildLog].slice(
        -100
      );
    }

    if (status === "success" || status === "failed") {
      updateData.completedAt = new Date();
    }

    await deployment.update(updateData);
    return deployment;
  }

  /**
   * 获取工程的发布版本列表
   * @param {string} projectId - 工程ID
   * @param {Object} options - 查询选项
   */
  async listByProject(projectId, options = {}) {
    const { page = 1, pageSize = 20, status, type, mode } = options;
    const offset = (page - 1) * pageSize;

    const where = { projectId };
    if (status) where.status = status;
    if (type) where.type = type;
    if (mode) where.mode = mode;

    const { rows, count } = await Deployment.findAndCountAll({
      where,
      include: [
        {
          model: User,
          as: "deployer",
          attributes: ["id", "username"],
        },
      ],
      order: [["createdAt", "DESC"]],
      limit: pageSize,
      offset,
    });

    return {
      items: rows,
      total: count,
      page,
      pageSize,
      totalPages: Math.ceil(count / pageSize),
    };
  }

  /**
   * 获取发布版本详情
   * @param {string} deploymentId - 发布版本ID
   */
  async getById(deploymentId) {
    const deployment = await Deployment.findByPk(deploymentId, {
      include: [
        {
          model: Project,
          as: "project",
          attributes: ["id", "name", "code"],
        },
        {
          model: User,
          as: "deployer",
          attributes: ["id", "username"],
        },
        {
          model: NodeDeployment,
          as: "nodeDeployments",
          include: [
            {
              model: Node,
              as: "node",
              attributes: ["id", "name", "status", "ipAddress"],
            },
          ],
        },
      ],
    });

    if (!deployment) {
      throw new Error(`发布版本 ${deploymentId} 不存在`);
    }

    return deployment;
  }

  /**
   * DEV模式直接部署（无需版本号，连接开发库）
   * @param {string} deploymentId - 部署记录ID（DEV模式下是源版本）
   * @param {string} nodeId - 节点ID
   * @param {string} deployedBy - 部署者ID
   */
  async deployDevMode(deploymentId, nodeId, deployedBy) {
    const sourceDeployment = await Deployment.findByPk(deploymentId);
    if (!sourceDeployment) {
      throw new Error("源部署记录不存在");
    }

    // 检查节点是否已有部署
    const existingDeployment = await NodeDeployment.findOne({
      where: {
        nodeId,
        projectId: sourceDeployment.projectId,
        deletedAt: null,
      },
    });

    if (existingDeployment) {
      // 检查当前模式
      if (existingDeployment.mode === "RELEASE") {
        throw new Error(
          "该节点当前处于RELEASE模式，请先撤销部署后再切换到DEV模式"
        );
      }

      // 已经是DEV模式，直接更新状态为运行
      if (existingDeployment.mode === "DEV") {
        await existingDeployment.update({
          status: "running",
          deployLog: [
            ...(existingDeployment.deployLog || []),
            {
              time: new Date().toISOString(),
              status: "running",
              message: "DEV模式重新部署",
            },
          ],
        });

        // 更新节点当前工程信息
        await Node.update(
          {
            currentProjectId: sourceDeployment.projectId,
            currentVersion: sourceDeployment.version,
            currentDeploymentId: existingDeployment.id,
          },
          { where: { id: nodeId } }
        );

        return existingDeployment;
      }
    }

    // 创建新的DEV模式部署记录
    const id = crypto.randomUUID();
    const nodeDeployment = await NodeDeployment.create({
      id,
      nodeId,
      deploymentId: sourceDeployment.id,
      projectId: sourceDeployment.projectId,
      version: sourceDeployment.version, // 使用源版本号
      mode: "DEV",
      status: "pending",
      runtimeConfig: { syncMode: "development_database" },
      deployedBy,
      deployLog: [
        {
          time: new Date().toISOString(),
          status: "pending",
          message: "DEV模式部署任务已创建",
        },
      ],
    });

    // 更新节点当前工程信息
    await Node.update(
      {
        currentProjectId: sourceDeployment.projectId,
        currentVersion: sourceDeployment.version,
        currentDeploymentId: id,
      },
      { where: { id: nodeId } }
    );

    return nodeDeployment;
  }

  /**
   * RELEASE模式发布部署（需要版本号）
   * @param {string} deploymentId - 发布版本ID
   * @param {string} nodeId - 节点ID
   * @param {Object} runtimeConfig - 运行时配置
   * @param {string} deployedBy - 部署者ID
   */
  async deployReleaseMode(deploymentId, nodeId, runtimeConfig, deployedBy) {
    const deployment = await Deployment.findByPk(deploymentId);
    if (!deployment) {
      throw new Error("发布版本不存在");
    }

    // 检查节点是否已有DEV模式部署
    const existingDevDeployment = await NodeDeployment.findOne({
      where: {
        nodeId,
        projectId: deployment.projectId,
        mode: "DEV",
        deletedAt: null,
      },
    });

    if (existingDevDeployment) {
      // 自动停止DEV实例
      await existingDevDeployment.update({
        status: "stopped",
        stoppedAt: new Date(),
        deployLog: [
          ...(existingDevDeployment.deployLog || []),
          {
            time: new Date().toISOString(),
            status: "stopped",
            message: "切换到RELEASE模式，停止DEV实例",
          },
        ],
      });
    }

    // 检查是否有其他RELEASE部署
    const existingRelease = await NodeDeployment.findOne({
      where: {
        nodeId,
        projectId: deployment.projectId,
        mode: "RELEASE",
        status: { [Op.in]: ["pending", "deploying", "running"] },
        deletedAt: null,
      },
    });

    if (existingRelease) {
      await existingRelease.update({
        status: "stopped",
        stoppedAt: new Date(),
      });
    }

    // 创建新的RELEASE部署记录
    const id = crypto.randomUUID();
    const nodeDeployment = await NodeDeployment.create({
      id,
      nodeId,
      deploymentId,
      projectId: deployment.projectId,
      version: deployment.version,
      mode: "RELEASE",
      status: "pending",
      runtimeConfig: { ...runtimeConfig, syncMode: "local_database" },
      deployedBy,
      deployLog: [
        {
          time: new Date().toISOString(),
          status: "pending",
          message: "RELEASE模式部署任务已创建",
        },
      ],
    });

    // 更新节点当前工程信息
    await Node.update(
      {
        currentProjectId: deployment.projectId,
        currentVersion: deployment.version,
        currentDeploymentId: id,
      },
      { where: { id: nodeId } }
    );

    return nodeDeployment;
  }

  /**
   * 批量部署到多个节点（支持DEV和RELEASE模式）
   * @param {string} deploymentId - 发布版本ID
   * @param {Array<string>} nodeIds - 节点ID列表
   * @param {string} mode - 运行模式 DEV | RELEASE
   * @param {Object} runtimeConfig - 运行时配置
   * @param {string} deployedBy - 部署者ID
   */
  async deployToNodes(deploymentId, nodeIds, mode, runtimeConfig, deployedBy) {
    const results = [];

    for (const nodeId of nodeIds) {
      try {
        let result;
        if (mode === "DEV") {
          result = await this.deployDevMode(deploymentId, nodeId, deployedBy);
        } else {
          result = await this.deployReleaseMode(
            deploymentId,
            nodeId,
            runtimeConfig,
            deployedBy
          );
        }
        results.push({ nodeId, success: true, deploymentId: result.id });
      } catch (error) {
        results.push({ nodeId, success: false, error: error.message });
      }
    }

    return results;
  }

  /**
   * 部署到节点（兼容旧接口，默认RELEASE模式）
   * @param {string} deploymentId - 发布版本ID
   * @param {string} nodeId - 节点ID
   * @param {Object} runtimeConfig - 运行时配置
   * @param {string} deployedBy - 部署操作者ID
   */
  async deployToNode(deploymentId, nodeId, runtimeConfig, deployedBy) {
    return this.deployReleaseMode(
      deploymentId,
      nodeId,
      runtimeConfig,
      deployedBy
    );
  }

  /**
   * 启动工程
   * @param {string} nodeDeploymentId - 节点部署记录ID
   */
  async start(nodeDeploymentId) {
    const nodeDeployment = await NodeDeployment.findByPk(nodeDeploymentId);
    if (!nodeDeployment) {
      throw new Error("部署记录不存在");
    }

    await nodeDeployment.update({
      status: "running",
      startedAt: new Date(),
      deployLog: [
        ...(nodeDeployment.deployLog || []),
        {
          time: new Date().toISOString(),
          status: "running",
          message: "收到启动指令",
        },
      ],
    });

    // 更新节点当前工程信息
    await Node.update(
      {
        currentProjectId: nodeDeployment.projectId,
        currentVersion: nodeDeployment.version,
        currentDeploymentId: nodeDeploymentId,
      },
      { where: { id: nodeDeployment.nodeId } }
    );

    return nodeDeployment;
  }

  /**
   * 停止工程
   * @param {string} nodeDeploymentId - 节点部署记录ID
   */
  async stop(nodeDeploymentId) {
    const nodeDeployment = await NodeDeployment.findByPk(nodeDeploymentId);
    if (!nodeDeployment) {
      throw new Error("部署记录不存在");
    }

    await nodeDeployment.update({
      status: "stopped",
      stoppedAt: new Date(),
      deployLog: [
        ...(nodeDeployment.deployLog || []),
        {
          time: new Date().toISOString(),
          status: "stopped",
          message: "收到停止指令",
        },
      ],
    });

    // 清除节点当前工程信息
    await Node.update(
      {
        currentProjectId: null,
        currentVersion: null,
        currentDeploymentId: null,
      },
      { where: { id: nodeDeployment.nodeId } }
    );

    return nodeDeployment;
  }

  /**
   * 重启工程
   * @param {string} nodeDeploymentId - 节点部署记录ID
   */
  async restart(nodeDeploymentId) {
    const nodeDeployment = await NodeDeployment.findByPk(nodeDeploymentId);
    if (!nodeDeployment) {
      throw new Error("部署记录不存在");
    }

    // 先停止
    await nodeDeployment.update({
      status: "stopped",
      stoppedAt: new Date(),
      deployLog: [
        ...(nodeDeployment.deployLog || []),
        {
          time: new Date().toISOString(),
          status: "stopped",
          message: "收到重启指令（停止）",
        },
      ],
    });

    // 再启动
    await nodeDeployment.update({
      status: "running",
      startedAt: new Date(),
      deployLog: [
        ...(nodeDeployment.deployLog || []),
        {
          time: new Date().toISOString(),
          status: "running",
          message: "收到重启指令（启动）",
        },
      ],
    });

    // 更新节点当前工程信息
    await Node.update(
      {
        currentProjectId: nodeDeployment.projectId,
        currentVersion: nodeDeployment.version,
        currentDeploymentId: nodeDeploymentId,
      },
      { where: { id: nodeDeployment.nodeId } }
    );

    return nodeDeployment;
  }

  /**
   * 撤销部署（释放节点资源）
   * @param {string} nodeDeploymentId - 节点部署记录ID
   */
  async undeploy(nodeDeploymentId) {
    const nodeDeployment = await NodeDeployment.findByPk(nodeDeploymentId);
    if (!nodeDeployment) {
      throw new Error("部署记录不存在");
    }

    await nodeDeployment.update({
      status: "stopped",
      stoppedAt: new Date(),
      deployLog: [
        ...(nodeDeployment.deployLog || []),
        {
          time: new Date().toISOString(),
          status: "stopped",
          message: "已撤销部署",
        },
      ],
    });

    // 清除节点当前工程信息
    await Node.update(
      {
        currentProjectId: null,
        currentVersion: null,
        currentDeploymentId: null,
      },
      { where: { id: nodeDeployment.nodeId } }
    );

    return nodeDeployment;
  }

  /**
   * 获取节点上工程的当前模式
   * @param {string} projectId - 工程ID
   * @param {string} nodeId - 节点ID
   */
  async getNodeProjectMode(projectId, nodeId) {
    const deployment = await NodeDeployment.findOne({
      where: {
        projectId,
        nodeId,
        status: "running",
        deletedAt: null,
      },
    });

    return deployment ? deployment.mode : null;
  }

  /**
   * 停止节点上的工程（旧接口兼容）
   * @param {string} nodeDeploymentId - 节点部署记录ID
   */
  async stopNodeDeployment(nodeDeploymentId) {
    return this.stop(nodeDeploymentId);
  }

  /**
   * 回滚到指定版本（仅RELEASE模式）
   * @param {string} nodeId - 节点ID
   * @param {string} deploymentId - 目标发布版本ID
   * @param {string} deployedBy - 操作者ID
   */
  async rollback(nodeId, deploymentId, deployedBy) {
    // 回滚仅作用于RELEASE模式
    return this.deployReleaseMode(deploymentId, nodeId, {}, deployedBy);
  }

  /**
   * 获取节点的部署历史
   * @param {string} nodeId - 节点ID
   * @param {Object} options - 查询选项
   */
  async getNodeDeploymentHistory(nodeId, options = {}) {
    const { page = 1, pageSize = 20 } = options;
    const offset = (page - 1) * pageSize;

    const { rows, count } = await NodeDeployment.findAndCountAll({
      where: { nodeId },
      include: [
        {
          model: Deployment,
          as: "deployment",
          attributes: ["id", "version", "name", "type", "mode"],
        },
        {
          model: Project,
          as: "project",
          attributes: ["id", "name"],
        },
        {
          model: User,
          as: "deployer",
          attributes: ["id", "username"],
        },
      ],
      order: [["createdAt", "DESC"]],
      limit: pageSize,
      offset,
    });

    return {
      items: rows,
      total: count,
      page,
      pageSize,
      totalPages: Math.ceil(count / pageSize),
    };
  }
}

module.exports = new DeploymentService();
