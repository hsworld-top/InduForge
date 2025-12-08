<template>
    <div class="text-component-editor">
        <div class="section-title">文本属性</div>
        <el-form label-position="left" label-width="80px" size="small" class="editor-form">
            <!-- 文本内容 -->
            <el-form-item label="文本内容">
                <el-input v-model="localProps.text" type="textarea" :rows="3" placeholder="输入文本内容" @change="handleChange" />
            </el-form-item>

            <!-- 字体 -->
            <el-form-item label="字体">
                <el-select v-model="localStyle.fontFamily" @change="handleStyleChange" placeholder="选择字体">
                    <el-option label="系统默认" value="system-ui" />
                    <el-option label="宋体" value="SimSun" />
                    <el-option label="黑体" value="SimHei" />
                    <el-option label="微软雅黑" value="Microsoft YaHei" />
                    <el-option label="Arial" value="Arial" />
                    <el-option label="Times New Roman" value="Times New Roman" />
                    <el-option label="Courier New" value="Courier New" />
                </el-select>
            </el-form-item>

            <!-- 字号 -->
            <el-form-item label="字号">
                <el-input-number v-model="localStyle.fontSize" :min="12" :max="72" :step="1" :controls="false" @change="handleStyleChange" />
            </el-form-item>

            <!-- 字重 -->
            <el-form-item label="字重">
                <el-select v-model="localStyle.fontWeight" @change="handleStyleChange" placeholder="字重">
                    <el-option label="正常 (400)" value="400" />
                    <el-option label="中等 (500)" value="500" />
                    <el-option label="粗体 (600)" value="600" />
                    <el-option label="加粗 (700)" value="700" />
                    <el-option label="特粗 (900)" value="900" />
                </el-select>
            </el-form-item>

            <!-- 颜色 -->
            <el-form-item label="颜色">
                <el-color-picker v-model="localStyle.color" show-alpha @change="handleStyleChange" />
            </el-form-item>

            <!-- 对齐方式 -->
            <el-form-item label="对齐">
                <el-radio-group v-model="localStyle.textAlign" @change="handleStyleChange">
                    <el-radio-button value="left">
                        <el-icon><AlignLeft /></el-icon>
                    </el-radio-button>
                    <el-radio-button value="center">
                        <el-icon><AlignCenter /></el-icon>
                    </el-radio-button>
                    <el-radio-button value="right">
                        <el-icon><AlignRight /></el-icon>
                    </el-radio-button>
                    <el-radio-button value="justify">
                        <el-icon><Document /></el-icon>
                    </el-radio-button>
                </el-radio-group>
            </el-form-item>

            <!-- 行高 -->
            <el-form-item label="行高">
                <el-input-number v-model="localStyle.lineHeight" :min="1" :max="3" :step="0.1" :precision="1" :controls="false" @change="handleStyleChange" />
            </el-form-item>

            <!-- 字间距 -->
            <el-form-item label="字间距">
                <el-input-number v-model="localStyle.letterSpacing" :min="-5" :max="10" :step="0.5" :controls="false" @change="handleStyleChange" />
            </el-form-item>

            <!-- 文本装饰 -->
            <el-form-item label="装饰">
                <el-checkbox-group v-model="textDecorations" @change="handleTextDecorationChange">
                    <el-checkbox value="underline">下划线</el-checkbox>
                    <el-checkbox value="line-through">删除线</el-checkbox>
                    <el-checkbox value="overline">上划线</el-checkbox>
                </el-checkbox-group>
            </el-form-item>

            <!-- 文本样式 -->
            <el-form-item label="样式">
                <el-select v-model="localStyle.fontStyle" @change="handleStyleChange" placeholder="样式">
                    <el-option label="正常" value="normal" />
                    <el-option label="斜体" value="italic" />
                    <el-option label="倾斜" value="oblique" />
                </el-select>
            </el-form-item>
        </el-form>
    </div>
</template>

<script setup>
/**
 * TextComponentEditor - 文本组件编辑器
 * Task 6.3: 实现组件专有属性编辑器
 *
 * 功能：
 * - 编辑文本内容
 * - 编辑字体、字号、字重
 * - 编辑颜色、对齐方式
 * - 编辑行高、字间距
 * - 编辑文本装饰和样式
 */
import { ref, computed, watch } from 'vue';
import IconTablerAlignLeft from '~icons/tabler/align-left';
import IconTablerAlignCenter from '~icons/tabler/align-center';
import IconTablerAlignRight from '~icons/tabler/align-right';
import IconTablerAlignJustified from '~icons/tabler/align-justified';

// Props
const props = defineProps({
    /**
     * 组件 props
     */
    props: {
        type: Object,
        default: () => ({}),
    },
    /**
     * 组件 style
     */
    style: {
        type: Object,
        default: () => ({}),
    },
});

// Emits
const emit = defineEmits(['change-props', 'change-style']);

// Local props
const localProps = ref({
    text: props.props.text || '文本内容',
});

// Local style
const localStyle = ref({
    fontFamily: props.style.fontFamily || 'system-ui',
    fontSize: props.style.fontSize || 14,
    fontWeight: props.style.fontWeight || '400',
    color: props.style.color || '#000000',
    textAlign: props.style.textAlign || 'left',
    lineHeight: props.style.lineHeight || 1.5,
    letterSpacing: props.style.letterSpacing || 0,
    textDecoration: props.style.textDecoration || 'none',
    fontStyle: props.style.fontStyle || 'normal',
});

// 文本装饰（多选）
const textDecorations = ref([]);

// 初始化文本装饰
if (localStyle.value.textDecoration && localStyle.value.textDecoration !== 'none') {
    textDecorations.value = localStyle.value.textDecoration.split(' ');
}

// 监听 props 变化
watch(
    () => props.props,
    (newProps) => {
        localProps.value = {
            text: newProps.text || '文本内容',
        };
    },
    { deep: true }
);

watch(
    () => props.style,
    (newStyle) => {
        localStyle.value = {
            fontFamily: newStyle.fontFamily || 'system-ui',
            fontSize: newStyle.fontSize || 14,
            fontWeight: newStyle.fontWeight || '400',
            color: newStyle.color || '#000000',
            textAlign: newStyle.textAlign || 'left',
            lineHeight: newStyle.lineHeight || 1.5,
            letterSpacing: newStyle.letterSpacing || 0,
            textDecoration: newStyle.textDecoration || 'none',
            fontStyle: newStyle.fontStyle || 'normal',
        };
    },
    { deep: true }
);

/**
 * 处理文本装饰变化
 */
function handleTextDecorationChange() {
    localStyle.value.textDecoration = textDecorations.value.length > 0 ? textDecorations.value.join(' ') : 'none';
    handleStyleChange();
}

/**
 * 处理 props 变化
 */
function handleChange() {
    emit('change-props', { ...localProps.value });
}

/**
 * 处理 style 变化
 */
function handleStyleChange() {
    emit('change-style', { ...localStyle.value });
}
</script>

<style scoped>
.text-component-editor {
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

:deep(.el-radio-button__inner) {
    padding: 8px 12px;
}
</style>

