<template>
  <el-button
    class="ui-button-component"
    :id="resolvedDomId || null"
    :type="mergedProps.type"
    :size="mergedProps.size"
    :disabled="mergedProps.disabled"
    :loading="mergedProps.loading"
    :plain="mergedProps.plain"
    :round="mergedProps.round"
    :circle="mergedProps.circle"
    :icon="resolvedIcon"
    v-bind="listeners"
    :style="mergedStyle"
    @click="handleEvent('click', $event)"
    @mousedown="handleEvent('mousedown', $event)"
    @mouseup="handleEvent('mouseup', $event)"
    @mouseenter="handleEvent('mouseenter', $event)"
    @mouseleave="handleEvent('mouseleave', $event)"
    @focus="handleEvent('focus', $event)"
    @blur="handleEvent('blur', $event)"
  >
    <template v-if="resolvedIcon" #icon>
      <component :is="resolvedIcon" />
    </template>
    <span>{{ mergedProps.text || '按钮' }}</span>
  </el-button>
</template>

<script setup>
import { computed, useAttrs, watch, onBeforeUnmount, ref } from 'vue';
import * as ElementPlusIconsVue from '@element-plus/icons-vue';

defineOptions({
  name: 'UIButtonComponent',
});

const emit = defineEmits(['click', 'mousedown', 'mouseup', 'mouseenter', 'mouseleave', 'focus', 'blur']);

const props = defineProps({
  text: { type: String, default: '按钮' },
  type: { type: String, default: 'primary' },
  size: { type: String, default: 'default' },
  disabled: { type: Boolean, default: false },
  loading: { type: Boolean, default: false },
  plain: { type: Boolean, default: false },
  round: { type: Boolean, default: false },
  circle: { type: Boolean, default: false },
  icon: { type: String, default: '' },
  domId: { type: String, default: '' },
  // 自定义样式配置（JSON 或 key:value 文本）
  styleConfig: { type: [String, Object], default: '' },
  // 详细配置（JSON/对象字面量），可覆盖按钮属性，支持 styleConfig
  advancedConfig: { type: [String, Object], default: '' },
  events: { type: Object, default: () => ({}) },
});

const attrs = useAttrs();
const listeners = computed(() => {
  const res = {};
  Object.keys(attrs).forEach((key) => {
    if (key.startsWith('on')) {
      res[key] = attrs[key];
    }
  });
  return res;
});

function parseStyleConfig(config) {
  if (!config) return { style: {}, domId: '', cssText: '' };

  if (typeof config === 'object') {
    const { domId = '', ...rest } = config;
    return { style: { ...rest }, domId, cssText: '' };
  }

  try {
    const parsed = JSON.parse(config);
    if (parsed && typeof parsed === 'object') {
      return parseStyleConfig(parsed);
    }
  } catch (err) {
    // ignore JSON parse errors
  }

  const text = String(config).trim();
  if (text.includes('{') && text.includes('}')) {
    const idMatch = text.match(/#([\w-]+)/);
    return { style: {}, domId: idMatch ? idMatch[1] : '', cssText: text };
  }

  const style = {};
  let domId = '';
  const lines = text
    .split('\n')
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith('//') && !l.startsWith('/*') && !l.startsWith('*'));

  lines.forEach((line) => {
    const [k, ...rest] = line.split(':');
    if (!k || rest.length === 0) return;
    const key = k.trim();
    const value = rest.join(':').replace(/;$/, '').trim();
    if (!key) return;
    if (key === 'domId') {
      domId = value;
      return;
    }
    style[key] = value;
  });

  return { style, domId, cssText: '' };
}

function parseAdvancedConfig(config) {
  if (!config) return {};
  if (typeof config === 'object') return { ...config };

  try {
    const parsed = JSON.parse(config);
    if (parsed && typeof parsed === 'object') return parsed;
  } catch (err) {
    // ignore JSON errors
  }

  try {
    // eslint-disable-next-line no-new-func
    const fn = new Function(`return (${config})`);
    const res = fn();
    return res && typeof res === 'object' ? res : {};
  } catch (err) {
    return {};
  }
}

const mergedProps = computed(() => {
  const adv = parseAdvancedConfig(props.advancedConfig);
  return {
    ...props,
    ...adv,
  };
});

const customStyle = computed(() => parseStyleConfig(mergedProps.value.styleConfig));

const resolvedIcon = computed(() => {
  if (!mergedProps.value.icon) return null;
  const match = Object.keys(ElementPlusIconsVue).find(
    (key) => key.toLowerCase() === mergedProps.value.icon.toLowerCase(),
  );
  return match ? ElementPlusIconsVue[match] : null;
});

const resolvedDomId = computed(() => mergedProps.value.domId || customStyle.value.domId || '');

const mergedStyle = computed(() => {
  const styleAttr = attrs.style || {};
  const { left, top, right, bottom, position, zIndex, ...rest } = styleAttr;
  return {
    width: '100%',
    height: '100%',
    ...rest, // 来自画布的样式（大小、颜色等）
    ...customStyle.value.style, // 来自“样式配置”文本
  };
});

const styleEl = ref(null);

const applyCssText = (cssText) => {
  if (typeof document === 'undefined') return;
  if (styleEl.value) {
    styleEl.value.remove();
    styleEl.value = null;
  }
  if (cssText) {
    const el = document.createElement('style');
    el.setAttribute('data-ui-button-style', 'true');
    el.textContent = cssText;
    document.head.appendChild(el);
    styleEl.value = el;
  }
};

watch(
  () => customStyle.value.cssText,
  (css) => applyCssText(css),
  { immediate: true },
);

onBeforeUnmount(() => {
  if (styleEl.value) {
    styleEl.value.remove();
    styleEl.value = null;
  }
});

const compiledHandlers = computed(() => {
  const map = {};
  const events = mergedProps.value.events || {};
  Object.entries(events).forEach(([name, code]) => {
    if (typeof code === 'function') {
      map[name] = code;
    } else if (typeof code === 'string' && code.trim()) {
      try {
        // eslint-disable-next-line no-new-func
        map[name] = new Function('event', 'emit', 'props', code);
      } catch (err) {
        console.error(`[Button] 无法编译事件 ${name}:`, err);
      }
    }
  });
  return map;
});

function handleEvent(name, evt) {
  const fn = compiledHandlers.value[name];
  if (fn) {
    try {
      fn(evt, emit, mergedProps.value);
    } catch (err) {
      console.error(`[Button] 事件 ${name} 执行失败:`, err);
    }
  }
  emit(name, evt);
}
</script>

<style scoped>
.ui-button-component {
  width: 100%;
  height: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 12px;
  box-sizing: border-box;
}
</style>
