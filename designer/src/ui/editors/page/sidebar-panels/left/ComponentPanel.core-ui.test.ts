import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { i18n } from "@/i18n";
import { registerBuiltinComponents } from "@/editor-core/registry/builtin-manifests";
import { componentRegistry } from "@/editor-core/registry/component-registry";
import ComponentPanel from "./ComponentPanel.vue";

const globalStubs = {
  ElInput: {
    inheritAttrs: false,
    props: ["modelValue"],
    emits: ["update:modelValue"],
    template:
      '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
  ElCollapse: {
    template: "<div><slot /></div>",
  },
  ElCollapseItem: {
    template: '<section><header><slot name="title" /></header><slot /></section>',
  },
};

function mountPanel() {
  return mount(ComponentPanel, {
    global: {
      plugins: [i18n],
      stubs: globalStubs,
    },
  });
}

describe("ComponentPanel core UI components", () => {
  beforeEach(() => {
    i18n.global.locale.value = "zh";
    setActivePinia(createPinia());
    componentRegistry.clear();
    vi.spyOn(console, "warn").mockImplementation(() => undefined);
    registerBuiltinComponents();
  });

  afterEach(() => {
    i18n.global.locale.value = "zh";
    vi.restoreAllMocks();
    componentRegistry.clear();
  });

  it("展示第一批核心 UI 组件", () => {
    const wrapper = mountPanel();

    ["输入框", "数字输入", "选择器", "单选框", "多选框", "开关", "表格", "分页"].forEach(
      (name) => {
        expect(wrapper.text()).toContain(name);
      },
    );
  });

  it("核心 UI 组件支持按类型搜索", async () => {
    const wrapper = mountPanel();

    await wrapper.find("input").setValue("Pagination");

    expect(wrapper.text()).toContain("分页");
    expect(wrapper.text()).not.toContain("输入框");
  });
});
