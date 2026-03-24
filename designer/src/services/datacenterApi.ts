/**
 * 数据中心 API：连接、查询、数据点
 */

import request from "@/utils/request";

export const datacenterApi = {
  getConnections(projectId: string, params: Record<string, unknown> = {}) {
    return request.get(`/data/projects/${projectId}/connections`, {
      params,
    });
  },

  getQueries(projectId: string, params: Record<string, unknown> = {}) {
    return request.get(`/data/projects/${projectId}/queries`, {
      params,
    });
  },

  getDatapoints(projectId: string, connectionId: string) {
    return request.get(
      `/data/projects/${projectId}/connections/${connectionId}/datapoints`,
    );
  },

  getDataPoints(projectId: string, params: Record<string, unknown> = {}) {
    return request.get(`/data/projects/${projectId}/datapoints`, {
      params,
    });
  },

  getDatapointStatus(projectId: string, datapointIds: string[]) {
    return request.post(`/data/projects/${projectId}/datapoints/status`, {
      datapointIds,
    });
  },

  getDatapointValues(projectId: string, datapointIds: string[]) {
    return request.post(`/data/projects/${projectId}/datapoints/values`, {
      datapointIds,
    });
  },

  writeDatapointValue(projectId: string, datapointId: string, value: unknown) {
    return request.post(
      `/data/projects/${projectId}/datapoints/${datapointId}/write`,
      { value },
    );
  },

  executeQuery(queryId: string, parameters: Record<string, unknown> = {}) {
    return request.post(`/data/queries/${queryId}/execute`, {
      parameters,
    });
  },
};

export default datacenterApi;
