function buildTenantWhere(baseWhere, req) {
  const where = { ...(baseWhere || {}) };
  if (req.user && req.user.role !== 'SUPER_ADMIN') {
    where.tenantId = req.user.tenantId;
  }
  return where;
}

module.exports = { buildTenantWhere };


