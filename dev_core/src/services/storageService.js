const Minio = require('minio');
const dayjs = require('dayjs');
const { logger } = require('../utils/logger');
const { TIME_FORMAT } = require('../constants/time');

let minioClient = null;
let storageStatus = {
  connected: false,
  lastError: null,
  lastErrorTime: null,
};

const getStorageConfig = () => {
  const endpoint = process.env.MINIO_ENDPOINT || '127.0.0.1';
  const port = Number(process.env.MINIO_PORT || 9000);
  const useSSL = String(process.env.MINIO_USE_SSL || 'false') === 'true';
  const accessKey = process.env.MINIO_ACCESS_KEY || '';
  const secretKey = process.env.MINIO_SECRET_KEY || '';
  const bucketIfp = process.env.MINIO_BUCKET_IFP || 'ifp-artifacts';
  const bucketDesign = process.env.MINIO_BUCKET_DESIGN || 'design-assets';
  const region = process.env.MINIO_REGION || 'us-east-1';

  return { endpoint, port, useSSL, accessKey, secretKey, bucketIfp, bucketDesign, region };
};

const buildClient = () => {
  const { endpoint, port, useSSL, accessKey, secretKey } = getStorageConfig();
  return new Minio.Client({
    endPoint: endpoint,
    port,
    useSSL,
    accessKey,
    secretKey,
  });
};

const ensureBucket = async (client, bucket, region) => {
  const exists = await client.bucketExists(bucket);
  if (!exists) {
    await client.makeBucket(bucket, region);
    logger.info('MinIO bucket created', { bucket, region });
  }
};

/**
 * 初始化 MinIO 连接并确保 Bucket 存在
 * @returns {Promise<Minio.Client>}
 */
const initStorage = async () => {
  if (minioClient) {
    return minioClient;
  }

  const { bucketIfp, bucketDesign, region } = getStorageConfig();
  const client = buildClient();

  try {
    await ensureBucket(client, bucketIfp, region);
    await ensureBucket(client, bucketDesign, region);
    storageStatus.connected = true;
    storageStatus.lastError = null;
    storageStatus.lastErrorTime = null;
    minioClient = client;
    return minioClient;
  } catch (error) {
    storageStatus.connected = false;
    storageStatus.lastError = error.message;
    storageStatus.lastErrorTime = dayjs().format(TIME_FORMAT);
    throw error;
  }
};

const getClient = async () => {
  if (!minioClient) {
    return initStorage();
  }
  return minioClient;
};

const getBucketName = (type = 'ifp') => {
  const { bucketIfp, bucketDesign } = getStorageConfig();
  return type === 'design' ? bucketDesign : bucketIfp;
};

/**
 * 上传对象到 MinIO
 * @param {string} bucketType
 * @param {string} objectKey
 * @param {Readable} stream
 * @param {number} size
 * @param {Object} meta
 */
const uploadObject = async (bucketType, objectKey, stream, size, meta = {}) => {
  const client = await getClient();
  const bucket = getBucketName(bucketType);
  return client.putObject(bucket, objectKey, stream, size, meta);
};

/**
 * 获取对象可读流
 * @param {string} bucketType
 * @param {string} objectKey
 * @returns {Promise<Readable>}
 */
const getObjectStream = async (bucketType, objectKey) => {
  const client = await getClient();
  const bucket = getBucketName(bucketType);
  return client.getObject(bucket, objectKey);
};

/**
 * 获取对象元信息
 * @param {string} bucketType
 * @param {string} objectKey
 * @returns {Promise<Object>}
 */
const statObject = async (bucketType, objectKey) => {
  const client = await getClient();
  const bucket = getBucketName(bucketType);
  return client.statObject(bucket, objectKey);
};

/**
 * 删除对象
 * @param {string} bucketType
 * @param {string} objectKey
 */
const removeObject = async (bucketType, objectKey) => {
  const client = await getClient();
  const bucket = getBucketName(bucketType);
  return client.removeObject(bucket, objectKey);
};

const getStorageStatus = () => ({
  ...storageStatus,
});

module.exports = {
  initStorage,
  getClient,
  getBucketName,
  uploadObject,
  getObjectStream,
  statObject,
  removeObject,
  getStorageStatus,
};
