<template>
    <div class="data-source-panel h-full flex flex-col bg-white dark:bg-gray-800">
        <!-- 头部 -->
        <div class="flex items-center justify-between p-3 border-b border-gray-200 dark:border-gray-700">
            <h3 class="text-sm font-medium text-gray-900 dark:text-white">数据源</h3>
            <el-button type="primary" size="small" @click="showAddDialog = true">
                <el-icon class="mr-1"><Plus /></el-icon>
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
                                <el-icon :class="getTypeIcon(ds.type).color">
                                    <component :is="getTypeIcon(ds.type).icon" />
                                </el-icon>
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
                                <el-icon><Refresh /></el-icon>
                            </el-button>
                            <el-button size="small" text @click.stop="editDataSource(ds)">
                                <el-icon><Edit /></el-icon>
                            </el-button>
                            <el-button size="small" text type="danger" @click.stop="removeDataSource(ds.id)">
                                <el-icon><Delete /></el-icon>
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
                <el-form-item label="数据源ID" prop="id">
                    <el-input v-model="form.id" placeholder="ds_example" :disabled="!!editingDataSource" />
                </el-form-item>

                <el-form-item label="类型" prop="type">
                    <el-select v-model="form.type" placeholder="选择类型" class="w-full" @change="onTypeChange">
                        <el-option label="数据中心" value="dataCenter" />
                        <el-option label="HTTP请求" value="http" />
                        <el-option label="静态数据" value="static" />
                        <el-option label="计算数据" value="computed" />
                    </el-select>
                </el-form-item>

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
import { ref, computed, onMounted, watch } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Plus, Refresh, Edit, Delete, Database, Link, Document, Calculator } from '@element-plus/icons-vue';
import { useDesignStore } from '@/store/design';

const store = useDesignStore();

const showAddDialog = ref(false);
const editingDataSource = ref(null);
const selectedDataSourceId = ref(null);
const formRef = ref(null);

const connections = ref([]);
const queries = ref([]);
const loadingConnections = ref(false);
const loadingQueries = ref(false);

const form = ref({
    id: '',
    type: 'dataCenter',
    config: {
        sourceType: 'query',
        queryId: '',
        connectionId: '',
        tags: [],
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

const rules = {
    id: [
        { required: true, message: '请输入数据源ID', trigger: 'blur' },
        { pattern: /^[a-zA-Z_][a-zA-Z0-9_]*$/, message: 'ID只能包含字母、数字和下划线，且以字母或下划线开头', trigger: 'blur' },
    ],
    type: [{ required: true, message: '请选择类型', trigger: 'change' }],
};

const dataSources = computed(() => {
    return Array.from(store.dataSourceManager?.dataSources.values() || []);
});

const getTypeLabel = (type) => {
    const labels = {
        dataCenter: '数据中心',
        http: 'HTTP请求',
        static: '静态数据',
        computed: '计算数据',
    };
    return labels[type] || type;
};

const getTypeIcon = (type) => {
    const icons = {
        dataCenter: { icon: Database, color: 'text-blue-500' },
        http: { icon: Link, color: 'text-green-500' },
        static: { icon: Document, color: 'text-gray-500' },
        computed: { icon: Calculator, color: 'text-purple-500' },
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
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;

    if (diff < 60000) {
        return '刚刚';
    } else if (diff < 3600000) {
        return `${Math.floor(diff / 60000)}分钟前`;
    } else if (diff < 86400000) {
        return `${Math.floor(diff / 3600000)}小时前`;
    } else {
        return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
    }
};

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

const onTypeChange = () => {
    // 重置配置
    form.value.config = {
        sourceType: 'query',
        queryId: '',
        connectionId: '',
        tags: [],
        url: '',
        method: 'GET',
        dataJson: '{}',
        dependencies: [],
        compute: '',
    };
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
        type: 'dataCenter',
        config: {
            sourceType: 'query',
            queryId: '',
            connectionId: '',
            tags: [],
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

watch(
    () => showAddDialog.value,
    (val) => {
        if (val) {
            loadConnections();
            loadQueries();
        }
    },
);

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
