import { mount } from "@vue/test-utils";
import { defineComponent } from "vue";
import { describe, expect, it } from "vitest";
import PropEditor from "./PropEditor.vue";

const componentStub = (name: string, template: string) =>
  defineComponent({
    name,
    props: ["modelValue", "type"],
    template,
  });

describe("PropEditor type mapping", () => {
  const mountEditor = (prop: Record<string, unknown>, modelValue: unknown) =>
    mount(PropEditor, {
      props: {
        prop,
        modelValue: modelValue as string | number | boolean | Record<string, unknown> | unknown[],
      },
      global: {
        stubs: {
          MonacoEditor: true,
          FriendlyColorPicker: true,
          "el-switch": componentStub("ElSwitch", "<div class=\"el-switch-stub\" />"),
          "el-select": componentStub("ElSelect", "<div class=\"el-select-stub\"><slot /></div>"),
          "el-option": componentStub("ElOption", "<div class=\"el-option-stub\" />"),
          "el-input-number": componentStub("ElInputNumber", "<div class=\"el-input-number-stub\" />"),
          "el-input": componentStub("ElInput", "<textarea v-if=\"type === 'textarea'\" /><input v-else />"),
          "el-autocomplete": true,
        },
      },
    });

  it("renders boolean, enum, number and JSON props with matched controls", () => {
    expect(mountEditor({ type: "boolean" }, true).findComponent({ name: "ElSwitch" }).exists()).toBe(
      true,
    );

    expect(
      mountEditor(
        {
          type: "enum",
          options: [
            { label: "默认", value: "default" },
            { label: "小", value: "small" },
          ],
        },
        "default",
      )
        .findComponent({ name: "ElSelect" })
        .exists(),
    ).toBe(true);

    expect(mountEditor({ type: "number", min: 1 }, 1).findComponent({ name: "ElInputNumber" }).exists()).toBe(
      true,
    );

    const jsonEditor = mountEditor({ type: "array" }, [{ label: "选项一", value: "option1" }]);
    const input = jsonEditor.findComponent({ name: "ElInput" });
    expect(input.exists()).toBe(true);
    expect(input.props("type")).toBe("textarea");
  });
});
