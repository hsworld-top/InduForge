import type { Ref } from "vue";
import { onBeforeUnmount, onMounted, ref } from "vue";
import { Storage } from "@/utils/storage";

const SAVE_SETTINGS_STORAGE_KEY = "designer_save_settings";

export interface DesignerPageTabLike {
  id: string;
  isDirty: boolean;
}

export interface EditorStoreForAutoSave {
  saveCurrentPage: () => Promise<unknown>;
}

/**
 * 设计器自动保存：本地设置持久化 + 定时节流保存当前页
 */
export function useDesignerAutoSave(options: {
  editorStore: EditorStoreForAutoSave;
  pageTabs: Ref<DesignerPageTabLike[]>;
  currentPageId: Ref<string>;
  readonlyState: Ref<{ readonly?: boolean } | undefined>;
  isSaving: Ref<boolean>;
  isDirty: Ref<boolean>;
  onAutoSaveSuccess?: () => void;
}) {
  const {
    editorStore,
    pageTabs,
    currentPageId,
    readonlyState,
    isSaving,
    isDirty,
    onAutoSaveSuccess,
  } = options;

  const saveSettings = ref({
    autoSave: false,
    intervalMinutes: 5,
  });
  const autoSaveTimer = ref<ReturnType<typeof setInterval> | null>(null);
  const autoSaving = ref(false);

  async function handleAutoSave(): Promise<void> {
    if (!saveSettings.value.autoSave) return;
    if (readonlyState.value?.readonly) return;
    if (!currentPageId.value || autoSaving.value || isSaving.value) return;
    if (!isDirty.value) return;
    try {
      autoSaving.value = true;
      await editorStore.saveCurrentPage();
      const tab = pageTabs.value.find((t) => t.id === currentPageId.value);
      if (tab) {
        tab.isDirty = false;
      }
      onAutoSaveSuccess?.();
    } catch (error) {
      console.warn("自动保存失败:", error);
    } finally {
      autoSaving.value = false;
    }
  }

  function clearAutoSaveTimer(): void {
    if (autoSaveTimer.value) {
      clearInterval(autoSaveTimer.value);
      autoSaveTimer.value = null;
    }
  }

  function syncAutoSaveTimer(): void {
    clearAutoSaveTimer();
    if (!saveSettings.value.autoSave) return;
    const interval = Number(saveSettings.value.intervalMinutes) || 5;
    autoSaveTimer.value = setInterval(
      () => {
        void handleAutoSave();
      },
      interval * 60 * 1000,
    );
  }

  function handleSaveSettingsChange(settings: {
    autoSave?: boolean;
    intervalMinutes?: number;
  }): void {
    const nextSettings = {
      autoSave: Boolean(settings?.autoSave),
      intervalMinutes: Number(settings?.intervalMinutes) || 5,
    };
    saveSettings.value = nextSettings;
    Storage.set(SAVE_SETTINGS_STORAGE_KEY, nextSettings);
    syncAutoSaveTimer();
  }

  onMounted(() => {
    const cached = Storage.get(SAVE_SETTINGS_STORAGE_KEY, null) as {
      autoSave?: boolean;
      intervalMinutes?: number;
    } | null;
    if (cached && typeof cached === "object") {
      saveSettings.value = {
        autoSave: Boolean(cached.autoSave),
        intervalMinutes: Number(cached.intervalMinutes) || 5,
      };
    }
    syncAutoSaveTimer();
  });

  onBeforeUnmount(() => {
    clearAutoSaveTimer();
  });

  return {
    saveSettings,
    autoSaving,
    handleSaveSettingsChange,
  };
}
