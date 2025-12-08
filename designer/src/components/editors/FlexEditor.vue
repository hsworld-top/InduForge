<template>
    <div class="flex-editor">
        <div class="section-title">Flexbox 布局</div>
        <el-form label-position="left" label-width="90px" size="small" class="editor-form">
            <!-- Flex Direction -->
            <el-form-item label="排列方向">
                <el-select v-model="localProps.flexDirection" @change="handleChange" placeholder="选择方向">
                <el-option label="水平 (row)" value="row">
                    <div class="option-with-icon">
                        <IconTablerArrowRight />
                        <span>水平 (row)</span>
                    </div>
                </el-option>
                <el-option label="水平反向 (row-reverse)" value="row-reverse">
                    <div class="option-with-icon">
                        <IconTablerArrowLeft />
                        <span>水平反向</span>
                    </div>
                </el-option>
                <el-option label="垂直 (column)" value="column">
                    <div class="option-with-icon">
                        <IconTablerArrowDown />
                        <span>垂直 (column)</span>
                    </div>
                </el-option>
                <el-option label="垂直反向 (column-reverse)" value="column-reverse">
                    <div class="option-with-icon">
                        <IconTablerArrowUp />
                        <span>垂直反向</span>
                    </div>
                </el-option>
                </el-select>
            </el-form-item>

            <!-- Justify Content -->
            <el-form-item label="主轴对齐">
                <el-select v-model="localProps.justifyContent" @change="handleChange" placeholder="主轴对齐">
                    <el-option label="起始对齐" value="flex-start" />
                    <el-option label="结束对齐" value="flex-end" />
                    <el-option label="居中对齐" value="center" />
                    <el-option label="两端对齐" value="space-between" />
                    <el-option label="均匀分布" value="space-around" />
                    <el-option label="等距分布" value="space-evenly" />
                </el-select>
            </el-form-item>

            <!-- Align Items -->
            <el-form-item label="交叉轴对齐">
                <el-select v-model="localProps.alignItems" @change="handleChange" placeholder="交叉轴对齐">
                    <el-option label="拉伸" value="stretch" />
                    <el-option label="起始对齐" value="flex-start" />
                    <el-option label="结束对齐" value="flex-end" />
                    <el-option label="居中对齐" value="center" />
                    <el-option label="基线对齐" value="baseline" />
                </el-select>
            </el-form-item>

            <!-- Flex Wrap -->
            <el-form-item label="换行">
                <el-select v-model="localProps.flexWrap" @change="handleChange" placeholder="换行">
                    <el-option label="不换行" value="nowrap" />
                    <el-option label="换行" value="wrap" />
                    <el-option label="反向换行" value="wrap-reverse" />
                </el-select>
            </el-form-item>

            <!-- Align Content (多行时) -->
            <el-form-item label="多行对齐" v-if="localProps.flexWrap !== 'nowrap'">
                <el-select v-model="localProps.alignContent" @change="handleChange" placeholder="多行对齐">
                    <el-option label="起始对齐" value="flex-start" />
                    <el-option label="结束对齐" value="flex-end" />
                    <el-option label="居中对齐" value="center" />
                    <el-option label="两端对齐" value="space-between" />
                    <el-option label="均匀分布" value="space-around" />
                    <el-option label="拉伸" value="stretch" />
                </el-select>
            </el-form-item>

            <!-- Gap -->
            <el-form-item label="间距 (Gap)">
                <el-input-number v-model="localProps.gap" :min="0" :step="4" :controls="false" @change="handleChange" />
            </el-form-item>
        </el-form>
    </div>
</template>

<script setup>
/**
 * FlexEditor - Flexbox 布局编辑器
 * Task 6.4: 实现布局属性编辑器
 *
 * 功能：
 * - 编辑 flexDirection, justifyContent, alignItems
 * - 编辑 flexWrap, alignContent
 * - 编辑 gap
 */
import { ref, watch } from 'vue';
import IconTablerArrowRight from '~icons/tabler/arrow-right';
import IconTablerArrowLeft from '~icons/tabler/arrow-left';
import IconTablerArrowDown from '~icons/tabler/arrow-down';
import IconTablerArrowUp from '~icons/tabler/arrow-up';

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
    flexDirection: props.props.flexDirection || 'row',
    justifyContent: props.props.justifyContent || 'flex-start',
    alignItems: props.props.alignItems || 'stretch',
    flexWrap: props.props.flexWrap || 'nowrap',
    alignContent: props.props.alignContent || 'stretch',
    gap: props.props.gap || 0,
});

// 监听 props 变化
watch(
    () => props.props,
    (newProps) => {
        localProps.value = {
            flexDirection: newProps.flexDirection || 'row',
            justifyContent: newProps.justifyContent || 'flex-start',
            alignItems: newProps.alignItems || 'stretch',
            flexWrap: newProps.flexWrap || 'nowrap',
            alignContent: newProps.alignContent || 'stretch',
            gap: newProps.gap || 0,
        };
    },
    { deep: true }
);

/**
 * 处理属性变化
 */
function handleChange() {
    emit('change', { ...localProps.value });
}
</script>

<style scoped>
.flex-editor {
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

.option-with-icon {
    display: flex;
    align-items: center;
    gap: 8px;
}

.el-input-number {
    width: 100%;
}

:deep(.el-input-number .el-input__inner) {
    text-align: left;
}
</style>

