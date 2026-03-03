/**
 * 节点服务 - 节点注册、心跳、状态管理
 * @description 处理 NodeAgent 的注册、心跳上报、状态查询等功能
 */
const crypto = require("crypto");
const { Node, NodeDeployment, NodeCommand, Deployment, Project, Tenant, User, Log } = require("../models");
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
 * 生成稳定命令ID（同一记录同一状态同一更新时间 => 同一 commandId）
 * @param {string} type - 命令类型
 * @param {Object} record - 数据记录
 * @returns {string}
 */
function buildCommandId(type, record) {
  const updatedAt = record?.updatedAt ? new Date(record.updatedAt).getTime() : 0;
  const raw = [type, record?.id || "", record?.status || "", updatedAt].join(":");
  return crypto.createHash("sha256").update(raw).digest("hex").slice(0, 24);
}

const TERMINAL_COMMAND_STATUS = ["completed", "failed", "dead_letter"];

/**
 * 规范化 IP 地址（处理 ::ffff: 前缀与本地回环）
 * @param {string} ip
 * @returns {string}
 */
function normalizeIp(ip) {
  if (!ip) return ip;
  if (ip === "::1") return "127.0.0.1";
  if (ip.startsWith("::ffff:")) return ip.slice(7);
  return ip;
}

/**
 * 校验节点注册令牌
 * @param {Object} node - 节点模型
 * @param {string} registrationToken - 请求携带令牌
 */
function ensureNodeTokenValid(node, registrationToken) {
  const expectedToken = String(node?.registrationToken || "").trim();
  const providedToken = String(registrationToken || "").trim();
  if (!expectedToken) {
    throw new Error(`节点 ${node?.id} 缺少注册令牌，请重新审批节点`);
  }
  if (!providedToken || providedToken !== expectedToken) {
    throw new Error(`节点 ${node?.id} 令牌无效`);
  }
}

/**
 * 校验节点名称是否可复用
 * @param {string} tenantId - 租户ID
 * @param {string} nodeName - 节点名称
 * @throws {Error} 当存在未删除同名节点时抛出错误
 */
async function ensureNodeNameReusable(tenantId, nodeName) {
  const activeNode = await Node.findOne({
    where: { tenantId, name: nodeName, deletedAt: null },
  });
  if (activeNode) {
    throw new Error(`节点名称 "${nodeName}" 已存在`);
  }

  const softDeletedNode = await Node.findOne({
    where: { tenantId, name: nodeName },
    paranoid: false,
  });

  // 兼容历史软删除数据：若同名节点已删除，物理清理后允许重新注册
  if (softDeletedNode && softDeletedNode.deletedAt) {
    await softDeletedNode.destroy({ force: true });
  }
}

/**
 * 规范化节点注册错误
 * @param {Error} error - 原始错误
 * @param {string} nodeName - 节点名称
 * @returns {Error}
 */
function normalizeRegisterError(error, nodeName) {
  if (error && error.name === "SequelizeUniqueConstraintError") {
    const message = String(error.message || "").toLowerCase();
    const conflictField = Object.keys(error.fields || {}).join(",").toLowerCase();
    if (message.includes("primary") || conflictField.includes("id")) {
      return new Error("节点ID已存在，请检查当前运维代理机器码");
    }
    return new Error(`节点名称 "${nodeName}" 已存在`);
  }
  return error;
}

/**
 * 规范化节点ID为可落库格式（36位）。
 * 规则：
 * 1) 为空时返回随机 UUID；
 * 2) 长度 <= 36 时直接使用；
 * 3) 长度 > 36 时基于原值生成稳定 UUID（同输入同输出）。
 * @param {string} rawNodeId - 原始节点ID
 * @returns {string}
 */
function normalizeNodeId(rawNodeId) {
  const value = String(rawNodeId || "").trim();
  if (!value) {
    return crypto.randomUUID();
  }
  if (value.length <= 36) {
    return value;
  }

  const hash = crypto.createHash("sha256").update(value).digest("hex");
  return `${hash.slice(0, 8)}-${hash.slice(8, 12)}-${hash.slice(12, 16)}-${hash.slice(16, 20)}-${hash.slice(20, 32)}`;
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
   * @param {string} data.role - 创建者角色
   * @returns {Promise<Object>} 节点信息及注册令牌
   */
  async register(data) {
    const { tenantId, name, agentVersion, ipAddress, port, createdBy, description, role } = data;

    // 检查节点名称可用性（仅未删除同名视为冲突）
    await ensureNodeNameReusable(tenantId, name);

    const id = crypto.randomUUID();

    // 具备运维权限的用户自动审批通过
    const hasOpsPermission = ["OPS_ADMIN", "SYSTEM_ADMIN"].includes(role);
    const approvalStatus = hasOpsPermission ? "approved" : "pending";
    const registrationToken = hasOpsPermission ? generateRegistrationToken() : null;

    let node;
    try {
      node = await Node.create({
        id,
        tenantId,
        name,
        description,
        agentVersion,
        ipAddress: normalizeIp(ipAddress),
        port: port || 8080,
        status: "offline", // 注册后默认为离线，等待审批
        approvalStatus,
        registrationToken,
        lastHeartbeatAt: null,
        approvedAt: hasOpsPermission ? new Date() : null,
        approvedBy: hasOpsPermission ? createdBy : null,
        createdBy,
        updatedBy: createdBy,
      });
    } catch (error) {
      throw normalizeRegisterError(error, name);
    }

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
   * 带用户认证的节点注册
   * @param {Object} data - 注册数据
   * @param {string} data.username - 用户名
   * @param {string} data.password - 密码
   * @param {string} data.nodeId - 节点ID（可选，优先使用 Agent 机器码）
   * @param {string} data.nodeName - 节点名称
   * @param {string} data.nodeDescription - 节点描述
   * @param {string} data.agentVersion - Agent版本
   * @param {string} data.ipAddress - IP地址
   * @param {number} data.port - 端口
   * @param {string} data.mode - 节点模式(online/offline)
   * @returns {Promise<Object>} 注册结果
   */
  async registerWithAuth(data) {
    const {
      username,
      password,
      nodeId,
      nodeName,
      nodeDescription,
      agentVersion,
      ipAddress,
      port,
      mode = "online",
      userAgent,
    } = data;

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
    const hasOpsPermission = ["OPS_ADMIN", "SYSTEM_ADMIN"].includes(user.role);
    const approvalStatus = hasOpsPermission ? "approved" : "pending";
    const registrationToken = generateRegistrationToken();
    const finalNodeId = normalizeNodeId(nodeId);

    const existingNodeByName = await Node.findOne({
      where: { tenantId, name: nodeName, deletedAt: null },
    });

    let node;
    let isResubmitted = false;

    // 同名且已拒绝：允许复用并重新提交流程。
    if (existingNodeByName && existingNodeByName.approvalStatus === "rejected") {
      isResubmitted = true;
      await existingNodeByName.update({
        description: nodeDescription,
        agentVersion: agentVersion || "1.0.0",
        ipAddress: normalizeIp(ipAddress),
        port: port || 8080,
        mode,
        status: "offline",
        approvalStatus,
        registrationToken: hasOpsPermission ? registrationToken : null,
        approvedAt: hasOpsPermission ? new Date() : null,
        approvedBy: hasOpsPermission ? user.id : null,
        registeredBy: user.id,
        updatedBy: user.id,
        lastErrorMessage: null,
        lastErrorAt: null,
        lastHeartbeatAt: null,
      });
      node = existingNodeByName;
    } else {
      // 检查节点名称可用性（仅未删除同名视为冲突）
      await ensureNodeNameReusable(tenantId, nodeName);

      const existingNodeById = await Node.findByPk(finalNodeId, { paranoid: false });
      if (existingNodeById) {
        if (existingNodeById.deletedAt) {
          await existingNodeById.destroy({ force: true });
        } else {
          throw new Error("节点ID已存在，请检查当前运维代理机器码");
        }
      }

      try {
        node = await Node.create({
          id: finalNodeId,
          tenantId,
          name: nodeName,
          description: nodeDescription,
          agentVersion: agentVersion || "1.0.0",
          ipAddress: normalizeIp(ipAddress),
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
      } catch (error) {
        throw normalizeRegisterError(error, nodeName);
      }
    }

    // 记录节点注册日志，供“最近活动”展示。
    await Log.create({
      level: "info",
      message: hasOpsPermission
        ? `${isResubmitted ? "节点重新注册并自动审批通过" : "节点注册并自动审批通过"}，节点：${node.name}，操作者：${user.username}（${user.role}）`
        : `${isResubmitted ? "节点注册申请已重新提交" : "节点注册申请已提交"}，节点：${node.name}，申请人：${user.username}`,
      action: hasOpsPermission ? "node-register.auto-approve" : "node-register.create",
      resource: "node-register",
      resourceId: node.id,
      userId: user.id,
      tenantId,
      ip: normalizeIp(ipAddress) || null,
      userAgent: userAgent || null,
      createdAt: new Date(),
    });

    if (!hasOpsPermission) {
      socketService.broadcastNodePendingRequest(tenantId, {
        nodeId: node.id,
        nodeName: node.name,
        applicant: {
          id: user.id,
          username: user.username,
          role: user.role,
        },
        approvalStatus: node.approvalStatus,
        createdAt: node.createdAt,
      });
    }

    return {
      nodeId: node.id,
      nodeName: node.name,
      approvalStatus: node.approvalStatus,
      registrationToken: hasOpsPermission ? registrationToken : null,
      message: hasOpsPermission
        ? "自动审批通过"
        : isResubmitted
          ? "申请已重新提交，等待管理员审批"
          : "申请已提交，等待管理员审批",
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
  async heartbeat(nodeId, heartbeatData, registrationToken = "") {
    const { agentVersion, metrics, projects, ipAddress } = heartbeatData;

    const node = await Node.findByPk(nodeId);
    if (!node) {
      throw new Error(`节点 ${nodeId} 不存在`);
    }

    // 检查审批状态
    if (node.approvalStatus !== "approved") {
      throw new Error(`节点 ${nodeId} 未通过审批，无法上报心跳`);
    }
    ensureNodeTokenValid(node, registrationToken);
    const previousStatus = node.status;

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
      updateData.ipAddress = normalizeIp(ipAddress);
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
    if (updateData.status !== previousStatus) {
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
    // 兼容旧数据：若存在 pending 部署但命令表无 deploy 命令，自动补一条
    const pendingDeployments = await NodeDeployment.findAll({
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

    for (const pendingDeployment of pendingDeployments) {
      const exists = await NodeCommand.findOne({
        where: {
          deploymentId: pendingDeployment.id,
          type: "deploy",
          status: { [Op.notIn]: TERMINAL_COMMAND_STATUS },
          deletedAt: null,
        },
      });
      if (exists) continue;
      const commandId = crypto.randomUUID();
      await NodeCommand.create({
        id: commandId,
        tenantId: pendingDeployment.tenantId || (await Node.findByPk(nodeId, { attributes: ["tenantId"] })).tenantId,
        nodeId,
        deploymentId: pendingDeployment.id,
        projectId: pendingDeployment.projectId,
        type: "deploy",
        status: "pending",
        payload: {
          commandId,
          deploymentId: pendingDeployment.id,
          projectId: pendingDeployment.projectId,
          artifactUrl: pendingDeployment.deployment?.artifactUrl,
          artifactHash: pendingDeployment.deployment?.artifactHash,
          version: pendingDeployment.deployment?.version || pendingDeployment.version,
          runtimeConfig: pendingDeployment.runtimeConfig || {},
        },
        attempts: 0,
        maxAttempts: 3,
        timeoutSeconds: 30,
        requestedAt: new Date(),
      });
    }

    const queue = await NodeCommand.findAll({
      where: {
        nodeId,
        status: { [Op.in]: ["pending", "issued"] },
        deletedAt: null,
      },
      order: [["requestedAt", "ASC"]],
      limit: 20,
    });

    const commands = [];
    for (const command of queue) {
      if (command.status === "pending") {
        await command.update({
          status: "issued",
          issuedAt: command.issuedAt || new Date(),
        });
      }
      commands.push({
        type: command.type,
        payload: {
          commandId: command.id,
          deploymentId: command.deploymentId,
          projectId: command.projectId,
          issuedAt: command.issuedAt || command.requestedAt,
          ...(command.payload || {}),
        },
      });
    }

    return commands;
  }

  /**
   * 扫描并处理命令超时（重试/死信）
   * @returns {Promise<{retried:number,deadLetter:number}>}
   */
  async processCommandTimeouts() {
    const now = new Date();
    const commands = await NodeCommand.findAll({
      where: {
        status: "issued",
        deletedAt: null,
      },
      order: [["issuedAt", "ASC"]],
      limit: 200,
    });

    let retried = 0;
    let deadLetter = 0;

    for (const command of commands) {
      const timeoutSeconds = Math.max(5, Number(command.timeoutSeconds || 30));
      const issuedAt = command.issuedAt ? new Date(command.issuedAt).getTime() : 0;
      if (!issuedAt) continue;
      if (now.getTime() - issuedAt < timeoutSeconds * 1000) continue;

      const nextAttempts = Number(command.attempts || 0) + 1;
      if (nextAttempts >= Number(command.maxAttempts || 3)) {
        const deadLetterMessage = `命令执行超时（>${timeoutSeconds}s），已进入死信队列`;
        await command.update({
          status: "dead_letter",
          attempts: nextAttempts,
          lastError: deadLetterMessage,
          completedAt: now,
        });
        deadLetter += 1;

        const nodeDeployment = await NodeDeployment.findByPk(command.deploymentId);
        await NodeDeployment.update(
          {
            status: "error",
            errorMessage: `命令 ${command.type} 执行超时，已超过最大重试次数`,
          },
          { where: { id: command.deploymentId } }
        );
        if (nodeDeployment) {
          const tenantId = (await Node.findByPk(nodeDeployment.nodeId, { attributes: ["tenantId"] }))?.tenantId;
          if (tenantId) {
            socketService.broadcastDeployStatus(tenantId, {
              nodeId: nodeDeployment.nodeId,
              deploymentId: nodeDeployment.id,
              projectId: nodeDeployment.projectId,
              status: "error",
              errorMessage: deadLetterMessage,
            });
          }
        }
        continue;
      }

      await command.update({
        status: "pending",
        attempts: nextAttempts,
        issuedAt: null,
        lastError: `命令执行超时（>${timeoutSeconds}s），准备第 ${nextAttempts} 次重试`,
      });
      retried += 1;
    }

    return { retried, deadLetter };
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
      distinct: true,
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
            {
              model: NodeCommand,
              as: "commands",
              attributes: [
                "id",
                "type",
                "status",
                "attempts",
                "maxAttempts",
                "requestedAt",
                "issuedAt",
                "acknowledgedAt",
                "completedAt",
                "lastError",
              ],
              required: false,
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

    await node.destroy({ force: true }); // 物理删除，避免同名唯一约束残留
    return true;
  }

  /**
   * 节点主动下线（Agent 正常停止时调用）
   * @param {string} nodeId - 节点ID
   * @param {Object} data - 下线附加信息
   * @param {string} data.reason - 下线原因
   * @returns {Promise<Object>} 下线后的节点状态
   */
  async offline(nodeId, data = {}, registrationToken = "") {
    const node = await Node.findByPk(nodeId);
    if (!node) {
      throw new Error(`节点 ${nodeId} 不存在`);
    }
    ensureNodeTokenValid(node, registrationToken);

    const previousStatus = node.status;
    const updateData = {
      status: "offline",
      updatedAt: new Date(),
    };
    if (data?.reason) {
      updateData.lastErrorMessage = String(data.reason);
      updateData.lastErrorAt = new Date();
    }

    await node.update(updateData);

    if (previousStatus !== "offline") {
      socketService.broadcastNodeStatus(node.tenantId, nodeId, "offline");
    }

    return {
      nodeId: node.id,
      status: "offline",
    };
  }

  /**
   * 检查并标记离线节点
   * @param {number} timeoutMinutes - 超时分钟数
   * @returns {Promise<number>} 标记为离线的节点数量
   */
  async markOfflineNodes(timeoutMinutes = 5) {
    const threshold = new Date(Date.now() - timeoutMinutes * 60 * 1000);

    const staleNodes = await Node.findAll({
      attributes: ["id", "tenantId"],
      where: {
        status: "online",
        lastHeartbeatAt: { [Op.lt]: threshold },
        deletedAt: null,
      },
    });
    if (staleNodes.length === 0) {
      return 0;
    }

    const nodeIds = staleNodes.map((item) => item.id);
    const [affectedCount] = await Node.update(
      { status: "offline" },
      {
        where: {
          id: { [Op.in]: nodeIds },
          status: "online",
          deletedAt: null,
        },
      }
    );

    // 推送离线状态，确保运维前端实时更新而无需刷新页面。
    staleNodes.forEach((node) => {
      socketService.broadcastNodeStatus(node.tenantId, node.id, "offline");
    });

    return affectedCount;
  }

  /**
   * 计算下一次离线检测的建议延迟（毫秒）
   * @param {number} timeoutSeconds - 心跳超时秒数
   * @param {number} fallbackIntervalMs - 回退检测间隔
   * @returns {Promise<number>} 建议延迟毫秒
   */
  async getNextOfflineCheckDelayMs(timeoutSeconds = 30, fallbackIntervalMs = 5000) {
    const oldestOnline = await Node.findOne({
      attributes: ["lastHeartbeatAt"],
      where: {
        status: "online",
        deletedAt: null,
      },
      order: [["lastHeartbeatAt", "ASC"]],
    });

    // 没有在线节点时，降低检测频率
    if (!oldestOnline || !oldestOnline.lastHeartbeatAt) {
      return Math.max(fallbackIntervalMs * 6, 30000);
    }

    const expireAt = new Date(oldestOnline.lastHeartbeatAt).getTime() + timeoutSeconds * 1000;
    const now = Date.now();
    const delay = expireAt - now;

    // 保障边界：最短 1 秒，最长 30 秒
    if (delay <= 1000) return 1000;
    if (delay >= 30000) return 30000;
    return delay;
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

    const updatedNode = node.toJSON();
    delete updatedNode.registrationToken;
    return updatedNode;
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
  async updateDeploymentStatus(nodeId, deploymentId, statusData, registrationToken = "") {
    const { status, error, startedAt, stoppedAt } = statusData;

    const node = await Node.findByPk(nodeId);
    if (!node) {
      throw new Error(`节点 ${nodeId} 不存在`);
    }
    ensureNodeTokenValid(node, registrationToken);

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

    // 命令回执：将该部署下最近一条未完成命令标记为完成/失败
    const activeCommand = await NodeCommand.findOne({
      where: {
        deploymentId,
        nodeId,
        status: { [Op.in]: ["pending", "issued", "acknowledged"] },
        deletedAt: null,
      },
      order: [["updatedAt", "DESC"]],
    });
    let commandAck = null;
    if (activeCommand) {
      const isFailure = status === "error";
      const acknowledgedAt = new Date();
      const completedAt = new Date();
      const commandStatus = isFailure ? "failed" : "completed";
      const commandError = isFailure
        ? (error?.message || statusData?.message || "节点执行失败")
        : null;
      await activeCommand.update({
        status: commandStatus,
        acknowledgedAt,
        completedAt,
        lastError: commandError,
      });
      commandAck = {
        id: activeCommand.id,
        status: commandStatus,
        acknowledgedAt,
        completedAt,
        lastError: commandError,
      };
    }

    socketService.broadcastDeployStatus(node.tenantId, {
      nodeId,
      deploymentId: nodeDeployment.id,
      projectId: nodeDeployment.projectId,
      status,
      startedAt: updateData.startedAt || nodeDeployment.startedAt || null,
      stoppedAt: updateData.stoppedAt || nodeDeployment.stoppedAt || null,
      errorMessage: updateData.errorMessage || "",
      commandAck,
    });

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
