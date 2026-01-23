/**
 * 节点服务 - 节点注册、心跳、状态管理
 * @description 处理 NodeAgent 的注册、心跳上报、状态查询等功能
 */
const crypto = require("crypto");
const { Node, NodeDeployment, Deployment, Project, Tenant, User } = require("../models");
const { Op } = require("sequelize");
const socketService = require("./socketService");
const bcrypt = require("bcryptjs");

/**
 * 生成节点注册令牌
 */
function generateRegistrationToken() {
  return crypto.randomBytes(32).toString("hex");
}

/**
 * 节点服务类
 */
class NodeService {
  /**
   * 注册新节点
   * @param {Object} data - 节点注册数据
   * @param {string} data.tenantId - 租户ID
   * @param {string} data.name - 节点名称
   * @param {string} data.agentVersion - Agent版本
   * @param {string} data.ipAddress - IP地址
   * @param {number} data.port - 端口
   * @param {string} data.createdBy - 创建者ID
   * @returns {Promise<Object>} 节点信息及注册令牌
   */
  async register(data) {
    const { tenantId, name, agentVersion, ipAddress, port, createdBy, description } = data;

    // 检查节点名称是否重复
    const existing = await Node.findOne({
      where: { tenantId, name, deletedAt: null },
    });
    if (existing) {
      throw new Error(`节点名称 "${name}" 已存在`);
    }

    const id = crypto.randomUUID();
    const registrationToken = generateRegistrationToken();

    const node = await Node.create({
      id,
      tenantId,
      name,
      description,
      agentVersion,
      ipAddress,
      port: port || 8080,
      status: "offline", // 注册后默认为离线，等待审批
      approvalStatus: "pending",
      registrationToken,
      lastHeartbeatAt: null,
      createdBy,
      updatedBy: createdBy,
    });

    return {
      id: node.id,
      name: node.name,
      registrationToken,
      status: node.status,
    };
  }

  /**
   * 带用户认证的节点注册
   * @param {Object} data - 注册数据
   * @param {string} data.username - 用户名
   * @param {string} data.password - 密码
   * @param {string} data.nodeName - 节点名称
   * @param {string} data.nodeDescription - 节点描述
   * @param {string} data.agentVersion - Agent版本
   * @param {string} data.ipAddress - IP地址
   * @param {number} data.port - 端口
   * @param {string} data.mode - 节点模式(online/offline)
   * @returns {Promise<Object>} 注册结果
   */
  async registerWithAuth(data) {
    const { username, password, nodeName, nodeDescription, agentVersion, ipAddress, port, mode = 'online' } = data;

    // 验证节点名称格式
    const namePattern = /^[a-zA-Z0-9_-]{3,50}$/;
    if (!namePattern.test(nodeName)) {
      throw new Error("节点名称格式不正确，仅允许字母、数字、中划线、下划线，长度3-50个字符");
    }

    // 验证用户身份
    const user = await User.findOne({
      where: { username, status: "active" },
      include: [{ model: Tenant, as: "tenant" }],
    });

    if (!user) {
      throw new Error("用户名或密码错误");
    }

    // 验证密码
    const isPasswordValid = await bcrypt.compare(password, user.password);
    if (!isPasswordValid) {
      throw new Error("用户名或密码错误");
    }

    // 检查租户状态
    if (user.tenant && user.tenant.status !== "active") {
      throw new Error("租户已被禁用");
    }

    const tenantId = user.tenantId;

    // 检查节点名称是否重复
    const existing = await Node.findOne({
      where: { tenantId, name: nodeName, deletedAt: null },
    });
    if (existing) {
      throw new Error(`节点名称 "${nodeName}" 已存在`);
    }

    const nodeId = crypto.randomUUID();
    const registrationToken = generateRegistrationToken();

    // 检查用户角色，决定是否自动审批
    const hasOpsPermission = ["OPS_ADMIN", "SYSTEM_ADMIN"].includes(user.role);
    const approvalStatus = hasOpsPermission ? "approved" : "pending";

    const node = await Node.create({
      id: nodeId,
      tenantId,
      name: nodeName,
      description: nodeDescription,
      agentVersion: agentVersion || "1.0.0",
      ipAddress,
      port: port || 8080,
      mode,
      status: "offline",
      approvalStatus,
      registrationToken: hasOpsPermission ? registrationToken : null, // 仅自动审批时生成token
      approvedAt: hasOpsPermission ? new Date() : null,
      approvedBy: hasOpsPermission ? user.id : null,
      registeredBy: user.id,
      createdBy: user.id,
      updatedBy: user.id,
    });

    return {
      nodeId: node.id,
      nodeName: node.name,
      approvalStatus: node.approvalStatus,
      registrationToken: hasOpsPermission ? registrationToken : null,
      message: hasOpsPermission ? "自动审批通过" : "申请已提交，等待管理员审批",
      autoApproved: hasOpsPermission,
    };
  }

  /**
   * 获取节点审批状态（无需认证）
   * @param {string} nodeId - 节点ID
   * @returns {Promise<Object>} 审批状态信息
   */
  async getApprovalStatus(nodeId) {
    const node = await Node.findByPk(nodeId, {
      attributes: ["id", "name", "approvalStatus", "registrationToken"],
    });

    if (!node) {
      throw new Error(`节点 ${nodeId} 不存在`);
    }

    const result = {
      nodeId: node.id,
      nodeName: node.name,
      approvalStatus: node.approvalStatus,
    };

    // 仅在审批通过时返回 token
    if (node.approvalStatus === "approved" && node.registrationToken) {
      result.registrationToken = node.registrationToken;
    }

    return result;
  }

  /**
   * 节点心跳上报
   * @param {string} nodeId - 节点ID
   * @param {Object} heartbeatData - 心跳数据
   * @param {string} heartbeatData.agentVersion - Agent版本
   * @param {Object} heartbeatData.metrics - 运行指标
   * @param {boolean} heartbeatData.runtimeHealthy - RuntimeEngine是否健康
   * @param {Object} heartbeatData.runningProject - 当前运行的工程信息
   * @returns {Promise<Object>} 更新后的节点状态及待执行指令
   */
  async heartbeat(nodeId, heartbeatData) {
    const { agentVersion, metrics, projects, ipAddress } = heartbeatData;

    const node = await Node.findByPk(nodeId);
    if (!node) {
      throw new Error(`节点 ${nodeId} 不存在`);
    }

    // 检查审批状态
    if (node.approvalStatus !== "approved") {
      throw new Error(`节点 ${nodeId} 未通过审批，无法上报心跳`);
    }

    // 更新节点状态
    const updateData = {
      status: "online", // 收到心跳即认为 Agent 在线
      lastHeartbeatAt: new Date(),
      agentVersion: agentVersion || node.agentVersion,
      metrics: {
        ...node.metrics,
        ...metrics,
        lastUpdated: new Date().toISOString(),
      },
    };

    if (ipAddress) {
      updateData.ipAddress = ipAddress;
    }

    // 更新各工程运行状态与指标
    if (projects && Array.isArray(projects)) {
      for (const p of projects) {
        await NodeDeployment.update(
          {
            status: p.status,
            runtimeMetrics: p.metrics || {},
            updatedAt: new Date(),
          },
          {
            where: {
              nodeId,
              projectId: p.projectId,
              deletedAt: null,
            },
          }
        );
      }
    }

    // 如果有错误信息
    if (heartbeatData.error) {
      updateData.lastErrorMessage = heartbeatData.error.message;
      updateData.lastErrorAt = new Date();
      updateData.status = "error";
    }

    await node.update(updateData);

    // 发送 WebSocket 广播
    socketService.broadcastNodeMetrics(node.tenantId, nodeId, updateData.metrics);
    if (updateData.status !== node.status) {
      socketService.broadcastNodeStatus(node.tenantId, nodeId, updateData.status);
    }
    if (projects && Array.isArray(projects)) {
      for (const p of projects) {
        socketService.broadcastProjectMetrics(node.tenantId, nodeId, p.projectId, p.metrics);
      }
    }

    // 查询待下发的指令（可扩展）
    const pendingCommands = await this.getPendingCommands(nodeId);

    return {
      status: updateData.status,
      commands: pendingCommands,
    };
  }

  /**
   * 获取待下发指令
   * @param {string} nodeId - 节点ID
   * @returns {Promise<Array>} 待执行指令列表
   */
  async getPendingCommands(nodeId) {
    // 检查是否有待部署的任务
    const pendingDeployment = await NodeDeployment.findOne({
      where: {
        nodeId,
        status: "pending",
        deletedAt: null,
      },
      include: [
        {
          model: Deployment,
          as: "deployment",
          attributes: ["id", "artifactUrl", "artifactHash", "version"],
        },
      ],
    });

    if (pendingDeployment) {
      return [
        {
          type: "deploy",
          payload: {
            deploymentId: pendingDeployment.id,
            artifactUrl: pendingDeployment.deployment.artifactUrl,
            artifactHash: pendingDeployment.deployment.artifactHash,
            version: pendingDeployment.deployment.version,
            runtimeConfig: pendingDeployment.runtimeConfig,
          },
        },
      ];
    }

    return [];
  }

  /**
   * 获取节点列表
   * @param {string} tenantId - 租户ID
   * @param {Object} options - 查询选项
   * @returns {Promise<Object>} 节点列表及分页信息
   */
  async list(tenantId, options = {}) {
    const { page = 1, pageSize = 20, status, search } = options;
    const offset = (page - 1) * pageSize;

    const where = { tenantId };
    if (status) {
      where.status = status;
    }
    if (options.approvalStatus) {
      where.approvalStatus = options.approvalStatus;
    }
    if (search) {
      where[Op.or] = [
        { name: { [Op.like]: `%${search}%` } },
        { ipAddress: { [Op.like]: `%${search}%` } },
      ];
    }

    const { rows, count } = await Node.findAndCountAll({
      where,
      include: [
        {
          model: NodeDeployment,
          as: "deployments",
          include: [
            {
              model: Project,
              as: "project",
              attributes: ["id", "name", "code"],
            },
          ],
          where: { deletedAt: null },
          required: false,
        },
        {
          model: User,
          as: "registrant",
          attributes: ["id", "username", "role"],
          required: false,
        },
      ],
      order: [["lastHeartbeatAt", "DESC"]],
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
   * 获取节点详情
   * @param {string} nodeId - 节点ID
   * @returns {Promise<Object>} 节点详细信息
   */
  async getById(nodeId) {
    const node = await Node.findByPk(nodeId, {
      include: [
        {
          model: Project,
          as: "currentProject",
          attributes: ["id", "name", "code"],
        },
        {
          model: NodeDeployment,
          as: "currentDeployment",
          include: [
            {
              model: Deployment,
              as: "deployment",
              attributes: ["id", "version", "name", "status"],
            },
          ],
        },
        {
          model: Tenant,
          as: "tenant",
          attributes: ["id", "name"],
        },
      ],
    });

    if (!node) {
      throw new Error(`节点 ${nodeId} 不存在`);
    }

    return node;
  }

  /**
   * 更新节点信息
   * @param {string} nodeId - 节点ID
   * @param {Object} data - 更新数据
   * @param {string} userId - 操作用户ID
   * @returns {Promise<Object>} 更新后的节点
   */
  async update(nodeId, data, userId) {
    const node = await Node.findByPk(nodeId);
    if (!node) {
      throw new Error(`节点 ${nodeId} 不存在`);
    }

    const { name, description, port, config } = data;
    const updateData = { updatedBy: userId };

    if (name !== undefined) {
      // 检查名称重复
      const existing = await Node.findOne({
        where: {
          tenantId: node.tenantId,
          name,
          id: { [Op.ne]: nodeId },
          deletedAt: null,
        },
      });
      if (existing) {
        throw new Error(`节点名称 "${name}" 已存在`);
      }
      updateData.name = name;
    }

    if (description !== undefined) updateData.description = description;
    if (port !== undefined) updateData.port = port;
    if (config !== undefined) updateData.config = { ...node.config, ...config };

    await node.update(updateData);
    return node;
  }

  /**
   * 删除节点
   * @param {string} nodeId - 节点ID
   * @returns {Promise<boolean>} 是否成功
   */
  async delete(nodeId) {
    const node = await Node.findByPk(nodeId);
    if (!node) {
      throw new Error(`节点 ${nodeId} 不存在`);
    }

    // 检查是否有正在运行的工程
    const runningCount = await NodeDeployment.count({
      where: {
        nodeId,
        status: "running",
        deletedAt: null
      }
    });

    if (runningCount > 0 && node.status === "online") {
      throw new Error(`节点仍有 ${runningCount} 个工程在运行中，请先停止后再删除`);
    }

    await node.destroy(); // 软删除
    return true;
  }

  /**
   * 检查并标记离线节点
   * @param {number} timeoutMinutes - 超时分钟数
   * @returns {Promise<number>} 标记为离线的节点数量
   */
  async markOfflineNodes(timeoutMinutes = 5) {
    const threshold = new Date(Date.now() - timeoutMinutes * 60 * 1000);

    const [affectedCount] = await Node.update(
      { status: "offline" },
      {
        where: {
          status: "online",
          lastHeartbeatAt: { [Op.lt]: threshold },
        },
      }
    );

    return affectedCount;
  }

  /**
   * 审批通过节点
   * @param {string} nodeId - 节点ID
   * @param {string} userId - 审批人ID
   */
  async approve(nodeId, userId) {
    const node = await Node.findByPk(nodeId);
    if (!node) {
      throw new Error(`节点 ${nodeId} 不存在`);
    }

    // 如果尚未生成 registrationToken，则生成一个
    const registrationToken = node.registrationToken || generateRegistrationToken();

    await node.update({
      approvalStatus: "approved",
      approvedAt: new Date(),
      approvedBy: userId,
      registrationToken,
      updatedBy: userId,
    });

    return {
      ...node.toJSON(),
      registrationToken, // 确保返回 token
    };
  }

  /**
   * 拒绝节点注册
   * @param {string} nodeId - 节点ID
   * @param {string} userId - 审批人ID
   */
  async reject(nodeId, userId) {
    const node = await Node.findByPk(nodeId);
    if (!node) {
      throw new Error(`节点 ${nodeId} 不存在`);
    }

    await node.update({
      approvalStatus: "rejected",
      updatedBy: userId,
      status: "offline",
    });

    return node;
  }

  /**
   * 更新部署状态回调（NodeAgent 上报）
   * @param {string} nodeId - 节点ID
   * @param {string} deploymentId - 节点部署记录ID
   * @param {Object} statusData - 状态数据
   */
  async updateDeploymentStatus(nodeId, deploymentId, statusData) {
    const { status, error, startedAt, stoppedAt } = statusData;

    const nodeDeployment = await NodeDeployment.findOne({
      where: { id: deploymentId, nodeId },
    });

    if (!nodeDeployment) {
      throw new Error(`部署记录 ${deploymentId} 不存在`);
    }

    const updateData = { status };
    if (error) {
      updateData.errorMessage = error.message;
      updateData.errorStack = error.stack;
    }
    if (startedAt) updateData.startedAt = new Date(startedAt);
    if (stoppedAt) updateData.stoppedAt = new Date(stoppedAt);

    // 添加部署日志
    const log = {
      time: new Date().toISOString(),
      status,
      message: statusData.message || `状态变更为 ${status}`,
    };
    updateData.deployLog = [...(nodeDeployment.deployLog || []), log].slice(-50);

    await nodeDeployment.update(updateData);

    // 更新节点当前状态指标
    if (status === "running") {
      await Node.update(
        {
          currentDeploymentId: deploymentId,
          currentProjectId: nodeDeployment.projectId,
          currentVersion: nodeDeployment.version,
        },
        { where: { id: nodeId } }
      );
    } else if (status === "stopped" || status === "error") {
      // 只有当没有其他正在运行的工程时，才清空节点的当前工程信息
      const otherRunning = await NodeDeployment.findOne({
        where: {
          nodeId,
          status: "running",
          id: { [Op.ne]: deploymentId },
          deletedAt: null
        }
      });

      if (!otherRunning) {
        await Node.update(
          {
            currentDeploymentId: null,
            currentProjectId: null,
            currentVersion: null,
          },
          { where: { id: nodeId } }
        );
      }
    }

    return nodeDeployment;
  }
}

module.exports = new NodeService();
