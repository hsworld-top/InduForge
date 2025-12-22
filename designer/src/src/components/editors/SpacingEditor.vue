<template>
    <div class="spacing-editor">
        <div class="section-title">间距</div>
        <el-form label-position="left" label-width="70px" size="small" class="editor-form">
            <!-- Margin -->
            <div class="spacing-group">
                <div class="group-header">
                    <span>外边距 (Margin)</span>
                    <el-button size="small" text @click="toggleMarginUnified">
                        <el-icon>
                            <component :is="marginUnified ? 'Lock' : 'Unlock'" />
                        </el-icon>
                    </el-button>
                </div>

                <div v-if="marginUnified" class="unified-input">
                    <el-form-item label="全部">
                        <el-input-number v-model="unifiedMargin" :step="1" :controls="false" @change="handleUnifiedMarginChange" />
                    </el-form-item>
                </div>

                <div v-else class="spacing-box">
                    <div class="spacing-input top">
                        <el-input-number v-model="localStyle.marginTop" :step="1" :controls="false" placeholder="上" @change="handleChange" />
                    </div>
                    <div class="spacing-middle">
                        <div class="spacing-input left">
                            <el-input-number v-model="localStyle.marginLeft" :step="1" :controls="false" placeholder="左" @change="handleChange" />
                        </div>
                        <div class="spacing-center">
                            <span class="center-label">M</span>
                        </div>
                        <div class="spacing-input right">
                            <el-input-number v-model="localStyle.marginRight" :step="1" :controls="false" placeholder="右" @change="handleChange" />
                        </div>
                    </div>
                    <div class="spacing-input bottom">
                        <el-input-number v-model="localStyle.marginBottom" :step="1" :controls="false" placeholder="下" @change="handleChange" />
                    </div>
                </div>
            </div>

            <!-- Padding -->
            <div class="spacing-group">
                <div class="group-header">
                    <span>内边距 (Padding)</span>
                    <el-button size="small" text @click="togglePaddingUnified">
                        <el-icon>
                            <component :is="paddingUnified ? 'Lock' : 'Unlock'" />
                        </el-icon>
                    </el-button>
                </div>

                <div v-if="paddingUnified" class="unified-input">
                    <el-form-item label="全部">
                        <el-input-number v-model="unifiedPadding" :step="1" :controls="false" @change="handleUnifiedPaddingChange" />
                    </el-form-item>
                </div>

                <div v-else class="spacing-box">
                    <div class="spacing-input top">
                        <el-input-number v-model="localStyle.paddingTop" :step="1" :controls="false" placeholder="上" @change="handleChange" />
                    </div>
                    <div class="spacing-middle">
                        <div class="spacing-input left">
                            <el-input-number v-model="localStyle.paddingLeft" :step="1" :controls="false" placeholder="左" @change="handleChange" />
                        </div>
                        <div class="spacing-center">
                            <span class="center-label">P</span>
                        </div>
                        <div class="spacing-input right">
                            <el-input-number v-model="localStyle.paddingRight" :step="1" :controls="false" placeholder="右" @change="handleChange" />
                        </div>
                    </div>
                    <div class="spacing-input bottom">
                        <el-input-number v-model="localStyle.paddingBottom" :step="1" :controls="false" placeholder="下" @change="handleChange" />
                    </div>
                </div>
            </div>
        </el-form>
    </div>
</template>

<script setup>
/**
 * SpacingEditor - 间距编辑器
 * Task 6.2: 实现通用属性编辑器
 *
 * 功能：
 * - 编辑 margin (marginTop, marginRight, marginBottom, marginLeft)
 * - 编辑 padding (paddingTop, paddingRight, paddingBottom, paddingLeft)
 * - 支持统一设置或分别设置
 */
import { ref, computed, watch } from 'vue';
import IconTablerLock from '~icons/tabler/lock';
import IconTablerLockOpen from '~icons/tabler/lock-open';

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
const marginUnified = ref(false);
const paddingUnified = ref(false);
const unifiedMargin = ref(0);
const unifiedPadding = ref(0);

// Local style
const localStyle = ref({
    marginTop: props.style.marginTop || 0,
    marginRight: props.style.marginRight || 0,
    marginBottom: props.style.marginBottom || 0,
    marginLeft: props.style.marginLeft || 0,
    paddingTop: props.style.paddingTop || 0,
    paddingRight: props.style.paddingRight || 0,
    paddingBottom: props.style.paddingBottom || 0,
    paddingLeft: props.style.paddingLeft || 0,
});

// 监听 props 变化
watch(
    () => props.style,
    (newStyle) => {
        localStyle.value = {
            marginTop: newStyle.marginTop || 0,
            marginRight: newStyle.marginRight || 0,
            marginBottom: newStyle.marginBottom || 0,
            marginLeft: newStyle.marginLeft || 0,
            paddingTop: newStyle.paddingTop || 0,
            paddingRight: newStyle.paddingRight || 0,
            paddingBottom: newStyle.paddingBottom || 0,
            paddingLeft: newStyle.paddingLeft || 0,
        };
    },
    { deep: true }
);

/**
 * 切换 margin 统一设置
 */
function toggleMarginUnified() {
    marginUnified.value = !marginUnified.value;
    if (marginUnified.value) {
        // 取第一个值作为统一值
        unifiedMargin.value = localStyle.value.marginTop || 0;
    }
}

/**
 * 切换 padding 统一设置
 */
function togglePaddingUnified() {
    paddingUnified.value = !paddingUnified.value;
    if (paddingUnified.value) {
        // 取第一个值作为统一值
        unifiedPadding.value = localStyle.value.paddingTop || 0;
    }
}

/**
 * 处理统一 margin 变化
 */
function handleUnifiedMarginChange() {
    localStyle.value.marginTop = unifiedMargin.value;
    localStyle.value.marginRight = unifiedMargin.value;
    localStyle.value.marginBottom = unifiedMargin.value;
    localStyle.value.marginLeft = unifiedMargin.value;
    handleChange();
}

/**
 * 处理统一 padding 变化
 */
function handleUnifiedPaddingChange() {
    localStyle.value.paddingTop = unifiedPadding.value;
    localStyle.value.paddingRight = unifiedPadding.value;
    localStyle.value.paddingBottom = unifiedPadding.value;
    localStyle.value.paddingLeft = unifiedPadding.value;
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
.spacing-editor {
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

.spacing-group {
    margin-bottom: 20px;
}

.group-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    font-size: 12px;
    color: #606266;
}

.unified-input {
    margin-top: 8px;
}

.spacing-box {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
    background-color: #f5f7fa;
    border-radius: 4px;
}

.spacing-middle {
    display: flex;
    gap: 8px;
    align-items: center;
}

.spacing-input {
    flex: 1;
}

.spacing-input.top,
.spacing-input.bottom {
    display: flex;
    justify-content: center;
}

.spacing-input.top .el-input-number,
.spacing-input.bottom .el-input-number {
    width: 80px;
}

.spacing-center {
    flex: 0 0 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: #e4e7ed;
    border-radius: 4px;
    height: 32px;
}

.center-label {
    font-size: 12px;
    font-weight: 600;
    color: #909399;
}

.el-input-number {
    width: 100%;
}

:deep(.el-input-number .el-input__inner) {
    text-align: center;
    padding: 0 8px;
}
</style>

