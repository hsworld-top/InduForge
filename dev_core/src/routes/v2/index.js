const express = require('express');
const dayjs = require('dayjs');
const { TIME_FORMAT } = require('../../constants/time');

const router = express.Router();


const { logger } = require('../../utils/logger');
/**
 * @swagger
 * /api/v2/test:
 *   get:
 *     summary: V2 测试 API
 *     tags: [V2测试]
 *     responses:
 *       200:
 *         description: 测试成功
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 message:
 *                   type: string
 *                 version:
 *                   type: string
 *                 timestamp:
 *                   type: string
 */
router.get('/test', (req, res) => {
  res.json({
    message: 'This is a test API from v2',
    version: '2.0.0',
    timestamp: dayjs().format(TIME_FORMAT)
  });
});

function buildV2Router(options = {}) {
  logger.info('V2 router mounted successfully');
  return router;
}

module.exports = { buildV2Router };

