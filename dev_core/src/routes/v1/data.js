const express = require("express");
const router = express.Router();
const { authenticateToken } = require("../../middlewares/auth");
const dataConnectionService = require("../../services/dataConnectionService");
const dataQueryService = require("../../services/dataQueryService");
const mqttConnectionController = require("../../controllers/mqttConnectionController");
const mqttTagGroupController = require("../../controllers/mqttTagGroupController");
const mqttTagController = require("../../controllers/mqttTagController");
const AppError = require("../../utils/AppError");
const ErrorCodes = require("../../constants/errorCodes");

/**
 * 权限检查中间件
 * 检查用户是否有权限访问指定工程
 */
async function ensureProjectAccess(req, projectId) {
  if (req.user.role === "SUPER_ADMIN" || req.user.role === "SYSTEM_ADMIN") {
    return true;
  }
  const userProjects = await req.user.getProjects();
  const projectIds = userProjects.map((p) => p.id);
  return projectIds.includes(projectId);
}

/**
 * 权限检查包装器
 */
async function checkProjectAccess(req, projectId) {
  const hasAccess = await ensureProjectAccess(req, projectId);
  if (!hasAccess) {
    throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
      message: "无权访问此工程",
    });
  }
}

// ===========================================
// 数据连接相关API
// ===========================================

/**
 * 获取数据连接列表
 * GET /api/v1/projects/:projectId/connections
 */
router.get(
  "/projects/:projectId/connections",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);

      const { page, limit, type, status } = req.query;
      const result = await dataConnectionService.getConnections(projectId, {
        page,
        limit,
        type,
        status,
      });

      res.json({
        success: true,
        data: result,
      });
    } catch (error) {
      console.error("获取数据连接列表失败:", error);
      next(error);
    }
  }
);

/**
 * 创建数据连接
 * POST /api/v1/projects/:projectId/connections
 */
router.post(
  "/projects/:projectId/connections",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);

      const connection = await dataConnectionService.createConnection(
        projectId,
        req.body,
        req.user.id
      );

      res.status(201).json({
        success: true,
        message: "数据连接创建成功",
        data: connection,
      });
    } catch (error) {
      console.error("创建数据连接失败:", error);
      next(error);
    }
  }
);

/**
 * 更新数据连接
 * PUT /api/v1/projects/:projectId/connections/:connectionId
 */
router.put(
  "/projects/:projectId/connections/:connectionId",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId, connectionId } = req.params;
      await checkProjectAccess(req, projectId);

      const connection = await dataConnectionService.updateConnection(
        projectId,
        connectionId,
        req.body,
        req.user.id
      );

      res.json({
        success: true,
        message: "数据连接更新成功",
        data: connection,
      });
    } catch (error) {
      console.error("更新数据连接失败:", error);
      next(error);
    }
  }
);

/**
 * 删除数据连接
 * DELETE /api/v1/projects/:projectId/connections/:connectionId
 */
router.delete(
  "/projects/:projectId/connections/:connectionId",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId, connectionId } = req.params;
      await checkProjectAccess(req, projectId);

      await dataConnectionService.deleteConnection(projectId, connectionId);

      res.json({
        success: true,
        message: "数据连接删除成功",
      });
    } catch (error) {
      console.error("删除数据连接失败:", error);
      next(error);
    }
  }
);

/**
 * 更新连接状态
 * PATCH /api/v1/projects/:projectId/connections/:connectionId/status
 */
router.patch(
  "/projects/:projectId/connections/:connectionId/status",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId, connectionId } = req.params;
      await checkProjectAccess(req, projectId);

      const { status } = req.body;

      if (!status) {
        return res.status(400).json({
          success: false,
          message: "缺少状态参数",
        });
      }

      const connection = await dataConnectionService.updateConnectionStatus(
        projectId,
        connectionId,
        status
      );

      res.json({
        success: true,
        message: "连接状态更新成功",
        data: connection,
      });
    } catch (error) {
      console.error("更新连接状态失败:", error);
      next(error);
    }
  }
);

/**
 * 测试数据连接
 * POST /api/v1/projects/:projectId/connections/test
 */
router.post(
  "/projects/:projectId/connections/test",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);

      const { type, config } = req.body;

      if (!type || !config) {
        return res.status(400).json({
          success: false,
          message: "缺少连接类型或配置",
        });
      }

      await dataConnectionService.testConnection(type, config);

      res.json({
        success: true,
        message: "数据库连接成功",
      });
    } catch (error) {
      console.error("测试数据连接失败:", error);
      next(error);
    }
  }
);

/**
 * 获取关系数据库的表列表
 * GET /api/v1/projects/:projectId/connections/:connectionId/tables
 */
router.get(
  "/projects/:projectId/connections/:connectionId/tables",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId, connectionId } = req.params;
      await checkProjectAccess(req, projectId);

      const tables = await dataConnectionService.getTables(
        projectId,
        connectionId
      );

      res.json({
        success: true,
        data: { tables },
      });
    } catch (error) {
      console.error("获取表列表失败:", error);
      next(error);
    }
  }
);

/**
 * 直接执行SQL查询
 * POST /api/v1/projects/:projectId/connections/:connectionId/execute-sql
 */
router.post(
  "/projects/:projectId/connections/:connectionId/execute-sql",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId, connectionId } = req.params;
      await checkProjectAccess(req, projectId);

      const { sql, parameters = [] } = req.body;

      const result = await dataConnectionService.executeSql(
        projectId,
        connectionId,
        sql,
        parameters
      );

      res.json({
        success: true,
        data: result.data,
        executionTime: result.executionTime,
      });
    } catch (error) {
      console.error("执行SQL查询失败:", error);
      next(error);
    }
  }
);

/**
 * 获取指定表的数据
 * GET /api/v1/projects/:projectId/connections/:connectionId/tables/:tableName/data
 */
router.get(
  "/projects/:projectId/connections/:connectionId/tables/:tableName/data",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId, connectionId, tableName } = req.params;
      await checkProjectAccess(req, projectId);

      const { page = 1, limit = 100 } = req.query;

      const result = await dataConnectionService.getTableData(
        projectId,
        connectionId,
        tableName,
        { page, limit }
      );

      res.json({
        success: true,
        data: result,
      });
    } catch (error) {
      console.error("获取表数据失败:", error);
      next(error);
    }
  }
);

/**
 * 获取表结构信息
 * GET /api/v1/projects/:projectId/connections/:connectionId/tables/:tableName/structure
 */
router.get(
  "/projects/:projectId/connections/:connectionId/tables/:tableName/structure",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId, connectionId, tableName } = req.params;
      await checkProjectAccess(req, projectId);

      const result = await dataConnectionService.getTableStructure(
        projectId,
        connectionId,
        tableName
      );

      res.json({
        success: true,
        data: result,
      });
    } catch (error) {
      console.error("获取表结构失败:", error);
      next(error);
    }
  }
);

// ===========================================
// 数据查询相关API
// ===========================================

/**
 * 获取数据查询列表
 * GET /api/v1/projects/:projectId/queries
 */
router.get(
  "/projects/:projectId/queries",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);

      const { page, limit, connectionId, queryType, isActive } = req.query;

      const result = await dataQueryService.getQueries(projectId, {
        page,
        limit,
        connectionId,
        queryType,
        isActive,
      });

      res.json({
        success: true,
        data: result,
      });
    } catch (error) {
      console.error("获取数据查询列表失败:", error);
      next(error);
    }
  }
);

/**
 * 创建数据查询
 * POST /api/v1/projects/:projectId/queries
 */
router.post(
  "/projects/:projectId/queries",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);

      const query = await dataQueryService.createQuery(
        projectId,
        req.body,
        req.user.id
      );

      res.status(201).json({
        success: true,
        message: "数据查询创建成功",
        data: query,
      });
    } catch (error) {
      console.error("创建数据查询失败:", error);
      next(error);
    }
  }
);

/**
 * 执行数据查询
 * POST /api/v1/queries/:id/execute
 */
router.post(
  "/queries/:id/execute",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { id } = req.params;
      const { parameters = {} } = req.body;

      // 权限检查：先获取查询，检查工程权限
      const { DataQuery } = require("../../models");
      const query = await DataQuery.findByPk(id);

      if (query) {
        await checkProjectAccess(req, query.projectId);
      }

      const result = await dataQueryService.executeQuery(
        id,
        parameters,
        req.user.id
      );

      res.json({
        success: true,
        data: result.data,
        executionTime: result.executionTime,
      });
    } catch (error) {
      console.error("执行数据查询失败:", error);
      next(error);
    }
  }
);

/**
 * 更新数据查询
 * PUT /api/v1/data/queries/:id
 */
router.put(
  "/queries/:id",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { id } = req.params;
      const { DataQuery } = require("../../models");
      const query = await DataQuery.findByPk(id);
      if (query) {
        await checkProjectAccess(req, query.projectId);
      }

      const result = await dataQueryService.updateQuery(id, req.body, req.user.id);

      res.json({
        success: true,
        data: result,
        message: "数据查询更新成功",
      });
    } catch (error) {
      console.error("更新数据查询失败:", error);
      next(error);
    }
  }
);

/**
 * 删除数据查询
 * DELETE /api/v1/data/queries/:id
 */
router.delete(
  "/queries/:id",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { id } = req.params;
      const { DataQuery } = require("../../models");
      const query = await DataQuery.findByPk(id);
      if (query) {
        await checkProjectAccess(req, query.projectId);
      }

      await dataQueryService.deleteQuery(id, req.user.id);

      res.json({
        success: true,
        message: "数据查询删除成功",
      });
    } catch (error) {
      console.error("删除数据查询失败:", error);
      next(error);
    }
  }
);

// ===========================================
// 数据点相关API
// ===========================================

const dataPointController = require("../../controllers/dataPointController");

/**
 * 获取数据点列表
 * GET /api/v1/data/projects/:projectId/datapoints
 */
router.get(
  "/projects/:projectId/datapoints",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);
      await dataPointController.getDataPoints(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取数据点值
 * GET /api/v1/data/projects/:projectId/datapoints/value
 */
router.get(
  "/projects/:projectId/datapoints/value",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);
      await dataPointController.getDataPointValue(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取数据点详情
 * GET /api/v1/data/projects/:projectId/datapoints/:id
 */
router.get(
  "/projects/:projectId/datapoints/:id",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);
      await dataPointController.getDataPoint(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 更新数据点扩展属性
 * PUT /api/v1/data/projects/:projectId/datapoints/:id
 */
router.put(
  "/projects/:projectId/datapoints/:id",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);
      await dataPointController.updateDataPoint(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 删除失效数据点
 * DELETE /api/v1/data/projects/:projectId/datapoints/:id
 */
router.delete(
  "/projects/:projectId/datapoints/:id",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);
      await dataPointController.deleteDataPoint(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 批量删除失效数据点
 * POST /api/v1/data/projects/:projectId/datapoints/delete-batch
 */
router.post(
  "/projects/:projectId/datapoints/delete-batch",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);
      await dataPointController.deleteDataPointsBatch(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取数据点使用情况
 * GET /api/v1/data/projects/:projectId/datapoints/:id/usages
 */
router.get(
  "/projects/:projectId/datapoints/:id/usages",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);
      await dataPointController.getDataPointUsages(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

// ===========================================
// MQTT 连接相关API
// ===========================================

/**
 * 创建 MQTT 连接
 * POST /api/v1/data/projects/:projectId/mqtt/connections
 */
router.post(
  "/projects/:projectId/mqtt/connections",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttConnectionController.create(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取 MQTT 连接列表
 * GET /api/v1/data/projects/:projectId/mqtt/connections
 */
router.get(
  "/projects/:projectId/mqtt/connections",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttConnectionController.list(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取单个 MQTT 连接
 * GET /api/v1/data/mqtt/connections/:id
 */
router.get(
  "/mqtt/connections/:id",
  authenticateToken,
  async (req, res, next) => {
    try {
      // 从连接中获取 projectId 进行权限检查
      const { DataConnection } = require("../../models");
      const connection = await DataConnection.findByPk(req.params.id);
      if (connection) {
        await checkProjectAccess(req, connection.projectId);
      }
      await mqttConnectionController.getOne(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 更新 MQTT 连接
 * PUT /api/v1/data/mqtt/connections/:id
 */
router.put(
  "/mqtt/connections/:id",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { DataConnection } = require("../../models");
      const connection = await DataConnection.findByPk(req.params.id);
      if (connection) {
        await checkProjectAccess(req, connection.projectId);
      }
      await mqttConnectionController.update(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 删除 MQTT 连接
 * DELETE /api/v1/data/mqtt/connections/:id
 */
router.delete(
  "/mqtt/connections/:id",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { DataConnection } = require("../../models");
      const connection = await DataConnection.findByPk(req.params.id);
      if (connection) {
        await checkProjectAccess(req, connection.projectId);
      }
      await mqttConnectionController.delete(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 测试 MQTT 连接
 * POST /api/v1/data/mqtt/connections/test
 */
router.post(
  "/mqtt/connections/test",
  authenticateToken,
  async (req, res, next) => {
    try {
      await mqttConnectionController.test(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 启动 MQTT 连接
 * POST /api/v1/data/mqtt/connections/:id/start
 */
router.post(
  "/mqtt/connections/:id/start",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { DataConnection } = require("../../models");
      const connection = await DataConnection.findByPk(req.params.id);
      if (connection) {
        await checkProjectAccess(req, connection.projectId);
      }
      await mqttConnectionController.start(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 停止 MQTT 连接
 * POST /api/v1/data/mqtt/connections/:id/stop
 */
router.post(
  "/mqtt/connections/:id/stop",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { DataConnection } = require("../../models");
      const connection = await DataConnection.findByPk(req.params.id);
      if (connection) {
        await checkProjectAccess(req, connection.projectId);
      }
      await mqttConnectionController.stop(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取 MQTT 连接状态
 * GET /api/v1/data/mqtt/connections/:id/status
 */
router.get(
  "/mqtt/connections/:id/status",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { DataConnection } = require("../../models");
      const connection = await DataConnection.findByPk(req.params.id);
      if (connection) {
        await checkProjectAccess(req, connection.projectId);
      }
      await mqttConnectionController.getStatus(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

// ===========================================
// MQTT 订阅相关API
// ===========================================

const mqttSubscriptionController = require("../../controllers/mqttSubscriptionController");

/**
 * 获取指定连接的 MQTT 订阅列表
 * GET /api/v1/data/projects/:projectId/mqtt/connections/:connectionId/subscriptions
 */
router.get(
  "/projects/:projectId/mqtt/connections/:connectionId/subscriptions",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);
      await mqttSubscriptionController.getSubscriptions(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 创建 MQTT 订阅
 * POST /api/v1/data/projects/:projectId/mqtt/connections/:connectionId/subscriptions
 */
router.post(
  "/projects/:projectId/mqtt/connections/:connectionId/subscriptions",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { projectId } = req.params;
      await checkProjectAccess(req, projectId);
      await mqttSubscriptionController.createSubscription(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取单个 MQTT 订阅
 * GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId
 */
router.get(
  "/projects/:projectId/mqtt/subscriptions/:subscriptionId",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttSubscriptionController.getSubscription(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 更新 MQTT 订阅
 * PUT /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId
 */
router.put(
  "/projects/:projectId/mqtt/subscriptions/:subscriptionId",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttSubscriptionController.updateSubscription(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 删除 MQTT 订阅
 * DELETE /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId
 */
router.delete(
  "/projects/:projectId/mqtt/subscriptions/:subscriptionId",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttSubscriptionController.deleteSubscription(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 启用/禁用 MQTT 订阅
 * PATCH /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/toggle
 */
router.patch(
  "/projects/:projectId/mqtt/subscriptions/:subscriptionId/toggle",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttSubscriptionController.toggleSubscription(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取订阅的实时消息
 * GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/messages
 */
router.get(
  "/projects/:projectId/mqtt/subscriptions/:subscriptionId/messages",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttSubscriptionController.getMessages(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

// ===========================================
// MQTT Tag (变量) 管理路由
// ===========================================

// ============================================
// MQTT 变量组管理路由
// ============================================

/**
 * 获取订阅的所有变量组
 * GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tag-groups
 */
router.get(
  "/projects/:projectId/mqtt/subscriptions/:subscriptionId/tag-groups",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttTagGroupController.getGroups(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 创建变量组
 * POST /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tag-groups
 */
router.post(
  "/projects/:projectId/mqtt/subscriptions/:subscriptionId/tag-groups",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttTagGroupController.createGroup(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 批量更新变量组顺序
 * PUT /api/v1/data/mqtt/tag-groups/order
 * 注意：此路由必须在 /:groupId 路由之前，否则 order 会被当作 groupId
 */
router.put(
  "/mqtt/tag-groups/order",
  authenticateToken,
  async (req, res, next) => {
    try {
      await mqttTagGroupController.updateGroupsOrder(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取单个变量组
 * GET /api/v1/data/mqtt/tag-groups/:groupId
 */
router.get(
  "/mqtt/tag-groups/:groupId",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { DataMqttTagGroup } = require("../../models");
      const group = await DataMqttTagGroup.findByPk(req.params.groupId);
      if (group) {
        await checkProjectAccess(req, group.projectId);
      }
      await mqttTagGroupController.getGroup(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 更新变量组
 * PUT /api/v1/data/mqtt/tag-groups/:groupId
 */
router.put(
  "/mqtt/tag-groups/:groupId",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { DataMqttTagGroup } = require("../../models");
      const group = await DataMqttTagGroup.findByPk(req.params.groupId);
      if (group) {
        await checkProjectAccess(req, group.projectId);
      }
      await mqttTagGroupController.updateGroup(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 删除变量组
 * DELETE /api/v1/data/mqtt/tag-groups/:groupId
 */
router.delete(
  "/mqtt/tag-groups/:groupId",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { DataMqttTagGroup } = require("../../models");
      const group = await DataMqttTagGroup.findByPk(req.params.groupId);
      if (group) {
        await checkProjectAccess(req, group.projectId);
      }
      await mqttTagGroupController.deleteGroup(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

// ============================================
// MQTT 变量管理路由
// ============================================

/**
 * 创建 Tag
 * POST /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags
 */
router.post(
  "/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttTagController.createTag(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 批量创建 Tag
 * POST /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags/batch
 */
router.post(
  "/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags/batch",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttTagController.createTagsBatch(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取订阅的所有 Tag
 * GET /api/v1/data/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags
 */
router.get(
  "/projects/:projectId/mqtt/subscriptions/:subscriptionId/tags",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttTagController.getTagsBySubscription(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取项目的所有 Tag
 * GET /api/v1/data/projects/:projectId/mqtt/tags
 */
router.get(
  "/projects/:projectId/mqtt/tags",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      await mqttTagController.getTagsByProject(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取 Tag 详情
 * GET /api/v1/data/mqtt/tags/:id
 */
router.get("/mqtt/tags/:id", authenticateToken, async (req, res, next) => {
  try {
    const { DataMqttTag } = require("../../models");
    const tag = await DataMqttTag.findByPk(req.params.id);
    if (tag) {
      await checkProjectAccess(req, tag.projectId);
    }
    await mqttTagController.getTag(req, res, next);
  } catch (error) {
    next(error);
  }
});

/**
 * 更新 Tag
 * PUT /api/v1/data/mqtt/tags/:id
 */
router.put("/mqtt/tags/:id", authenticateToken, async (req, res, next) => {
  try {
    const { DataMqttTag } = require("../../models");
    const tag = await DataMqttTag.findByPk(req.params.id);
    if (tag) {
      await checkProjectAccess(req, tag.projectId);
    }
    await mqttTagController.updateTag(req, res, next);
  } catch (error) {
    next(error);
  }
});

/**
 * 删除 Tag
 * DELETE /api/v1/data/mqtt/tags/:id
 */
router.delete("/mqtt/tags/:id", authenticateToken, async (req, res, next) => {
  try {
    const { DataMqttTag } = require("../../models");
    const tag = await DataMqttTag.findByPk(req.params.id);
    if (tag) {
      await checkProjectAccess(req, tag.projectId);
    }
    await mqttTagController.deleteTag(req, res, next);
  } catch (error) {
    next(error);
  }
});

/**
 * 切换 Tag 启用状态
 * PATCH /api/v1/data/mqtt/tags/:id/toggle
 */
router.patch(
  "/mqtt/tags/:id/toggle",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { DataMqttTag } = require("../../models");
      const tag = await DataMqttTag.findByPk(req.params.id);
      if (tag) {
        await checkProjectAccess(req, tag.projectId);
      }
      await mqttTagController.toggleTag(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 更新 Tags 顺序
 * PUT /api/v1/data/mqtt/tags/order
 */
router.put("/mqtt/tags/order", authenticateToken, async (req, res, next) => {
  try {
    await mqttTagController.updateTagsOrder(req, res, next);
  } catch (error) {
    next(error);
  }
});

/**
 * 获取 Tag 的当前值
 * GET /api/v1/data/mqtt/tags/:id/value
 */
router.get(
  "/mqtt/tags/:id/value",
  authenticateToken,
  async (req, res, next) => {
    try {
      const { DataMqttTag } = require("../../models");
      const tag = await DataMqttTag.findByPk(req.params.id);
      if (tag) {
        await checkProjectAccess(req, tag.projectId);
      }
      await mqttTagController.getTagValue(req, res, next);
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 获取多个 Tag 的当前值
 * POST /api/v1/data/mqtt/tags/values
 */
router.post("/mqtt/tags/values", authenticateToken, async (req, res, next) => {
  try {
    await mqttTagController.getTagValues(req, res, next);
  } catch (error) {
    next(error);
  }
});

// ===========================================
// MQTT 连接管理路由
// ===========================================

/**
 * 启动 MQTT 连接
 * POST /api/v1/data/projects/:projectId/mqtt/connections/:connectionId/start
 */
router.post(
  "/projects/:projectId/mqtt/connections/:connectionId/start",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      const mqttService = require("../../services/mqttService");
      const result = await mqttService.startConnection(req.params.connectionId);
      res.json({
        success: true,
        data: result,
        message: "MQTT连接已启动",
      });
    } catch (error) {
      next(error);
    }
  }
);

/**
 * 停止 MQTT 连接
 * POST /api/v1/data/projects/:projectId/mqtt/connections/:connectionId/stop
 */
router.post(
  "/projects/:projectId/mqtt/connections/:connectionId/stop",
  authenticateToken,
  async (req, res, next) => {
    try {
      await checkProjectAccess(req, req.params.projectId);
      const mqttService = require("../../services/mqttService");
      const result = await mqttService.stopConnection(req.params.connectionId);
      res.json({
        success: true,
        data: result,
        message: "MQTT连接已停止",
      });
    } catch (error) {
      next(error);
    }
  }
);

module.exports = router;
