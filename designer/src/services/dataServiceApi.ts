/**
 * data_service 接入层。
 *
 * 设计器开发阶段只请求相对路径 `/api/v1/data/...`，由 Vite 代理转发到 data_service；
 * 正式环境由 nginx 代理承接同一路径，前端不直接跨域访问数据服务。
 */

import request from "@/utils/request";
import { requireDatapointsPagePayload } from "@/utils/datapoint-payload";
import { unwrapApiData } from "@/types/api";

export interface ListDataPointsParams {
  page?: number;
  pageSize?: number;
  search?: string;
  status?: string;
  type?: string;
  sourceId?: string;
}

export interface DataPointValuesRequest {
  datapointIds?: string[];
  paths?: string[];
}

export interface DataPointValuesResponse {
  values?: Record<string, unknown>;
}

export interface NormalizedDataPointsPage {
  datapoints: unknown[];
  pagination: Record<string, unknown>;
}

function compactParams(params: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(
    Object.entries(params).filter(([, value]) => value !== undefined && value !== null && value !== ""),
  );
}

export const dataServiceApi = {
  getConnections(projectId: string, params: Record<string, unknown> = {}) {
    return request.get(`/data/projects/${projectId}/connections`, {
      params: compactParams(params),
    });
  },

  createPreviewSession(projectId: string) {
    return request.post(`/data/projects/${projectId}/preview/sessions`, {});
  },

  heartbeatPreviewSession(sessionId: string) {
    return request.post(`/data/preview/sessions/${sessionId}/heartbeat`, {});
  },

  deletePreviewSession(sessionId: string) {
    return request.delete(`/data/preview/sessions/${sessionId}`);
  },

  getQueries(projectId: string, params: Record<string, unknown> = {}) {
    return request.get(`/data/projects/${projectId}/queries`, {
      params: compactParams(params),
    });
  },

  getDatapoints(projectId: string, connectionId: string) {
    return request.get(`/data/projects/${projectId}/connections/${connectionId}/datapoints`);
  },

  async listDataPoints(
    projectId: string,
    params: ListDataPointsParams = {},
  ): Promise<NormalizedDataPointsPage> {
    const result = await request.get(`/data/projects/${projectId}/datapoints`, {
      params: compactParams({ ...params }),
    });
    return requireDatapointsPagePayload(unwrapApiData(result));
  },

  getDataPoints(projectId: string, params: Record<string, unknown> = {}) {
    return request.get(`/data/projects/${projectId}/datapoints`, {
      params: compactParams(params),
    });
  },

  getDatapointStatus(projectId: string, datapointIds: string[]) {
    return request.post(`/data/projects/${projectId}/datapoints/status`, {
      datapointIds,
    });
  },

  getDataPointValues(
    projectId: string,
    payload: DataPointValuesRequest,
  ): Promise<DataPointValuesResponse> {
    return request.post<DataPointValuesResponse>(
      `/data/projects/${projectId}/datapoints/values`,
      payload,
    );
  },

  getDatapointValues(projectId: string, datapointIds: string[]) {
    return request.post(`/data/projects/${projectId}/datapoints/values`, {
      datapointIds,
    });
  },

  writeDatapointValue(projectId: string, datapointId: string, value: unknown) {
    return request.post(`/data/projects/${projectId}/datapoints/${datapointId}/write`, { value });
  },

  executeQuery(queryId: string, parameters: Record<string, unknown> = {}) {
    return request.post(`/data/queries/${queryId}/execute`, {
      parameters,
    });
  },
};

export default dataServiceApi;
