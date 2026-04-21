import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { i18n } from "@/i18n";
import PageIdentitySection from "./PageIdentitySection.vue";
import type { PageInspectorFormState } from "./page-inspector-types";

function createForm(): PageInspectorFormState {
  return {
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
    backgroundType: "color",
    backgroundValue: "#ffffff",
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
  };
}

describe("PageIdentitySection", () => {
  it("renders runtime title label and usage hint", () => {
    const wrapper = mount(PageIdentitySection, {
      props: {
        form: createForm(),
        isSystemPage: false,
        pageId: "page-home",
      },
      global: {
        plugins: [i18n],
        stubs: {
          "el-input": {
            props: ["modelValue", "disabled", "placeholder", "size"],
            template:
              "<input :value='modelValue' :disabled='disabled' :placeholder='placeholder' />",
          },
          "el-tooltip": {
            props: ["content"],
            template: "<span class='tooltip-stub'><slot />{{ content }}</span>",
          },
        },
      },
    });

    expect(wrapper.text()).toContain("运行标题");
    expect(wrapper.text()).toContain("主页面激活时可同步浏览器标题");
    expect(wrapper.find("input[placeholder='请输入运行标题']").exists()).toBe(true);
  });

  it("renders page id as readonly text without extra copy actions", () => {
    const wrapper = mount(PageIdentitySection, {
      props: {
        form: createForm(),
        isSystemPage: false,
        pageId: "page-home",
      },
      global: {
        plugins: [i18n],
        stubs: {
          "el-input": {
            props: ["modelValue", "disabled", "placeholder", "size"],
            template:
              "<input :value='modelValue' :disabled='disabled' :placeholder='placeholder' />",
          },
          "el-button": {
            template: "<button type='button'><slot /></button>",
          },
          "el-tooltip": {
            props: ["content"],
            template: "<span class='tooltip-stub'><slot />{{ content }}</span>",
          },
        },
      },
    });

    expect(wrapper.text()).toContain("页面 ID");
    expect(wrapper.text()).not.toContain("复制");
    expect(wrapper.text()).not.toContain("页面唯一标识");
    expect(wrapper.find("button").exists()).toBe(false);
    expect(wrapper.find("input[value='page-home']").exists()).toBe(true);
  });
});
