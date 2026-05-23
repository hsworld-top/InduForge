<template>
  <section class="mqtt-subscription-tree-branch">
    <button
      type="button"
      class="mqtt-subscription-tree-branch__group"
      @click="expanded = !expanded"
      @contextmenu.prevent.stop="$emit('groupContextmenu', $event, node)"
    >
      <IconTablerChevronRight
        class="mqtt-subscription-tree-branch__chevron"
        :class="{ 'is-open': expanded }"
      />
      <IconTablerFolderOpen v-if="expanded" class="mqtt-subscription-tree-branch__icon" />
      <IconTablerFolder v-else class="mqtt-subscription-tree-branch__icon" />
      <span class="mqtt-subscription-tree-branch__group-name">
        {{ node.name }}
      </span>
      <span class="mqtt-subscription-tree-branch__count">{{ totalCount }}</span>
    </button>

    <div v-if="expanded" class="mqtt-subscription-tree-branch__children">
      <button
        v-for="subscription in node.subscriptions"
        :key="String(subscription.id)"
        type="button"
        class="mqtt-subscription-tree-branch__subscription"
        :class="{ 'is-active': String(subscription.id) === selectedSubscriptionId }"
        @click="$emit('selectSubscription', subscription)"
        @dblclick="$emit('openTags', subscription)"
        @keydown.enter="$emit('openTags', subscription)"
        @contextmenu.prevent.stop="$emit('subscriptionContextmenu', $event, subscription)"
      >
        <IconTablerRss class="mqtt-subscription-tree-branch__subscription-icon" />
        <el-tooltip
          :content="subscription.name || subscription.topic || subscription.id"
          placement="top"
          :show-after="400"
        >
          <span class="mqtt-subscription-tree-branch__subscription-name">
            {{ subscription.name || subscription.topic }}
          </span>
        </el-tooltip>
      </button>

      <MqttSubscriptionTreeBranch
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selected-subscription-id="selectedSubscriptionId"
        @select-subscription="$emit('selectSubscription', $event)"
        @open-tags="$emit('openTags', $event)"
        @group-contextmenu="(mouseEvent, group) => $emit('groupContextmenu', mouseEvent, group)"
        @subscription-contextmenu="
          (mouseEvent, subscription) => $emit('subscriptionContextmenu', mouseEvent, subscription)
        "
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderOpen from '~icons/tabler/folder-open'
import IconTablerRss from '~icons/tabler/rss'
import type { MqttSubscription, MqttSubscriptionGroupNode } from './mqttSubscriptionTreeModel'

defineOptions({ name: 'MqttSubscriptionTreeBranch' })

const props = defineProps<{
  node: MqttSubscriptionGroupNode
  selectedSubscriptionId?: string | null
}>()

defineEmits<{
  (event: 'selectSubscription', subscription: MqttSubscription): void
  (event: 'openTags', subscription: MqttSubscription): void
  (event: 'subscriptionContextmenu', mouseEvent: MouseEvent, subscription: MqttSubscription): void
  (event: 'groupContextmenu', mouseEvent: MouseEvent, group: MqttSubscriptionGroupNode): void
}>()

const expanded = ref(true)

const countSubscriptions = (node: MqttSubscriptionGroupNode): number =>
  node.subscriptions.length +
  node.children.reduce((sum, child) => sum + countSubscriptions(child), 0)

const totalCount = computed(() => countSubscriptions(props.node))
</script>

<style scoped>
.mqtt-subscription-tree-branch {
  display: grid;
  gap: 2px;
}

.mqtt-subscription-tree-branch__group,
.mqtt-subscription-tree-branch__subscription {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  text-align: left;
}

.mqtt-subscription-tree-branch__group {
  min-height: 28px;
  display: grid;
  grid-template-columns: 16px 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 5px;
  padding: 0 6px;
  font-size: 13px;
  font-weight: 700;
}

.mqtt-subscription-tree-branch__group:hover,
.mqtt-subscription-tree-branch__subscription:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.mqtt-subscription-tree-branch__chevron {
  width: 14px;
  height: 14px;
  transform: rotate(0deg);
  transition: transform 0.16s ease;
}

.mqtt-subscription-tree-branch__chevron.is-open {
  transform: rotate(90deg);
}

.mqtt-subscription-tree-branch__icon,
.mqtt-subscription-tree-branch__subscription-icon {
  width: 16px;
  height: 16px;
  color: var(--dc-primary);
}

.mqtt-subscription-tree-branch__group-name,
.mqtt-subscription-tree-branch__subscription-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-subscription-tree-branch__count {
  min-width: 20px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  text-align: center;
}

.mqtt-subscription-tree-branch__children {
  display: grid;
  gap: 2px;
  margin-left: 10px;
  padding-left: 6px;
}

.mqtt-subscription-tree-branch__subscription {
  min-height: 30px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  padding: 3px 6px;
}

.mqtt-subscription-tree-branch__subscription.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.mqtt-subscription-tree-branch__subscription-name {
  display: block;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}
</style>
