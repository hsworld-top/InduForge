const { randomUUID } = require('crypto');

function requestIdMiddleware(req, res, next) {
  const headerId = req.headers['x-request-id'];
  const id = (typeof headerId === 'string' && headerId.trim()) ? headerId : randomUUID();
  req.requestId = id;
  res.locals.requestId = id; // 设置到 res.locals 以便 ApiResponse 使用
  res.setHeader('X-Request-Id', id);
  next();
}

module.exports = { requestIdMiddleware };


