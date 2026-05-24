const { QueryTypes } = require('sequelize')

describe('projectSettingsStore', () => {
  let sequelize
  let projectSettingsStore

  beforeEach(() => {
    jest.resetModules()
    sequelize = {
      query: jest.fn(),
    }
    projectSettingsStore = require('../projectSettingsStore')
  })

  test('getProjectSettingsRow 使用 PostgreSQL 占位符查询设置', async () => {
    const expectedRow = {
      globalVariables: { definitions: { a: 1 }, groups: [] },
      globalScripts: { system: {} },
      i18n: { enabled: true },
    }
    sequelize.query.mockResolvedValue([expectedRow])

    const row = await projectSettingsStore.getProjectSettingsRow(sequelize, 'project-1')

    expect(row).toEqual(expectedRow)
    expect(sequelize.query).toHaveBeenCalledWith(
      expect.stringContaining('WHERE "projectId" = $1'),
      expect.objectContaining({
        bind: ['project-1'],
        type: QueryTypes.SELECT,
      }),
    )
  })

  test('upsertProjectSettings 使用 ON CONFLICT 覆盖已有记录', async () => {
    sequelize.query.mockResolvedValue([])

    await projectSettingsStore.upsertProjectSettings(sequelize, {
      projectId: 'project-1',
      schemaVersion: '1.0.0',
      globalVariables: { definitions: { a: 1 }, groups: [] },
      globalScripts: { system: {} },
      i18n: { enabled: true },
      updatedBy: 'user-1',
      updatedAt: new Date('2026-04-20T10:00:00.000Z'),
    })

    expect(sequelize.query).toHaveBeenCalledWith(
      expect.stringContaining('ON CONFLICT ("projectId") DO UPDATE'),
      expect.objectContaining({
        bind: [
          'project-1',
          '1.0.0',
          JSON.stringify({ definitions: { a: 1 }, groups: [] }),
          JSON.stringify({ system: {} }),
          JSON.stringify({ enabled: true }),
          'user-1',
          new Date('2026-04-20T10:00:00.000Z'),
        ],
      }),
    )
  })
})
