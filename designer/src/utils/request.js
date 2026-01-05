import axios from 'axios';
import { ElMessage } from 'element-plus';
import { Storage } from '@/utils/storage';
import { STORAGE_KEYS } from '@/constants';

// 创建 axios 实例
const request = axios.create({
    baseURL: '/api/v1',
    timeout: 10000,
    headers: {
        'Content-Type': 'application/json',
    },
});

// 刷新token的状态标志
let isRefreshing = false;
let failedQueue = [];

/**
 * 处理失败的请求队列。
 * @param {Error|null} error - 错误
 * @param {string|null} token - 新令牌
 */
const processQueue = (error, token = null) => {
    failedQueue.forEach((prom) => {
        if (error) {
            prom.reject(error);
        } else {
            prom.resolve(token);
        }
    });

    failedQueue = [];
};

/**
 * 刷新访问令牌。
 * @param {string} refreshToken - 刷新令牌
 * @returns {Promise} 刷新结果
 */
const refreshAccessToken = (refreshToken) => {
    return axios.post('/api/v1/auth/refresh', { refreshToken });
};

// 请求拦截器
request.interceptors.request.use(
    (config) => {
        // 添加认证 token（从 LocalStorage 读取，与 dev_ide 共享）
        const token = Storage.getToken();
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }

        // 添加租户 ID
        const tenantId = Storage.getTenantId();
        if (tenantId) {
            config.headers['X-Tenant-ID'] = tenantId;
        }

        return config;
    },
    (error) => {
        return Promise.reject(error);
    },
);

// 响应拦截器
request.interceptors.response.use(
    (response) => {
        return response.data;
    },
    (error) => {
        const { response, config } = error;

        if (response) {
            const { status, data } = response;

            switch (status) {
                case 401:
                    // 刷新 token 失败时直接跳转登录页
                    if (config.url.includes('/auth/refresh')) {
                        Storage.remove(STORAGE_KEYS.TOKEN);
                        Storage.remove(STORAGE_KEYS.REFRESH_TOKEN);
                        Storage.remove(STORAGE_KEYS.USER_INFO);
                        Storage.remove(STORAGE_KEYS.TENANT_ID);
                        window.location.href = '/login';
                        return Promise.reject(error);
                    }

                    const refreshToken = Storage.getRefreshToken();
                    if (!refreshToken) {
                        Storage.remove(STORAGE_KEYS.TOKEN);
                        Storage.remove(STORAGE_KEYS.REFRESH_TOKEN);
                        Storage.remove(STORAGE_KEYS.USER_INFO);
                        Storage.remove(STORAGE_KEYS.TENANT_ID);
                        window.location.href = '/login';
                        return Promise.reject(error);
                    }

                    if (!isRefreshing) {
                        isRefreshing = true;

                        return refreshAccessToken(refreshToken)
                            .then((result) => {
                                const payload = result?.data?.data || result?.data || {};
                                const { accessToken, refreshToken: newRefreshToken } = payload;

                                Storage.setToken(accessToken);
                                if (newRefreshToken) {
                                    Storage.setRefreshToken(newRefreshToken);
                                }

                                processQueue(null, accessToken);

                                config.headers.Authorization = `Bearer ${accessToken}`;
                                return request(config);
                            })
                            .catch((refreshError) => {
                                processQueue(refreshError, null);
                                Storage.remove(STORAGE_KEYS.TOKEN);
                                Storage.remove(STORAGE_KEYS.REFRESH_TOKEN);
                                Storage.remove(STORAGE_KEYS.USER_INFO);
                                Storage.remove(STORAGE_KEYS.TENANT_ID);
                                window.location.href = '/login';
                                return Promise.reject(refreshError);
                            })
                            .finally(() => {
                                isRefreshing = false;
                            });
                    }

                    return new Promise((resolve, reject) => {
                        failedQueue.push({
                            resolve: (token) => {
                                config.headers.Authorization = `Bearer ${token}`;
                                resolve(request(config));
                            },
                            reject,
                        });
                    });
                case 403:
                    ElMessage.error('没有权限访问此资源');
                    break;
                case 404:
                    ElMessage.error('请求的资源不存在');
                    break;
                case 422:
                    // 验证错误
                    if (data.errors) {
                        const errorMessages = Object.values(data.errors).flat();
                        ElMessage.error(errorMessages.join('; '));
                    } else {
                        ElMessage.error(data.message || '请求参数错误');
                    }
                    break;
                case 500:
                    ElMessage.error('服务器内部错误');
                    break;
                default:
                    ElMessage.error(data.message || '请求失败');
            }
        } else {
            // 网络错误
            ElMessage.error('网络连接失败，请检查网络设置');
        }

        return Promise.reject(error);
    },
);

export default request;
