<template>
  <button
    type="button"
    class="dc-status-indicator"
    :class="`dc-status-indicator--${resolvedState}`"
    :title="label"
    @click="$emit('click', resolvedState)"
  >
    <span class="dc-status-indicator__dot" aria-hidden="true"></span>
    <span class="dc-status-indicator__text">{{ label }}</span>
  </button>
</template>

<script setup lang="ts">
import { computed } from "vue";

type PreviewState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting";

const props = defineProps<{
  state?: PreviewState;
  status?: string;
}>();

defineEmits<{
  (event: "click", state: PreviewState): void;
}>();

const legacyStatusMap: Record<string, PreviewState> = {
  connected: "connected",
  disconnected: "disconnected",
  error: "disconnected",
  unknown: "disconnected",
};

const stateLabelMap: Record<PreviewState, string> = {
  disconnected: "预览已断开",
  connecting: "预览连接中",
  connected: "预览已连接",
  reconnecting: "预览重连中",
};

const resolvedState = computed<PreviewState>(() => {
  if (props.state) {
    return props.state;
  }

  return legacyStatusMap[props.status || ""] || "disconnected";
});

const label = computed(() => stateLabelMap[resolvedState.value]);
</script>

<style scoped>
.dc-status-indicator {
  min-height: 24px;
  max-width: 100%;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease;
}

.dc-status-indicator:hover {
  border-color: rgba(29, 78, 216, 0.22);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.dc-status-indicator:focus-visible {
  outline: none;
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}

.dc-status-indicator__dot {
  width: 7px;
  height: 7px;
  flex: 0 0 7px;
  border-radius: 999px;
  background: currentColor;
}

.dc-status-indicator__text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dc-status-indicator--connected {
  border-color: rgba(22, 163, 74, 0.2);
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.dc-status-indicator--connecting,
.dc-status-indicator--reconnecting {
  border-color: rgba(217, 119, 6, 0.22);
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.dc-status-indicator--disconnected {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
}
</style>
