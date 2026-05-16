import request from "@/utils/request";
import { listResponseSchema } from "./schemas/common.schema";
import {
  DatapointSchema,
  DatapointUpdateSchema,
  type Datapoint,
  type DatapointUpdate,
} from "./schemas/datapoint.schema";

const datapointListSchema = listResponseSchema(DatapointSchema);

type DatapointListResp = {
  list: Datapoint[];
  pagination?: {
    page?: number;
    pageSize?: number;
    total?: number;
  };
};

/** 获取数据点列表 */
export async function getDatapoints(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<DatapointListResp> {
  const res = await request({
    url: `/data/projects/${projectId}/datapoints`,
    method: "get",
    params,
  });
  // request 已解包 data 层，res 即后端 data 字段
  return datapointListSchema.parse(res);
}

/** 更新数据点 */
export async function updateDatapoint(
  projectId: string,
  datapointId: string,
  data: DatapointUpdate,
): Promise<Datapoint> {
  const body = DatapointUpdateSchema.parse(data);
  const res = await request({
    url: `/data/projects/${projectId}/datapoints/${datapointId}`,
    method: "put",
    data: body,
  });
  return DatapointSchema.parse(res);
}

/** 更新数据点运行态权限 */
export async function updateDatapointRuntimeGrant(
  projectId: string,
  datapointId: string,
  data: Record<string, unknown>,
): Promise<Datapoint> {
  const res = await request({
    url: `/data/projects/${projectId}/datapoints/${datapointId}/runtime-permissions`,
    method: "put",
    data,
  });
  return DatapointSchema.parse(res);
}

/** 删除数据点 */
export async function deleteDatapoint(
  projectId: string,
  datapointId: string,
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/datapoints/${datapointId}`,
    method: "delete",
  });
}

/** 批量删除数据点 */
export async function deleteDatapointsBatch(
  projectId: string,
  ids: string[],
): Promise<void> {
  await request({
    url: `/data/projects/${projectId}/datapoints/delete-batch`,
    method: "post",
    data: { ids },
  });
}
