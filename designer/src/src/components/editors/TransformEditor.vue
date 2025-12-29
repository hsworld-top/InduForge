<template>
    <div class="transform-editor">
        <div class="section-title">变换</div>
        <el-form label-position="left" label-width="70px" size="small" class="editor-form">
            <!-- 旋转 -->
            <el-form-item label="旋转">
                <div class="slider-input">
                    <el-slider v-model="localStyle.rotate" :min="0" :max="360" :step="1" show-input @change="handleChange" />
                    <span class="unit">°</span>
                </div>
            </el-form-item>

            <!-- 缩放 X -->
            <el-form-item label="缩放 X">
                <div class="slider-input">
                    <el-slider v-model="localStyle.scaleX" :min="0.1" :max="3" :step="0.1" show-input @change="handleChange" />
                </div>
            </el-form-item>

            <!-- 缩放 Y -->
            <el-form-item label="缩放 Y">
                <div class="slider-input">
                    <el-slider v-model="localStyle.scaleY" :min="0.1" :max="3" :step="0.1" show-input @change="handleChange" />
                </div>
            </el-form-item>

            <!-- 锁定缩放比例 -->
            <el-form-item label="锁定缩放">
                <el-switch v-model="lockScale" @change="handleLockScaleChange" />
            </el-form-item>

            <!-- 透明度 -->
            <el-form-item label="透明度">
                <div class="slider-input">
                    <el-slider v-model="localStyle.opacity" :min="0" :max="1" :step="0.01" show-input @change="handleChange" />
                </div>
            </el-form-item>

            <!-- 倾斜 X -->
            <el-form-item label="倾斜 X">
                <div class="slider-input">
                    <el-slider v-model="localStyle.skewX" :min="-45" :max="45" :step="1" show-input @change="handleChange" />
                    <span class="unit">°</span>
                </div>
            </el-form-item>

            <!-- 倾斜 Y -->
            <el-form-item label="倾斜 Y">
                <div class="slider-input">
                    <el-slider v-model="localStyle.skewY" :min="-45" :max="45" :step="1" show-input @change="handleChange" />
                    <span class="unit">°</span>
                </div>
            </el-form-item>

            <!-- 重置按钮 -->
            <el-form-item>
                <el-button size="small" @click="handleReset">重置变换</el-button>
            </el-form-item>
        </el-form>
    </div>
</template>

<script setup>
/**
 * TransformEditor - 变换编辑器
 * Task 6.2: 实现通用属性编辑器
 *
 * 功能：
 * - 编辑 rotation（旋转）
 * - 编辑 scaleX, scaleY（缩放）
 * - 编辑 opacity（透明度）
 * - 编辑 skewX, skewY（倾斜）
 * - 支持锁定缩放比例
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
const lockScale = ref(false);

// Local style
const localStyle = ref({
    rotate: props.style.rotate || 0,
    scaleX: props.style.scaleX || 1,
    scaleY: props.style.scaleY || 1,
    opacity: props.style.opacity !== undefined ? props.style.opacity : 1,
    skewX: props.style.skewX || 0,
    skewY: props.style.skewY || 0,
});

// 监听 props 变化
watch(
    () => props.style,
    (newStyle) => {
        localStyle.value = {
            rotate: newStyle.rotate || 0,
            scaleX: newStyle.scaleX || 1,
            scaleY: newStyle.scaleY || 1,
            opacity: newStyle.opacity !== undefined ? newStyle.opacity : 1,
            skewX: newStyle.skewX || 0,
            skewY: newStyle.skewY || 0,
        };
    },
    { deep: true }
);

// 监听 scaleX 变化（锁定比例时）
watch(
    () => localStyle.value.scaleX,
    (newScaleX, oldScaleX) => {
        if (lockScale.value && newScaleX !== oldScaleX) {
            localStyle.value.scaleY = newScaleX;
            handleChange();
        }
    }
);

// 监听 scaleY 变化（锁定比例时）
watch(
    () => localStyle.value.scaleY,
    (newScaleY, oldScaleY) => {
        if (lockScale.value && newScaleY !== oldScaleY) {
            localStyle.value.scaleX = newScaleY;
            handleChange();
        }
    }
);

/**
 * 处理锁定缩放比例变化
 */
function handleLockScaleChange() {
    if (lockScale.value) {
        // 锁定时，让 Y 与 X 保持一致
        localStyle.value.scaleY = localStyle.value.scaleX;
        handleChange();
    }
}

/**
 * 重置变换
 */
function handleReset() {
    localStyle.value = {
        rotate: 0,
        scaleX: 1,
        scaleY: 1,
        opacity: 1,
        skewX: 0,
        skewY: 0,
    };
    handleChange();
}

/**
 * 处理属性变化
 */
function handleChange() {
    emit('change', { ...localStyle.value });
}
</script>

<style scoped>
.transform-editor {
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

.slider-input {
    display: flex;
    align-items: center;
    gap: 8px;
}

.slider-input .el-slider {
    flex: 1;
}

.unit {
    font-size: 12px;
    color: #909399;
    min-width: 20px;
    text-align: right;
}

:deep(.el-slider__input) {
    width: 80px;
}
</style>

