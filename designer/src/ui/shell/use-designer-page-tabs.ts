import type { Ref } from "vue";
import type { DesignerPageTab, DesignerStorePageRow } from "./designer-view-types";
import { ElMessageBox } from "element-plus";
import { ref, watch } from "vue";
import { ElMessage } from "./el-message-compat";

export interface DesignerPageTabStateSnapshot {
  tabs?: DesignerPageTab[];
  activeId?: string;
}

/** useDesignerPageTabs 所需的最小 store 面 */
export interface DesignerPageTabsStore {
  pages: DesignerStorePageRow[];
  setCurrentPage: (id: string) => void | Promise<void>;
  saveCurrentPage: () => void | Promise<void>;
  saveCurrentPageDraft: () => void;
  setPageTabState: (tabs: DesignerPageTab[], activeId: string) => void;
}

/**
 * 设计器底部/顶部页面标签：打开、关闭、脏状态与 store 同步
 */
export function useDesignerPageTabs(options: {
  editorStore: DesignerPageTabsStore;
  pageTabState: Ref<DesignerPageTabStateSnapshot | undefined>;
  currentPageId: Ref<string>;
  isDirty: Ref<boolean>;
  leftActiveKey: Ref<string>;
}) {
  const { editorStore, pageTabState, currentPageId, isDirty, leftActiveKey } = options;

  const pageTabs = ref<DesignerPageTab[]>([]);
  const activePageTabId = ref("");

  {
    const initialTabState = pageTabState.value;
    if (initialTabState?.tabs?.length) {
      pageTabs.value = initialTabState.tabs.map((item: DesignerPageTab) => ({
        ...item,
      }));
      activePageTabId.value = initialTabState.activeId || "";
    }
  }

  function openPageTab(pageId: string) {
    const page = editorStore.pages.find((p: DesignerStorePageRow) => p.id === pageId);
    if (!page) return;

    const existingTab = pageTabs.value.find((t) => t.id === pageId);
    if (existingTab) {
      activePageTabId.value = pageId;
      return;
    }

    pageTabs.value.push({
      id: pageId,
      name: page.name || "未命名页面",
      isDirty: false,
    });

    activePageTabId.value = pageId;
  }

  async function handleClosePageTab(tabId: string) {
    const index = pageTabs.value.findIndex((t) => t.id === tabId);
    if (index === -1) return;

    const tab = pageTabs.value[index];
    if (!tab) return;

    if (tab.isDirty) {
      try {
        const action = await ElMessageBox.confirm(
          `页面 "${tab.name}" 有未保存的修改，是否保存后关闭？`,
          "关闭确认",
          {
            distinguishCancelAndClose: true,
            confirmButtonText: "保存并关闭",
            cancelButtonText: "不保存",
            type: "warning",
          },
        );

        if (action === "confirm") {
          if (currentPageId.value !== tabId) {
            editorStore.setCurrentPage(tabId);
            await new Promise((resolve) => setTimeout(resolve, 100));
          }
          await editorStore.saveCurrentPage();
          ElMessage.success("页面已保存");
        }
      } catch (action: unknown) {
        if (action === "close") {
          return;
        }
      }
    }

    pageTabs.value.splice(index, 1);

    if (activePageTabId.value === tabId) {
      if (pageTabs.value.length > 0) {
        const newActiveTab = pageTabs.value[Math.min(index, pageTabs.value.length - 1)];
        if (!newActiveTab) {
          activePageTabId.value = "";
          return;
        }
        activePageTabId.value = newActiveTab.id;
        editorStore.setCurrentPage(newActiveTab.id);
      } else {
        activePageTabId.value = "";
      }
    }
  }

  function updateTabDirtyState() {
    const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
    if (tab) {
      tab.isDirty = isDirty.value;
    }
  }

  function updateTabName() {
    const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
    if (tab) {
      const page = editorStore.pages.find(
        (p: DesignerStorePageRow) => p.id === currentPageId.value,
      );
      tab.name = page?.name || "未命名页面";
    }
  }

  function syncPageTabsWithPages() {
    const pageIdSet = new Set(editorStore.pages.map((page: DesignerStorePageRow) => page.id));
    const nextTabs = pageTabs.value.filter((tab) => pageIdSet.has(tab.id));
    if (nextTabs.length !== pageTabs.value.length) {
      pageTabs.value = nextTabs;
    }
    if (activePageTabId.value && !pageIdSet.has(activePageTabId.value)) {
      activePageTabId.value =
        currentPageId.value && pageIdSet.has(currentPageId.value)
          ? currentPageId.value
          : nextTabs[0]?.id || "";
    }
  }

  watch(isDirty, updateTabDirtyState);

  watch(
    [() => editorStore.pages, currentPageId],
    () => {
      syncPageTabsWithPages();
      updateTabName();
    },
    { deep: true },
  );

  watch(
    activePageTabId,
    async (newTabId, oldTabId) => {
      if (newTabId && newTabId !== oldTabId) {
        if (isDirty.value && currentPageId.value) {
          editorStore.saveCurrentPageDraft();
        }
        await editorStore.setCurrentPage(newTabId);
      }
    },
    { flush: "sync" },
  );

  watch(
    [pageTabs, activePageTabId],
    () => {
      editorStore.setPageTabState(pageTabs.value, activePageTabId.value);
    },
    { deep: true },
  );

  watch(
    currentPageId,
    (newPageId, oldPageId) => {
      if (newPageId && !pageTabs.value.some((t) => t.id === newPageId)) {
        const pageExists = editorStore.pages.some((p: DesignerStorePageRow) => p.id === newPageId);
        if (pageExists) {
          openPageTab(newPageId);
        }
      }
      if (newPageId && oldPageId && newPageId !== oldPageId) {
        leftActiveKey.value = "material";
      }
    },
    { immediate: true },
  );

  return {
    pageTabs,
    activePageTabId,
    openPageTab,
    handleClosePageTab,
  };
}
