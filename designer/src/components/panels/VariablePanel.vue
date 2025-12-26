<template>
    <div class="variable-panel">
        <div class="variable-header">
            <div class="toolbar">
                <el-button class="toolbar-btn" size="small" :icon="Plus" @click="openAddDialog" />
                <el-button class="toolbar-btn" size="small" :icon="Edit" :disabled="!selectedVariable" @click="openEditDialog" />
                <el-button class="toolbar-btn" size="small" :icon="Delete" :disabled="!selectedVariable" @click="removeSelected" />
                <el-divider direction="vertical" />
                <el-button class="toolbar-btn" size="small" :icon="Upload" @click="exportVariables" />
                <el-button class="toolbar-btn" size="small" :icon="Download" @click="triggerImport" />
                <input ref="fileInput" class="file-input" type="file" accept=".xlsx,.xls" @change="handleImport" />
            </div>
        </div>

        <div class="variable-content">
            <el-table
                v-if="variableRows.length"
                :data="variableRows"
                size="small"
                highlight-current-row
                @current-change="handleCurrentChange"
                @row-dblclick="openEditDialog">
                <el-table-column prop="name" label="变量名" min-width="100" />
                <el-table-column label="默认值" min-width="110">
                    <template #default="{ row }">
                        <span class="value-cell">{{ formatValue(row.value) }}</span>
                    </template>
                </el-table-column>
                <el-table-column label="类型" min-width="70">
                    <template #default="{ row }">
                        <span>{{ getVariableType(row) }}</span>
                    </template>
                </el-table-column>
            </el-table>
            <div v-else class="empty-state">
                <el-empty description="暂无变量" :image-size="60" />
            </div>
        </div>

        <el-dialog v-model="showDialog" :title="dialogTitle" width="520px" :close-on-click-modal="false" :lock-scroll="false">
            <el-form ref="formRef" :model="form" :rules="rules" label-width="70px">
                <el-form-item label="名称" prop="name">
                    <el-input v-model="form.name" placeholder="var_name" />
                </el-form-item>
                <el-form-item label="类型" prop="type">
                    <el-select v-model="form.type" class="w-full" @change="handleTypeChange">
                        <el-option label="String" value="String" />
                        <el-option label="Number" value="Number" />
                        <el-option label="Boolean" value="Boolean" />
                        <el-option label="Object" value="Object" />
                        <el-option label="Array" value="Array" />
                    </el-select>
                </el-form-item>
                <el-form-item label="默认值" prop="defaultValue">
                    <el-input v-if="form.type === 'String' || form.type === 'Number'" v-model="form.defaultValue" />
                    <el-switch v-else-if="form.type === 'Boolean'" v-model="form.defaultValue" />
                    <el-input v-else v-model="form.defaultValue" type="textarea" :rows="3" placeholder='{"key":"value"}' />
                </el-form-item>
                <el-form-item label="开放性" prop="access">
                    <el-select v-model="form.access" class="w-full">
                        <el-option label="公开" value="public" />
                        <el-option label="私有" value="private" />
                    </el-select>
                </el-form-item>
                <el-form-item label="描述" prop="description">
                    <el-input v-model="form.description" type="textarea" :rows="3" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="showDialog = false">取消</el-button>
                <el-button type="primary" @click="saveVariable">保存</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Plus, Edit, Delete, Download, Upload } from '@element-plus/icons-vue';
import { useDesignStore } from '@/store/design';
import * as XLSX from 'xlsx';

const designStore = useDesignStore();

const showDialog = ref(false);
const formRef = ref(null);
const fileInput = ref(null);
const selectedName = ref('');
const isEditing = ref(false);
const editingName = ref('');
const isHydratingForm = ref(false);

const form = ref({
    name: '',
    type: 'String',
    defaultValue: '',
    access: 'public',
    description: '',
});

const rules = {
    name: [
        { required: true, message: '请输入变量名称', trigger: 'blur' },
        { pattern: /^[a-zA-Z_][a-zA-Z0-9_]*$/, message: '名称只能包含字母、数字和下划线，且以字母或下划线开头', trigger: 'blur' },
    ],
    type: [{ required: true, message: '请选择类型', trigger: 'change' }],
};

const currentPage = computed(() => designStore.currentPage);
const dialogTitle = computed(() => (isEditing.value ? '编辑变量' : '变量'));
const variableRows = computed(() => {
    const vars = currentPage.value?.variables || {};
    return Object.entries(vars).map(([name, value]) => ({ name, value }));
});
const selectedVariable = computed(() => {
    if (!currentPage.value) return null;
    const name = selectedName.value;
    if (!name) return null;
    const vars = currentPage.value.variables || {};
    if (!Object.prototype.hasOwnProperty.call(vars, name)) return null;
    return { name, value: vars[name] };
});

const formatValue = (value) => {
    if (value === null || value === undefined) return 'null';
    if (typeof value === 'object') return JSON.stringify(value);
    return String(value);
};

const ensureVariableMeta = () => {
    if (!currentPage.value) return;
    if (!currentPage.value.variableMeta || typeof currentPage.value.variableMeta !== 'object') {
        currentPage.value.variableMeta = {};
    }
};

const handleTypeChange = (type) => {
    if (isHydratingForm.value) return;
    if (type === 'Boolean') {
        form.value.defaultValue = false;
        return;
    }
    if (type === 'Object') {
        form.value.defaultValue = '{}';
        return;
    }
    if (type === 'Array') {
        form.value.defaultValue = '[]';
        return;
    }
    form.value.defaultValue = '';
};

const applyFormValues = async (values) => {
    isHydratingForm.value = true;
    Object.assign(form.value, values);
    await nextTick();
    isHydratingForm.value = false;
    formRef.value?.clearValidate();
};

const openAddDialog = async () => {
    isEditing.value = false;
    editingName.value = '';
    showDialog.value = true;
    await applyFormValues({
        name: '',
        type: 'String',
        defaultValue: '',
        access: 'public',
        description: '',
    });
};

const inferType = (value) => {
    if (Array.isArray(value)) return 'Array';
    if (value === null || value === undefined) return 'String';
    if (typeof value === 'object') return 'Object';
    if (typeof value === 'number') return 'Number';
    if (typeof value === 'boolean') return 'Boolean';
    return 'String';
};

const getVariableType = (row) => {
    const name = row?.name;
    if (!name) return 'String';
    const meta = currentPage.value?.variableMeta?.[name];
    return meta?.type || inferType(row?.value);
};

const formatDefaultValueForForm = (type, value) => {
    if (type === 'Boolean') return Boolean(value);
    if (type === 'Number') return value === null || value === undefined ? '' : String(value);
    if (type === 'Object') return value && typeof value === 'object' && !Array.isArray(value) ? JSON.stringify(value, null, 2) : '{}';
    if (type === 'Array') return Array.isArray(value) ? JSON.stringify(value, null, 2) : '[]';
    return value === null || value === undefined ? '' : String(value);
};

const resolveVariable = (row) => {
    if (row?.name) return row;
    if (!currentPage.value) return null;
    const name = selectedName.value;
    if (!name) return null;
    const vars = currentPage.value.variables || {};
    if (!Object.prototype.hasOwnProperty.call(vars, name)) return null;
    return { name, value: vars[name] };
};

const openEditDialog = async (row = null) => {
    if (!currentPage.value) return;
    const target = resolveVariable(row);
    if (!target) return;
    const name = target.name;
    const value = target.value;
    const meta = currentPage.value.variableMeta?.[name] || {};
    const type = meta.type || inferType(value);

    isEditing.value = true;
    editingName.value = name;
    selectedName.value = name;
    showDialog.value = true;
    await applyFormValues({
        name,
        type,
        defaultValue: formatDefaultValueForForm(type, value),
        access: meta.access || 'public',
        description: meta.description || '',
    });
};

const parseDefaultValue = () => {
    const { type, defaultValue } = form.value;
    if (type === 'Number') {
        if (defaultValue === '' || defaultValue === null || defaultValue === undefined) return 0;
        const num = Number(defaultValue);
        if (Number.isNaN(num)) {
            throw new Error('默认值必须为数字');
        }
        return num;
    }
    if (type === 'Boolean') {
        if (typeof defaultValue === 'boolean') return defaultValue;
        if (defaultValue === 'true') return true;
        if (defaultValue === 'false') return false;
        return Boolean(defaultValue);
    }
    if (type === 'Object') {
        if (!defaultValue) return {};
        try {
            const parsed = JSON.parse(defaultValue);
            if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
                return parsed;
            }
            throw new Error('默认值必须是对象');
        } catch (error) {
            throw new Error('默认值必须是合法的JSON对象');
        }
    }
    if (type === 'Array') {
        if (!defaultValue) return [];
        try {
            const parsed = JSON.parse(defaultValue);
            if (Array.isArray(parsed)) {
                return parsed;
            }
            throw new Error('默认值必须是数组');
        } catch (error) {
            throw new Error('默认值必须是合法的JSON数组');
        }
    }
    return String(defaultValue ?? '');
};

const escapeRegExp = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

const replaceVarInString = (text, oldName, newName) => {
    if (typeof text !== 'string') return text;
    const escaped = escapeRegExp(oldName);
    let next = text.replace(new RegExp(`\\bvars\\.${escaped}\\b`, 'g'), `vars.${newName}`);
    next = next.replace(new RegExp(`\\bvars\\[['"]${escaped}['"]\\]`, 'g'), `vars["${newName}"]`);
    return next;
};

const replaceVarInValue = (value, oldName, newName) => {
    if (typeof value === 'string') {
        return replaceVarInString(value, oldName, newName);
    }
    if (Array.isArray(value)) {
        return value.map((item) => replaceVarInValue(item, oldName, newName));
    }
    if (value && typeof value === 'object') {
        const next = {};
        Object.entries(value).forEach(([key, val]) => {
            if (key === 'variable' && val === oldName) {
                next[key] = newName;
                return;
            }
            next[key] = replaceVarInValue(val, oldName, newName);
        });
        return next;
    }
    return value;
};

const normalizeHeader = (value) => String(value ?? '').trim().toLowerCase();

const resolveHeaderIndex = (headers, candidates) => {
    const normalized = headers.map((item) => normalizeHeader(item));
    const match = candidates.map((item) => item.toLowerCase());
    return normalized.findIndex((item) => match.includes(item));
};

const normalizeType = (typeValue) => {
    const text = String(typeValue || '').trim().toLowerCase();
    if (!text) return '';
    if (['string', 'str', '字符串'].includes(text)) return 'String';
    if (['number', 'num', '数字', '数值'].includes(text)) return 'Number';
    if (['boolean', 'bool', '布尔', '布尔值'].includes(text)) return 'Boolean';
    if (['object', 'obj', '对象'].includes(text)) return 'Object';
    if (['array', 'arr', '数组'].includes(text)) return 'Array';
    return String(typeValue).trim();
};

const normalizeAccess = (accessValue) => {
    const text = String(accessValue || '').trim().toLowerCase();
    if (!text) return 'public';
    if (['public', '公开', 'open'].includes(text)) return 'public';
    if (['private', '私有', '私密'].includes(text)) return 'private';
    return text === '1' ? 'public' : text === '0' ? 'private' : 'public';
};

const parseImportedValue = (type, cellValue) => {
    if (type === 'Number') {
        if (cellValue === null || cellValue === undefined || cellValue === '') return 0;
        const num = typeof cellValue === 'number' ? cellValue : Number(cellValue);
        if (Number.isNaN(num)) {
            throw new Error('默认值必须为数字');
        }
        return num;
    }
    if (type === 'Boolean') {
        if (typeof cellValue === 'boolean') return cellValue;
        const text = String(cellValue || '').trim().toLowerCase();
        if (text === 'true' || text === '1' || text === '是') return true;
        if (text === 'false' || text === '0' || text === '否') return false;
        return Boolean(cellValue);
    }
    if (type === 'Object') {
        if (cellValue && typeof cellValue === 'object' && !Array.isArray(cellValue)) return cellValue;
        if (!cellValue) return {};
        try {
            const parsed = typeof cellValue === 'string' ? JSON.parse(cellValue) : cellValue;
            if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
                return parsed;
            }
            throw new Error('默认值必须是对象');
        } catch (error) {
            throw new Error('默认值必须是合法的JSON对象');
        }
    }
    if (type === 'Array') {
        if (Array.isArray(cellValue)) return cellValue;
        if (!cellValue) return [];
        try {
            const parsed = typeof cellValue === 'string' ? JSON.parse(cellValue) : cellValue;
            if (Array.isArray(parsed)) {
                return parsed;
            }
            throw new Error('默认值必须是数组');
        } catch (error) {
            throw new Error('默认值必须是合法的JSON数组');
        }
    }
    if (cellValue === null || cellValue === undefined) return '';
    return String(cellValue);
};

const formatExportValue = (value) => {
    if (value === null || value === undefined) return '';
    if (typeof value === 'object') return JSON.stringify(value);
    return value;
};

const sanitizeExcelValue = (value) => {
    if (value === null || value === undefined) return '';
    const text = typeof value === 'string' ? value : String(value);
    if (/^[=+\-@]/.test(text)) {
        return `'${text}`;
    }
    return text;
};

const downloadBlob = (blob, filename) => {
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
};

const renameVariableReferences = (oldName, newName) => {
    if (!currentPage.value || oldName === newName) return;

    if (currentPage.value.events) {
        currentPage.value.events = replaceVarInValue(currentPage.value.events, oldName, newName);
    }

    if (Array.isArray(currentPage.value.dataSources)) {
        currentPage.value.dataSources = currentPage.value.dataSources.map((source) => {
            const next = { ...source };
            if (next.config) {
                next.config = replaceVarInValue(next.config, oldName, newName);
            }
            if (typeof next.transformer === 'string') {
                next.transformer = replaceVarInString(next.transformer, oldName, newName);
            }
            if (typeof next.errorHandler === 'string') {
                next.errorHandler = replaceVarInString(next.errorHandler, oldName, newName);
            }
            return next;
        });
    }

    const updateComponent = (component) => {
        if (!component || typeof component !== 'object') return;

        if (component.bindings && typeof component.bindings === 'object') {
            const nextBindings = {};
            Object.entries(component.bindings).forEach(([path, expression]) => {
                nextBindings[path] = typeof expression === 'string' ? replaceVarInString(expression, oldName, newName) : expression;
            });
            component.bindings = nextBindings;
            if (component.id) {
                designStore.bindComponent(component.id, nextBindings);
            }
        }

        if (component.props?.events && typeof component.props.events === 'object') {
            const nextEvents = {};
            Object.entries(component.props.events).forEach(([eventName, code]) => {
                nextEvents[eventName] = typeof code === 'string' ? replaceVarInString(code, oldName, newName) : code;
            });
            component.props = { ...component.props, events: nextEvents };
        }

        if (component.events && typeof component.events === 'object') {
            component.events = replaceVarInValue(component.events, oldName, newName);
        }

        if (Array.isArray(component.children)) {
            component.children.forEach(updateComponent);
        }
    };

    (currentPage.value.components || []).forEach(updateComponent);
};

const saveVariable = async () => {
    if (!currentPage.value) return;
    try {
        await formRef.value.validate();
        const name = form.value.name.trim();
        if (!name) return;

        const existing = currentPage.value.variables || {};
        if (!isEditing.value && Object.prototype.hasOwnProperty.call(existing, name)) {
            ElMessage.warning('变量已存在');
            return;
        }
        if (isEditing.value && name !== editingName.value && Object.prototype.hasOwnProperty.call(existing, name)) {
            ElMessage.warning('变量已存在');
            return;
        }

        const value = parseDefaultValue();
        const renamed = isEditing.value && editingName.value && editingName.value !== name;
        if (renamed) {
            renameVariableReferences(editingName.value, name);
        }
        const nextVars = { ...existing };
        if (renamed) {
            delete nextVars[editingName.value];
        }
        nextVars[name] = value;
        currentPage.value.variables = nextVars;

        ensureVariableMeta();
        const nextMeta = { ...currentPage.value.variableMeta };
        if (renamed) {
            delete nextMeta[editingName.value];
        }
        nextMeta[name] = {
            type: form.value.type,
            access: form.value.access,
            description: form.value.description || '',
            defaultValue: value,
        };
        currentPage.value.variableMeta = nextMeta;

        designStore.isDirty = true;
        designStore.saveHistory(isEditing.value ? '编辑变量' : '新增变量');
        showDialog.value = false;
        selectedName.value = name;
        isEditing.value = false;
        editingName.value = '';
    } catch (error) {
        if (error?.message) {
            ElMessage.error(error.message);
        }
    }
};

const handleCurrentChange = (row) => {
    selectedName.value = row?.name || '';
};

const removeSelected = async () => {
    if (!currentPage.value || !selectedVariable.value) return;
    try {
        await ElMessageBox.confirm('确定删除选中的变量吗？', '确认删除', { type: 'warning', lockScroll: false });
    } catch (error) {
        return;
    }

    const name = selectedVariable.value.name;
    const nextVars = { ...(currentPage.value.variables || {}) };
    delete nextVars[name];
    currentPage.value.variables = nextVars;

    if (currentPage.value.variableMeta) {
        const nextMeta = { ...currentPage.value.variableMeta };
        delete nextMeta[name];
        currentPage.value.variableMeta = nextMeta;
    }

    designStore.isDirty = true;
    designStore.saveHistory('删除变量');
    selectedName.value = '';
    ElMessage.success('变量已删除');
};

const exportVariables = () => {
    try {
        if (!currentPage.value) {
            ElMessage.warning('页面未加载，将导出空模板');
        }
        const vars = currentPage.value?.variables || {};
        const meta = currentPage.value?.variableMeta || {};
        const header = ['变量名', '类型', '默认值', '公开性', '描述'];
        const data = [header];

        Object.entries(vars).forEach(([name, value]) => {
            const type = meta[name]?.type || inferType(value);
            const access = meta[name]?.access === 'private' ? '私有' : '公开';
            const description = meta[name]?.description || '';
            data.push([
                sanitizeExcelValue(name),
                sanitizeExcelValue(type),
                sanitizeExcelValue(formatExportValue(value)),
                sanitizeExcelValue(access),
                sanitizeExcelValue(description),
            ]);
        });

        const sheet = XLSX.utils.aoa_to_sheet(data);
        const workbook = XLSX.utils.book_new();
        XLSX.utils.book_append_sheet(workbook, sheet, 'Variables');
        const output = XLSX.write(workbook, { bookType: 'xlsx', type: 'array' });
        downloadBlob(new Blob([output], { type: 'application/octet-stream' }), 'variables.xlsx');
    } catch (error) {
        ElMessage.error('导出失败，请检查控制台');
        console.error(error);
    }
};

const triggerImport = () => {
    if (fileInput.value) {
        fileInput.value.click();
    }
};

const handleImport = async (event) => {
    const file = event.target.files?.[0];
    if (!file || !currentPage.value) return;
    try {
        const buffer = await file.arrayBuffer();
        const workbook = XLSX.read(buffer, { type: 'array' });
        const sheetName = workbook.SheetNames[0];
        if (!sheetName) {
            ElMessage.error('Excel 文件为空');
            return;
        }
        const sheet = workbook.Sheets[sheetName];
        const rows = XLSX.utils.sheet_to_json(sheet, { header: 1, defval: '' });
        if (!rows.length) {
            ElMessage.error('Excel 文件为空');
            return;
        }

        const headers = rows[0] || [];
        const nameIndex = resolveHeaderIndex(headers, ['变量名', '名称', 'name', 'variable', '变量']);
        const typeIndex = resolveHeaderIndex(headers, ['类型', 'type']);
        const defaultIndex = resolveHeaderIndex(headers, ['默认值', 'default', 'defaultvalue', '默认']);
        const accessIndex = resolveHeaderIndex(headers, ['公开性', '访问', 'access', '公开']);
        const descIndex = resolveHeaderIndex(headers, ['描述', 'description', 'desc']);

        if (nameIndex === -1) {
            ElMessage.error('Excel 缺少“变量名”列');
            return;
        }

        const variables = {};
        const variableMeta = {};

        rows.slice(1).forEach((row, rowIndex) => {
            const name = String(row[nameIndex] || '').trim();
            if (!name) return;

            const rawType = typeIndex >= 0 ? row[typeIndex] : '';
            const normalizedType = normalizeType(rawType) || inferType(row[defaultIndex]);
            const type = ['String', 'Number', 'Boolean', 'Object', 'Array'].includes(normalizedType)
                ? normalizedType
                : 'String';
            const value = parseImportedValue(type, defaultIndex >= 0 ? row[defaultIndex] : '');
            const access = accessIndex >= 0 ? normalizeAccess(row[accessIndex]) : 'public';
            const description = descIndex >= 0 ? String(row[descIndex] || '').trim() : '';

            variables[name] = value;
            variableMeta[name] = {
                type,
                access,
                description,
                defaultValue: value,
            };
        });

        if (Object.keys(variables).length === 0) {
            ElMessage.error('未读取到变量数据');
            return;
        }

        const hasExisting = Object.keys(currentPage.value.variables || {}).length > 0;
        if (hasExisting) {
            await ElMessageBox.confirm('导入将覆盖现有变量，是否继续？', '导入确认', { type: 'warning', lockScroll: false });
        }

        currentPage.value.variables = { ...variables };
        currentPage.value.variableMeta = { ...variableMeta };
        selectedName.value = '';

        designStore.isDirty = true;
        designStore.saveHistory('导入变量');
        ElMessage.success('导入成功');
    } catch (error) {
        if (error !== 'cancel') {
            ElMessage.error('导入失败，请检查文件内容');
        }
    } finally {
        event.target.value = '';
    }
};
</script>

<style scoped>
.variable-panel {
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: #fff;
}

.variable-header {
    padding: 8px 12px;
    border-bottom: 1px solid #e4e7ed;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
}

.header-title {
    font-size: 14px;
    color: #303133;
    font-weight: 500;
}

.toolbar {
    display: flex;
    align-items: center;
    gap: 6px;
}

.toolbar-btn {
    min-width: 28px;
}

.file-input {
    display: none;
}

.variable-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.value-cell {
    font-family: monospace;
}

.empty-state {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
}

:deep(.el-table) {
    flex: 1;
}
</style>
