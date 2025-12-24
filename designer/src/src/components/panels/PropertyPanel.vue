<template>
    <div class="property-panel">
        <!-- 空状态 -->
        <div v-if="showCanvasEvents" class="panel-tab-content">
            <el-scrollbar>
                <div class="section-group">
                    <div class="section-title title-row space-between">
                        <span>事件</span>
                    </div>
                    <div class="canvas-event-toolbar">
                        <el-button
                            class="canvas-event-toolbar-btn"
                            size="small"
                            :icon="Plus"
                            @click="openAddCanvasEventDialog" />
                        <el-button
                            class="canvas-event-toolbar-btn"
                            size="small"
                            :icon="Delete"
                            :disabled="!canRemoveCanvasEvent"
                            @click="removeSelectedCanvasEvent" />
                    </div>
                    <el-form label-position="left" label-width="160px" size="small">
                        <template v-for="entry in canvasEventEntries" :key="entry.key">
                            <el-form-item :label="entry.label">
                                <template #label>
                                    <span
                                        class="event-name canvas-event-name"
                                        :class="{ 'is-selected': entry.removable && entry.id === selectedCanvasEventId }"
                                        @click="selectCanvasEvent(entry)">
                                        {{ entry.label }}
                                    </span>
                                </template>
                                <div class="event-row">
                                    <el-button
                                        class="event-config-button"
                                        size="small"
                                        :type="isEventConfigured(entry.eventName, 'canvas', entry.itemId) ? 'success' : 'primary'"
                                        @click="openEventDialog(entry.eventName, 'canvas', entry.itemId)">
                                        {{ isEventConfigured(entry.eventName, 'canvas', entry.itemId) ? '已配置' : '配置' }}
                                    </el-button>
                                </div>
                            </el-form-item>
                        </template>
                    </el-form>
                </div>
            </el-scrollbar>
        </div>

        <div v-else-if="!selectedComponent && selectedComponents.length === 0" class="empty-state">
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

            <!-- 样式配置弹窗（通用，按钮同款） -->
            <el-dialog v-model="styleDialogVisible" title="样式配置" width="720px" draggable :lock-scroll="false">
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

        <!-- 添加画布事件弹窗 -->
        <el-dialog v-model="addCanvasEventDialogVisible" title="添加事件" width="520px" :close-on-click-modal="false">
            <div class="add-canvas-event-dialog">
                <el-radio-group v-model="addCanvasEventType" class="add-canvas-event-group">
                    <div class="event-option-row">
                        <div class="event-option-radio">
                            <el-radio label="variableChange">变量改变</el-radio>
                        </div>
                        <span class="event-option-label">变量名：</span>
                        <el-select
                            v-model="addCanvasEventVariable"
                            size="small"
                            placeholder="请选择变量"
                            class="event-option-input">
                            <el-option v-for="name in variableOptions" :key="name" :label="name" :value="name" />
                        </el-select>
                    </div>
                    <el-divider />
                    <div class="event-option-row">
                        <div class="event-option-radio">
                            <el-radio label="timer">定时器</el-radio>
                        </div>
                        <span class="event-option-label">连接名：</span>
                        <el-input v-model="addCanvasEventTimerName" size="small" placeholder="请输入连接名" class="event-option-input" />
                    </div>
                </el-radio-group>
            </div>
            <template #footer>
                <el-button @click="addCanvasEventDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="handleAddCanvasEvent">保存</el-button>
            </template>
        </el-dialog>

        <!-- 事件配置弹窗 -->
        <el-dialog v-model="eventDialogVisible" title="事件配置" width="720px" :lock-scroll="false">
            <div class="event-config-dialog">
                <div class="event-config-header">
                    <span class="event-config-name">{{ currentEventLabel }}</span>
                </div>
                <MonacoEditor v-model="eventCode" language="javascript" :theme="monacoTheme" height="360px" />
                <div v-if="eventError" class="error-tip">{{ eventError }}</div>
            </div>
            <template #footer>
                <el-button @click="eventDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="handleEventSave">确定</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue';
import { ElMessage } from 'element-plus';
import { Plus, Delete } from '@element-plus/icons-vue';
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
const currentEventScope = ref('component');
const currentEventItemId = ref('');
const eventCode = ref('');
const eventError = ref('');
const monacoTheme = ref('vs');
const styleDialogVisible = ref(false);
const styleInput = ref('');
const styleError = ref('');
const addCanvasEventDialogVisible = ref(false);
const addCanvasEventType = ref('variableChange');
const addCanvasEventVariable = ref('');
const addCanvasEventTimerName = ref('');
const selectedCanvasEventId = ref('');
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
const currentPage = computed(() => designStore.currentPage);

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
const defaultVariableOptions = ['变量A', '变量B', '变量C'];
const availableVariables = computed(() => Object.keys(currentPage.value?.variables || {}));
const variableOptions = computed(() =>
    availableVariables.value.length > 0 ? availableVariables.value : defaultVariableOptions,
);
const canvasEventsSchema = {
    created: { label: '创建时', hideKey: true },
    unmounted: { label: '关闭时', hideKey: true },
    variableChange: { label: '变量改变', hideKey: true },
    timer: { label: '定时器', hideKey: true },
};
const baseCanvasEvents = ['created', 'unmounted'];
const canvasEventBindings = computed(() => currentPage.value?.events || {});
const showCanvasEvents = computed(
    () => activeTab.value === 'events' && !selectedComponent.value && selectedComponents.value.length === 0 && currentPage.value,
);
const canvasEventEntries = computed(() => {
    const entries = [];
    baseCanvasEvents.forEach((eventName) => {
        const meta = canvasEventsSchema[eventName];
        entries.push({
            key: `static:${eventName}`,
            id: eventName,
            itemId: null,
            eventName,
            label: displayEventLabel(meta, eventName),
            removable: false,
        });
    });

    const variableItems = getCanvasEventList('variableChange');
    variableItems.forEach((item) => {
        entries.push({
            key: `variableChange:${item.id}`,
            id: item.id,
            itemId: item.id,
            eventName: 'variableChange',
            label: item.variable || '变量改变',
            removable: true,
        });
    });

    const timerItems = getCanvasEventList('timer');
    timerItems.forEach((item) => {
        entries.push({
            key: `timer:${item.id}`,
            id: item.id,
            itemId: item.id,
            eventName: 'timer',
            label: item.name || '定时器',
            removable: true,
        });
    });

    return entries;
});
const selectedCanvasEvent = computed(() =>
    canvasEventEntries.value.find((entry) => entry.removable && entry.id === selectedCanvasEventId.value),
);
const canRemoveCanvasEvent = computed(() => Boolean(selectedCanvasEvent.value));
const currentEventLabel = computed(() => {
    if (currentEventScope.value === 'canvas') {
        if (currentEventItemId.value) {
            const entry = canvasEventEntries.value.find(
                (item) => item.eventName === currentEventKey.value && item.itemId === currentEventItemId.value,
            );
            if (entry?.label) return entry.label;
        }
        const meta = canvasEventsSchema[currentEventKey.value];
        return meta?.label || currentEventKey.value;
    }
    return currentEventKey.value;
});

watch(canvasEventEntries, (entries) => {
    if (!selectedCanvasEventId.value) return;
    const exists = entries.some((entry) => entry.removable && entry.id === selectedCanvasEventId.value);
    if (!exists) {
        selectedCanvasEventId.value = '';
    }
});

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

const isEventConfigured = (eventName, scope = 'component', itemId = null) => {
    const bindings = scope === 'canvas' ? canvasEventBindings.value : eventBindings.value;
    const value = bindings?.[eventName];
    if (scope === 'canvas' && Array.isArray(value)) {
        if (!itemId) return false;
        const item = value.find((entry) => entry.id === itemId);
        return Boolean(item?.code);
    }
    return Boolean(value);
};

function displayEventLabel(meta, eventName) {
    const label = meta?.label;
    if (meta?.hideKey && label) return label;
    if (label && label !== eventName) return `${label} ${eventName}`;
    return label || eventName;
}

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

function createCanvasEventId() {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
        return crypto.randomUUID();
    }
    return `evt_${Date.now()}_${Math.random().toString(16).slice(2)}`;
}

function getCanvasEventList(eventName) {
    const list = canvasEventBindings.value?.[eventName];
    return Array.isArray(list) ? list : [];
}

function findCanvasEventItem(eventName, itemId) {
    if (!itemId) return null;
    const list = getCanvasEventList(eventName);
    return list.find((item) => item.id === itemId) || null;
}

function selectCanvasEvent(entry) {
    if (!entry?.removable) {
        selectedCanvasEventId.value = '';
        return;
    }
    selectedCanvasEventId.value = entry.id;
}

function openAddCanvasEventDialog() {
    const options = variableOptions.value;
    addCanvasEventType.value = options.length > 0 ? 'variableChange' : 'timer';
    addCanvasEventVariable.value = options.length > 0 ? options[0] : '';
    addCanvasEventTimerName.value = '';
    addCanvasEventDialogVisible.value = true;
}

function handleAddCanvasEvent() {
    if (!currentPage.value) return;
    if (addCanvasEventType.value === 'variableChange') {
        const variableName = (addCanvasEventVariable.value || '').trim();
        if (!variableName) {
            ElMessage.warning('请选择变量名');
            return;
        }
        const list = getCanvasEventList('variableChange');
        if (list.some((item) => item.variable === variableName)) {
            ElMessage.warning('该变量已存在');
            return;
        }
        const item = { id: createCanvasEventId(), variable: variableName, code: '' };
        designStore.updatePageEvent('variableChange', [...list, item]);
        designStore.saveHistory('新增画布事件');
        selectedCanvasEventId.value = item.id;
    } else {
        const timerName = (addCanvasEventTimerName.value || '').trim();
        if (!timerName) {
            ElMessage.warning('请输入连接名');
            return;
        }
        const list = getCanvasEventList('timer');
        if (list.some((item) => item.name === timerName)) {
            ElMessage.warning('该连接名已存在');
            return;
        }
        const item = { id: createCanvasEventId(), name: timerName, code: '' };
        designStore.updatePageEvent('timer', [...list, item]);
        designStore.saveHistory('新增画布事件');
        selectedCanvasEventId.value = item.id;
    }
    addCanvasEventDialogVisible.value = false;
}

function removeSelectedCanvasEvent() {
    const entry = selectedCanvasEvent.value;
    if (!entry) return;
    const list = getCanvasEventList(entry.eventName);
    const next = list.filter((item) => item.id !== entry.id);
    designStore.updatePageEvent(entry.eventName, next);
    designStore.saveHistory('删除画布事件');
    selectedCanvasEventId.value = '';
}

function handleEventChange(eventName, val) {
    if (!selectedComponent.value) return;
    const current = selectedComponent.value.props?.events || {};
    const next = { ...current, [eventName]: val };
    handlePropsChange({ ...selectedComponent.value.props, events: next });
    designStore.saveHistory('更新组件事件');
}

function handleCanvasEventChange(eventName, val, itemId = '') {
    if (!currentPage.value) return;
    if (itemId) {
        const list = getCanvasEventList(eventName);
        const next = list.map((item) => (item.id === itemId ? { ...item, code: val } : item));
        designStore.updatePageEvent(eventName, next);
    } else {
        designStore.updatePageEvent(eventName, val);
    }
    designStore.saveHistory('更新画布事件');
}

function openEventDialog(eventName, scope = 'component', itemId = '') {
    const bindings = scope === 'canvas' ? canvasEventBindings.value : eventBindings.value;
    currentEventScope.value = scope;
    currentEventItemId.value = scope === 'canvas' ? itemId : '';
    let current = '';
    if (scope === 'canvas') {
        if (itemId) {
            const item = findCanvasEventItem(eventName, itemId);
            current = item?.code || '';
        } else {
            current = typeof bindings?.[eventName] === 'string' ? bindings[eventName] : '';
        }
    } else {
        current = bindings?.[eventName] || '';
    }
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
    if (currentEventScope.value === 'canvas') {
        handleCanvasEventChange(currentEventKey.value, eventCode.value, currentEventItemId.value);
    } else {
        handleEventChange(currentEventKey.value, eventCode.value);
    }
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

.canvas-event-toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    background-color: #f5f7fa;
    border: 1px solid #e4e7ed;
    border-radius: 4px;
    margin-bottom: 8px;
}

.canvas-event-toolbar-btn {
    min-width: 28px;
}

.canvas-event-name {
    cursor: pointer;
}

.canvas-event-name.is-selected {
    color: #409eff;
    font-weight: 600;
}

.add-canvas-event-dialog {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.add-canvas-event-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.event-option-row {
    display: grid;
    grid-template-columns: 110px 70px 220px;
    align-items: center;
    column-gap: 16px;
}

.event-option-radio {
    display: flex;
    align-items: center;
    justify-content: flex-start;
}

.event-option-label {
    font-size: 12px;
    color: #606266;
    text-align: right;
}

.event-option-input {
    width: 100%;
    min-width: 0;
}

.event-option-input :deep(.el-select),
.event-option-input :deep(.el-input) {
    width: 100%;
}

.event-option-input :deep(.el-input__wrapper) {
    width: 100%;
    box-sizing: border-box;
}

.event-option-radio :deep(.el-radio) {
    margin-right: 0;
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
