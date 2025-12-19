<template>
  <div class="ui-element-plus-auto" :style="mergedStyle">
    <RenderRoot />
  </div>
</template>

<script setup>
import { computed, defineComponent, h, resolveDynamicComponent, useAttrs } from 'vue';
import { extractLayoutFreeStyle } from '../utils/styleHelpers.js';

defineOptions({
  name: 'UIElementPlusAuto',
});

const props = defineProps({
  componentName: { type: String, required: true },
  componentProps: { type: [String, Object], default: '' },
  slotText: { type: String, default: '' },
  advancedProps: { type: [String, Object], default: '' },
});

const attrs = useAttrs();

const listeners = computed(() => {
  const res = {};
  Object.entries(attrs).forEach(([key, value]) => {
    if (key.startsWith('on')) res[key] = value;
  });
  return res;
});

const mergedStyle = computed(() => extractLayoutFreeStyle(attrs));

const parseToObject = (val) => {
  if (!val) return {};
  if (typeof val === 'object') return { ...val };
  try {
    const parsed = JSON.parse(val);
    return parsed && typeof parsed === 'object' ? parsed : {};
  } catch (err) {
    return {};
  }
};

const baseProps = computed(() => parseToObject(props.componentProps));
const extraProps = computed(() => parseToObject(props.advancedProps));
const resolvedProps = computed(() => ({
  ...baseProps.value,
  ...extraProps.value,
}));

const hComp = (name, compProps = {}, children) => h(resolveDynamicComponent(name), compProps, children);

const slotBuilders = {
  ElRow: (text) => ({
    default: () => [
      hComp('ElCol', { span: 8 }, () => text || 'Col 1'),
      hComp('ElCol', { span: 8 }, () => text || 'Col 2'),
      hComp('ElCol', { span: 8 }, () => text || 'Col 3'),
    ],
  }),
  ElContainer: (text) => ({
    default: () => [
      hComp('ElHeader', { height: '36px' }, () => text || 'Header'),
      hComp('ElMain', {}, () => text || 'Main'),
    ],
  }),
  ElTabs: (text) => ({
    default: () => [
      hComp('ElTabPane', { label: 'Tab 1', name: '1' }, () => text || 'Tab 1 内容'),
      hComp('ElTabPane', { label: 'Tab 2', name: '2' }, () => text || 'Tab 2 内容'),
    ],
  }),
  ElSteps: (text) => ({
    default: () => [
      hComp('ElStep', { title: '步骤一', description: text || '描述 1' }),
      hComp('ElStep', { title: '步骤二', description: text || '描述 2' }),
      hComp('ElStep', { title: '步骤三', description: text || '描述 3' }),
    ],
  }),
  ElBreadcrumb: () => ({
    default: () => [
      hComp('ElBreadcrumbItem', {}, () => '首页'),
      hComp('ElBreadcrumbItem', {}, () => '列表'),
      hComp('ElBreadcrumbItem', {}, () => '详情'),
    ],
  }),
  ElDropdown: (text) => ({
    default: () => text || '下拉菜单',
    dropdown: () =>
      hComp('ElDropdownMenu', {}, () => [
        hComp('ElDropdownItem', { command: 'A' }, () => '选项 A'),
        hComp('ElDropdownItem', { command: 'B' }, () => '选项 B'),
        hComp('ElDropdownItem', { command: 'C', divided: true }, () => '选项 C'),
      ]),
  }),
  ElMenu: (text) => ({
    default: () => [
      hComp('ElMenuItem', { index: '1' }, () => text || '菜单一'),
      hComp('ElSubMenu', { index: '2' }, {
        title: () => '子菜单',
        default: () => [
          hComp('ElMenuItem', { index: '2-1' }, () => '选项 1'),
          hComp('ElMenuItem', { index: '2-2' }, () => '选项 2'),
        ],
      }),
      hComp('ElMenuItem', { index: '3' }, () => text || '菜单三'),
    ],
  }),
  ElCollapse: (text) => ({
    default: () => [
      hComp('ElCollapseItem', { name: '1', title: '面板 1' }, () => text || '内容 1'),
      hComp('ElCollapseItem', { name: '2', title: '面板 2' }, () => text || '内容 2'),
    ],
  }),
  ElCarousel: (text) => ({
    default: () => [
      hComp('ElCarouselItem', {}, () => text || '轮播 1'),
      hComp('ElCarouselItem', {}, () => text || '轮播 2'),
      hComp('ElCarouselItem', {}, () => text || '轮播 3'),
    ],
  }),
  ElTimeline: (text) => ({
    default: () => [
      hComp('ElTimelineItem', { timestamp: '2025-01-01', type: 'primary' }, () => text || '节点 1'),
      hComp('ElTimelineItem', { timestamp: '2025-01-02', type: 'success' }, () => text || '节点 2'),
      hComp('ElTimelineItem', { timestamp: '2025-01-03', type: 'warning' }, () => text || '节点 3'),
    ],
  }),
  ElDescriptions: (text) => ({
    default: () => [
      hComp('ElDescriptionsItem', { label: '用户名' }, () => text || 'admin'),
      hComp('ElDescriptionsItem', { label: '角色' }, () => 'Editor'),
      hComp('ElDescriptionsItem', { label: '邮箱' }, () => 'admin@example.com'),
    ],
  }),
  ElDialog: (text) => ({
    default: () => text || '对话框内容',
    footer: () =>
      h('div', { style: 'text-align: right;' }, [
        hComp('ElButton', { size: 'small' }, () => '取消'),
        hComp('ElButton', { type: 'primary', size: 'small' }, () => '确定'),
      ]),
  }),
  ElDrawer: (text) => ({
    default: () => text || '抽屉内容',
  }),
  ElPopover: (text) => ({
    default: () => text || '气泡卡片内容',
    reference: () => hComp('ElButton', { size: 'small' }, () => '参考元素'),
  }),
  ElTooltip: (text) => ({
    default: () => hComp('ElButton', { size: 'small' }, () => 'Hover 我'),
    content: () => text || '提示内容',
  }),
  ElPopconfirm: (text) => ({
    reference: () => hComp('ElButton', { size: 'small', type: 'danger' }, () => '删除'),
    default: () => text || '确认删除？',
  }),
  ElCard: (text) => ({
    header: () => '卡片标题',
    default: () => text || '卡片内容',
  }),
  ElRadioGroup: (text) => ({
    default: () => [
      hComp('ElRadio', { label: 'A' }, () => text || '选项 A'),
      hComp('ElRadio', { label: 'B' }, () => text || '选项 B'),
      hComp('ElRadio', { label: 'C' }, () => text || '选项 C'),
    ],
  }),
  ElCheckboxGroup: (text) => ({
    default: () => [
      hComp('ElCheckbox', { label: 'A' }, () => text || '选项 A'),
      hComp('ElCheckbox', { label: 'B' }, () => text || '选项 B'),
      hComp('ElCheckbox', { label: 'C' }, () => text || '选项 C'),
    ],
  }),
  ElSpace: (text) => ({
    default: () => [
      hComp('ElButton', { size: 'small', type: 'primary' }, () => text || '按钮 1'),
      hComp('ElButton', { size: 'small' }, () => text || '按钮 2'),
      hComp('ElButton', { size: 'small', type: 'success' }, () => text || '按钮 3'),
    ],
  }),
  ElAffix: (text) => ({
    default: () => hComp('ElButton', { type: 'primary', size: 'small' }, () => text || '固定按钮'),
  }),
};

const resolvedSlots = computed(() => {
  const builder = slotBuilders[props.componentName];
  if (builder) return builder(props.slotText);
  const text = props.slotText || props.componentName;
  return { default: () => text };
});

const renderNode = computed(() => {
  const comp = resolveDynamicComponent(props.componentName);
  if (!comp) {
    return h(
      'div',
      { class: 'ui-element-plus-auto__missing' },
      `未找到组件 ${props.componentName}`,
    );
  }
  return h(comp, { ...resolvedProps.value, ...listeners.value }, resolvedSlots.value);
});

const RenderRoot = defineComponent({
  name: 'ElementPlusAutoRender',
  setup: () => () => renderNode.value,
});
</script>

<style scoped>
.ui-element-plus-auto {
  width: 100%;
  height: 100%;
}

.ui-element-plus-auto__missing {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
  color: #909399;
  border: 1px dashed #dcdfe6;
  font-size: 12px;
  text-align: center;
  padding: 8px;
  box-sizing: border-box;
}
</style>
