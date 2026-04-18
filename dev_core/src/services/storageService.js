const Minio = require("minio");
const dayjs = require("dayjs");
const { logger } = require("../utils/logger");
const { TIME_FORMAT } = require("../constants/time");

let storageClient = null;
const storageStatus = {
  connected: false,
  lastError: null,
  lastErrorTime: null,
};

const pickEnvValue = (...values) =>
  values.find((value) => value !== undefined && value !== null && String(value).trim() !== "");

const parsePort = (value, defaultPort) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : defaultPort;
};

const parseBoolean = (value, defaultValue = false) => {
  if (typeof value === "boolean") {
    return value;
  }

  if (typeof value !== "string") {
    return defaultValue;
  }

  const normalized = value.trim().toLowerCase();
  if (["true", "1", "yes", "on"].includes(normalized)) {
    return true;
  }
  if (["false", "0", "no", "off"].includes(normalized)) {
    return false;
  }

  return defaultValue;
};

const getStorageConfig = () => {
  const endpoint = pickEnvValue(process.env.SEAWEEDFS_ENDPOINT, process.env.MINIO_ENDPOINT, "127.0.0.1");
  const port = parsePort(pickEnvValue(process.env.SEAWEEDFS_PORT, process.env.MINIO_PORT), 25000);
  const useSSL = parseBoolean(pickEnvValue(process.env.SEAWEEDFS_USE_SSL, process.env.MINIO_USE_SSL), false);
  const accessKey = pickEnvValue(process.env.SEAWEEDFS_ACCESS_KEY, process.env.MINIO_ACCESS_KEY, "");
  const secretKey = pickEnvValue(process.env.SEAWEEDFS_SECRET_KEY, process.env.MINIO_SECRET_KEY, "");
  const bucketIfp = pickEnvValue(process.env.SEAWEEDFS_BUCKET_IFP, process.env.MINIO_BUCKET_IFP, "ifp-artifacts");
  const bucketDesign = pickEnvValue(
    process.env.SEAWEEDFS_BUCKET_DESIGN,
    process.env.MINIO_BUCKET_DESIGN,
    "design-assets"
  );
  const region = pickEnvValue(process.env.SEAWEEDFS_REGION, process.env.MINIO_REGION, "us-east-1");
  const provider = pickEnvValue(process.env.OBJECT_STORAGE_PROVIDER, process.env.STORAGE_PROVIDER, "seaweedfs");

  return {
    endpoint,
    port,
    useSSL,
    accessKey,
    secretKey,
    bucketIfp,
    bucketDesign,
    region,
    provider,
  };
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

const ensureBucket = async (client, bucket, region, provider) => {
  const exists = await client.bucketExists(bucket);
  if (!exists) {
    await client.makeBucket(bucket, region);
    logger.info("对象存储 bucket 已创建", { provider, bucket, region });
  }
};

/**
 * 初始化对象存储连接并确保 Bucket 存在
 * @returns {Promise<Minio.Client>}
 */
const initStorage = async () => {
  if (storageClient) {
    return storageClient;
  }

  const { bucketIfp, bucketDesign, region, provider } = getStorageConfig();
  const client = buildClient();

  try {
    await ensureBucket(client, bucketIfp, region, provider);
    await ensureBucket(client, bucketDesign, region, provider);
    storageStatus.connected = true;
    storageStatus.lastError = null;
    storageStatus.lastErrorTime = null;
    storageClient = client;
    return storageClient;
  } catch (error) {
    storageStatus.connected = false;
    storageStatus.lastError = error.message;
    storageStatus.lastErrorTime = dayjs().format(TIME_FORMAT);
    throw error;
  }
};

const getClient = async () => {
  if (!storageClient) {
    return initStorage();
  }
  return storageClient;
};

const getBucketName = (type = "ifp") => {
  const { bucketIfp, bucketDesign } = getStorageConfig();
  return type === "design" ? bucketDesign : bucketIfp;
};

/**
 * 上传对象到对象存储
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

/**
 * 复制对象
 * @param {string} bucketType
 * @param {string} sourceKey
 * @param {string} targetKey
 */
const copyObject = async (bucketType, sourceKey, targetKey) => {
  const client = await getClient();
  const bucket = getBucketName(bucketType);
  const source = `/${bucket}/${sourceKey}`;
  return client.copyObject(bucket, targetKey, source);
};

const getStorageStatus = () => {
  const { provider, endpoint, port } = getStorageConfig();
  return {
    ...storageStatus,
    provider,
    endpoint,
    port,
  };
};

module.exports = {
  initStorage,
  getClient,
  getBucketName,
  uploadObject,
  getObjectStream,
  statObject,
  removeObject,
  copyObject,
  getStorageStatus,
};
