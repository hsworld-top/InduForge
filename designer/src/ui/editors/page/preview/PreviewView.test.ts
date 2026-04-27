import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";
import PreviewView from "./PreviewView.vue";

const mocks = vi.hoisted(() => {
  const createRef = <T>(value: T) => {
    const ref = { value } as { value: T; __v_isRef: true };
    Object.defineProperty(ref, "__v_isRef", {
      value: true,
    });
    return ref;
  };

  const currentPage = createRef<{
    id?: string;
    name?: string;
    rootNodeId?: string;
    lifecycle?: Record<string, unknown>;
    config?: Record<string, unknown>;
  } | null>(null);
  const currentPageId = createRef("page-1");
  const doc = createRef<Record<string, unknown> | null>(null);
  const docVersion = createRef(1);
  const projectVariables = createRef<Record<string, unknown>>({});
  const globalScripts = createRef<Record<string, unknown>>({});
  const projectId = createRef("project-1");

  const editorStoreMock = {
    currentPage: currentPage.value,
    currentPageId: currentPageId.value,
    doc: doc.value,
    docVersion: docVersion.value,
    projectVariables: projectVariables.value,
    globalScripts: globalScripts.value,
    projectId: projectId.value,
    loadProject: vi.fn(),
  };

  const initPreviewRuntime = vi.fn(() => ({
    start: vi.fn().mockResolvedValue(undefined),
    stop: vi.fn().mockResolvedValue(undefined),
  }));
  const clearPreviewRuntime = vi.fn();
  const routerPush = vi.fn();

  return {
    currentPage,
    currentPageId,
    doc,
    docVersion,
    projectVariables,
    globalScripts,
    projectId,
    editorStoreMock,
    initPreviewRuntime,
    clearPreviewRuntime,
    routerPush,
  };
});

vi.mock("pinia", () => ({
  storeToRefs: () => ({
    currentPage: mocks.currentPage,
    currentPageId: mocks.currentPageId,
    doc: mocks.doc,
    docVersion: mocks.docVersion,
    projectVariables: mocks.projectVariables,
    globalScripts: mocks.globalScripts,
    projectId: mocks.projectId,
  }),
}));

vi.mock("@/stores/editor-store", () => ({
  useEditorStore: () => mocks.editorStoreMock,
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({
    meta: { project: { id: "project-1" } },
    query: { pid: "project-1", pageId: "page-1" },
  }),
  useRouter: () => ({
    push: mocks.routerPush,
  }),
}));

vi.mock("./previewRuntime", () => ({
  clearPreviewRuntime: mocks.clearPreviewRuntime,
  initPreviewRuntime: mocks.initPreviewRuntime,
}));

vi.mock("@/ui/editors/page/canvas/NodeRenderer.vue", () => ({
  default: {
    name: "NodeRenderer",
    props: ["nodeId", "isRoot", "readonly"],
    template: `<div class="node-renderer-stub" />`,
  },
}));

vi.mock("~icons/ep/arrow-left", () => ({
  default: {
    name: "IconEpArrowLeft",
    template: `<span class="icon-arrow-left-stub" />`,
  },
}));

vi.mock("~icons/ep/refresh", () => ({
  default: {
    name: "IconEpRefresh",
    template: `<span class="icon-refresh-stub" />`,
  },
}));

const globalStubs = {
  ElButton: {
    template: `<button><slot /></button>`,
  },
  ElDivider: {
    template: `<hr />`,
  },
  ElRadioGroup: {
    template: `<div><slot /></div>`,
  },
  ElRadioButton: {
    template: `<span><slot /></span>`,
  },
};

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
}

describe("PreviewView", () => {
  beforeEach(() => {
    mocks.currentPage.value = null;
    mocks.currentPageId.value = "page-1";
    mocks.doc.value = null;
    mocks.docVersion.value = 1;
    mocks.projectVariables.value = {};
    mocks.globalScripts.value = {};
    mocks.projectId.value = "project-1";
    mocks.editorStoreMock.currentPage = null;
    mocks.editorStoreMock.currentPageId = "page-1";
    mocks.editorStoreMock.doc = null;
    mocks.editorStoreMock.docVersion = 1;
    mocks.editorStoreMock.projectVariables = {};
    mocks.editorStoreMock.globalScripts = {};
    mocks.editorStoreMock.projectId = "project-1";
    mocks.editorStoreMock.loadProject.mockReset();
    mocks.initPreviewRuntime.mockClear();
    mocks.clearPreviewRuntime.mockClear();
    mocks.routerPush.mockReset();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("prefers grouped viewport and background config over legacy flat fields", async () => {
    mocks.currentPage.value = {
      id: "page-1",
      name: "首页",
      rootNodeId: "root-1",
      lifecycle: { onMounted: [] },
      config: {
        viewport: {
          width: 1440,
          height: 900,
          autoFit: false,
        },
        background: {
          kind: "image",
          value: "/assets/grouped-bg.png",
          size: "contain",
          position: "top center",
          repeat: "repeat-x",
        },
        width: 1024,
        height: 768,
        autoFit: true,
        backgroundColor: "#112233",
        backgroundImage: "/assets/legacy-bg.png",
        backgroundSize: "cover",
        backgroundRepeat: "repeat",
        backgroundPosition: "left top",
      },
    };
    mocks.editorStoreMock.currentPage = mocks.currentPage.value;

    const wrapper = mount(PreviewView, {
      global: {
        stubs: globalStubs,
      },
    });

    await flushPromises();

    const frame = wrapper.get(".preview-frame").element as HTMLElement;
    const canvas = wrapper.get(".preview-canvas").element as HTMLElement;

    expect(frame.style.width).toBe("1440px");
    expect(frame.style.height).toBe("900px");
    expect(canvas.style.width).toBe("1440px");
    expect(canvas.style.height).toBe("900px");
    expect(canvas.style.backgroundColor).toBe("rgb(17, 34, 51)");
    expect(canvas.style.backgroundImage).toContain("/assets/grouped-bg.png");
    expect(canvas.style.backgroundSize).toBe("contain");
    expect(canvas.style.backgroundRepeat).toBe("repeat-x");

    wrapper.unmount();
  });

  it("falls back to legacy flat viewport and background fields when grouped config is absent", async () => {
    mocks.currentPage.value = {
      id: "page-legacy",
      name: "旧页面",
      rootNodeId: "root-legacy",
      config: {
        width: 1200,
        height: 800,
        autoFit: false,
        backgroundColor: "#223344",
        backgroundImage: "/assets/legacy-only.png",
        backgroundSize: "cover",
        backgroundRepeat: "no-repeat",
        backgroundPosition: "center",
      },
    };
    mocks.editorStoreMock.currentPage = mocks.currentPage.value;

    const wrapper = mount(PreviewView, {
      global: {
        stubs: globalStubs,
      },
    });

    await flushPromises();

    const frame = wrapper.get(".preview-frame").element as HTMLElement;
    const canvas = wrapper.get(".preview-canvas").element as HTMLElement;

    expect(frame.style.width).toBe("1200px");
    expect(frame.style.height).toBe("800px");
    expect(canvas.style.width).toBe("1200px");
    expect(canvas.style.height).toBe("800px");
    expect(canvas.style.backgroundColor).toBe("rgb(34, 51, 68)");
    expect(canvas.style.backgroundImage).toContain("/assets/legacy-only.png");
    expect(canvas.style.backgroundSize).toBe("cover");
    expect(canvas.style.backgroundRepeat).toBe("no-repeat");

    wrapper.unmount();
  });

  it("shows preview adaptation summary and keeps aspect-ratio autofit visible in preset containers", async () => {
    mocks.currentPage.value = {
      id: "page-fit",
      name: "适配页面",
      rootNodeId: "root-fit",
      config: {
        viewport: {
          width: 1440,
          height: 900,
          autoFit: true,
          lockAspectRatio: true,
          minWidth: 640,
          minHeight: 400,
          overflowMode: "scroll",
        },
        background: {
          kind: "color",
          value: "#ffffff",
        },
      },
    };
    mocks.editorStoreMock.currentPage = mocks.currentPage.value;

    const wrapper = mount(PreviewView, {
      global: {
        stubs: globalStubs,
      },
    });

    await flushPromises();
    (wrapper.vm as any).viewKey = "phonePortrait";
    await wrapper.vm.$nextTick();

    const headerText = wrapper.get("header").text();
    const frame = wrapper.get(".preview-frame").element as HTMLElement;
    const canvas = wrapper.get(".preview-canvas").element as HTMLElement;

    expect(headerText).toContain("设计尺寸：1440 × 900");
    expect(headerText).toContain("容器：480 × 800");
    expect(headerText).toContain("适配：自适应-等比");
    expect(headerText).toContain("最小尺寸：640 × 400");
    expect(headerText).toContain("滚动：始终显示");
    expect(frame.style.width).toBe("480px");
    expect(frame.style.height).toBe("800px");
    expect(frame.style.overflow).toBe("scroll");
    expect(canvas.style.width).toBe("640px");
    expect(canvas.style.height).toBe("400px");

    wrapper.unmount();
  });

  it("uses page transition config as preview page transition", async () => {
    mocks.currentPage.value = {
      id: "page-transition",
      name: "动画页面",
      rootNodeId: "root-transition",
      config: {
        viewport: {
          width: 1366,
          height: 768,
          autoFit: false,
        },
        background: {
          kind: "color",
          value: "#ffffff",
        },
        transition: {
          type: "slide",
        },
      },
    };
    mocks.editorStoreMock.currentPage = mocks.currentPage.value;

    const wrapper = mount(PreviewView, {
      global: {
        stubs: globalStubs,
      },
    });

    await flushPromises();

    expect(wrapper.get(".preview-frame").attributes("data-page-transition")).toBe("slide");
    expect(wrapper.get(".preview-canvas").attributes("data-page-style-root")).toBe("page-1");

    wrapper.unmount();
  });
});
