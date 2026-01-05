const ApiResponse = require("../utils/response");
const AppError = require("../utils/AppError");
const { ErrorCodes } = require("../constants/errorCodes");
const dataPointService = require("../services/dataPointService");

/**
 * 数据点控制器
 * 处理数据点列表、详情与更新接口
 */
class DataPointController {
  /**
   * 获取数据点列表
   * GET /api/v1/data/projects/:projectId/datapoints
   */
  async getDataPoints(req, res, next) {
    try {
      const { projectId } = req.params;
      const result = await dataPointService.getDataPoints(projectId, req.query);
      return ApiResponse.paginated(
        res,
        { datapoints: result.datapoints },
        result.pagination
      );
    } catch (error) {
      return next(error);
    }
  }

  /**
   * 获取数据点详情
   * GET /api/v1/data/projects/:projectId/datapoints/:id
   */
  async getDataPoint(req, res, next) {
    try {
      const { projectId, id } = req.params;
      const datapoint = await dataPointService.getDataPoint(projectId, id);
      return ApiResponse.success(res, { datapoint });
    } catch (error) {
      return next(error);
    }
  }

  /**
   * 更新数据点扩展属性
   * PUT /api/v1/data/projects/:projectId/datapoints/:id
   */
  async updateDataPoint(req, res, next) {
    try {
      const { projectId, id } = req.params;
      const datapoint = await dataPointService.updateDataPoint(
        projectId,
        id,
        req.body,
        req.user.id
      );
      return ApiResponse.success(res, { datapoint });
    } catch (error) {
      return next(error);
    }
  }

  /**
   * 删除失效数据点
   * DELETE /api/v1/data/projects/:projectId/datapoints/:id
   */
  async deleteDataPoint(req, res, next) {
    try {
      const { projectId, id } = req.params;
      await dataPointService.deleteInvalidDataPoint(projectId, id);
      return ApiResponse.success(res, null);
    } catch (error) {
      return next(error);
    }
  }

  /**
   * 批量删除失效数据点
   * POST /api/v1/data/projects/:projectId/datapoints/delete-batch
   */
  async deleteDataPointsBatch(req, res, next) {
    try {
      const { projectId } = req.params;
      const { ids } = req.body;
      const deletedCount = await dataPointService.deleteInvalidDataPoints(
        projectId,
        ids
      );
      return ApiResponse.success(res, { deletedCount });
    } catch (error) {
      return next(error);
    }
  }

  /**
   * 获取数据点值
   * GET /api/v1/data/projects/:projectId/datapoints/value
   */
  async getDataPointValue(req, res, next) {
    try {
      const { projectId } = req.params;
      const { path } = req.query;

      if (!path) {
        throw new AppError(ErrorCodes.VALIDATION_REQUIRED, 400, {
          message: "缺少数据点路径",
        });
      }

      const value = await dataPointService.getDataPointValue(projectId, path);
      return ApiResponse.success(res, value);
    } catch (error) {
      return next(error);
    }
  }

  /**
   * 获取数据点使用情况
   * GET /api/v1/data/projects/:projectId/datapoints/:id/usages
   */
  async getDataPointUsages(req, res, next) {
    try {
      const { projectId } = req.params;
      await dataPointService.getDataPoint(projectId, req.params.id);
      return ApiResponse.success(res, { usages: [] });
    } catch (error) {
      return next(error);
    }
  }
}

module.exports = new DataPointController();
