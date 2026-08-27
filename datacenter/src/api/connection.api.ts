import type { AxiosRequestConfig } from 'axios'
import request from '@/utils/request'
import {
  ConnectionPageSchema,
  ConnectionSchema,
  ConnectionTestResultSchema,
  type Connection,
  type ConnectionPage,
  type ConnectionTestResult,
} from './schemas/connection.schema'

const unwrapData = (value: unknown): unknown => {
  if (value && typeof value === 'object' && 'code' in value && 'data' in value) {
    return (value as { data?: unknown }).data
  }
  return value
}

const requestData = async (config: AxiosRequestConfig): Promise<unknown> =>
  unwrapData(await request(config))

export async function listConnections(
  projectId: string,
  params?: Record<string, unknown>,
): Promise<Connection[] | ConnectionPage> {
  const data = await requestData({
    url: `/data/projects/${projectId}/connections`,
    method: 'get',
    params,
  })
  return params ? ConnectionPageSchema.parse(data) : ConnectionSchema.array().parse(data)
}

export async function getConnection(projectId: string, connectionId: string): Promise<Connection> {
  return ConnectionSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/connections/${connectionId}`,
      method: 'get',
    }),
  )
}

export async function createConnection(
  projectId: string,
  data: Record<string, unknown>,
): Promise<Connection> {
  return ConnectionSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/connections`,
      method: 'post',
      data,
    }),
  )
}

export async function updateConnection(
  projectId: string,
  connectionId: string,
  data: Record<string, unknown>,
): Promise<Connection> {
  return ConnectionSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/connections/${connectionId}`,
      method: 'put',
      data,
    }),
  )
}

export async function deleteConnection(projectId: string, connectionId: string): Promise<void> {
  await requestData({
    url: `/data/projects/${projectId}/connections/${connectionId}`,
    method: 'delete',
  })
}

export async function testConnectionDraft(
  projectId: string,
  data: Record<string, unknown>,
): Promise<ConnectionTestResult> {
  return ConnectionTestResultSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/connections/test`,
      method: 'post',
      data,
    }),
  )
}

export async function testSavedConnection(
  projectId: string,
  connectionId: string,
): Promise<ConnectionTestResult> {
  return ConnectionTestResultSchema.parse(
    await requestData({
      url: `/data/projects/${projectId}/connections/${connectionId}/test`,
      method: 'post',
    }),
  )
}

export async function listDatapointSourceOptions(
  projectId: string,
  params: Record<string, unknown>,
): Promise<unknown> {
  return requestData({
    url: `/data/projects/${projectId}/datapoint-source-options`,
    method: 'get',
    params,
  })
}
