<template>
  <div
    class="canvas-component"
    :class="{
      'is-selected': isSelected,
      'is-locked': component.locked,
      'is-hidden': !component.visible,
    }"
    :style="componentStyle"
    @click.stop="handleClick"
    @mousedown.stop="handleMouseDown"
  >
    <!-- 组件内容渲染 -->
    <component
      :is="renderComponent"
      v-bind="component.props"
      class="component-content"
    />
    
    <!-- 递归渲染子组件 -->
    <CanvasComponent
      v-for="child in component.children"
      :key="child.id"
      :component="child"
      :scale="scale"
      @select="$emit('select', $event)"
    />
    
    <!-- 锁定图标 -->
    <div v-if="component.locked" class="lock-indicator">
      <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
        <path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"/>
      </svg>
    </div>
  </div>
</template>

<script setup>
/**
 * CanvasComponent - 画布组件渲染器
 * 根据 component schema 渲染组件
 * Requirements: 2.1, 2.2
 */
import { computed, h, defineAsyncComponent } from 'vue'
import { useDesignStore } from '@/store/design'

// Props
const props = defineProps({
  component: {
    type: Object,
    required: true,
  },
  scale: {
    type: Number,
    default: 1,
  },
})

// Emits
const emit = defineEmits(['select'])

// Store
const designStore = useDesignStore()

// Computed
const isSelected = computed(() => {
  return designStore.selectedComponentId === props.component.id
})

/**
 * 组件样式
 * Requirements: 2.1 - 根据 style 属性渲染组件
 * Requirements: 2.2 - 绝对定位组件放置在指定的 left/top 坐标
 * Property 1: Component Style Rendering
 */
const componentStyle = computed(() => {
  const style = props.component.style || {}
  
  return {
    position: style.position || 'absolute',
    left: typeof style.left === 'number' ? `${style.left}px` : style.left || '0px',
    top: typeof style.top === 'number' ? `${style.top}px` : style.top || '0px',
    width: typeof style.width === 'number' ? `${style.width}px` : style.width || 'auto',
    height: typeof style.height === 'number' ? `${style.height}px` : style.height || 'auto',
    zIndex: style.zIndex || 'auto',
    opacity: props.component.visible === false ? 0.3 : 1,
    pointerEvents: props.component.locked ? 'none' : 'auto',
  }
})

/**
 * 渲染组件
 * 根据组件类型返回对应的渲染组件
 */
const renderComponent = computed(() => {
  const type = props.component.type
  
  // 基础组件映射
  const componentMap = {
    Container: ContainerRenderer,
    Text: TextRenderer,
    Button: ButtonRenderer,
    Image: ImageRenderer,
    Input: InputRenderer,
  }
  
  return componentMap[type] || PlaceholderRenderer
})

// 内置渲染器组件

/**
 * 容器渲染器
 */
const ContainerRenderer = {
  name: 'ContainerRenderer',
  props: ['layout', 'gap', 'padding', 'backgroundColor', 'borderRadius', 'border'],
  setup(props, { slots }) {
    return () => h('div', {
      class: 'renderer-container',
      style: {
        display: props.layout === 'flex' ? 'flex' : props.layout === 'grid' ? 'grid' : 'block',
        gap: props.gap ? `${props.gap}px` : undefined,
        padding: props.padding ? `${props.padding}px` : undefined,
        backgroundColor: props.backgroundColor,
        borderRadius: props.borderRadius ? `${props.borderRadius}px` : undefined,
        border: props.border,
        width: '100%',
        height: '100%',
      },
    }, slots.default?.())
  },
}

/**
 * 文本渲染器
 */
const TextRenderer = {
  name: 'TextRenderer',
  props: ['content', 'fontSize', 'fontWeight', 'color', 'textAlign', 'lineHeight'],
  setup(props) {
    return () => h('span', {
      class: 'renderer-text',
      style: {
        fontSize: props.fontSize ? `${props.fontSize}px` : '14px',
        fontWeight: props.fontWeight || 'normal',
        color: props.color || '#333333',
        textAlign: props.textAlign || 'left',
        lineHeight: props.lineHeight || 1.5,
        display: 'block',
        width: '100%',
        height: '100%',
        overflow: 'hidden',
      },
    }, props.content || '文本内容')
  },
}

/**
 * 按钮渲染器
 */
const ButtonRenderer = {
  name: 'ButtonRenderer',
  props: ['text', 'type', 'disabled', 'backgroundColor', 'textColor', 'borderRadius'],
  setup(props) {
    const typeColors = {
      primary: { bg: '#409eff', text: '#ffffff' },
      success: { bg: '#67c23a', text: '#ffffff' },
      warning: { bg: '#e6a23c', text: '#ffffff' },
      danger: { bg: '#f56c6c', text: '#ffffff' },
      default: { bg: '#ffffff', text: '#606266' },
    }
    
    const colors = typeColors[props.type] || typeColors.default
    
    return () => h('button', {
      class: 'renderer-button',
      disabled: props.disabled,
      style: {
        backgroundColor: props.backgroundColor || colors.bg,
        color: props.textColor || colors.text,
        borderRadius: props.borderRadius ? `${props.borderRadius}px` : '4px',
        border: props.type === 'default' ? '1px solid #dcdfe6' : 'none',
        padding: '8px 16px',
        cursor: props.disabled ? 'not-allowed' : 'pointer',
        opacity: props.disabled ? 0.6 : 1,
        width: '100%',
        height: '100%',
        fontSize: '14px',
      },
    }, props.text || '按钮')
  },
}

/**
 * 图片渲染器
 */
const ImageRenderer = {
  name: 'ImageRenderer',
  props: ['src', 'alt', 'objectFit', 'borderRadius'],
  setup(props) {
    return () => h('img', {
      class: 'renderer-image',
      src: props.src || 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="100" height="100"%3E%3Crect fill="%23f0f0f0" width="100" height="100"/%3E%3Ctext x="50%25" y="50%25" dominant-baseline="middle" text-anchor="middle" fill="%23999"%3EImage%3C/text%3E%3C/svg%3E',
      alt: props.alt || '',
      style: {
        width: '100%',
        height: '100%',
        objectFit: props.objectFit || 'cover',
        borderRadius: props.borderRadius ? `${props.borderRadius}px` : undefined,
        display: 'block',
      },
    })
  },
}

/**
 * 输入框渲染器
 */
const InputRenderer = {
  name: 'InputRenderer',
  props: ['placeholder', 'disabled', 'type', 'maxLength'],
  setup(props) {
    return () => h('input', {
      class: 'renderer-input',
      type: props.type || 'text',
      placeholder: props.placeholder || '请输入',
      disabled: props.disabled,
      maxLength: props.maxLength,
      style: {
        width: '100%',
        height: '100%',
        padding: '8px 12px',
        border: '1px solid #dcdfe6',
        borderRadius: '4px',
        fontSize: '14px',
        boxSizing: 'border-box',
        outline: 'none',
      },
    })
  },
}

/**
 * 占位符渲染器（未知组件类型）
 */
const PlaceholderRenderer = {
  name: 'PlaceholderRenderer',
  setup(props, { attrs }) {
    return () => h('div', {
      class: 'renderer-placeholder',
      style: {
        width: '100%',
        height: '100%',
        backgroundColor: '#f5f5f5',
        border: '1px dashed #ccc',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: '#999',
        fontSize: '12px',
      },
    }, `未知组件: ${attrs.type || 'Unknown'}`)
  },
}

// Methods

/**
 * 处理点击事件
 * Requirements: 3.1 - 点击组件时选中
 */
function handleClick() {
  // 锁定的组件不能选中
  if (props.component.locked) {
    return
  }
  emit('select', props.component.id)
}

/**
 * 处理鼠标按下事件
 */
function handleMouseDown(event) {
  // 锁定的组件不能操作
  if (props.component.locked) {
    return
  }
  // 选中组件
  emit('select', props.component.id)
}
</script>

<style scoped>
.canvas-component {
  box-sizing: border-box;
  cursor: pointer;
  transition: box-shadow 0.2s ease;
}

.canvas-component:hover:not(.is-locked) {
  box-shadow: 0 0 0 1px rgba(64, 158, 255, 0.5);
}

.canvas-component.is-selected {
  box-shadow: 0 0 0 2px #409eff;
}

.canvas-component.is-locked {
  cursor: not-allowed;
}

.canvas-component.is-hidden {
  opacity: 0.3;
}

.component-content {
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.lock-indicator {
  position: absolute;
  top: 2px;
  right: 2px;
  background: rgba(0, 0, 0, 0.5);
  color: white;
  padding: 2px;
  border-radius: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 渲染器样式 */
.renderer-container {
  box-sizing: border-box;
}

.renderer-text {
  word-wrap: break-word;
}

.renderer-button {
  font-family: inherit;
}

.renderer-input:focus {
  border-color: #409eff;
}

.renderer-input:disabled {
  background-color: #f5f7fa;
  cursor: not-allowed;
}
</style>
