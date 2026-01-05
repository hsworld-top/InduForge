<template>
    <div class="data-source-panel h-full flex flex-col bg-white dark:bg-gray-800">
        <!-- 头部 -->
        <div class="flex items-center justify-between p-3 border-b border-gray-200 dark:border-gray-700">
            <h3 class="text-sm font-medium text-gray-900 dark:text-white">数据源</h3>
            <el-button type="primary" size="small" @click="showAddDialog = true">
                <IconTablerPlus class="mr-1" />
                添加
            </el-button>
        </div>

        <!-- 数据源列表 -->
        <div class="flex-1 overflow-y-auto p-3">
            <div v-if="dataSources.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
                <div class="text-sm">暂无数据源</div>
                <div class="text-xs mt-1">点击"添加"按钮创建数据源</div>
            </div>

            <div v-else class="space-y-2">
                <div
                    v-for="ds in dataSources"
                    :key="ds.id"
                    :class="[
                        'p-3 rounded-lg border cursor-pointer transition-all',
                        selectedDataSourceId === ds.id ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20' : 'border-gray-200 dark:border-gray-700 hover:border-blue-300 dark:hover:border-blue-600',
                    ]"
                    @click="selectDataSource(ds.id)">
                    <div class="flex items-start justify-between">
                        <div class="flex-1 min-w-0">
                            <div class="flex items-center space-x-2">
                                <component :is="getTypeIcon(ds.type).icon" :class="getTypeIcon(ds.type).color" />
                                <span class="text-sm font-medium text-gray-900 dark:text-white truncate">
                                    {{ ds.id }}
                                </span>
                                <el-tag :type="getStatusType(ds.status)" size="small">
                                    {{ getStatusLabel(ds.status) }}
                                </el-tag>
                            </div>
                            <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                                {{ getTypeLabel(ds.type) }}
                            </div>
                            <div v-if="ds.lastUpdate" class="text-xs text-gray-400 dark:text-gray-500 mt-1">更新: {{ formatTime(ds.lastUpdate) }}</div>
                        </div>
                        <div class="flex items-center space-x-1 ml-2">
                            <el-button size="small" text @click.stop="refreshDataSource(ds.id)" :loading="ds.status === 'loading'">
                                <IconTablerRefresh />
                            </el-button>
                            <el-button size="small" text @click.stop="editDataSource(ds)">
                                <IconTablerPencil />
                            </el-button>
                            <el-button size="small" text type="danger" @click.stop="removeDataSource(ds.id)">
                                <IconTablerTrash />
                            </el-button>
                        </div>
                    </div>

                    <div v-if="ds.error" class="mt-2 text-xs text-red-500 dark:text-red-400">错误: {{ ds.error }}</div>
                </div>
            </div>
        </div>

        <!-- 添加/编辑数据源对话框 -->
        <el-dialog v-model="showAddDialog" :title="editingDataSource ? '编辑数据源' : '添加数据源'" width="600px" :close-on-click-modal="false" :lock-scroll="false">
            <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">

                <!-- DataCenter配置 -->
                <template v-if="form.type === 'dataCenter'">
                        <el-form-item label="数据类型" prop="config.sourceType">
                        <el-select v-model="form.config.sourceType" placeholder="选择数据类型" class="w-full">
                            <el-option label="查询" value="query" />
                            <el-option label="点位订阅" value="tags" />
                        </el-select>
                    </el-form-item>

                    <template v-if="form.config.sourceType === 'query'">
                        <el-form-item label="查询" prop="config.queryId">
                            <el-select v-model="form.config.queryId" placeholder="选择查询" class="w-full" filterable :loading="loadingQueries">
                                <el-option v-for="query in queries" :key="query.id" :label="query.name" :value="query.id" />
                            </el-select>
                        </el-form-item>
                    </template>

                    <template v-if="form.config.sourceType === 'tags'">
                        <el-form-item label="连接" prop="config.connectionId">
                            <el-select v-model="form.config.connectionId" placeholder="选择连接" class="w-full" filterable :loading="loadingConnections">
                                <el-option v-for="conn in connections" :key="conn.id" :label="conn.name" :value="conn.id" />
                            </el-select>
                        </el-form-item>

                        <el-form-item label="点位标签" prop="config.tags">
                            <el-select v-model="form.config.tags" placeholder="输入点位标签" class="w-full" multiple filterable allow-create />
                        </el-form-item>
                    </template>

                    <el-form-item label="模式" prop="mode">
                        <el-radio-group v-model="form.mode">
                            <el-radio label="request">单次请求</el-radio>
                            <el-radio label="poll">轮询</el-radio>
                            <el-radio label="subscription">订阅</el-radio>
                        </el-radio-group>
                    </el-form-item>

                    <el-form-item v-if="form.mode !== 'request'" label="间隔(ms)" prop="interval">
                        <el-input-number v-model="form.interval" :min="1000" :step="1000" class="w-full" />
                    </el-form-item>
                </template>

                <!-- 数据点配置 -->
                <template v-if="form.type === 'datapoint'">
                    <el-form-item label="来源类型">
                        <el-select v-model="datapointTypeFilter" placeholder="全部类型" class="w-full" @change="loadDataPoints">
                            <el-option label="全部类型" value="" />
                            <el-option label="关系库查询" value="db.query" />
                            <el-option label="MQTT 主题" value="mqtt.subscription" />
                            <el-option label="MQTT 变量" value="mqtt.tag" />
                            <el-option label="计算输出" value="calc.output" />
                        </el-select>
                    </el-form-item>

                    <el-form-item label="搜索">
                        <el-input v-model="datapointSearch" placeholder="搜索名称或路径" clearable />
                    </el-form-item>

                    <el-form-item label="数据点" prop="config.datapointPath">
                        <el-select
                            v-model="form.config.datapointPath"
                            placeholder="选择数据点"
                            class="w-full"
                            filterable
                            :loading="loadingDatapoints"
                            @change="handleDatapointChange">
                            <el-option-group v-for="group in datapointGroups" :key="group.key" :label="group.label">
                                <el-option v-for="item in group.items" :key="item.id" :label="item.label" :value="item.path" />
                            </el-option-group>
                        </el-select>
                    </el-form-item>

                    <el-form-item label="模式" prop="mode">
                        <el-radio-group v-model="form.mode">
                            <el-radio label="request">单次请求</el-radio>
                            <el-radio label="poll">轮询</el-radio>
                            <el-radio label="subscription">订阅</el-radio>
                        </el-radio-group>
                    </el-form-item>

                    <el-form-item v-if="form.mode !== 'request'" label="间隔(ms)" prop="interval">
                        <el-input-number v-model="form.interval" :min="1000" :step="1000" class="w-full" />
                    </el-form-item>
                </template>

                <!-- HTTP配置 -->
                <template v-if="form.type === 'http'">
                    <el-form-item label="URL" prop="config.url">
                        <el-input v-model="form.config.url" placeholder="https://api.example.com/data" />
                    </el-form-item>

                    <el-form-item label="方法" prop="config.method">
                        <el-select v-model="form.config.method" class="w-full">
                            <el-option label="GET" value="GET" />
                            <el-option label="POST" value="POST" />
                            <el-option label="PUT" value="PUT" />
                            <el-option label="DELETE" value="DELETE" />
                        </el-select>
                    </el-form-item>

                    <el-form-item label="模式" prop="mode">
                        <el-radio-group v-model="form.mode">
                            <el-radio label="request">单次请求</el-radio>
                            <el-radio label="poll">轮询</el-radio>
                        </el-radio-group>
                    </el-form-item>

                    <el-form-item v-if="form.mode === 'poll'" label="间隔(ms)" prop="interval">
                        <el-input-number v-model="form.interval" :min="1000" :step="1000" class="w-full" />
                    </el-form-item>
                </template>

                <!-- Static配置 -->
                <template v-if="form.type === 'static'">
                    <el-form-item label="数据" prop="config.data">
                        <el-input v-model="form.config.dataJson" type="textarea" :rows="6" placeholder='{"key": "value"}' />
                    </el-form-item>
                </template>

                <!-- Computed配置 -->
                <template v-if="form.type === 'computed'">
                    <el-form-item label="依赖" prop="config.dependencies">
                        <el-select v-model="form.config.dependencies" placeholder="选择依赖的数据源" class="w-full" multiple filterable>
                            <el-option v-for="ds in dataSources" :key="ds.id" :label="ds.id" :value="ds.id" />
                        </el-select>
                    </el-form-item>

                    <el-form-item label="计算函数" prop="config.compute">
                        <el-input v-model="form.config.compute" type="textarea" :rows="6" placeholder="(sources) => ({ result: sources.ds1.value + sources.ds2.value })" />
                    </el-form-item>
                </template>

                <el-form-item label="自动启动">
                    <el-switch v-model="form.options.autoStart" />
                </el-form-item>
            </el-form>

            <template #footer>
                <el-button @click="showAddDialog = false">取消</el-button>
                <el-button type="primary" @click="saveDataSource">保存</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, computed, onBeforeUnmount, onMounted, watch } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import IconTablerPlus from '~icons/tabler/plus';
import IconTablerRefresh from '~icons/tabler/refresh';
import IconTablerPencil from '~icons/tabler/pencil';
import IconTablerTrash from '~icons/tabler/trash';
import IconTablerDatabase from '~icons/tabler/database';
import IconTablerLink from '~icons/tabler/link';
import IconTablerFileText from '~icons/tabler/file-text';
import IconTablerChartBar from '~icons/tabler/chart-bar';
import { useDesignStore } from '@/store/design';
import dayjs from 'dayjs';
import { TIME_FORMAT } from '@/constants';

const store = useDesignStore();

const showAddDialog = ref(false);
const editingDataSource = ref(null);
const selectedDataSourceId = ref(null);
const formRef = ref(null);

const connections = ref([]);
const queries = ref([]);
const loadingConnections = ref(false);
const loadingQueries = ref(false);
const datapoints = ref([]);
const loadingDatapoints = ref(false);
const datapointTypeFilter = ref('');
const datapointSearch = ref('');
const datapointSearchTimer = ref(null);

const form = ref({
    id: '',
    type: 'datapoint',
    config: {
        sourceType: 'query',
        queryId: '',
        connectionId: '',
        tags: [],
        datapointId: '',
        datapointPath: '',
        datapointType: '',
        url: '',
        method: 'GET',
        dataJson: '{}',
        dependencies: [],
        compute: '',
    },
    mode: 'request',
    interval: 5000,
    options: {
        autoStart: true,
    },
});

/**
 * 校验数据点配置
 * @param {Object} rule - 校验规则
 * @param {string} value - 当前值
 * @param {Function} callback - 回调
 */
const validateDatapointPath = (rule, value, callback) => {
    if (form.value.type === 'datapoint' && !form.value.config.datapointPath) {
        callback(new Error('请选择数据点'));
        return;
    }
    callback();
};

const rules = {
    'config.datapointPath': [{ validator: validateDatapointPath, trigger: 'change' }],
};

const dataSources = computed(() => {
    return Array.from(store.dataSourceManager?.dataSources.values() || []);
});

const getTypeLabel = (type) => {
    const labels = {
        dataCenter: '数据中心',
        datapoint: '数据点',
        http: 'HTTP请求',
        static: '静态数据',
        computed: '计算数据',
    };
    return labels[type] || type;
};

const getTypeIcon = (type) => {
    const icons = {
        dataCenter: { icon: IconTablerDatabase, color: 'text-blue-500' },
        datapoint: { icon: IconTablerDatabase, color: 'text-indigo-500' },
        http: { icon: IconTablerLink, color: 'text-green-500' },
        static: { icon: IconTablerFileText, color: 'text-gray-500' },
        computed: { icon: IconTablerChartBar, color: 'text-purple-500' },
    };
    return icons[type] || { icon: Database, color: 'text-gray-500' };
};

const getStatusLabel = (status) => {
    const labels = {
        idle: '空闲',
        loading: '加载中',
        ready: '就绪',
        error: '错误',
    };
    return labels[status] || status;
};

const getStatusType = (status) => {
    const types = {
        idle: 'info',
        loading: 'warning',
        ready: 'success',
        error: 'danger',
    };
    return types[status] || 'info';
};

const formatTime = (timestamp) => {
    const date = dayjs(timestamp);
    if (!date.isValid()) return '-';
    const diff = dayjs().diff(date);

    if (diff < 60000) {
        return '刚刚';
    } else if (diff < 3600000) {
        return `${Math.floor(diff / 60000)}分钟前`;
    } else if (diff < 86400000) {
        return `${Math.floor(diff / 3600000)}小时前`;
    } else {
        return date.format(TIME_FORMAT);
    }
};

const datapointGroups = computed(() => {
    const keyword = datapointSearch.value.trim().toLowerCase();
    const filtered = datapoints.value.filter((item) => {
        if (!keyword) return true;
        const name = item.name || '';
        const path = item.path || '';
        return name.toLowerCase().includes(keyword) || path.toLowerCase().includes(keyword);
    });

    const groups = new Map();
    filtered.forEach((item) => {
        const key = item.sourceType || 'other';
        if (!groups.has(key)) {
            groups.set(key, []);
        }
        groups.get(key).push({
            id: item.id,
            path: item.path,
            label: item.name ? `${item.name} (${item.path})` : item.path,
            sourceType: item.sourceType,
        });
    });

    const typeLabels = {
        'db.query': '关系库查询',
        'mqtt.subscription': 'MQTT 主题',
        'mqtt.tag': 'MQTT 变量',
        'calc.output': '计算输出',
    };

    return Array.from(groups.entries()).map(([key, items]) => ({
        key,
        label: typeLabels[key] || key,
        items,
    }));
});

const selectDataSource = (id) => {
    selectedDataSourceId.value = id;
};

const refreshDataSource = async (id) => {
    try {
        await store.dataSourceManager.refresh(id);
        ElMessage.success('刷新成功');
    } catch (error) {
        ElMessage.error('刷新失败: ' + error.message);
    }
};

const editDataSource = (ds) => {
    editingDataSource.value = ds;
    form.value = {
        id: ds.id,
        type: ds.type,
        config: { ...ds.config },
        mode: ds.mode,
        interval: ds.interval,
        options: { autoStart: true },
    };

    if (ds.type === 'static' && ds.config.data) {
        form.value.config.dataJson = JSON.stringify(ds.config.data, null, 2);
    }

    if (ds.type === 'datapoint') {
        datapointTypeFilter.value = ds.config?.datapointType || '';
    }

    showAddDialog.value = true;
};

const removeDataSource = async (id) => {
    try {
        await ElMessageBox.confirm('确定要删除此数据源吗？', '确认删除', {
            type: 'warning',
            lockScroll: false,
        });

        store.dataSourceManager.unregister(id);

        // 从页面配置中移除
        if (store.currentPage?.dataSources) {
            const index = store.currentPage.dataSources.findIndex((ds) => ds.id === id);
            if (index !== -1) {
                store.currentPage.dataSources.splice(index, 1);
                store.isDirty = true;
            }
        }

        ElMessage.success('删除成功');
    } catch (error) {
        if (error !== 'cancel') {
            ElMessage.error('删除失败: ' + error.message);
        }
    }
};

const saveDataSource = async () => {
    try {
        await formRef.value.validate();

        const config = { ...form.value };

        // 处理静态数据
        if (config.type === 'static') {
            try {
                config.config.data = JSON.parse(config.config.dataJson);
                delete config.config.dataJson;
            } catch (error) {
                ElMessage.error('数据格式错误，请输入有效的JSON');
                return;
            }
        }

        // 添加到页面配置
        if (!store.currentPage.dataSources) {
            store.currentPage.dataSources = [];
        }

        if (!config.id) {
            config.id = buildDatapointId(config.config?.datapointId || config.config?.datapointPath || '');
        }

        const existingIndex = store.currentPage.dataSources.findIndex((ds) => ds.id === config.id);
        if (existingIndex !== -1) {
            store.currentPage.dataSources[existingIndex] = config;
        } else {
            store.currentPage.dataSources.push(config);
        }

        // 注册到管理器
        if (editingDataSource.value) {
            store.dataSourceManager.unregister(config.id);
        }
        store.dataSourceManager.register(config);

        store.isDirty = true;
        showAddDialog.value = false;
        editingDataSource.value = null;
        resetForm();

        ElMessage.success(editingDataSource.value ? '更新成功' : '添加成功');
    } catch (error) {
        console.error('Save data source error:', error);
    }
};

const resetForm = () => {
    form.value = {
        id: '',
        type: 'datapoint',
        config: {
            sourceType: 'query',
            queryId: '',
            connectionId: '',
            tags: [],
            datapointId: '',
            datapointPath: '',
            datapointType: '',
            url: '',
            method: 'GET',
            dataJson: '{}',
            dependencies: [],
            compute: '',
        },
        mode: 'request',
        interval: 5000,
        options: {
            autoStart: true,
        },
    };
    datapointTypeFilter.value = '';
    datapointSearch.value = '';
};

const loadConnections = async () => {
    if (!store.projectId) return;

    loadingConnections.value = true;
    try {
        // 使用DataSourceManager的API方法
        if (store.dataSourceManager) {
            connections.value = await store.dataSourceManager.getConnectionsAPI(store.projectId);
        }
    } catch (error) {
        console.error('Load connections error:', error);
    } finally {
        loadingConnections.value = false;
    }
};

const loadQueries = async () => {
    if (!store.projectId) return;

    loadingQueries.value = true;
    try {
        // 使用DataSourceManager的API方法
        if (store.dataSourceManager) {
            queries.value = await store.dataSourceManager.getQueriesAPI(store.projectId);
        }
    } catch (error) {
        console.error('Load queries error:', error);
    } finally {
        loadingQueries.value = false;
    }
};

/**
 * 加载数据点列表
 */
const loadDataPoints = async () => {
    if (!store.projectId) return;

    loadingDatapoints.value = true;
    try {
        if (store.dataSourceManager) {
            datapoints.value = await store.dataSourceManager.getDataPointsAPI(store.projectId, {
                page: 1,
                pageSize: 200,
                status: 'active',
                type: datapointTypeFilter.value || undefined,
                search: datapointSearch.value || undefined,
            });
        }
    } catch (error) {
        console.error('Load datapoints error:', error);
    } finally {
        loadingDatapoints.value = false;
    }
};

/**
 * 处理数据点选择
 * @param {string} path - 数据点路径
 */
const handleDatapointChange = (path) => {
    const selected = datapoints.value.find((item) => item.path === path);
    if (!selected) {
        form.value.config.datapointId = '';
        form.value.config.datapointType = '';
        return;
    }

    form.value.config.datapointId = selected.id;
    form.value.config.datapointType = selected.sourceType;
    if (!editingDataSource.value) {
        form.value.id = buildDatapointId(selected.id);
    }
};

/**
 * 构建数据点数据源ID
 * @param {string} seed - 数据点ID或路径
 * @returns {string}
 */
const buildDatapointId = (seed) => {
    const normalized = String(seed || '').replace(/[^a-zA-Z0-9_]/g, '_');
    const baseId = normalized ? `dp_${normalized}` : `dp_${Date.now()}`;
    const existingIds = new Set(dataSources.value.map((ds) => ds.id));

    if (!existingIds.has(baseId)) {
        return baseId;
    }

    let index = 1;
    let candidate = `${baseId}_${index}`;
    while (existingIds.has(candidate)) {
        index += 1;
        candidate = `${baseId}_${index}`;
    }
    return candidate;
};

watch(
    () => showAddDialog.value,
    (val) => {
        if (val) {
            loadConnections();
            loadQueries();
            if (form.value.type === 'datapoint') {
                loadDataPoints();
            }
        }
    },
);

watch(
    () => form.value.type,
    (type) => {
        if (showAddDialog.value && type === 'datapoint') {
            loadDataPoints();
        }
    },
);

watch(
    () => datapointSearch.value,
    () => {
        if (datapointSearchTimer.value) {
            clearTimeout(datapointSearchTimer.value);
        }
        datapointSearchTimer.value = setTimeout(() => {
            if (showAddDialog.value && form.value.type === 'datapoint') {
                loadDataPoints();
            }
        }, 300);
    },
);

onBeforeUnmount(() => {
    if (datapointSearchTimer.value) {
        clearTimeout(datapointSearchTimer.value);
    }
});

onMounted(() => {
    // 初始化已有的数据源
    if (store.currentPage?.dataSources) {
        store.currentPage.dataSources.forEach((ds) => {
            if (!store.dataSourceManager.dataSources.has(ds.id)) {
                store.dataSourceManager.register(ds);
            }
        });
    }
});
</script>

<style scoped>
.data-source-panel {
    min-width: 280px;
}
</style>
