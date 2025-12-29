<template>
    <div class="props-editor">
        <el-form v-if="hasProps" label-position="left" label-width="80px" size="small">
            <template v-for="(schema, key) in propsSchema" :key="key">
                <el-form-item v-if="isPropVisible(schema)" :label="schema.label || key">
                    <div v-if="schema.format === 'json'" class="json-config-row">
                        <el-button
                            size="small"
                            :type="isJsonConfigured(key) ? 'success' : 'primary'"
                            @click="openJsonDialog(key, schema)">
                            {{ isJsonConfigured(key) ? '已配置' : '配置' }}
                        </el-button>
                    </div>

                    <!-- Boolean 类型 - Requirements: 4.4 -->
                    <el-switch v-else-if="schema.type === 'boolean'" :model-value="getPropValue(key, schema)" @change="(val) => handlePropChange(key, val)" />

                    <!-- Number 类型 - Requirements: 4.5 -->
                    <el-input-number
                        v-else-if="schema.type === 'number'"
                        :model-value="getPropValue(key, schema)"
                        :min="schema.min"
                        :max="schema.max"
                        :step="schema.step || 1"
                        @change="(val) => handlePropChange(key, val)" />

                    <!-- Enum 类型 - Requirements: 4.7 -->
                    <el-select v-else-if="schema.type === 'enum'" :model-value="getPropValue(key, schema)" @change="(val) => handlePropChange(key, val)">
                        <el-option v-for="opt in schema.options" :key="opt.value" :label="opt.label" :value="opt.value" />
                    </el-select>

                    <!-- String 类型 (多行) - Requirements: 4.6 -->
                    <el-input
                        v-else-if="schema.type === 'string' && schema.multiline"
                        type="textarea"
                        :rows="3"
                        :model-value="getPropValue(key, schema)"
                        @change="(val) => handlePropChange(key, val)" />

                    <!-- String 类型 (颜色) -->
                    <el-color-picker v-else-if="schema.type === 'string' && schema.format === 'color'" :model-value="getPropValue(key, schema)" @change="(val) => handlePropChange(key, val)" />

                    <!-- String 类型 (默认) - Requirements: 4.6 -->
                    <el-input v-else :model-value="getPropValue(key, schema)" :placeholder="schema.description" @change="(val) => handlePropChange(key, val)" />

                    <!-- 描述信息 -->
                    <div v-if="schema.description" class="prop-description">
                        {{ schema.description }}
                    </div>
                </el-form-item>
            </template>
        </el-form>

        <!-- 空状态 -->
        <div v-else class="empty-state">
            <span class="empty-text">无可编辑属性</span>
        </div>

        <el-dialog v-model="jsonDialogVisible" :title="jsonDialogTitle" width="720px" draggable :lock-scroll="false">
            <div class="json-config-dialog">
                <MonacoEditor v-model="jsonInput" language="json" :theme="monacoTheme" height="320px" />
                <div v-if="jsonError" class="error-tip">{{ jsonError }}</div>
            </div>
            <template #footer>
                <el-button @click="jsonDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="handleJsonSave">确定</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
/**
 * PropsEditor - 属性编辑器组件
 * 根据 propsSchema 动态生成表单
 * 支持 boolean/number/string/enum 类型
 * Requirements: 4.1, 4.2, 4.4, 4.5, 4.6, 4.7
 */
import { computed, ref, watch, onBeforeUnmount } from 'vue';
import MonacoEditor from '@/components/common/MonacoEditor.vue';

const props = defineProps({
    /**
     * 组件属性对象
     */
    props: {
        type: Object,
        default: () => ({}),
    },
    /**
     * 属性 Schema 定义
     * Requirements: 4.1 - 显示组件属性表单
     */
    propsSchema: {
        type: Object,
        default: () => ({}),
    },
});

const emit = defineEmits(['update:props', 'change']);

const jsonDialogVisible = ref(false);
const jsonDialogKey = ref('');
const jsonDialogLabel = ref('');
const jsonInput = ref('');
const jsonError = ref('');
const monacoTheme = ref('vs');
let mediaQuery;
let mediaHandler;

function detectTheme() {
    if (typeof document !== 'undefined') {
        const html = document.documentElement;
        const body = document.body;
        const isDark = (el) => el && (el.classList?.contains('dark') || el.dataset?.theme === 'dark');
        if (isDark(html) || isDark(body)) return 'vs-dark';
    }
    if (typeof window !== 'undefined' && window.matchMedia) {
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'vs-dark' : 'vs';
    }
    return 'vs';
}

monacoTheme.value = detectTheme();
if (typeof window !== 'undefined' && window.matchMedia) {
    mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaHandler = (e) => {
        monacoTheme.value = e.matches ? 'vs-dark' : 'vs';
    };
    mediaQuery.addEventListener('change', mediaHandler);
}

onBeforeUnmount(() => {
    if (mediaQuery && mediaHandler) {
        mediaQuery.removeEventListener('change', mediaHandler);
    }
});

watch(
    () => jsonInput.value,
    () => {
        jsonError.value = '';
    },
);

/**
 * 是否有可编辑的属性
 */
const hasProps = computed(() => {
    return props.propsSchema && Object.keys(props.propsSchema).length > 0;
});

const jsonDialogTitle = computed(() => {
    if (jsonDialogLabel.value) return `${jsonDialogLabel.value} 配置`;
    return 'JSON 配置';
});

/**
 * 获取属性值
 * @param {string} key - 属性名
 * @param {Object} schema - 属性 schema
 * @returns {any} 属性值
 */
function getByPath(obj, path) {
    if (!obj) return undefined;
    return path.split('.').reduce((acc, k) => (acc && acc[k] !== undefined ? acc[k] : undefined), obj);
}

function setByPath(obj, path, value) {
    const parts = path.split('.');
    const next = { ...(obj || {}) };
    let cur = next;
    parts.forEach((p, idx) => {
        if (idx === parts.length - 1) {
            cur[p] = value;
        } else {
            cur[p] = { ...(cur[p] || {}) };
            cur = cur[p];
        }
    });
    return next;
}

function getPropValue(key, schema) {
    const value = getByPath(props.props, key);
    // 如果值为 undefined，返回默认值
    if (value === undefined && schema?.default !== undefined) {
        return schema.default;
    }
    if (schema?.format === 'json' && typeof value === 'object') {
        try {
            return JSON.stringify(value, null, 2);
        } catch (err) {
            return '';
        }
    }
    return value;
}

function getJsonEditorValue(key, schema) {
    const value = getByPath(props.props, key);
    if ((value === undefined || value === null) && schema?.default !== undefined) {
        if (typeof schema.default === 'string') return schema.default;
        try {
            return JSON.stringify(schema.default, null, 2);
        } catch (err) {
            return '';
        }
    }
    if (value === undefined || value === null) return '';
    if (typeof value === 'string') return value;
    try {
        return JSON.stringify(value, null, 2);
    } catch (err) {
        return '';
    }
}

function isJsonConfigured(key) {
    const value = getByPath(props.props, key);
    if (value === undefined || value === null) return false;
    if (typeof value === 'string') return Boolean(value.trim());
    if (Array.isArray(value)) return value.length > 0;
    if (typeof value === 'object') return Object.keys(value).length > 0;
    return true;
}

function openJsonDialog(key, schema) {
    jsonDialogKey.value = key;
    jsonDialogLabel.value = schema?.label || key;
    jsonInput.value = getJsonEditorValue(key, schema);
    jsonError.value = '';
    jsonDialogVisible.value = true;
}

function handleJsonSave() {
    if (!jsonDialogKey.value) return;
    const text = (jsonInput.value || '').trim();
    if (!text) {
        jsonError.value = '';
        handlePropChange(jsonDialogKey.value, '');
        jsonDialogVisible.value = false;
        return;
    }
    try {
        const parsed = JSON.parse(text);
        jsonError.value = '';
        handlePropChange(jsonDialogKey.value, parsed);
        jsonDialogVisible.value = false;
    } catch (err) {
        jsonError.value = err?.message || 'JSON 解析失败，请输入有效的 JSON';
    }
}

/**
 * 判断属性是否可见
 * @param {Object} schema - 属性 schema
 * @returns {boolean} 是否可见
 */
function isPropVisible(schema) {
    if (typeof schema.visible === 'function') {
        return schema.visible(props.props || {});
    }
    return true;
}

/**
 * 处理属性变更
 * Requirements: 4.2 - 修改属性时更新 schema
 * @param {string} key - 属性名
 * @param {any} value - 新值
 */
function handlePropChange(key, value) {
    let finalValue = value;
    const schema = props.propsSchema[key];
    if (schema?.format === 'json' && typeof value === 'string') {
        const trimmed = value.trim();
        if (!trimmed) {
            finalValue = '';
        } else {
            try {
                finalValue = JSON.parse(trimmed);
            } catch (err) {
                // 保留原始字符串，避免直接清空
                finalValue = value;
            }
        }
    }
    const newProps = setByPath(props.props, key, finalValue);
    emit('update:props', newProps);
    emit('change', key, finalValue);
}
</script>

<style scoped>
.props-editor {
    padding: 12px;
}

.empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
}

.empty-text {
    font-size: 12px;
    color: #909399;
}

.prop-description {
    font-size: 11px;
    color: #909399;
    margin-top: 4px;
    line-height: 1.4;
}

.json-config-row {
    display: flex;
    justify-content: flex-end;
    width: 100%;
}

.json-config-dialog {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.error-tip {
    color: #f56c6c;
    font-size: 12px;
}

:deep(.el-form-item) {
    margin-bottom: 12px;
}

:deep(.el-form-item:last-child) {
    margin-bottom: 0;
}

:deep(.el-form-item__label) {
    font-size: 12px;
    color: #606266;
}

:deep(.el-input-number) {
    width: 100%;
}

:deep(.el-select) {
    width: 100%;
}
</style>
