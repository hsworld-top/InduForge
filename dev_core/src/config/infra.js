const pickEnvValue = (...values) =>
  values.find((value) => value !== undefined && value !== null && String(value).trim() !== '')

const toNumber = (value, fallback) => {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

const toBoolean = (value, fallback = false) => {
  if (typeof value === 'boolean') return value
  if (typeof value !== 'string') return fallback

  const normalized = value.trim().toLowerCase()
  if (['true', '1', 'yes', 'on'].includes(normalized)) return true
  if (['false', '0', 'no', 'off'].includes(normalized)) return false
  return fallback
}

const buildMetaStoreConfig = () => ({
  host: pickEnvValue(process.env.IF_META_STORE_HOST, process.env.DB_HOST, '127.0.0.1'),
  port: toNumber(pickEnvValue(process.env.IF_META_STORE_PORT, process.env.DB_PORT), 18432),
  user: pickEnvValue(process.env.IF_META_STORE_USER, process.env.DB_USER, 'postgres'),
  password: pickEnvValue(process.env.IF_META_STORE_PASSWORD, process.env.DB_PASSWORD, 'postgres'),
  database: pickEnvValue(process.env.IF_META_STORE_CORE_DB, process.env.DB_NAME, 'if_core'),
  adminDatabase: pickEnvValue(
    process.env.IF_META_STORE_ADMIN_DATABASE,
    process.env.DB_ADMIN_NAME,
    'postgres',
  ),
  connectTimeout: toNumber(process.env.DB_CONNECT_TIMEOUT, 10000),
  queryTimeout: toNumber(process.env.DB_QUERY_TIMEOUT, 30000),
  sslEnabled: toBoolean(pickEnvValue(process.env.IF_META_STORE_SSL, process.env.DB_SSL), false),
})

const buildCacheStoreConfig = () => ({
  host: pickEnvValue(process.env.IF_CACHE_STORE_HOST, process.env.REDIS_HOST, '127.0.0.1'),
  port: toNumber(pickEnvValue(process.env.IF_CACHE_STORE_PORT, process.env.REDIS_PORT), 18379),
  password: pickEnvValue(process.env.IF_CACHE_STORE_PASSWORD, process.env.REDIS_PASSWORD, ''),
  db: toNumber(pickEnvValue(process.env.IF_CACHE_STORE_CORE_DB, process.env.REDIS_DB), 0),
  maxRetries: toNumber(
    pickEnvValue(process.env.IF_CACHE_STORE_MAX_RETRIES, process.env.REDIS_MAX_RETRIES),
    20,
  ),
  degradedRetryInterval: toNumber(
    pickEnvValue(
      process.env.IF_CACHE_STORE_DEGRADED_RETRY_INTERVAL,
      process.env.REDIS_DEGRADED_RETRY_INTERVAL,
    ),
    30000,
  ),
})

const buildObjectStoreConfig = () => ({
  endpoint: pickEnvValue(
    process.env.IF_OBJECT_STORE_ENDPOINT,
    process.env.SEAWEEDFS_ENDPOINT,
    '127.0.0.1',
  ),
  port: toNumber(pickEnvValue(process.env.IF_OBJECT_STORE_PORT, process.env.SEAWEEDFS_PORT), 18500),
  useSSL: toBoolean(
    pickEnvValue(process.env.IF_OBJECT_STORE_USE_SSL, process.env.SEAWEEDFS_USE_SSL),
    false,
  ),
  accessKey: pickEnvValue(
    process.env.IF_OBJECT_STORE_ACCESS_KEY,
    process.env.SEAWEEDFS_ACCESS_KEY,
    '',
  ),
  secretKey: pickEnvValue(
    process.env.IF_OBJECT_STORE_SECRET_KEY,
    process.env.SEAWEEDFS_SECRET_KEY,
    '',
  ),
  bucketIfp: pickEnvValue(
    process.env.IF_OBJECT_STORE_BUCKET_IFP,
    process.env.SEAWEEDFS_BUCKET_IFP,
    'ifp-artifacts',
  ),
  bucketDesign: pickEnvValue(
    process.env.IF_OBJECT_STORE_BUCKET_DESIGN,
    process.env.SEAWEEDFS_BUCKET_DESIGN,
    'design-assets',
  ),
  region: pickEnvValue(
    process.env.IF_OBJECT_STORE_REGION,
    process.env.SEAWEEDFS_REGION,
    'us-east-1',
  ),
  provider: pickEnvValue(
    process.env.OBJECT_STORAGE_PROVIDER,
    process.env.STORAGE_PROVIDER,
    'induforge-object-store',
  ),
})

module.exports = {
  pickEnvValue,
  toBoolean,
  toNumber,
  buildMetaStoreConfig,
  buildCacheStoreConfig,
  buildObjectStoreConfig,
}
