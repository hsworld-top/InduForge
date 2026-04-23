const {
  buildAccessTokenPayload,
  hasCapability,
  listAccessibleProjectIds,
} = require('../authz');

describe('authz token payload helpers', () => {
  test('SYSTEM_ADMIN 令牌应携带 capability 通配符与 projectIds 通配符', () => {
    const payload = buildAccessTokenPayload({
      userId: 'u-1',
      username: 'admin',
      role: 'SYSTEM_ADMIN',
      tenantId: 't-1',
      projectIds: ['p-1'],
    });

    expect(payload).toMatchObject({
      userId: 'u-1',
      username: 'admin',
      role: 'SYSTEM_ADMIN',
      tenantId: 't-1',
      capabilities: ['*'],
      projectIds: ['*'],
    });
  });

  test('租户级角色会把租户下工程列表写入 JWT projectIds', async () => {
    const ProjectModel = {
      findAll: jest.fn().mockResolvedValue([
        { id: 'p-1' },
        { id: 'p-2' },
        { id: 'p-2' },
      ]),
    };

    await expect(
      listAccessibleProjectIds(ProjectModel, {
        tenantId: 'tenant-1',
        role: 'PROJECT_ADMIN',
      }),
    ).resolves.toEqual(['p-1', 'p-2']);
    expect(ProjectModel.findAll).toHaveBeenCalledWith({
      where: { tenantId: 'tenant-1' },
      attributes: ['id'],
    });
  });

  test('权限判断会识别角色的 capability 通配符', () => {
    expect(hasCapability('SYSTEM_ADMIN', 'project:read')).toBe(true);
    expect(hasCapability('USER', 'project:write')).toBe(false);
  });
});
