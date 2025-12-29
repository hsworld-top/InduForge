<template>
    <div class="grid-editor">
        <div class="section-title">Grid 布局</div>
        <el-form label-position="left" label-width="100px" size="small" class="editor-form">
            <!-- Grid Template Columns -->
            <el-form-item label="列模板">
                <el-input v-model="localProps.gridTemplateColumns" placeholder="例如: 1fr 1fr 1fr" @change="handleChange">
                    <template #append>
                    <el-dropdown @command="handleColumnsPreset">
                        <el-button>
                            <IconTablerLayoutGrid />
                        </el-button>
                            <template #dropdown>
                                <el-dropdown-menu>
                                    <el-dropdown-item command="1fr">1 列</el-dropdown-item>
                                    <el-dropdown-item command="1fr 1fr">2 列</el-dropdown-item>
                                    <el-dropdown-item command="1fr 1fr 1fr">3 列</el-dropdown-item>
                                    <el-dropdown-item command="1fr 1fr 1fr 1fr">4 列</el-dropdown-item>
                                    <el-dropdown-item command="repeat(auto-fill, minmax(200px, 1fr))">自适应</el-dropdown-item>
                                </el-dropdown-menu>
                            </template>
                        </el-dropdown>
                    </template>
                </el-input>
            </el-form-item>

            <!-- Grid Template Rows -->
            <el-form-item label="行模板">
                <el-input v-model="localProps.gridTemplateRows" placeholder="例如: auto auto" @change="handleChange">
                    <template #append>
                    <el-dropdown @command="handleRowsPreset">
                        <el-button>
                            <IconTablerLayoutGrid />
                        </el-button>
                            <template #dropdown>
                                <el-dropdown-menu>
                                    <el-dropdown-item command="auto">1 行</el-dropdown-item>
                                    <el-dropdown-item command="auto auto">2 行</el-dropdown-item>
                                    <el-dropdown-item command="auto auto auto">3 行</el-dropdown-item>
                                    <el-dropdown-item command="repeat(auto-fill, minmax(100px, 1fr))">自适应</el-dropdown-item>
                                </el-dropdown-menu>
                            </template>
                        </el-dropdown>
                    </template>
                </el-input>
            </el-form-item>

            <!-- Gap -->
            <el-form-item label="间距 (Gap)">
                <el-input-number v-model="localProps.gap" :min="0" :step="4" :controls="false" @change="handleChange" />
            </el-form-item>

            <!-- Grid Auto Flow -->
            <el-form-item label="自动流">
                <el-select v-model="localProps.gridAutoFlow" @change="handleChange" placeholder="自动流">
                    <el-option label="按行填充 (row)" value="row" />
                    <el-option label="按列填充 (column)" value="column" />
                    <el-option label="密集行 (row dense)" value="row dense" />
                    <el-option label="密集列 (column dense)" value="column dense" />
                </el-select>
            </el-form-item>

            <!-- Justify Items -->
            <el-form-item label="单元格水平对齐">
                <el-select v-model="localProps.justifyItems" @change="handleChange" placeholder="水平对齐">
                    <el-option label="拉伸" value="stretch" />
                    <el-option label="起始对齐" value="start" />
                    <el-option label="结束对齐" value="end" />
                    <el-option label="居中对齐" value="center" />
                </el-select>
            </el-form-item>

            <!-- Align Items -->
            <el-form-item label="单元格垂直对齐">
                <el-select v-model="localProps.alignItems" @change="handleChange" placeholder="垂直对齐">
                    <el-option label="拉伸" value="stretch" />
                    <el-option label="起始对齐" value="start" />
                    <el-option label="结束对齐" value="end" />
                    <el-option label="居中对齐" value="center" />
                </el-select>
            </el-form-item>

            <!-- Justify Content -->
            <el-form-item label="网格水平对齐">
                <el-select v-model="localProps.justifyContent" @change="handleChange" placeholder="水平对齐">
                    <el-option label="起始对齐" value="start" />
                    <el-option label="结束对齐" value="end" />
                    <el-option label="居中对齐" value="center" />
                    <el-option label="拉伸" value="stretch" />
                    <el-option label="两端对齐" value="space-between" />
                    <el-option label="均匀分布" value="space-around" />
                    <el-option label="等距分布" value="space-evenly" />
                </el-select>
            </el-form-item>

            <!-- Align Content -->
            <el-form-item label="网格垂直对齐">
                <el-select v-model="localProps.alignContent" @change="handleChange" placeholder="垂直对齐">
                    <el-option label="起始对齐" value="start" />
                    <el-option label="结束对齐" value="end" />
                    <el-option label="居中对齐" value="center" />
                    <el-option label="拉伸" value="stretch" />
                    <el-option label="两端对齐" value="space-between" />
                    <el-option label="均匀分布" value="space-around" />
                    <el-option label="等距分布" value="space-evenly" />
                </el-select>
            </el-form-item>
        </el-form>
    </div>
</template>

<script setup>
/**
 * GridEditor - Grid 布局编辑器
 * Task 6.4: 实现布局属性编辑器
 *
 * 功能：
 * - 编辑 gridTemplateColumns, gridTemplateRows
 * - 编辑 gap, gridAutoFlow
 * - 编辑对齐属性
 * - 提供常用布局预设
 */
import { ref, watch } from 'vue';
import IconTablerLayoutGrid from '~icons/tabler/layout-grid';

// Props
const props = defineProps({
    /**
     * 组件 props
     */
    props: {
        type: Object,
        default: () => ({}),
    },
});

// Emits
const emit = defineEmits(['change']);

// Local props
const localProps = ref({
    gridTemplateColumns: props.props.gridTemplateColumns || 'repeat(3, 1fr)',
    gridTemplateRows: props.props.gridTemplateRows || 'auto',
    gap: props.props.gap || 0,
    gridAutoFlow: props.props.gridAutoFlow || 'row',
    justifyItems: props.props.justifyItems || 'stretch',
    alignItems: props.props.alignItems || 'stretch',
    justifyContent: props.props.justifyContent || 'start',
    alignContent: props.props.alignContent || 'start',
});

// 监听 props 变化
watch(
    () => props.props,
    (newProps) => {
        localProps.value = {
            gridTemplateColumns: newProps.gridTemplateColumns || 'repeat(3, 1fr)',
            gridTemplateRows: newProps.gridTemplateRows || 'auto',
            gap: newProps.gap || 0,
            gridAutoFlow: newProps.gridAutoFlow || 'row',
            justifyItems: newProps.justifyItems || 'stretch',
            alignItems: newProps.alignItems || 'stretch',
            justifyContent: newProps.justifyContent || 'start',
            alignContent: newProps.alignContent || 'start',
        };
    },
    { deep: true }
);

/**
 * 处理列模板预设
 */
function handleColumnsPreset(command) {
    localProps.value.gridTemplateColumns = command;
    handleChange();
}

/**
 * 处理行模板预设
 */
function handleRowsPreset(command) {
    localProps.value.gridTemplateRows = command;
    handleChange();
}

/**
 * 处理属性变化
 */
function handleChange() {
    emit('change', { ...localProps.value });
}
</script>

<style scoped>
.grid-editor {
    padding: 12px;
}

.section-title {
    font-size: 13px;
    font-weight: 600;
    color: #303133;
    margin-bottom: 12px;
    padding-bottom: 8px;
    border-bottom: 1px solid #dcdfe6;
}

.editor-form {
    margin-top: 8px;
}

.el-input-number {
    width: 100%;
}

:deep(.el-input-number .el-input__inner) {
    text-align: left;
}

:deep(.el-input-group__append) {
    padding: 0 8px;
}
</style>

