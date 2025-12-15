<template>
    <el-form
        class="ui-form-component"
        :model="model"
        :rules="rules"
        :label-width="labelWidth"
        :label-position="labelPosition"
        :inline="inline"
        :size="size"
        :disabled="disabled"
        :validate-on-rule-change="validateOnRuleChange"
        :hide-required-asterisk="hideRequiredAsterisk"
        :show-message="showMessage"
        :inline-message="inlineMessage"
        :status-icon="statusIcon"
        :style="mergedStyle">
        <slot>
            <el-form-item label="字段A">
                <el-input placeholder="请输入内容" />
            </el-form-item>
            <el-form-item label="字段B">
                <el-select placeholder="请选择">
                    <el-option label="选项1" value="1" />
                    <el-option label="选项2" value="2" />
                </el-select>
            </el-form-item>
        </slot>
    </el-form>
</template>

<script setup>
import { computed, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({
    name: 'UIFormComponent',
});

const props = defineProps({
    model: { type: Object, default: () => ({}) },
    rules: { type: Object, default: () => ({}) },
    labelWidth: { type: [String, Number], default: '80px' },
    labelPosition: { type: String, default: 'right' },
    inline: { type: Boolean, default: false },
    size: { type: String, default: 'default' },
    disabled: { type: Boolean, default: false },
    validateOnRuleChange: { type: Boolean, default: true },
    hideRequiredAsterisk: { type: Boolean, default: false },
    showMessage: { type: Boolean, default: true },
    inlineMessage: { type: Boolean, default: false },
    statusIcon: { type: Boolean, default: false },
});

const attrs = useAttrs();
const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));
</script>

<style scoped>
.ui-form-component {
    width: 100%;
    height: 100%;
    padding: 12px;
    box-sizing: border-box;
    background: #fff;
    border: 1px dashed #e5e7eb;
}
</style>
