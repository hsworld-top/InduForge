<template>
  <div class="style-editor">
    <!-- 位置 -->
    <div class="section-group">
      <div class="section-title">位置</div>
      <el-form label-position="left" label-width="60px" size="small">
        <el-row :gutter="8">
          <el-col :span="12">
            <el-form-item label="X">
              <el-input-number
                :model-value="getStyleValue('left')"
                @change="(val) => handleStyleChange('left', val)"
                :controls="false"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="Y">
              <el-input-number
                :model-value="getStyleValue('top')"
                @change="(val) => handleStyleChange('top', val)"
                :controls="false"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </div>
    
    <!-- 尺寸 -->
    <div class="section-group">
      <div class="section-title">尺寸</div>
      <el-form label-position="left" label-width="60px" size="small">
        <el-row :gutter="8">
          <el-col :span="12">
            <el-form-item label="宽">
              <el-input-number
                :model-value="getStyleValue('width')"
                @change="(val) => handleStyleChange('width', val)"
                :min="1"
                :controls="false"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="高">
              <el-input-number
                :model-value="getStyleValue('height')"
                @change="(val) => handleStyleChange('height', val)"
                :min="1"
                :controls="false"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </div>
    
    <!-- 层级 -->
    <div class="section-group">
      <div class="section-title">层级</div>
      <el-form label-position="left" label-width="60px" size="small">
        <el-form-item label="Z-Index">
          <el-input-number
            :model-value="getStyleValue('zIndex')"
            @change="(val) => handleStyleChange('zIndex', val)"
            :min="0"
          />
        </el-form-item>
      </el-form>
    </div>
    
    <!-- 背景 -->
    <div class="section-group">
      <div class="section-title">背景</div>
      <el-form label-position="left" label-width="60px" size="small">
        <el-form-item label="颜色">
          <el-color-picker
            :model-value="getStyleValue('backgroundColor')"
            @change="(val) => handleStyleChange('backgroundColor', val)"
            show-alpha
          />
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup>
/**
 * StyleEditor - 样式编辑器组件
 * 编辑 position, left, top, width, height, zIndex 等样式
 * Requirements: 4.3
 */

const props = defineProps({
  /**
   * 组件样式对象
   */
  style: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:style', 'change'])

/**
 * 获取样式值
 * @param {string} key - 样式属性名
 * @returns {any} 样式值
 */
function getStyleValue(key) {
  return props.style?.[key]
}

/**
 * 处理样式变更
 * Requirements: 4.3 - 修改样式时更新 schema 并重新渲染
 * @param {string} key - 样式属性名
 * @param {any} value - 新值
 */
function handleStyleChange(key, value) {
  const newStyle = { ...props.style, [key]: value }
  emit('update:style', newStyle)
  emit('change', key, value)
}
</script>

<style scoped>
.style-editor {
  padding: 12px;
}

.section-group {
  margin-bottom: 16px;
}

.section-group:last-child {
  margin-bottom: 0;
}

.section-title {
  font-size: 12px;
  font-weight: 500;
  color: #909399;
  margin-bottom: 8px;
  text-transform: uppercase;
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
</style>
