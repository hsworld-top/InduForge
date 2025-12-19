<template>
    <div class="property-panel">
        <!-- 空状态 -->
        <div v-if="!selectedComponent && selectedComponents.length === 0" class="empty-state">
            <el-empty description="请选择组件" :image-size="60" />
        </div>

        <!-- 多选状态 -->
        <div v-else-if="selectedComponents.length > 1" class="multi-select-state">
            <div class="panel-header">
                <span class="component-type">已选择 {{ selectedComponents.length }} 个组件</span>
            </div>

            <el-tabs v-model="activeTab" class="panel-tabs">
                <el-tab-pane label="样式" name="style">
                    <el-scrollbar>
                        <PositionEditor :style="commonStyle" @change="handleBatchStyleChange" />
                        <SpacingEditor :style="commonStyle" @change="handleBatchStyleChange" />
                        <TransformEditor :style="commonStyle" @change="handleBatchStyleChange" />
                    </el-scrollbar>
                </el-tab-pane>
            </el-tabs>
        </div>

        <!-- 单选状态 -->
        <template v-else-if="selectedComponent">
            <!-- 组件信息 -->
            <div class="panel-header panel-header-with-tabs" :class="{ 'no-tabs': !propsDef.showTabs }">
                <div class="panel-title">
                    <span class="component-type">{{ componentDef?.name || selectedComponent.type }}</span>
                    <span class="component-id">{{ selectedComponent.id.slice(0, 8) }}</span>
                </div>
                <div v-if="propsDef.showTabs" class="panel-tabs-bar">
                    <el-tabs v-model="activeTab" class="panel-tabs inline-tabs" type="card">
                        <el-tab-pane label="属性" name="props" />
                        <el-tab-pane label="样式" name="style" />
                        <el-tab-pane v-if="eventsSchema && Object.keys(eventsSchema).length" label="连接" name="events" />
                    </el-tabs>
                </div>
            </div>

            <!-- 标签页内容 -->
            <div class="panel-tab-content" v-if="activeTab === 'props'">
                <el-scrollbar>
                    <!-- 基础属性 -->
                    <div class="section-group">
                        <div class="section-title">基础</div>
                        <el-form label-position="left" label-width="80px" size="small">
                            <el-form-item label="标签">
                                <el-input v-model="componentLabel" @change="handleLabelChange" />
                            </el-form-item>
                            <el-form-item label="可见">
                                <el-switch v-model="componentVisible" @change="handleVisibleChange" />
                            </el-form-item>
                            <el-form-item label="锁定">
                                <el-switch v-model="componentLocked" @change="handleLockedChange" />
                            </el-form-item>
                        </el-form>
                    </div>

                    <!-- 组件专有属性编辑器 -->
                    <component
                        v-if="componentEditorName"
                        :is="componentEditorName"
                        :props="selectedComponent.props"
                        :style="selectedComponent.style"
                        @change-props="handlePropsChange"
                        @change-style="handleStyleObjectChange" />

                    <!-- 布局属性编辑器 -->
                    <FlexEditor v-if="isFlexContainer" :props="selectedComponent.props" @change="handlePropsChange" />
                    <GridEditor v-if="isGridContainer" :props="selectedComponent.props" @change="handlePropsChange" />

                    <!-- 样式配置（按钮同款弹窗） -->
                    <div v-if="showStyleConfigInProps" class="section-group">
                        <div class="section-title title-row space-between">
                            <span>样式配置</span>
                            <el-button
                                size="small"
                                :type="isStyleConfigured ? 'success' : 'primary'"
                                @click="openStyleDialog">
                                {{ isStyleConfigured ? '已配置' : '配置' }}
                            </el-button>
                        </div>
                    </div>

                    <!-- 通用属性 -->
                    <div v-if="filteredPropsSchema && Object.keys(filteredPropsSchema).length && !componentEditorName" class="section-group">
                        <div class="section-title">组件属性</div>
                        <PropsEditor :props="selectedComponent.props" :props-schema="filteredPropsSchema" @change="handlePropChange" />
                    </div>
                </el-scrollbar>
            </div>

            <div class="panel-tab-content" v-else-if="activeTab === 'style'">
                <el-scrollbar>
                    <PositionEditor :style="selectedComponent.style" @change="handleStyleObjectChange" />
                    <SpacingEditor :style="selectedComponent.style" @change="handleStyleObjectChange" />
                    <TransformEditor :style="selectedComponent.style" @change="handleStyleObjectChange" />
                    <StyleEditor v-if="selectedComponent" :style="selectedComponent.style" @change="handleStyleChange" />
                </el-scrollbar>
            </div>

            <div class="panel-tab-content" v-else-if="activeTab === 'events'">
                <el-scrollbar>
                    <div class="section-group">
                        <div class="section-title title-row space-between">
                            <span>事件</span>
                            <el-button
                                v-if="hasStyleConfigProp"
                                size="small"
                                :type="isStyleConfigured ? 'success' : 'primary'"
                                @click="openStyleDialog">
                                {{ isStyleConfigured ? '已配置' : '配置' }}
                            </el-button>
                        </div>
                        <el-form label-position="left" label-width="160px" size="small">
                            <template v-for="(meta, eventName) in eventsSchema" :key="eventName">
                                <el-form-item :label="meta.label || eventName">
                                    <template #label>
                                        <span class="event-name">{{ displayEventLabel(meta, eventName) }}</span>
                                    </template>
                                    <div class="event-row">
                                        <el-button
                                            class="event-config-button"
                                            size="small"
                                            :type="isEventConfigured(eventName) ? 'success' : 'primary'"
                                            @click="openEventDialog(eventName)">
                                            {{ isEventConfigured(eventName) ? '已配置' : '配置' }}
                                        </el-button>
                                    </div>
                                </el-form-item>
                            </template>
                        </el-form>
                    </div>
                </el-scrollbar>
            </div>

            <!-- 事件配置弹窗 -->
            <el-dialog v-model="eventDialogVisible" title="事件配置" width="720px">
                <div class="event-config-dialog">
                    <div class="event-config-header">
                        <span class="event-config-name">{{ currentEventKey }}</span>
                    </div>
                    <MonacoEditor v-model="eventCode" language="javascript" :theme="monacoTheme" height="360px" />
                    <div v-if="eventError" class="error-tip">{{ eventError }}</div>
                </div>
                <template #footer>
                    <el-button @click="eventDialogVisible = false">取消</el-button>
                    <el-button type="primary" @click="handleEventSave">确定</el-button>
                </template>
            </el-dialog>

            <!-- 样式配置弹窗（通用，按钮同款） -->
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
        </template>
    </div>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue';
import { useDesignStore } from '@/store/design';
import { getComponent } from '@/registry';
import { StyleEditor, PropsEditor } from '@/components/editors';
import MonacoEditor from '@/components/common/MonacoEditor.vue';
import PositionEditor from '@/components/editors/PositionEditor.vue';
import SpacingEditor from '@/components/editors/SpacingEditor.vue';
import TransformEditor from '@/components/editors/TransformEditor.vue';
import FlexEditor from '@/components/editors/FlexEditor.vue';
import GridEditor from '@/components/editors/GridEditor.vue';
import TextComponentEditor from '@/components/editors/TextComponentEditor.vue';
import ButtonComponentEditor from '@/components/editors/ButtonComponentEditor.vue';
import InputComponentEditor from '@/components/editors/InputComponentEditor.vue';
import ImageComponentEditor from '@/components/editors/ImageComponentEditor.vue';
import ChartComponentEditor from '@/components/editors/ChartComponentEditor.vue';

const designStore = useDesignStore();

const propsDef = defineProps({
    initialTab: { type: String, default: 'props' },
    showTabs: { type: Boolean, default: true },
});

const activeTab = ref(propsDef.initialTab || 'props');
watch(
    () => propsDef.initialTab,
    (val) => {
        if (val) activeTab.value = val;
    },
);

const eventDialogVisible = ref(false);
const currentEventKey = ref('');
const eventCode = ref('');
const eventError = ref('');
const monacoTheme = ref('vs');
const styleDialogVisible = ref(false);
const styleInput = ref('');
const styleError = ref('');
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

const selectedComponent = computed(() => designStore.selectedComponent);

const selectedComponents = computed(() => {
    const ids = designStore.selectedComponentIds || [];
    if (ids.length === 0) return [];

    return ids
        .map((id) => {
            const findById = (components) => {
                for (const comp of components) {
                    if (comp.id === id) return comp;
                    if (comp.children) {
                        const found = findById(comp.children);
                        if (found) return found;
                    }
                }
                return null;
            };
            return findById(designStore.components);
        })
        .filter(Boolean);
});

const commonStyle = computed(() => {
    if (selectedComponents.value.length === 0) return {};
    return selectedComponents.value[0].style || {};
});

const componentDef = computed(() => {
    if (!selectedComponent.value) return null;
    return getComponent(selectedComponent.value.type);
});

const propsSchema = computed(() => componentDef.value?.propsSchema || {});
const filteredPropsSchema = computed(() => {
    const schema = propsSchema.value || {};
    const rest = { ...schema };
    delete rest.styleConfig;
    return rest;
});
const eventsSchema = computed(() => componentDef.value?.eventsSchema || {});

const isFlexContainer = computed(() => {
    if (!selectedComponent.value) return false;
    return (
        selectedComponent.value.type === 'Flex' ||
        (selectedComponent.value.type === 'Container' && selectedComponent.value.props?.layoutMode === 'flex')
    );
});

const isGridContainer = computed(() => {
    if (!selectedComponent.value) return false;
    return (
        selectedComponent.value.type === 'Grid' ||
        (selectedComponent.value.type === 'Container' && selectedComponent.value.props?.layoutMode === 'grid')
    );
});

const componentEditorName = computed(() => {
    if (!selectedComponent.value) return null;

    const editorMap = {
        Text: TextComponentEditor,
        Button: ButtonComponentEditor,
        Input: InputComponentEditor,
        Image: ImageComponentEditor,
        LineChart: ChartComponentEditor,
        BarChart: ChartComponentEditor,
    };

    return editorMap[selectedComponent.value.type] || null;
});

const hasStyleConfigProp = computed(
    () => selectedComponent.value?.props?.styleConfig !== undefined || propsSchema.value?.styleConfig !== undefined,
);
const isStyleConfigured = computed(() => {
    const val = selectedComponent.value?.props?.styleConfig;
    return Boolean(val && String(val).trim());
});
const showStyleConfigInProps = computed(() => {
    if (componentEditorName.value === ButtonComponentEditor) return false;
    return hasStyleConfigProp.value;
});

const styleTagCache = new Map();

const escapeRegExp = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

const resolveCssText = (cssText, domId, realId) => {
    if (!cssText) return '';
    const target = domId || '';
    if (!target) return cssText;
    const escaped = escapeRegExp(target);
    const replacements = [
        { reg: new RegExp(`#${escaped}\\b`, 'g'), value: `#${realId}` },
        { reg: new RegExp(`\\[id=['"]${escaped}['"]\\]`, 'g'), value: `[id="${realId}"]` },
    ];
    return replacements.reduce((text, item) => text.replace(item.reg, item.value), cssText);
};

function applyCssText(componentId, cssText, domId) {
    if (typeof document === 'undefined') return;
    const existing = styleTagCache.get(componentId);
    if (existing) {
        existing.remove();
        styleTagCache.delete(componentId);
    }
    if (!cssText) return;
    const resolved = resolveCssText(cssText, domId, componentId);
    const el = document.createElement('style');
    el.setAttribute('data-style-config', componentId);
    el.textContent = resolved;
    document.head.appendChild(el);
    styleTagCache.set(componentId, el);
}

watch(
    () => selectedComponent.value?.props?.styleConfig,
    (val) => {
        styleInput.value = val || '';
        styleError.value = '';
    },
    { immediate: true },
);

const componentLabel = computed({
    get: () => selectedComponent.value?.label || '',
    set: () => {},
});

const componentVisible = computed({
    get: () => selectedComponent.value?.visible !== false,
    set: () => {},
});

const componentLocked = computed({
    get: () => selectedComponent.value?.locked === true,
    set: () => {},
});

const eventBindings = computed(() => selectedComponent.value?.props?.events || {});

const isEventConfigured = (eventName) => Boolean(eventBindings.value?.[eventName]);

const displayEventLabel = (meta, eventName) => {
    const label = meta?.label;
    if (label && label !== eventName) return `${label} ${eventName}`;
    return label || eventName;
};

function handleLabelChange(value) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, { label: value });
}

function handleVisibleChange(value) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, { visible: value });
}

function handleLockedChange(value) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, { locked: value });
}

function handlePropChange(key, value) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, {
        props: { [key]: value },
    });
}

function handlePropsChange(props) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, { props });
}

function handleEventChange(eventName, val) {
    if (!selectedComponent.value) return;
    const current = selectedComponent.value.props?.events || {};
    const next = { ...current, [eventName]: val };
    handlePropsChange({ ...selectedComponent.value.props, events: next });
    designStore.saveHistory('更新组件事件');
}

function openEventDialog(eventName) {
    const current = eventBindings.value?.[eventName] || '';
    currentEventKey.value = eventName;
    eventCode.value = current;
    eventError.value = '';
    eventDialogVisible.value = true;
}

function handleEventSave() {
    if (!currentEventKey.value) return;
    try {
        // 校验代码可编译，避免运行期直接报错
        // eslint-disable-next-line no-new-func
        new Function('event', 'emit', 'props', eventCode.value || '');
    } catch (err) {
        eventError.value = err?.message || '事件脚本存在语法错误';
        return;
    }
    handleEventChange(currentEventKey.value, eventCode.value);
    eventDialogVisible.value = false;
}

function handleStyleChange(key, value) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, {
        style: { [key]: value },
    });
}

function handleStyleObjectChange(style) {
    if (!selectedComponent.value) return;
    designStore.updateComponent(selectedComponent.value.id, { style });
}

function handleBatchStyleChange(style) {
    if (selectedComponents.value.length === 0) return;

    const ids = selectedComponents.value.map((comp) => comp.id);
    designStore.batchUpdateComponents(ids, { style });
}

function openStyleDialog() {
    styleInput.value = selectedComponent.value?.props?.styleConfig || '';
    styleError.value = '';
    styleDialogVisible.value = true;
}

function handleStyleSave() {
    try {
        const parsed = parseStyleInput(styleInput.value);
        styleError.value = '';
        if (!selectedComponent.value) return;
        const nextProps = { ...selectedComponent.value.props, styleConfig: styleInput.value };
        const nextStyle =
            parsed && parsed.style ? { ...(selectedComponent.value.style || {}), ...parsed.style } : selectedComponent.value.style;

        designStore.updateComponent(selectedComponent.value.id, {
            props: nextProps,
            style: nextStyle,
        });

        applyCssText(selectedComponent.value.id, parsed.cssText, parsed.domId);

        styleDialogVisible.value = false;
    } catch (err) {
        styleError.value = err?.message || '样式解析失败，请检查格式（key: value）';
    }
}

function parseStyleInput(input) {
    const text = (input || '').trim();
    if (!text) return { style: {}, domId: '', cssText: '' };

    // JSON 对象或对象字面量
    try {
        const parsed = JSON.parse(text);
        if (parsed && typeof parsed === 'object') {
            const { domId = '', ...rest } = parsed;
            return { style: rest, domId, cssText: '' };
        }
    } catch (e) {
        // ignore
    }

    if (text.includes('{') && text.includes('}')) {
        const idMatch = text.match(/#([\w-]+)/);
        return { style: {}, domId: idMatch ? idMatch[1] : '', cssText: text };
    }

    const lines = text
        .split('\n')
        .map((l) => l.trim())
        .filter((l) => l && !l.startsWith('//') && !l.startsWith('/*') && !l.startsWith('*'));

    const style = {};
    let domId = '';
    for (const line of lines) {
        const idx = line.indexOf(':');
        if (idx === -1) throw new Error(`格式错误：缺少冒号 -> "${line}"`);
        const key = line.slice(0, idx).trim();
        const value = line.slice(idx + 1).replace(/;$/, '').trim();
        if (!key) throw new Error(`格式错误：属性名为空 -> "${line}"`);
        if (!value) throw new Error(`格式错误：属性值为空 -> "${line}"`);
        if (key === 'domId') {
            domId = value;
        } else {
            style[key] = value;
        }
    }

    return { style, domId, cssText: '' };
}
</script>

<style scoped>
.property-panel {
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: #fff;
}

.empty-state {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
}

.panel-header {
    padding: 12px;
    border-bottom: 1px solid #e4e7ed;
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.panel-header-with-tabs {
    align-items: flex-end;
    gap: 12px;
    padding-bottom: 8px;
}

.panel-header-with-tabs.no-tabs {
    align-items: center;
    padding-bottom: 12px;
}

.panel-title {
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.panel-tabs-bar {
    flex: 1;
    display: flex;
    justify-content: flex-end;
    align-items: flex-end;
}

.component-type {
    font-size: 14px;
    font-weight: 500;
    color: #303133;
}

.component-id {
    font-size: 12px;
    color: #909399;
    font-family: monospace;
}

.panel-tabs {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.panel-tab-content {
    flex: 1;
    display: flex;
    flex-direction: column;
}

:deep(.inline-tabs .el-tabs__header) {
    margin: 0;
}

:deep(.inline-tabs .el-tabs__nav-wrap::after) {
    height: 0;
}

:deep(.el-tabs__content) {
    flex: 1;
    overflow: auto;
    padding: 0;
}

:deep(.el-tab-pane) {
    height: 100%;
}

.props-section,
.style-section {
    padding: 12px;
}

.section-group {
    padding: 12px;
}

.section-group:last-child {
    margin-bottom: 0;
}

.section-title {
    font-size: 12px;
    font-weight: 500;
    color: #909399;
    margin-bottom: 8px;
}

.title-row {
    display: flex;
    align-items: center;
    gap: 8px;
}

.space-between {
    justify-content: space-between;
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

:deep(.el-tabs__header) {
    margin: 0;
    padding: 0 12px;
}

:deep(.el-tabs__nav-wrap::after) {
    height: 1px;
}

:deep(.el-scrollbar) {
    height: 100%;
}

:deep(.el-scrollbar__wrap) {
    overflow-x: hidden;
}

.multi-select-state {
    height: 100%;
    display: flex;
    flex-direction: column;
}

.multi-select-state .panel-tabs {
    flex: 1;
    display: flex;
    flex-direction: column;
}

.event-row {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    width: 100%;
}

.event-name {
    font-size: 12px;
    color: #909399;
    white-space: nowrap;
}

.event-config-dialog {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.error-tip {
    color: #f56c6c;
    font-size: 12px;
}
</style>
