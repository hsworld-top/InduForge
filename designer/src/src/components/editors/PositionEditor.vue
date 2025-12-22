<template>
    <div class="position-editor">
        <div class="section-title">位置与尺寸</div>
        <el-form label-position="left" label-width="60px" size="small" class="editor-form">
            <!-- 位置 -->
            <div class="form-row">
                <el-form-item label="X">
                    <el-input-number v-model="localStyle.left" :min="0" :step="1" :controls="false" @change="handleChange" />
                </el-form-item>
                <el-form-item label="Y">
                    <el-input-number v-model="localStyle.top" :min="0" :step="1" :controls="false" @change="handleChange" />
                </el-form-item>
            </div>

            <!-- 尺寸 -->
            <div class="form-row">
                <el-form-item label="宽度">
                    <el-input-number v-model="localStyle.width" :min="0" :step="1" :controls="false" @change="handleChange" />
                </el-form-item>
                <el-form-item label="高度">
                    <el-input-number v-model="localStyle.height" :min="0" :step="1" :controls="false" @change="handleChange" />
                </el-form-item>
            </div>

            <!-- 锁定宽高比 -->
            <el-form-item label="锁定比例">
                <el-switch v-model="lockAspectRatio" />
            </el-form-item>

            <!-- 定位类型 -->
            <el-form-item label="定位">
                <el-select v-model="localStyle.position" @change="handleChange" placeholder="选择定位">
                    <el-option label="相对定位" value="relative" />
                    <el-option label="绝对定位" value="absolute" />
                    <el-option label="固定定位" value="fixed" />
                    <el-option label="静态定位" value="static" />
                </el-select>
            </el-form-item>

            <!-- Z-Index -->
            <el-form-item label="层级">
                <el-input-number v-model="localStyle.zIndex" :min="0" :step="1" :controls="false" @change="handleChange" />
            </el-form-item>
        </el-form>
    </div>
</template>

<script setup>
/**
 * PositionEditor - 位置与尺寸编辑器
 * Task 6.2: 实现通用属性编辑器
 *
 * 功能：
 * - 编辑组件的 left, top, width, height
 * - 支持锁定宽高比
 * - 支持 position 和 z-index
 */
import { ref, computed, watch } from 'vue';

// Props
const props = defineProps({
    /**
     * 样式对象
     */
    style: {
        type: Object,
        default: () => ({}),
    },
});

// Emits
const emit = defineEmits(['change']);

// State
const lockAspectRatio = ref(false);
const aspectRatio = ref(1);

// Local style
const localStyle = ref({
    left: props.style.left || 0,
    top: props.style.top || 0,
    width: props.style.width || 100,
    height: props.style.height || 100,
    position: props.style.position || 'absolute',
    zIndex: props.style.zIndex || 'auto',
});

// 监听 props 变化
watch(
    () => props.style,
    (newStyle) => {
        localStyle.value = {
            left: newStyle.left || 0,
            top: newStyle.top || 0,
            width: newStyle.width || 100,
            height: newStyle.height || 100,
            position: newStyle.position || 'absolute',
            zIndex: newStyle.zIndex || 'auto',
        };
    },
    { deep: true }
);

// 计算宽高比
watch(
    () => [localStyle.value.width, localStyle.value.height],
    ([width, height]) => {
        if (width && height && !lockAspectRatio.value) {
            aspectRatio.value = width / height;
        }
    }
);

// 监听宽度变化（锁定宽高比时）
watch(
    () => localStyle.value.width,
    (newWidth, oldWidth) => {
        if (lockAspectRatio.value && aspectRatio.value && newWidth !== oldWidth) {
            localStyle.value.height = Math.round(newWidth / aspectRatio.value);
            handleChange();
        }
    }
);

// 监听高度变化（锁定宽高比时）
watch(
    () => localStyle.value.height,
    (newHeight, oldHeight) => {
        if (lockAspectRatio.value && aspectRatio.value && newHeight !== oldHeight) {
            localStyle.value.width = Math.round(newHeight * aspectRatio.value);
            handleChange();
        }
    }
);

/**
 * 处理属性变化
 */
function handleChange() {
    emit('change', { ...localStyle.value });
}
</script>

<style scoped>
.position-editor {
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

.form-row {
    display: flex;
    gap: 8px;
}

.form-row .el-form-item {
    flex: 1;
    margin-bottom: 12px;
}

.el-input-number {
    width: 100%;
}

:deep(.el-input-number .el-input__inner) {
    text-align: left;
}
</style>

