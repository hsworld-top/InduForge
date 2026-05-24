/**
 * 部署服务 - 部署管理、启停、回滚
 * @description 处理工程部署到节点的相关操作，支持DEV/RELEASE模式
 */
const crypto = require('crypto')
const { Deployment, NodeDeployment, NodeCommand, Node, Project, User } = require('../models')
const { Op } = require('sequelize')
const socketService = require('./socketService')
const AppError = require('../utils/AppError')
const ErrorCodes = require('../constants/errorCodes')

/**
 * 部署服务类
 */
class DeploymentService {
  /**
   * 获取部署下仍在执行中的命令
   * @param {Object} nodeDeployment - 节点部署记录
   * @returns {Promise<Object|null>}
   */
  async getInFlightCommand(nodeDeployment) {
    return NodeCommand.findOne({
      where: {
        deploymentId: nodeDeployment.id,
        nodeId: nodeDeployment.nodeId,
        status: { [Op.in]: ['pending', 'issued', 'acknowledged'] },
        deletedAt: null,
      },
      order: [['updatedAt', 'DESC']],
    })
  }

  /**
   * 校验运行时指令是否可下发
   * @param {Object} nodeDeployment - 节点部署记录
   * @param {"start"|"stop"|"restart"} commandType - 指令类型
   * @returns {Promise<void>}
   */
  async ensureRuntimeCommandAllowed(nodeDeployment, commandType) {
    const inFlightCommand = await this.getInFlightCommand(nodeDeployment)
    if (inFlightCommand) {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: `当前存在未完成的 ${inFlightCommand.type} 指令，请稍后重试`,
      })
    }

    const currentStatus = nodeDeployment.status
    if (commandType === 'start') {
      if (!['stopped', 'error'].includes(currentStatus)) {
        throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
          message: `当前状态为 ${currentStatus}，仅 stopped/error 状态可启动`,
        })
      }
      return
    }

    if (commandType === 'stop') {
      if (currentStatus !== 'running') {
        throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
          message: `当前状态为 ${currentStatus}，仅 running 状态可停止`,
        })
      }
      return
    }

    if (commandType === 'restart') {
      if (currentStatus !== 'running') {
        throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
          message: `当前状态为 ${currentStatus}，仅 running 状态可重启`,
        })
      }
    }
  }

  /**
   * 清理同节点同工程的软删除部署记录，避免唯一索引冲突。
   * @param {string} nodeId - 节点ID
   * @param {string} projectId - 工程ID
   * @returns {Promise<void>}
   */
  async purgeSoftDeletedDeployments(nodeId, projectId) {
    const staleDeployments = await NodeDeployment.findAll({
      where: {
        nodeId,
        projectId,
      },
      paranoid: false,
    })

    for (const item of staleDeployments) {
      if (item.deletedAt) {
        await item.destroy({ force: true })
      }
    }
  }

  /**
   * 创建 deploy 命令记录，避免首次部署时命令时间线为空。
   * @param {Object} params - 参数
   * @param {string} params.tenantId - 租户ID
   * @param {Object} params.nodeDeployment - 节点部署记录
   * @param {Object} params.payload - 命令负载
   * @returns {Promise<void>}
   */
  async createDeployCommandRecord({ tenantId, nodeDeployment, payload = {} }) {
    const commandId = crypto.randomUUID()
    await NodeCommand.create({
      id: commandId,
      tenantId,
      nodeId: nodeDeployment.nodeId,
      deploymentId: nodeDeployment.id,
      projectId: nodeDeployment.projectId,
      type: 'deploy',
      status: 'pending',
      payload: {
        commandId,
        deploymentId: nodeDeployment.id,
        projectId: nodeDeployment.projectId,
        version: nodeDeployment.version,
        ...(payload || {}),
      },
      attempts: 0,
      maxAttempts: 3,
      timeoutSeconds: 30,
      requestedAt: new Date(),
    })
  }

  /**
   * 获取工程基础信息
   * @param {string} projectId - 工程ID
   * @returns {Promise<Object>} 工程信息
   */
  async getProjectBase(projectId) {
    const project = await Project.findByPk(projectId, {
      attributes: ['id', 'tenantId', 'name'],
    })
    if (!project) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '工程不存在' })
    }
    return project
  }

  /**
   * 获取或创建 DEV 模式源发布记录（无需业务版本号）
   * @param {string} projectId - 工程ID
   * @param {string} deployedBy - 操作者ID
   * @returns {Promise<Object>} DEV 源发布记录
   */
  async ensureDevSourceDeployment(projectId, deployedBy) {
    const project = await this.getProjectBase(projectId)
    const devVersion = '__DEV__'

    const existing = await Deployment.findOne({
      where: {
        projectId,
        version: devVersion,
        mode: 'DEV',
        deletedAt: null,
      },
    })
    if (existing) {
      if (existing.status !== 'success') {
        await existing.update({
          status: 'success',
          completedAt: existing.completedAt || new Date(),
        })
      }
      return existing
    }

    const id = crypto.randomUUID()
    return Deployment.create({
      id,
      projectId,
      tenantId: project.tenantId,
      version: devVersion,
      name: 'DEV 最新工程',
      description: 'DEV模式源记录（无需发布版本）',
      type: 'development',
      mode: 'DEV',
      status: 'success',
      deployedBy,
      startedAt: new Date(),
      completedAt: new Date(),
    })
  }
  /**
   * 根据发布记录ID获取工程ID
   * @param {string} deploymentId - 发布记录ID
   * @returns {Promise<string>} 工程ID
   */
  async getProjectIdByDeploymentId(deploymentId) {
    const deployment = await Deployment.findByPk(deploymentId, {
      attributes: ['id', 'projectId'],
    })
    if (!deployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '发布版本不存在' })
    }
    return deployment.projectId
  }

  /**
   * 根据节点部署记录ID获取工程ID
   * @param {string} nodeDeploymentId - 节点部署记录ID
   * @returns {Promise<string>} 工程ID
   */
  async getProjectIdByNodeDeploymentId(nodeDeploymentId) {
    const nodeDeployment = await NodeDeployment.findByPk(nodeDeploymentId, {
      attributes: ['id', 'projectId'],
    })
    if (!nodeDeployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '部署记录不存在' })
    }
    return nodeDeployment.projectId
  }

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
      type = 'development',
      mode = 'RELEASE',
      deployedBy,
    } = data

    // RELEASE模式检查版本号是否重复
    if (mode === 'RELEASE') {
      const existing = await Deployment.findOne({
        where: { projectId, version, deletedAt: null },
      })
      if (existing) {
        throw new AppError(ErrorCodes.RESOURCE_ALREADY_EXISTS, 409, {
          message: `版本 ${version} 已存在`,
        })
      }
    }

    const id = crypto.randomUUID()
    const deployment = await Deployment.create({
      id,
      projectId,
      tenantId,
      version,
      name: name || `v${version}`,
      description,
      type,
      mode,
      status: 'pending',
      deployedBy,
      startedAt: new Date(),
    })

    return deployment
  }

  /**
   * 更新发布状态（发布流水线调用）
   * @param {string} deploymentId - 发布版本ID
   * @param {Object} data - 更新数据
   */
  async updateDeploymentStatus(deploymentId, data) {
    const deployment = await Deployment.findByPk(deploymentId)
    if (!deployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: `发布版本 ${deploymentId} 不存在`,
      })
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
    } = data

    const updateData = {}
    if (status) updateData.status = status
    if (artifactUrl) updateData.artifactUrl = artifactUrl
    if (artifactHash) updateData.artifactHash = artifactHash
    if (artifactSize) updateData.artifactSize = artifactSize
    if (snapshotUrl) updateData.snapshotUrl = snapshotUrl
    if (snapshotHash) updateData.snapshotHash = snapshotHash
    if (manifest) updateData.manifest = manifest
    if (pageCount !== undefined) updateData.pageCount = pageCount
    if (componentCount !== undefined) updateData.componentCount = componentCount
    if (datapointCount !== undefined) updateData.datapointCount = datapointCount
    if (errorMessage) updateData.errorMessage = errorMessage
    if (buildLog) {
      updateData.buildLog = [...(deployment.buildLog || []), ...buildLog].slice(-100)
    }

    if (status === 'success' || status === 'failed') {
      updateData.completedAt = new Date()
    }

    await deployment.update(updateData)
    return deployment
  }

  /**
   * 获取工程的发布版本列表
   * @param {string} projectId - 工程ID
   * @param {Object} options - 查询选项
   */
  async listByProject(projectId, options = {}) {
    const { page = 1, pageSize = 20, status, type, mode } = options
    const offset = (page - 1) * pageSize

    const where = { projectId }
    if (status) where.status = status
    if (type) where.type = type
    if (mode) where.mode = mode

    const { rows, count } = await Deployment.findAndCountAll({
      where,
      include: [
        {
          model: User,
          as: 'deployer',
          attributes: ['id', 'username'],
        },
      ],
      order: [['createdAt', 'DESC']],
      limit: pageSize,
      offset,
    })

    return {
      items: rows,
      total: count,
      page,
      pageSize,
      totalPages: Math.ceil(count / pageSize),
    }
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
          as: 'project',
          attributes: ['id', 'name', 'code'],
        },
        {
          model: User,
          as: 'deployer',
          attributes: ['id', 'username'],
        },
        {
          model: NodeDeployment,
          as: 'nodeDeployments',
          include: [
            {
              model: Node,
              as: 'node',
              attributes: ['id', 'name', 'status', 'ipAddress'],
            },
          ],
        },
      ],
    })

    if (!deployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, {
        message: `发布版本 ${deploymentId} 不存在`,
      })
    }

    return deployment
  }

  /**
   * DEV模式直接部署（无需版本号，连接开发库）
   * @param {string} deploymentId - 部署记录ID（DEV模式下是源版本）
   * @param {string} nodeId - 节点ID
   * @param {string} deployedBy - 部署者ID
   */
  async deployDevMode(deploymentId, nodeId, deployedBy) {
    const sourceDeployment = await Deployment.findByPk(deploymentId)
    if (!sourceDeployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '源部署记录不存在' })
    }
    if (sourceDeployment.status !== 'success') {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '仅支持使用构建成功的版本进行DEV部署',
      })
    }

    // 检查节点是否已有部署
    const existingDeployment = await NodeDeployment.findOne({
      where: {
        nodeId,
        projectId: sourceDeployment.projectId,
        deletedAt: null,
      },
    })

    if (existingDeployment) {
      // 检查当前模式
      if (existingDeployment.mode === 'RELEASE') {
        throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
          message: '该节点当前处于RELEASE模式，请先撤销部署后再切换到DEV模式',
        })
      }

      // 已经是DEV模式，直接更新状态为运行
      if (existingDeployment.mode === 'DEV') {
        await existingDeployment.update({
          status: 'running',
          deployLog: [
            ...(existingDeployment.deployLog || []),
            {
              time: new Date().toISOString(),
              status: 'running',
              message: 'DEV模式重新部署',
            },
          ],
        })

        // 更新节点当前工程信息
        await Node.update(
          {
            currentProjectId: sourceDeployment.projectId,
            currentVersion: sourceDeployment.version,
            currentDeploymentId: existingDeployment.id,
          },
          { where: { id: nodeId } },
        )

        return existingDeployment
      }
    }

    // 清理软删除残留，避免唯一索引冲突后无法重新部署
    await this.purgeSoftDeletedDeployments(nodeId, sourceDeployment.projectId)

    // 创建新的DEV模式部署记录
    const id = crypto.randomUUID()
    const nodeDeployment = await NodeDeployment.create({
      id,
      nodeId,
      deploymentId: sourceDeployment.id,
      projectId: sourceDeployment.projectId,
      version: sourceDeployment.version, // 使用源版本号
      mode: 'DEV',
      status: 'pending',
      runtimeConfig: { syncMode: 'development_database' },
      deployedBy,
      deployLog: [
        {
          time: new Date().toISOString(),
          status: 'pending',
          message: 'DEV模式部署任务已创建',
        },
      ],
    })

    await this.createDeployCommandRecord({
      tenantId: sourceDeployment.tenantId,
      nodeDeployment,
      payload: {
        runtimeConfig: { syncMode: 'development_database' },
      },
    })

    // 更新节点当前工程信息
    await Node.update(
      {
        currentProjectId: sourceDeployment.projectId,
        currentVersion: sourceDeployment.version,
        currentDeploymentId: id,
      },
      { where: { id: nodeId } },
    )

    return nodeDeployment
  }

  /**
   * RELEASE模式发布部署（需要版本号）
   * @param {string} deploymentId - 发布版本ID
   * @param {string} nodeId - 节点ID
   * @param {Object} runtimeConfig - 运行时配置
   * @param {string} deployedBy - 部署者ID
   */
  async deployReleaseMode(deploymentId, nodeId, runtimeConfig, deployedBy) {
    const deployment = await Deployment.findByPk(deploymentId)
    if (!deployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '发布版本不存在' })
    }
    if (deployment.mode !== 'RELEASE') {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '仅支持RELEASE模式的发布版本进行部署',
      })
    }
    if (deployment.status !== 'success') {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '仅支持部署构建成功的发布版本',
      })
    }

    // 检查节点当前是否已有该工程部署记录（数据库层对 nodeId+projectId 做了唯一约束）
    const existingDeployment = await NodeDeployment.findOne({
      where: {
        nodeId,
        projectId: deployment.projectId,
        deletedAt: null,
      },
    })

    let nodeDeployment
    if (existingDeployment) {
      const nextLog = [...(existingDeployment.deployLog || [])]
      if (existingDeployment.mode === 'DEV') {
        nextLog.push({
          time: new Date().toISOString(),
          status: 'stopped',
          message: '切换到RELEASE模式，停止DEV实例',
        })
      }
      nextLog.push({
        time: new Date().toISOString(),
        status: 'pending',
        message: 'RELEASE模式部署任务已创建',
      })

      await existingDeployment.update({
        deploymentId,
        version: deployment.version,
        mode: 'RELEASE',
        status: 'pending',
        runtimeConfig: { ...runtimeConfig, syncMode: 'local_database' },
        deployedBy,
        stoppedAt: null,
        errorMessage: null,
        errorStack: null,
        deployLog: nextLog.slice(-50),
      })
      nodeDeployment = existingDeployment
    } else {
      // 清理软删除残留，避免唯一索引冲突后无法重新部署
      await this.purgeSoftDeletedDeployments(nodeId, deployment.projectId)

      // 创建新的RELEASE部署记录
      const id = crypto.randomUUID()
      nodeDeployment = await NodeDeployment.create({
        id,
        nodeId,
        deploymentId,
        projectId: deployment.projectId,
        version: deployment.version,
        mode: 'RELEASE',
        status: 'pending',
        runtimeConfig: { ...runtimeConfig, syncMode: 'local_database' },
        deployedBy,
        deployLog: [
          {
            time: new Date().toISOString(),
            status: 'pending',
            message: 'RELEASE模式部署任务已创建',
          },
        ],
      })
    }

    await this.createDeployCommandRecord({
      tenantId: deployment.tenantId,
      nodeDeployment,
      payload: {
        runtimeConfig: { ...runtimeConfig, syncMode: 'local_database' },
      },
    })

    // 更新节点当前工程信息
    await Node.update(
      {
        currentProjectId: deployment.projectId,
        currentVersion: deployment.version,
        currentDeploymentId: nodeDeployment.id,
      },
      { where: { id: nodeId } },
    )

    return nodeDeployment
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
    const results = []

    for (const nodeId of nodeIds) {
      try {
        let result
        if (mode === 'DEV') {
          result = await this.deployDevMode(deploymentId, nodeId, deployedBy)
        } else {
          result = await this.deployReleaseMode(deploymentId, nodeId, runtimeConfig, deployedBy)
        }
        results.push({ nodeId, success: true, deploymentId: result.id })
      } catch (error) {
        const detail =
          Array.isArray(error?.errors) && error.errors.length > 0
            ? error.errors.map((item) => item.message).join('; ')
            : ''
        const errorMessage = detail ? `${error.message}: ${detail}` : error.message
        results.push({ nodeId, success: false, error: errorMessage })
      }
    }

    return results
  }

  /**
   * 部署到节点（兼容旧接口，默认RELEASE模式）
   * @param {string} deploymentId - 发布版本ID
   * @param {string} nodeId - 节点ID
   * @param {Object} runtimeConfig - 运行时配置
   * @param {string} deployedBy - 部署操作者ID
   */
  async deployToNode(deploymentId, nodeId, runtimeConfig, deployedBy) {
    return this.deployReleaseMode(deploymentId, nodeId, runtimeConfig, deployedBy)
  }

  /**
   * 下发节点运行控制命令（通过心跳通道执行）
   * @param {Object} nodeDeployment - 节点部署记录
   * @param {string} commandType - 命令类型 start|stop|restart
   * @param {string} expectedStatus - 预期状态（用于界面即时反馈）
   * @returns {Promise<Object>} 更新后的记录
   */
  async enqueueRuntimeCommand(nodeDeployment, commandType, expectedStatus, commandMessage = '') {
    const commandId = crypto.randomUUID()
    const requestedAt = new Date()

    const updateData = {
      deployLog: [
        ...(nodeDeployment.deployLog || []),
        {
          time: requestedAt.toISOString(),
          status: expectedStatus || nodeDeployment.status,
          message: commandMessage || `已下发${commandType}指令，等待节点执行`,
        },
      ],
    }
    if (expectedStatus) {
      updateData.status = expectedStatus
    }
    await nodeDeployment.update(updateData)

    const node = await Node.findByPk(nodeDeployment.nodeId, {
      attributes: ['id', 'tenantId'],
    })
    if (!node) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '节点不存在' })
    }

    await NodeCommand.create({
      id: commandId,
      tenantId: node.tenantId,
      nodeId: nodeDeployment.nodeId,
      deploymentId: nodeDeployment.id,
      projectId: nodeDeployment.projectId,
      type: commandType,
      status: 'pending',
      payload: {
        commandId,
        deploymentId: nodeDeployment.id,
        projectId: nodeDeployment.projectId,
        version: nodeDeployment.version,
      },
      attempts: 0,
      maxAttempts: 3,
      timeoutSeconds: 30,
      requestedAt,
    })

    return nodeDeployment
  }

  /**
   * 启动工程
   * @param {string} nodeDeploymentId - 节点部署记录ID
   */
  async start(nodeDeploymentId) {
    const nodeDeployment = await NodeDeployment.findByPk(nodeDeploymentId)
    if (!nodeDeployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '部署记录不存在' })
    }
    await this.ensureRuntimeCommandAllowed(nodeDeployment, 'start')
    return this.enqueueRuntimeCommand(
      nodeDeployment,
      'start',
      'deploying',
      '已下发启动指令，等待节点执行',
    )
  }

  /**
   * 停止工程
   * @param {string} nodeDeploymentId - 节点部署记录ID
   */
  async stop(nodeDeploymentId) {
    const nodeDeployment = await NodeDeployment.findByPk(nodeDeploymentId)
    if (!nodeDeployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '部署记录不存在' })
    }
    await this.ensureRuntimeCommandAllowed(nodeDeployment, 'stop')
    return this.enqueueRuntimeCommand(nodeDeployment, 'stop', null, '已下发停止指令，等待节点执行')
  }

  /**
   * 重启工程
   * @param {string} nodeDeploymentId - 节点部署记录ID
   */
  async restart(nodeDeploymentId) {
    const nodeDeployment = await NodeDeployment.findByPk(nodeDeploymentId)
    if (!nodeDeployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '部署记录不存在' })
    }
    await this.ensureRuntimeCommandAllowed(nodeDeployment, 'restart')
    return this.enqueueRuntimeCommand(
      nodeDeployment,
      'restart',
      null,
      '已下发重启指令，等待节点执行',
    )
  }

  /**
   * 撤销部署（释放节点资源）
   * @param {string} nodeDeploymentId - 节点部署记录ID
   */
  async undeploy(nodeDeploymentId) {
    const nodeDeployment = await NodeDeployment.findByPk(nodeDeploymentId)
    if (!nodeDeployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '部署记录不存在' })
    }

    const now = new Date()
    const runtimeActiveStatuses = new Set(['pending', 'deploying', 'running'])
    const shouldStopRuntime = runtimeActiveStatuses.has(nodeDeployment.status)
    const undeployMessage = shouldStopRuntime
      ? '撤销部署：已停止运行并释放资源'
      : '撤销部署：已释放资源'

    await nodeDeployment.update({
      status: 'stopped',
      stoppedAt: now,
      deployLog: [
        ...(nodeDeployment.deployLog || []),
        {
          time: now.toISOString(),
          status: 'stopped',
          message: undeployMessage,
        },
      ],
    })

    // 仅当该部署正作为节点当前部署时才清除指针，避免误清理其他工程
    await Node.update(
      {
        currentProjectId: null,
        currentVersion: null,
        currentDeploymentId: null,
      },
      { where: { id: nodeDeployment.nodeId, currentDeploymentId: nodeDeployment.id } },
    )

    // 撤销部署后直接硬删除关系，避免唯一索引阻塞后续重新部署
    await nodeDeployment.destroy({ force: true })

    const node = await Node.findByPk(nodeDeployment.nodeId, {
      attributes: ['id', 'tenantId'],
    })
    if (node?.tenantId) {
      socketService.broadcastDeployStatus(node.tenantId, {
        nodeId: nodeDeployment.nodeId,
        projectId: nodeDeployment.projectId,
        deploymentId: nodeDeployment.id,
        status: 'undeployed',
        removed: true,
        stoppedAt: now.toISOString(),
      })
    }

    return nodeDeployment
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
        status: 'running',
        deletedAt: null,
      },
    })

    return deployment ? deployment.mode : null
  }

  /**
   * 停止节点上的工程（旧接口兼容）
   * @param {string} nodeDeploymentId - 节点部署记录ID
   */
  async stopNodeDeployment(nodeDeploymentId) {
    return this.stop(nodeDeploymentId)
  }

  /**
   * 回滚到指定版本（仅RELEASE模式）
   * @param {string} nodeId - 节点ID
   * @param {string} deploymentId - 目标发布版本ID
   * @param {string} deployedBy - 操作者ID
   */
  async rollback(nodeId, deploymentId, deployedBy) {
    const targetDeployment = await Deployment.findByPk(deploymentId, {
      attributes: ['id', 'mode', 'status'],
    })
    if (!targetDeployment) {
      throw new AppError(ErrorCodes.RESOURCE_NOT_FOUND, 404, { message: '回滚目标版本不存在' })
    }
    if (targetDeployment.mode !== 'RELEASE') {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '仅支持回滚到RELEASE模式版本',
      })
    }
    if (targetDeployment.status !== 'success') {
      throw new AppError(ErrorCodes.VALIDATION_FAILED, 400, {
        message: '仅支持回滚到构建成功的版本',
      })
    }

    // 回滚仅作用于RELEASE模式
    return this.deployReleaseMode(deploymentId, nodeId, {}, deployedBy)
  }

  /**
   * DEV 模式按工程直接部署（无需版本）
   * @param {string} projectId - 工程ID
   * @param {Array<string>} nodeIds - 节点ID列表
   * @param {string} deployedBy - 操作者ID
   * @returns {Promise<Array>} 部署结果
   */
  async deployDevToNodesByProject(projectId, nodeIds, deployedBy) {
    const devSource = await this.ensureDevSourceDeployment(projectId, deployedBy)
    return this.deployToNodes(devSource.id, nodeIds, 'DEV', {}, deployedBy)
  }

  /**
   * 获取节点的部署历史
   * @param {string} nodeId - 节点ID
   * @param {Object} options - 查询选项
   */
  async getNodeDeploymentHistory(nodeId, options = {}) {
    const { page = 1, pageSize = 20 } = options
    const offset = (page - 1) * pageSize

    const { rows, count } = await NodeDeployment.findAndCountAll({
      where: { nodeId },
      include: [
        {
          model: Deployment,
          as: 'deployment',
          attributes: ['id', 'version', 'name', 'type', 'mode'],
        },
        {
          model: Project,
          as: 'project',
          attributes: ['id', 'name'],
        },
        {
          model: User,
          as: 'deployer',
          attributes: ['id', 'username'],
        },
      ],
      order: [['createdAt', 'DESC']],
      limit: pageSize,
      offset,
    })

    return {
      items: rows,
      total: count,
      page,
      pageSize,
      totalPages: Math.ceil(count / pageSize),
    }
  }

  /**
   * 获取工程在各节点上的部署关系
   * @param {string} projectId - 工程ID
   * @returns {Promise<Array>} 节点部署列表
   */
  async listNodeDeploymentsByProject(projectId) {
    return NodeDeployment.findAll({
      where: { projectId, deletedAt: null },
      include: [
        {
          model: Node,
          as: 'node',
          attributes: ['id', 'name', 'status', 'ipAddress', 'port'],
        },
        {
          model: Deployment,
          as: 'deployment',
          attributes: ['id', 'version', 'name', 'type', 'mode'],
        },
        {
          model: Project,
          as: 'project',
          attributes: ['id', 'name'],
        },
      ],
      order: [['updatedAt', 'DESC']],
    })
  }
}

module.exports = new DeploymentService()
