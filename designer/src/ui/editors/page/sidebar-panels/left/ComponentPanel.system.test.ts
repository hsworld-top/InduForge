import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { i18n } from "@/i18n";
import { createDefaultProjectI18nSettings } from "@/editor-core/i18n/project-i18n";
import { registerBuiltinComponents } from "@/editor-core/registry/builtin-manifests";
import { componentRegistry } from "@/editor-core/registry/component-registry";
import { useEditorStore } from "@/stores/editor-store";
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

function mountPanel(enabled: boolean) {
  const store = useEditorStore();
  store.setProjectI18n({
    ...createDefaultProjectI18nSettings(),
    enabled,
  });
  return mount(ComponentPanel, {
    global: {
      plugins: [i18n],
      stubs: globalStubs,
    },
  });
}

describe("ComponentPanel system components", () => {
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

  it("国际化关闭时展示用户头像但不展示语言切换", () => {
    const wrapper = mountPanel(false);

    expect(wrapper.text()).toContain("系统组件");
    expect(wrapper.text()).toContain("用户头像");
    expect(wrapper.text()).not.toContain("语言切换");
  });

  it("国际化开启时展示用户头像和语言切换系统组件", () => {
    const wrapper = mountPanel(true);

    expect(wrapper.text()).toContain("系统组件");
    expect(wrapper.text()).toContain("用户头像");
    expect(wrapper.text()).toContain("语言切换");
  });

  it("系统组件支持按类型搜索", async () => {
    const wrapper = mountPanel(true);

    await wrapper.find("input").setValue("LanguageSwitcher");

    expect(wrapper.text()).toContain("系统组件");
    expect(wrapper.text()).toContain("语言切换");
  });

  it("系统组件支持按用户头像类型搜索", async () => {
    const wrapper = mountPanel(false);

    await wrapper.find("input").setValue("UserAvatarMenu");

    expect(wrapper.text()).toContain("系统组件");
    expect(wrapper.text()).toContain("用户头像");
  });
});
