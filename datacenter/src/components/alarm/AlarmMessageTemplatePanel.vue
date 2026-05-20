<template>
  <section class="alarm-message-template-panel">
    <header>
      <strong>消息模板</strong>
      <span>messageTemplate</span>
    </header>

    <div class="alarm-message-template-panel__tokens">
      <button
        v-for="token in tokens"
        :key="token"
        type="button"
        @click="appendToken(token)"
      >
        {{ token }}
      </button>
    </div>

    <textarea
      :value="draft.messageTemplate"
      rows="5"
      spellcheck="false"
      @input="updateMessage(inputValue($event))"
    />
  </section>
</template>

<script setup lang="ts">
import type { AlarmRuleDraft } from "@/components/alarm/alarmRuleModel";

const props = defineProps<{
  draft: AlarmRuleDraft;
}>();

const emit = defineEmits<{
  update: [patch: Partial<AlarmRuleDraft>];
}>();

const tokens = ["{{targetPath}}", "{{ruleType}}", "{{severity}}", "{{value}}"];

const inputValue = (event: Event) => (event.target as HTMLTextAreaElement).value;

const updateMessage = (messageTemplate: string) => {
  emit("update", { messageTemplate, dirty: true });
};

const appendToken = (token: string) => {
  updateMessage(`${props.draft.messageTemplate}${token}`);
};
</script>

<style scoped>
.alarm-message-template-panel {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.alarm-message-template-panel header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.alarm-message-template-panel strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-message-template-panel header span {
  color: var(--dc-text-muted);
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

.alarm-message-template-panel__tokens {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.alarm-message-template-panel__tokens button {
  height: 28px;
  padding: 0 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
  cursor: pointer;
  font-family: var(--dc-font-mono);
  font-size: 12px;
  font-weight: 700;
}

.alarm-message-template-panel__tokens button:hover {
  border-color: rgba(37, 99, 235, 0.26);
  background: var(--dc-primary-soft);
}

.alarm-message-template-panel textarea {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text);
  font-family: var(--dc-font-mono);
  font-size: 12px;
  line-height: 1.5;
  outline: none;
  padding: 9px;
  resize: vertical;
}
</style>
