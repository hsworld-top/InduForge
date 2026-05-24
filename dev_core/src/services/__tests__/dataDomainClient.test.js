const { DataDomainClient } = require('../dataDomainClient')
const ErrorCodes = require('../../constants/errorCodes')

const buildJsonResponse = (body, options = {}) => ({
  ok: options.ok ?? true,
  status: options.status ?? 200,
  statusText: options.statusText ?? 'OK',
  headers: {
    get: (name) => (name && name.toLowerCase() === 'content-type' ? 'application/json' : null),
  },
  json: jest.fn().mockResolvedValue(body),
  text: jest.fn().mockResolvedValue(JSON.stringify(body)),
})

describe('dataDomainClient', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    global.fetch = jest.fn()
  })

  afterEach(() => {
    delete global.fetch
  })

  test('getProjectSnapshot 会规范化 data_service 返回的 PascalCase 快照字段', async () => {
    global.fetch.mockResolvedValue(
      buildJsonResponse({
        success: true,
        data: {
          connections: [
            {
              ID: 'conn-1',
              ProjectID: 'project-1',
              Name: '主库',
              Type: 'relational',
              Status: 'connected',
              Config: {
                dbType: 'postgresql',
                host: '172.21.242.174',
              },
            },
          ],
          relationalConfigs: [
            {
              ConnectionID: 'conn-1',
              DBType: 'postgresql',
              Host: '172.21.242.174',
              Port: 5432,
              Database: 'data_service',
              Username: 'postgres',
              Password: 'postgres',
              SSL: false,
              SSLConfig: {},
            },
          ],
          queries: [
            {
              ID: 'query-1',
              ProjectID: 'project-1',
              ConnectionID: 'conn-1',
              Name: '查询1',
              QueryType: 'sql',
              Config: { sql: 'select 1' },
              IsEnabled: true,
              TimeoutMS: 30000,
              CacheEnabled: false,
              CacheTtlSeconds: 0,
            },
          ],
          datapoints: [
            {
              ID: 'dp-1',
              ProjectID: 'project-1',
              Path: 'device.temp',
              Name: '温度',
              SourceType: 'query',
              SourceID: 'query-1',
              SourceConfig: { column: 'temp' },
              DataType: 'number',
              Tags: [],
              RefreshMode: 'auto',
              Status: 'active',
            },
          ],
        },
      }),
    )

    const client = new DataDomainClient({
      baseUrl: 'http://data-service.test',
      timeoutMs: 2000,
    })

    const snapshot = await client.getProjectSnapshot('project-1', 'Bearer token-1')

    expect(global.fetch).toHaveBeenCalledWith(
      'http://data-service.test/api/v1/data/projects/project-1/snapshot',
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({
          Authorization: 'Bearer token-1',
          Accept: 'application/json',
        }),
      }),
    )
    expect(snapshot.connections[0]).toEqual({
      id: 'conn-1',
      projectId: 'project-1',
      name: '主库',
      type: 'relational',
      status: 'connected',
      config: {
        dbType: 'postgresql',
        host: '172.21.242.174',
      },
      createdAt: undefined,
      updatedAt: undefined,
    })
    expect(snapshot.queries[0].timeoutMs).toBe(30000)
    expect(snapshot.datapoints[0]).toEqual(
      expect.objectContaining({
        id: 'dp-1',
        sourceType: 'query',
        sourceId: 'query-1',
        dataType: 'number',
      }),
    )
  })

  test('replaceProjectSnapshot 会使用 PUT 调用 data_service 快照覆盖接口', async () => {
    global.fetch.mockResolvedValue(
      buildJsonResponse({
        success: true,
        data: { updated: true },
      }),
    )

    const client = new DataDomainClient({
      baseUrl: 'http://data-service.test',
      timeoutMs: 2000,
    })
    const snapshot = {
      connections: [
        { id: 'conn-1', name: '主库', type: 'relational', status: 'connected', config: {} },
      ],
      relationalConfigs: [],
      queries: [],
      mqttConfigs: [],
      mqttSubscriptions: [],
      mqttTagGroups: [],
      mqttTags: [],
      datapoints: [],
    }

    const result = await client.replaceProjectSnapshot('project-1', snapshot, 'Bearer token-2')

    expect(result).toEqual({ updated: true })
    expect(global.fetch).toHaveBeenCalledWith(
      'http://data-service.test/api/v1/data/projects/project-1/snapshot',
      expect.objectContaining({
        method: 'PUT',
        headers: expect.objectContaining({
          Authorization: 'Bearer token-2',
          'Content-Type': 'application/json',
        }),
        body: JSON.stringify(snapshot),
      }),
    )
  })

  test('getProjectArtifact 会规范化 artifact v1 并读取 mqtt/protocols 区块', async () => {
    global.fetch.mockResolvedValue(
      buildJsonResponse({
        success: true,
        data: {
          version: '1.0',
          projectId: 'project-1',
          generatedAt: '2026-04-20T12:00:00Z',
          connections: [
            {
              id: 'conn-kafka-1',
              name: 'kafka-main',
              type: 'kafka',
              status: 'connected',
              config: { brokers: '127.0.0.1:9092' },
            },
          ],
          queries: [],
          datapoints: [],
          mqtt: {
            connections: [
              {
                id: 'conn-mqtt-1',
                name: 'mqtt-main',
                type: 'mqtt',
                status: 'connected',
                brokerUrl: 'tcp://127.0.0.1:1883',
                protocol: 'mqtt',
                port: 1883,
                keepalive: 60,
                cleanSession: true,
                qos: 1,
                reconnectPeriod: 1000,
                connectTimeout: 30000,
                will: {},
                sslConfig: {},
              },
            ],
            subscriptions: [
              {
                id: 'sub-1',
                projectId: 'project-1',
                connectionId: 'conn-mqtt-1',
                name: 'sub-main',
                topic: 'factory/line1/temp',
                qos: 1,
                isEnabled: true,
                messageRetention: 100,
              },
            ],
            tagGroups: [],
            tags: [],
          },
          protocols: {
            kafka: [
              {
                id: 'conn-kafka-1',
                name: 'kafka-main',
                type: 'kafka',
                status: 'connected',
                config: { topic: 'factory.events' },
              },
            ],
            http: [],
            websocket: [],
            redis: [],
          },
        },
      }),
    )

    const client = new DataDomainClient({
      baseUrl: 'http://data-service.test',
      timeoutMs: 2000,
    })

    const artifact = await client.getProjectArtifact('project-1', 'Bearer token-artifact')

    expect(global.fetch).toHaveBeenCalledWith(
      'http://data-service.test/api/v1/data/projects/project-1/artifact',
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({
          Authorization: 'Bearer token-artifact',
          Accept: 'application/json',
        }),
      }),
    )
    expect(artifact.version).toBe('1.0')
    expect(artifact.projectId).toBe('project-1')
    expect(artifact.mqtt.connections[0]).toEqual(
      expect.objectContaining({
        id: 'conn-mqtt-1',
        brokerUrl: 'tcp://127.0.0.1:1883',
      }),
    )
    expect(artifact.protocols.kafka[0]).toEqual(
      expect.objectContaining({
        id: 'conn-kafka-1',
        type: 'kafka',
      }),
    )
  })

  test('request 在非 2xx 响应时会抛出 AppError，并使用技术异常状态边界 502', async () => {
    global.fetch.mockResolvedValue(
      buildJsonResponse(
        {
          message: 'upstream 404',
        },
        {
          ok: false,
          status: 404,
          statusText: 'Not Found',
        },
      ),
    )

    const client = new DataDomainClient({
      baseUrl: 'http://data-service.test',
      timeoutMs: 2000,
    })

    await expect(client.getProjectSnapshot('project-1', 'Bearer token-1')).rejects.toMatchObject({
      name: 'AppError',
      errorCode: ErrorCodes.EXTERNAL_SERVICE_ERROR,
      statusCode: 502,
      options: expect.objectContaining({
        message: 'upstream 404',
      }),
    })
  })
})
