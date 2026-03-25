import type { PendingComponentCall } from "./preview-runtime-helpers";
import type { PreviewComponentStubApi } from "./preview-runtime.types";
import { COMPONENT_SCRIPT_QUEUE_METHODS, queueComponentCall } from "./preview-runtime-helpers";

const COMPONENT_SCRIPT_VOID_METHODS = [
  "GetText",
  "GetSrc",
  "GetCommandItem",
  "GetMenuItem",
  "GetData",
  "GetRadioChecked",
  "GetRadioValue",
  "GetRadioLabel",
  "GetRadioEnable",
  "GetRadioVisible",
  "GetCheckState",
  "GetCheckEnable",
  "GetCheckVisible",
  "GetCheckedNodes",
  "GetCheckedKeys",
  "GetHalfCheckedNodes",
  "GetHalfCheckedKeys",
  "GetCurrentKey",
  "GetCurrentNode",
  "GetNode",
  "GetVisible",
  "GetActive",
  "GetUrl",
  "GetPage",
  "GetPageSize",
  "GetTotal",
  "GetActiveNames",
  "GetExpandedKeys",
  "GetSelection",
  "GetSelectionKeys",
  "GetPageData",
  "GetValue",
  "GetDate",
  "GetImage",
  "IsEmpty",
  "GetInputValue",
] as const;

const COMPONENT_SCRIPT_NULL_METHODS = [
  "elContainer",
  "elMain",
  "elLayout",
  "elLayoutRow",
  "elCol",
] as const;

/**
 * 构建未注册真实 ref 前的组件桩对象
 * @param {Map<string, PendingComponentCall[]>} pendingComponentCalls - 延迟调用队列
 * @param {string | null | undefined} pageId - 页面 ID
 * @param {string | null | undefined} name - 组件名
 * @returns {PreviewComponentStubApi}
 */
export function buildComponentStub(
  pendingComponentCalls: Map<string, PendingComponentCall[]>,
  pageId: string | null | undefined,
  name: string | null | undefined,
): PreviewComponentStubApi {
  const stub: Record<string, unknown> = {};

  Object.defineProperty(stub, "Name", {
    get() {
      return name || "";
    },
    enumerable: true,
    configurable: true,
  });
  Object.defineProperty(stub, "Comment", {
    get() {
      return "";
    },
    enumerable: true,
    configurable: true,
  });

  const location: Record<string, unknown> = {};
  Object.defineProperty(location, "X", {
    get() {
      return 0;
    },
    set(value: unknown) {
      const next = Number(value);
      if (!Number.isFinite(next)) return;
      queueComponentCall(pendingComponentCalls, pageId, name, "setStyle", [{ left: `${next}px` }]);
    },
    enumerable: true,
    configurable: true,
  });
  Object.defineProperty(location, "Y", {
    get() {
      return 0;
    },
    set(value: unknown) {
      const next = Number(value);
      if (!Number.isFinite(next)) return;
      queueComponentCall(pendingComponentCalls, pageId, name, "setStyle", [{ top: `${next}px` }]);
    },
    enumerable: true,
    configurable: true,
  });
  Object.defineProperty(stub, "Location", {
    get() {
      return location;
    },
    enumerable: true,
    configurable: true,
  });

  const size: Record<string, unknown> = {};
  Object.defineProperty(size, "Width", {
    get() {
      return 0;
    },
    set(value: unknown) {
      const next = Number(value);
      if (!Number.isFinite(next)) return;
      queueComponentCall(pendingComponentCalls, pageId, name, "setStyle", [{ width: `${next}px` }]);
    },
    enumerable: true,
    configurable: true,
  });
  Object.defineProperty(size, "Height", {
    get() {
      return 0;
    },
    set(value: unknown) {
      const next = Number(value);
      if (!Number.isFinite(next)) return;
      queueComponentCall(pendingComponentCalls, pageId, name, "setStyle", [
        { height: `${next}px` },
      ]);
    },
    enumerable: true,
    configurable: true,
  });
  Object.defineProperty(stub, "Size", {
    get() {
      return size;
    },
    enumerable: true,
    configurable: true,
  });

  Object.defineProperty(stub, "Visible", {
    get() {
      return true;
    },
    set(_value: unknown) {},
    enumerable: true,
    configurable: true,
  });
  Object.defineProperty(stub, "Enable", {
    get() {
      return true;
    },
    set(value: unknown) {
      queueComponentCall(pendingComponentCalls, pageId, name, "setProps", [{ disabled: !value }]);
    },
    enumerable: true,
    configurable: true,
  });
  Object.defineProperty(stub, "Caption", {
    get() {
      return "";
    },
    set(value: unknown) {
      queueComponentCall(pendingComponentCalls, pageId, name, "setProps", [
        { text: String(value ?? "") },
      ]);
    },
    enumerable: true,
    configurable: true,
  });
  Object.defineProperty(stub, "Image", {
    get() {
      return "";
    },
    set(value: unknown) {
      queueComponentCall(pendingComponentCalls, pageId, name, "setProps", [
        { src: String(value ?? "") },
      ]);
    },
    enumerable: true,
    configurable: true,
  });

  for (const method of COMPONENT_SCRIPT_QUEUE_METHODS) {
    stub[method] = (...args: unknown[]) =>
      queueComponentCall(pendingComponentCalls, pageId, name, method, args);
  }
  for (const method of COMPONENT_SCRIPT_VOID_METHODS) {
    stub[method] = () => undefined;
  }
  for (const method of COMPONENT_SCRIPT_NULL_METHODS) {
    stub[method] = () => null;
  }

  return stub as PreviewComponentStubApi;
}
