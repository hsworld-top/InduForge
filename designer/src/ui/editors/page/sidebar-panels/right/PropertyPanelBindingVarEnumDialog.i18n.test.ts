import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import { afterEach, describe, expect, it } from "vitest";
import { i18n } from "@/i18n";
import PropertyPanelBindingVarEnumDialog from "./PropertyPanelBindingVarEnumDialog.vue";

const sharedStubs = {
  "el-dialog": {
    props: ["title"],
    template: "<div><header>{{ title }}</header><slot /><slot name=\"footer\" /></div>",
  },
  "el-tabs": { template: "<div><slot /></div>" },
  "el-tab-pane": {
    props: ["label"],
    template: "<section>{{ label }}<slot /></section>",
  },
  "el-input": {
    props: ["placeholder"],
    template: "<div class=\"el-input-stub\" :data-placeholder=\"placeholder\"><slot /></div>",
  },
  "el-tree": { template: "<div><slot :data=\"{ label: 'group' }\" /></div>" },
  "el-table": { template: "<table><slot /></table>" },
  "el-table-column": {
    props: ["label"],
    template: "<th>{{ label }}</th>",
  },
  "el-button": { template: "<button><slot /></button>" },
  "el-icon": { template: "<span><slot /></span>" },
  IconEpFolder: true,
};

describe("PropertyPanel binding variable enum i18n", () => {
  afterEach(() => {
    i18n.global.locale.value = "zh";
  });

  it("会随 locale 切换更新绑定变量枚举文案", async () => {
    const wrapper = mount(PropertyPanelBindingVarEnumDialog, {
      props: {
        modelValue: true,
        bindingProjectGroupTree: [],
        bindingPageGroupTree: [],
        bindingProjectVariableRows: [],
        bindingPageVariableRows: [],
        filterBindingSidebarNode: () => true,
        bindingEnumProjectRowClass: () => "",
        bindingEnumPageRowClass: () => "",
        canConfirmInsert: true,
      },
      global: {
        plugins: [i18n],
        stubs: sharedStubs,
      },
    });

    expect(wrapper.text()).toContain("变量枚举");
    expect(wrapper.text()).toContain("工程变量");
    expect(wrapper.text()).toContain("页面变量");
    expect(wrapper.html()).toContain("搜索工程变量");

    i18n.global.locale.value = "en";
    await nextTick();

    expect(wrapper.text()).toContain("Variable Enum");
    expect(wrapper.text()).toContain("Project Variables");
    expect(wrapper.text()).toContain("Page Variables");
    expect(wrapper.html()).toContain("Search project variables");
  });
});
