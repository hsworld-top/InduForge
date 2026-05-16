import { ref, readonly } from "vue";

// 抽屉状态机，per-instance factory 函数（不是 store）
// 用法：const drawer = useDrawer<MyPayload>()

export function useDrawer<T = unknown>() {
  const visible = ref(false);
  // 固定（pin）后关闭按钮不会真正关闭，需要 unpin 后才能关
  const pinned = ref(false);
  const payload = ref<T | null>(null) as import("vue").Ref<T | null>;

  function open(data: T) {
    payload.value = data;
    visible.value = true;
  }

  function close() {
    if (pinned.value) return;
    visible.value = false;
    payload.value = null;
  }

  function forceClose() {
    pinned.value = false;
    visible.value = false;
    payload.value = null;
  }

  function pin() {
    pinned.value = true;
  }

  function unpin() {
    pinned.value = false;
  }

  return {
    visible: readonly(visible),
    pinned: readonly(pinned),
    payload: readonly(payload),
    open,
    close,
    forceClose,
    pin,
    unpin,
  };
}
