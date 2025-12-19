<template>
    <div class="props-editor">
        <el-form v-if="hasProps" label-position="left" label-width="80px" size="small">
            <template v-for="(schema, key) in propsSchema" :key="key">
                <el-form-item v-if="isPropVisible(schema)" :label="schema.label || key">
                    <!-- Boolean 类型 - Requirements: 4.4 -->
                    <el-switch v-if="schema.type === 'boolean'" :model-value="getPropValue(key, schema)" @change="(val) => handlePropChange(key, val)" />

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
    </div>
</template>

<script setup>
/**
 * PropsEditor - 属性编辑器组件
 * 根据 propsSchema 动态生成表单
 * 支持 boolean/number/string/enum 类型
 * Requirements: 4.1, 4.2, 4.4, 4.5, 4.6, 4.7
 */
import { computed } from 'vue';

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

/**
 * 是否有可编辑的属性
 */
const hasProps = computed(() => {
    return props.propsSchema && Object.keys(props.propsSchema).length > 0;
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
        try {
            finalValue = value ? JSON.parse(value) : {};
        } catch (err) {
            // 保留原始字符串，避免直接清空
            finalValue = value;
        }
    }
    const newProps = setByPath(props.props, key, finalValue);
    emit('update:props', newProps);
    emit('change', key, value);
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
