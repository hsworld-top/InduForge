import { mount } from "@vue/test-utils";
import { reactive } from "vue";
import { describe, expect, it, vi } from "vitest";
import { i18n } from "@/i18n";
import PageVisualSection from "./PageVisualSection.vue";
import type { PageInspectorFormState } from "./page-inspector-types";

vi.mock("@/ui/shared/widgets/base/FriendlyColorPicker.vue", () => ({
  default: {
    props: ["modelValue"],
    template: "<div class='color-picker-stub'>{{ modelValue }}</div>",
  },
}));

function createForm(backgroundType: PageInspectorFormState["backgroundType"]): PageInspectorFormState {
  return reactive({
    name: "首页",
    title: "首页标题",
    description: "首页描述",
    role: "home",
    routeMode: "auto",
    routePath: "/",
    routeSlug: "",
    parentRoutePath: "/",
    viewportPreset: "pc",
    width: 1366,
    height: 768,
    autoFit: true,
    lockAspectRatio: false,
    minWidth: 0,
    minHeight: 0,
    overflowMode: "auto",
    backgroundType,
    backgroundValue: backgroundType === "color" ? "#ffffff" : "asset://background",
    backgroundSize: "cover",
    backgroundPosition: "center",
    backgroundRepeat: "no-repeat",
    transitionType: "none",
    openMode: "replace",
    popupWidth: 800,
    popupHeight: 600,
    popupCenter: true,
    popupMaskClosable: true,
    permissionSummary: "未配置",
    cacheMode: "default",
    preloadMode: "lazy",
  });
}

function mountSection(form: PageInspectorFormState) {
  return mount(PageVisualSection, {
    props: { form },
    global: {
      plugins: [i18n],
      stubs: {
        "el-select": {
          props: ["modelValue", "disabled", "size"],
          template: "<select :disabled='disabled'><slot /></select>",
        },
        "el-option": {
          template: "<option><slot /></option>",
        },
        "el-input": {
          props: ["modelValue", "disabled", "placeholder", "size"],
          template:
            "<input :value='modelValue' :disabled='disabled' :placeholder='placeholder' />",
        },
      },
    },
  });
}

describe("PageVisualSection", () => {
  it("hides image-only background controls for solid color background", () => {
    const wrapper = mountSection(createForm("color"));

    expect(wrapper.text()).toContain("背景类型");
    expect(wrapper.text()).toContain("背景值");
    expect(wrapper.text()).not.toContain("背景填充");
    expect(wrapper.text()).not.toContain("背景定位");
    expect(wrapper.text()).not.toContain("背景重复");
  });

  it("shows image-only background controls for image background", () => {
    const wrapper = mountSection(createForm("image"));

    expect(wrapper.text()).toContain("背景填充");
    expect(wrapper.text()).toContain("背景定位");
    expect(wrapper.text()).toContain("背景重复");
  });

  it("hides image-only background controls for gradient background", () => {
    const wrapper = mountSection(createForm("gradient"));

    expect(wrapper.text()).not.toContain("背景填充");
    expect(wrapper.text()).not.toContain("背景定位");
    expect(wrapper.text()).not.toContain("背景重复");
  });
});
