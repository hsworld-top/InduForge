const express = require('express');
const router = express.Router();
const { authenticateToken } = require('../../middlewares/auth');
const dataConnectionService = require('../../services/dataConnectionService');
const dataQueryService = require('../../services/dataQueryService');
const AppError = require('../../utils/AppError');
const ErrorCodes = require('../../constants/errorCodes');

/**
 * 权限检查中间件
 * 检查用户是否有权限访问指定工程
 */
async function ensureProjectAccess(req, projectId) {
  if (req.user.role === 'SUPER_ADMIN' || req.user.role === 'SYSTEM_ADMIN') {
    return true;
  }
  const userProjects = await req.user.getProjects();
  const projectIds = userProjects.map(p => p.id);
  return projectIds.includes(projectId);
}

/**
 * 权限检查包装器
 */
async function checkProjectAccess(req, projectId) {
  const hasAccess = await ensureProjectAccess(req, projectId);
  if (!hasAccess) {
    throw new AppError(ErrorCodes.PERMISSION_DENIED, 403, {
      message: '无权访问此工程'
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
router.get('/projects/:projectId/connections', authenticateToken, async (req, res, next) => {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const { page, limit, type, status } = req.query;
    const result = await dataConnectionService.getConnections(projectId, {
      page,
      limit,
      type,
      status
    });

    res.json({
      success: true,
      data: result
    });
  } catch (error) {
    console.error('获取数据连接列表失败:', error);
    next(error);
  }
});

/**
 * 创建数据连接
 * POST /api/v1/projects/:projectId/connections
 */
router.post('/projects/:projectId/connections', authenticateToken, async (req, res, next) => {
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
      message: '数据连接创建成功',
      data: connection
    });
  } catch (error) {
    console.error('创建数据连接失败:', error);
    next(error);
  }
});

/**
 * 更新数据连接
 * PUT /api/v1/projects/:projectId/connections/:connectionId
 */
router.put('/projects/:projectId/connections/:connectionId', authenticateToken, async (req, res, next) => {
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
      message: '数据连接更新成功',
      data: connection
    });
  } catch (error) {
    console.error('更新数据连接失败:', error);
    next(error);
  }
});

/**
 * 删除数据连接
 * DELETE /api/v1/projects/:projectId/connections/:connectionId
 */
router.delete('/projects/:projectId/connections/:connectionId', authenticateToken, async (req, res, next) => {
  try {
    const { projectId, connectionId } = req.params;
    await checkProjectAccess(req, projectId);

    await dataConnectionService.deleteConnection(projectId, connectionId);

    res.json({
      success: true,
      message: '数据连接删除成功'
    });
  } catch (error) {
    console.error('删除数据连接失败:', error);
    next(error);
  }
});

/**
 * 更新连接状态
 * PATCH /api/v1/projects/:projectId/connections/:connectionId/status
 */
router.patch('/projects/:projectId/connections/:connectionId/status', authenticateToken, async (req, res, next) => {
  try {
    const { projectId, connectionId } = req.params;
    await checkProjectAccess(req, projectId);

    const { status } = req.body;

    if (!status) {
      return res.status(400).json({
        success: false,
        message: '缺少状态参数'
      });
    }

    const connection = await dataConnectionService.updateConnectionStatus(
      projectId,
      connectionId,
      status
    );

    res.json({
      success: true,
      message: '连接状态更新成功',
      data: connection
    });
  } catch (error) {
    console.error('更新连接状态失败:', error);
    next(error);
  }
});

/**
 * 测试数据连接
 * POST /api/v1/projects/:projectId/connections/test
 */
router.post('/projects/:projectId/connections/test', authenticateToken, async (req, res, next) => {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const { type, config } = req.body;

    if (!type || !config) {
      return res.status(400).json({
        success: false,
        message: '缺少连接类型或配置'
      });
    }

    await dataConnectionService.testConnection(type, config);

    res.json({
      success: true,
      message: '数据库连接成功'
    });
  } catch (error) {
    console.error('测试数据连接失败:', error);
    next(error);
  }
});

/**
 * 获取关系数据库的表列表
 * GET /api/v1/projects/:projectId/connections/:connectionId/tables
 */
router.get('/projects/:projectId/connections/:connectionId/tables', authenticateToken, async (req, res, next) => {
  try {
    const { projectId, connectionId } = req.params;
    await checkProjectAccess(req, projectId);

    const tables = await dataConnectionService.getTables(projectId, connectionId);

    res.json({
      success: true,
      data: { tables }
    });
  } catch (error) {
    console.error('获取表列表失败:', error);
    next(error);
  }
});

/**
 * 直接执行SQL查询
 * POST /api/v1/projects/:projectId/connections/:connectionId/execute-sql
 */
router.post('/projects/:projectId/connections/:connectionId/execute-sql', authenticateToken, async (req, res, next) => {
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
      executionTime: result.executionTime
    });
  } catch (error) {
    console.error('执行SQL查询失败:', error);
    next(error);
  }
});

/**
 * 获取指定表的数据
 * GET /api/v1/projects/:projectId/connections/:connectionId/tables/:tableName/data
 */
router.get('/projects/:projectId/connections/:connectionId/tables/:tableName/data', authenticateToken, async (req, res, next) => {
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
      data: result
    });
  } catch (error) {
    console.error('获取表数据失败:', error);
    next(error);
  }
});

// ===========================================
// 数据查询相关API
// ===========================================

/**
 * 获取数据查询列表
 * GET /api/v1/projects/:projectId/queries
 */
router.get('/projects/:projectId/queries', authenticateToken, async (req, res, next) => {
  try {
    const { projectId } = req.params;
    await checkProjectAccess(req, projectId);

    const { page, limit, connectionId, queryType, isActive } = req.query;

    const result = await dataQueryService.getQueries(projectId, {
      page,
      limit,
      connectionId,
      queryType,
      isActive
    });

    res.json({
      success: true,
      data: result
    });
  } catch (error) {
    console.error('获取数据查询列表失败:', error);
    next(error);
  }
});

/**
 * 创建数据查询
 * POST /api/v1/projects/:projectId/queries
 */
router.post('/projects/:projectId/queries', authenticateToken, async (req, res, next) => {
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
      message: '数据查询创建成功',
      data: query
    });
  } catch (error) {
    console.error('创建数据查询失败:', error);
    next(error);
  }
});

/**
 * 执行数据查询
 * POST /api/v1/queries/:id/execute
 */
router.post('/queries/:id/execute', authenticateToken, async (req, res, next) => {
  try {
    const { id } = req.params;
    const { parameters = {} } = req.body;

    // 权限检查：先获取查询，检查工程权限
    const { DataQuery } = require('../../models');
    const query = await DataQuery.findByPk(id);
    
    if (query) {
      await checkProjectAccess(req, query.projectId);
    }

    const result = await dataQueryService.executeQuery(id, parameters, req.user.id);

    res.json({
      success: true,
      data: result.data,
      executionTime: result.executionTime
    });
  } catch (error) {
    console.error('执行数据查询失败:', error);
    next(error);
  }
});

module.exports = router;
