<template>
    <div class="button-component-editor">
        <div class="editor-section">
            <div class="section-title">按钮属性</div>

            <div class="form-item">
                <label>文本</label>
                <el-input v-model="localProps.text" placeholder="按钮文本" @change="emitProps" />
            </div>

            <div class="form-item">
                <label>类型</label>
                <el-select v-model="localProps.type" placeholder="选择类型" @change="emitProps">
                    <el-option label="主要按钮" value="primary" />
                    <el-option label="成功按钮" value="success" />
                    <el-option label="警告按钮" value="warning" />
                    <el-option label="危险按钮" value="danger" />
                    <el-option label="信息按钮" value="info" />
                    <el-option label="文本按钮" value="text" />
                </el-select>
            </div>

            <div class="form-item">
                <label>尺寸</label>
                <el-select v-model="localProps.size" placeholder="选择尺寸" @change="emitProps">
                    <el-option label="大型" value="large" />
                    <el-option label="默认" value="default" />
                    <el-option label="小型" value="small" />
                </el-select>
            </div>

            <div class="form-item checkbox-group">
                <el-checkbox v-model="localProps.round" @change="emitProps">圆角按钮</el-checkbox>
                <el-checkbox v-model="localProps.circle" @change="emitProps">圆形按钮</el-checkbox>
                <el-checkbox v-model="localProps.disabled" @change="emitProps">禁用</el-checkbox>
            </div>

            <div class="form-item">
                <label>图标</label>
                <el-input v-model="localProps.icon" placeholder="图标名称 (Element Plus Icons)" @change="emitProps" />
            </div>
        </div>

        <div class="editor-section">
            <div class="section-title">
                样式配置
                <el-button
                    size="small"
                    :type="isStyleConfigured ? 'success' : 'primary'"
                    @click="styleDialogVisible = true">
                    {{ isStyleConfigured ? '已配置' : '配置' }}
                </el-button>
            </div>
        </div>

        <div class="editor-section">
            <div class="section-title">
                详细配置
                <el-button
                    size="small"
                    :type="isAdvancedConfigured ? 'success' : 'primary'"
                    @click="advancedDialogVisible = true">
                    {{ isAdvancedConfigured ? '已配置' : '配置' }}
                </el-button>
            </div>
        </div>

        <el-dialog v-model="styleDialogVisible" title="样式配置" width="720px" draggable>
            <div class="style-config-dialog">
                <MonacoEditor v-model="styleInput" language="css" :theme="monacoTheme" height="320px" />
                <div v-if="styleError" class="error-tip">{{ styleError }}</div>
            </div>
            <template #footer>
                <el-button @click="styleDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="handleStyleSave">确定</el-button>
            </template>
        </el-dialog>

        <el-dialog v-model="advancedDialogVisible" title="详细配置" width="720px" draggable>
            <div class="style-config-dialog">
                <MonacoEditor v-model="advancedInput" language="javascript" :theme="monacoTheme" height="320px" />
                <div v-if="advancedError" class="error-tip">{{ advancedError }}</div>
            </div>
            <template #footer>
                <el-button @click="advancedDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="handleAdvancedSave">确定</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, watch, onBeforeUnmount, computed } from 'vue';
import MonacoEditor from '@/components/common/MonacoEditor.vue';

const props = defineProps({
    props: {
        type: Object,
        default: () => ({}),
    },
    style: {
        type: Object,
        default: () => ({}),
    },
});

const emit = defineEmits(['change-props', 'change-style']);

const localProps = ref({
    text: props.props.text || '按钮',
    type: props.props.type || 'primary',
    size: props.props.size || 'default',
    round: props.props.round || false,
    circle: props.props.circle || false,
    disabled: props.props.disabled || false,
    icon: props.props.icon || '',
    styleConfig: props.props.styleConfig || '',
    advancedConfig: props.props.advancedConfig || '',
});

const styleDialogVisible = ref(false);
const styleInput = ref(props.props.styleConfig || '');
const styleError = ref('');
const monacoTheme = ref('vs');
const advancedDialogVisible = ref(false);
const advancedInput = ref(props.props.advancedConfig || '');
const advancedError = ref('');
let mediaQuery;
let mediaHandler;

const isStyleConfigured = computed(() => Boolean(localProps.value.styleConfig && localProps.value.styleConfig.trim()));
const isAdvancedConfigured = computed(() => Boolean(localProps.value.advancedConfig && localProps.value.advancedConfig.trim()));

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

watch(
    () => props.props,
    (newProps) => {
        localProps.value = {
            text: newProps.text || '按钮',
            type: newProps.type || 'primary',
            size: newProps.size || 'default',
            round: newProps.round || false,
            circle: newProps.circle || false,
            disabled: newProps.disabled || false,
            icon: newProps.icon || '',
            styleConfig: newProps.styleConfig || '',
            advancedConfig: newProps.advancedConfig || '',
        };
        styleInput.value = newProps.styleConfig || '';
        styleError.value = '';
        advancedInput.value = newProps.advancedConfig || '';
        advancedError.value = '';
    },
    { deep: true },
);

watch(
    () => styleInput.value,
    () => {
        styleError.value = '';
    },
);

function emitProps() {
    emit('change-props', { ...localProps.value });
}

function handleStyleSave() {
    try {
        parseStyleInput(styleInput.value);
        styleError.value = '';
        localProps.value.styleConfig = styleInput.value;
        emitProps();
        styleDialogVisible.value = false;
    } catch (err) {
        styleError.value = err.message || '样式解析失败，请检查格式（key: value）';
    }
}

function handleAdvancedSave() {
    try {
        parseAdvancedInput(advancedInput.value);
        advancedError.value = '';
        localProps.value.advancedConfig = advancedInput.value;
        emitProps();
        advancedDialogVisible.value = false;
    } catch (err) {
        advancedError.value = err.message || '详细配置解析失败，请检查格式（JSON / 对象字面量）';
    }
}

function parseStyleInput(input) {
    const text = (input || '').trim();
    if (!text) return {};

    // 如果包含标准 CSS 代码块（带 { }），直接视为合法，交由运行时注入
    if (text.includes('{') && text.includes('}')) {
        return {};
    }

    const lines = text
        .split('\n')
        .map((l) => l.trim())
        .filter((l) => l && !l.startsWith('//') && !l.startsWith('/*') && !l.startsWith('*'));

    const style = {};
    for (const line of lines) {
        const idx = line.indexOf(':');
        if (idx === -1) throw new Error(`格式错误：缺少冒号 -> "${line}"`);
        const key = line.slice(0, idx).trim();
        const value = line.slice(idx + 1).replace(/;$/, '').trim();
        if (!key) throw new Error(`格式错误：属性名为空 -> "${line}"`);
        if (!value) throw new Error(`格式错误：属性值为空 -> "${line}"`);
        style[key] = value;
    }

    return style;
}

function parseAdvancedInput(input) {
    if (!input || !input.trim()) return {};
    // 允许 JSON 或对象字面量
    try {
        const parsed = JSON.parse(input);
        if (parsed && typeof parsed === 'object') return parsed;
    } catch (err) {
        // ignore
    }
    try {
        // eslint-disable-next-line no-new-func
        const fn = new Function(`return (${input})`);
        const res = fn();
        if (res && typeof res === 'object') return res;
    } catch (err) {
        // ignore
    }
    throw new Error('格式错误：请输入合法的 JSON 或对象字面量，如 { "text": "确认", "type": "primary" }');
}

onBeforeUnmount(() => {
    if (mediaQuery && mediaHandler) {
        mediaQuery.removeEventListener('change', mediaHandler);
    }
});
</script>

<style scoped>
.button-component-editor {
    padding: 12px;
}

.editor-section {
    margin-bottom: 16px;
}

.section-title {
    font-size: 14px;
    font-weight: 600;
    color: #303133;
    margin-bottom: 12px;
    padding-bottom: 8px;
    border-bottom: 1px solid #e4e7ed;
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.form-item {
    margin-bottom: 14px;
}

.form-item label {
    display: block;
    font-size: 13px;
    color: #606266;
    margin-bottom: 8px;
}

.checkbox-group :deep(.el-checkbox) {
    margin-right: 12px;
}

.el-input,
.el-select {
    width: 100%;
}

.style-config-dialog {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.error-tip {
    margin-top: 6px;
    color: #f56c6c;
    font-size: 12px;
}
</style>
