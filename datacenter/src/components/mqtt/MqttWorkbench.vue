<template>
  <section class="mqtt-workbench">
    <aside class="mqtt-workbench__explorer">
      <div class="mqtt-workbench__source">
        <button
          type="button"
          class="mqtt-workbench__back"
          @click="$emit('back')"
        >
          <IconTablerArrowLeft />
          <span>返回接入源</span>
        </button>
        <span>MQTT Broker</span>
        <strong>{{ connection.name || "未命名 MQTT 接入源" }}</strong>
        <small>{{ endpointText }}</small>
      </div>

      <div class="mqtt-workbench__toolbar">
        <button type="button" title="订阅管理" @click="openSubscriptionList">
          <IconTablerListDetails />
        </button>
        <button type="button" title="刷新订阅" @click="loadSubscriptions">
          <IconTablerRefresh />
        </button>
        <button
          type="button"
          :title="connectionStarted ? '预览连接已启动' : '启动预览连接'"
          @click="ensureMqttPreview"
        >
          <IconTablerPlugConnected />
        </button>
      </div>

      <el-input
        v-model="filterText"
        class="mqtt-workbench__search"
        size="small"
        placeholder="筛选订阅或 Topic"
        clearable
      />

      <div class="mqtt-workbench__tree">
        <section class="mqtt-workbench__tree-section">
          <button
            type="button"
            class="mqtt-workbench__tree-head"
            @click="subscriptionsExpanded = !subscriptionsExpanded"
          >
            <IconTablerChevronDown v-if="subscriptionsExpanded" />
            <IconTablerChevronRight v-else />
            <span>订阅</span>
            <small>{{ filteredSubscriptions.length }}</small>
          </button>

          <div v-if="subscriptionsExpanded" class="mqtt-workbench__tree-body">
            <div v-if="loading" class="mqtt-workbench__loading">
              <IconTablerLoader2 />
              <span>加载订阅...</span>
            </div>
            <template v-else>
              <div
                v-for="subscription in filteredSubscriptions"
                :key="subscription.id"
                role="button"
                tabindex="0"
                class="mqtt-workbench__tree-item"
                :class="{
                  'is-active': selectedSubscription?.id === subscription.id,
                }"
                @click="selectedSubscription = subscription"
                @dblclick="openTagManager(subscription)"
                @keydown.enter="openTagManager(subscription)"
              >
                <IconTablerRss />
                <span>{{ subscription.name || subscription.topic }}</span>
                <small>QoS {{ subscription.qos ?? 0 }}</small>
                <button
                  type="button"
                  class="mqtt-workbench__inline-action"
                  title="查看消息"
                  @click.stop="openMessages(subscription)"
                >
                  <IconTablerMessages />
                </button>
                <button
                  type="button"
                  class="mqtt-workbench__inline-action"
                  title="变量管理"
                  @click.stop="openTagManager(subscription)"
                >
                  <IconTablerTags />
                </button>
              </div>
            </template>
            <div
              v-if="!loading && filteredSubscriptions.length === 0"
              class="mqtt-workbench__empty"
            >
              暂无订阅
            </div>
          </div>
        </section>
      </div>
    </aside>

    <main class="mqtt-workbench__main">
      <div class="mqtt-workbench__tabbar">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="mqtt-workbench__tab"
          :class="{ 'is-active': activeTabId === tab.id }"
          @click="activeTabId = tab.id"
        >
          <component :is="tab.icon" />
          <span>{{ tab.title }}</span>
          <IconTablerX
            v-if="tabs.length > 1"
            class="mqtt-workbench__tab-close"
            @click.stop="closeTab(tab.id)"
          />
        </button>
      </div>

      <div v-if="activeTab" class="mqtt-workbench__content">
        <MqttSubscriptionList
          v-if="activeTab.type === 'subscriptions'"
          ref="subscriptionListRef"
          :connection-id="connection.id"
          :project-id="projectIdText"
          @subscription-select="selectedSubscription = $event"
          @view-messages="openMessages"
          @manage-tags="openTagManager"
          @subscription-deleted="handleSubscriptionDeleted"
        />

        <MqttMessageViewer
          v-else-if="activeTab.type === 'messages'"
          :ref="(el) => setMessageViewerRef(activeTab.id, el)"
          :subscription="activeTab.subscription"
          :project-id="projectIdText"
          :connection-id="connection.id"
        />

        <div
          v-else-if="activeTab.type === 'tags'"
          class="mqtt-workbench__tag-panel"
        >
          <div class="mqtt-workbench__tag-list">
            <MqttTagList
              :project-id="projectIdText"
              :subscription-id="activeTab.subscription.id"
              :preview-session-id="previewSessionId"
            />
          </div>
          <div class="mqtt-workbench__tag-monitor">
            <MqttTagMonitor
              :project-id="projectIdText"
              :subscription-id="activeTab.subscription.id"
              :preview-session-id="previewSessionId"
            />
          </div>
        </div>
      </div>
    </main>

    <aside class="mqtt-workbench__inspector">
      <section>
        <div class="mqtt-workbench__panel-title">当前 Broker</div>
        <dl class="mqtt-workbench__facts">
          <div>
            <dt>地址</dt>
            <dd>{{ endpointText }}</dd>
          </div>
          <div>
            <dt>状态</dt>
            <dd>{{ connectionStarted ? "预览连接已启动" : "未启动" }}</dd>
          </div>
          <div>
            <dt>订阅</dt>
            <dd>{{ subscriptions.length }}</dd>
          </div>
        </dl>
      </section>

      <section>
        <div class="mqtt-workbench__panel-title">选中订阅</div>
        <dl class="mqtt-workbench__facts">
          <div>
            <dt>名称</dt>
            <dd>{{ selectedSubscription?.name || "-" }}</dd>
          </div>
          <div>
            <dt>Topic</dt>
            <dd>{{ selectedSubscription?.topic || "-" }}</dd>
          </div>
          <div>
            <dt>QoS</dt>
            <dd>{{ selectedSubscription?.qos ?? "-" }}</dd>
          </div>
        </dl>
      </section>

      <section>
        <div class="mqtt-workbench__panel-title">快捷动作</div>
        <button
          type="button"
          class="mqtt-workbench__quick-action"
          :disabled="!selectedSubscription"
          @click="selectedSubscription && openMessages(selectedSubscription)"
        >
          查看消息
        </button>
        <button
          type="button"
          class="mqtt-workbench__quick-action"
          :disabled="!selectedSubscription"
          @click="selectedSubscription && openTagManager(selectedSubscription)"
        >
          变量管理
        </button>
      </section>
    </aside>
  </section>
</template>

<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  shallowRef,
} from "vue";
import { ElMessage } from "element-plus";
import IconTablerArrowLeft from "~icons/tabler/arrow-left";
import IconTablerChevronDown from "~icons/tabler/chevron-down";
import IconTablerChevronRight from "~icons/tabler/chevron-right";
import IconTablerListDetails from "~icons/tabler/list-details";
import IconTablerLoader2 from "~icons/tabler/loader-2";
import IconTablerMessages from "~icons/tabler/messages";
import IconTablerPlugConnected from "~icons/tabler/plug-connected";
import IconTablerRefresh from "~icons/tabler/refresh";
import IconTablerRss from "~icons/tabler/rss";
import IconTablerTags from "~icons/tabler/tags";
import IconTablerX from "~icons/tabler/x";
import dataAPI from "@/api/data.api";
import { usePreviewSession } from "@/composables/usePreviewSession";
import { useMqttSocket } from "@/composables/useMqttSocket";
import { getApiErrorMessage } from "@/utils/request";
import MqttSubscriptionList from "./MqttSubscriptionList.vue";
import MqttMessageViewer from "./MqttMessageViewer.vue";
import MqttTagList from "./MqttTagList.vue";
import MqttTagMonitor from "./MqttTagMonitor.vue";

type MqttConnection = {
  id: string;
  name?: string;
  mqttConfig?: Record<string, any>;
  config?: Record<string, any>;
};

const props = defineProps<{
  projectId: string | number;
  connection: MqttConnection;
}>();

defineEmits<{
  (event: "back"): void;
}>();

const projectIdRef = computed(() => String(props.projectId || ""));
const projectIdText = computed(() => projectIdRef.value);
const config = computed(
  () => props.connection.mqttConfig || props.connection.config || {},
);
const endpointText = computed(() => {
  const host = config.value.brokerUrl || config.value.host || "未配置 Broker";
  const port = config.value.port ? `:${config.value.port}` : "";
  return `${host}${port}`;
});

const { sessionId: previewSessionId, ensureSession } = usePreviewSession(
  projectIdRef,
  { autoStart: false },
);
const { subscribeMessages } = useMqttSocket(projectIdRef, previewSessionId);

const loading = ref(false);
const subscriptions = ref<any[]>([]);
const selectedSubscription = ref<any>(null);
const subscriptionsExpanded = ref(true);
const filterText = ref("");
const connectionStarted = ref(false);
const tabs = ref<any[]>([]);
const activeTabId = ref("");
const subscriptionListRef = ref<any>(null);
const messageViewerRefs = shallowRef(new Map<string, any>());
const messageCleanups = new Map<string, () => void>();

const activeTab = computed(
  () => tabs.value.find((tab) => tab.id === activeTabId.value) || null,
);

const filteredSubscriptions = computed(() => {
  const keyword = filterText.value.trim().toLowerCase();
  if (!keyword) return subscriptions.value;
  return subscriptions.value.filter((subscription) => {
    const name = String(subscription.name || "").toLowerCase();
    const topic = String(subscription.topic || "").toLowerCase();
    return name.includes(keyword) || topic.includes(keyword);
  });
});

const loadSubscriptions = async () => {
  loading.value = true;
  try {
    const response = await dataAPI.getMqttSubscriptions(
      projectIdText.value,
      props.connection.id,
    );
    subscriptions.value = response.data || [];
    if (
      subscriptions.value.length > 0 &&
      !subscriptions.value.some(
        (subscription) => subscription.id === selectedSubscription.value?.id,
      )
    ) {
      selectedSubscription.value = subscriptions.value[0];
    }
    await subscriptionListRef.value?.loadSubscriptions?.();
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "加载 MQTT 订阅失败"));
  } finally {
    loading.value = false;
  }
};

const ensureMqttPreview = async () => {
  const session = await ensureSession();
  if (!session) {
    ElMessage.warning("预览会话不可用");
    return false;
  }
  if (connectionStarted.value) return true;

  try {
    await dataAPI.startMqttConnection(projectIdText.value, props.connection.id);
    connectionStarted.value = true;
    ElMessage.success("MQTT 预览连接已启动");
    return true;
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "启动 MQTT 预览连接失败"));
    return false;
  }
};

const addTab = (tab: any) => {
  const existing = tabs.value.find((item) => item.id === tab.id);
  if (existing) {
    activeTabId.value = existing.id;
    return existing;
  }
  tabs.value.push(tab);
  activeTabId.value = tab.id;
  return tab;
};

const openSubscriptionList = () => {
  addTab({
    id: `mqtt-subscriptions-${props.connection.id}`,
    type: "subscriptions",
    title: "订阅管理",
    icon: IconTablerListDetails,
  });
};

const openMessages = async (subscription: any) => {
  selectedSubscription.value = subscription;
  const ready = await ensureMqttPreview();
  if (!ready) return;

  const tab = addTab({
    id: `mqtt-messages-${subscription.id}`,
    type: "messages",
    title: `${subscription.name || subscription.topic} / 消息`,
    icon: IconTablerMessages,
    subscription,
  });

  if (!messageCleanups.has(tab.id)) {
    const cleanup = subscribeMessages(subscription.id, (data) => {
      const viewer = messageViewerRefs.value.get(tab.id);
      viewer?.addMessage?.(data.message);
      viewer?.setConnected?.(true);
    });
    messageCleanups.set(tab.id, cleanup);
  }

  await nextTick();
  messageViewerRefs.value.get(tab.id)?.setConnected?.(true);
};

const openTagManager = async (subscription: any) => {
  selectedSubscription.value = subscription;
  const ready = await ensureMqttPreview();
  if (!ready) return;

  addTab({
    id: `mqtt-tags-${subscription.id}`,
    type: "tags",
    title: `${subscription.name || subscription.topic} / 变量`,
    icon: IconTablerTags,
    subscription,
  });
};

const closeTab = (tabId: string) => {
  messageCleanups.get(tabId)?.();
  messageCleanups.delete(tabId);
  messageViewerRefs.value.delete(tabId);

  const index = tabs.value.findIndex((tab) => tab.id === tabId);
  if (index < 0) return;
  tabs.value.splice(index, 1);
  if (activeTabId.value === tabId) {
    activeTabId.value = tabs.value[index - 1]?.id || tabs.value[0]?.id || "";
  }
};

const setMessageViewerRef = (tabId: string, el: any) => {
  if (el) {
    messageViewerRefs.value.set(tabId, el);
    return;
  }
  messageViewerRefs.value.delete(tabId);
};

const handleSubscriptionDeleted = async (subscription: any) => {
  closeTab(`mqtt-messages-${subscription.id}`);
  closeTab(`mqtt-tags-${subscription.id}`);
  await loadSubscriptions();
};

onMounted(async () => {
  openSubscriptionList();
  await loadSubscriptions();
});

onBeforeUnmount(() => {
  Array.from(messageCleanups.values()).forEach((cleanup) => cleanup?.());
  messageCleanups.clear();
  messageViewerRefs.value.clear();
});
</script>

<style scoped>
.mqtt-workbench {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr) 260px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  overflow: hidden;
}

.mqtt-workbench__explorer,
.mqtt-workbench__inspector {
  min-height: 0;
  overflow: hidden;
  background: var(--dc-surface-muted);
}

.mqtt-workbench__explorer {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
}

.mqtt-workbench__inspector {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 14px;
  border-left: 1px solid var(--dc-border);
  overflow-y: auto;
}

.mqtt-workbench__source {
  padding: 14px;
  border-bottom: 1px solid var(--dc-border);
}

.mqtt-workbench__back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  padding: 6px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.mqtt-workbench__back svg {
  width: 14px;
  height: 14px;
}

.mqtt-workbench__back:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}

.mqtt-workbench__source span,
.mqtt-workbench__source small {
  display: block;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.mqtt-workbench__source strong {
  display: block;
  margin: 6px 0 4px;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-workbench__toolbar {
  display: flex;
  gap: 6px;
  padding: 10px 12px 6px;
}

.mqtt-workbench__toolbar button,
.mqtt-workbench__inline-action {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.mqtt-workbench__toolbar svg,
.mqtt-workbench__inline-action svg {
  width: 15px;
  height: 15px;
}

.mqtt-workbench__toolbar button:hover,
.mqtt-workbench__inline-action:hover {
  color: var(--dc-primary);
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
}

.mqtt-workbench__search {
  padding: 0 12px 10px;
}

.mqtt-workbench__tree {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding: 0 8px 12px;
}

.mqtt-workbench__tree-head,
.mqtt-workbench__tree-item,
.mqtt-workbench__quick-action {
  width: 100%;
  display: grid;
  align-items: center;
  border: 0;
  background: transparent;
  color: var(--dc-text-secondary);
  text-align: left;
}

.mqtt-workbench__tree-head {
  grid-template-columns: 18px minmax(0, 1fr) auto;
  min-height: 30px;
  padding: 5px 8px;
  font-size: 12px;
  font-weight: 700;
}

.mqtt-workbench__tree-head svg,
.mqtt-workbench__tree-item svg {
  width: 15px;
  height: 15px;
}

.mqtt-workbench__tree-head small,
.mqtt-workbench__tree-item small {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.mqtt-workbench__tree-body {
  padding-left: 8px;
}

.mqtt-workbench__tree-item {
  grid-template-columns: 20px minmax(0, 1fr) auto 30px 30px;
  gap: 6px;
  min-height: 32px;
  padding: 5px 4px 5px 8px;
  border-radius: var(--dc-radius-sm);
  font-size: 12px;
}

.mqtt-workbench__tree-item span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-workbench__tree-item:hover,
.mqtt-workbench__tree-item.is-active,
.mqtt-workbench__quick-action:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.mqtt-workbench__inline-action {
  opacity: 0;
}

.mqtt-workbench__tree-item:hover .mqtt-workbench__inline-action {
  opacity: 1;
}

.mqtt-workbench__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.mqtt-workbench__tabbar {
  display: flex;
  min-height: 38px;
  overflow-x: auto;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.mqtt-workbench__tab {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-width: 130px;
  max-width: 240px;
  padding: 0 10px;
  border: 0;
  border-right: 1px solid var(--dc-border);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.mqtt-workbench__tab.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font-weight: 700;
}

.mqtt-workbench__tab span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-workbench__tab svg {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
}

.mqtt-workbench__tab-close {
  margin-left: auto;
  color: var(--dc-text-muted);
}

.mqtt-workbench__content {
  min-height: 0;
  flex: 1;
  overflow: hidden;
}

.mqtt-workbench__tag-panel {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
}

.mqtt-workbench__tag-list {
  min-width: 0;
  min-height: 0;
  border-right: 1px solid var(--dc-border);
  overflow: hidden;
}

.mqtt-workbench__tag-monitor {
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.mqtt-workbench__panel-title {
  margin-bottom: 8px;
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
}

.mqtt-workbench__facts {
  margin: 0;
}

.mqtt-workbench__facts div {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--dc-border);
}

.mqtt-workbench__facts dt {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.mqtt-workbench__facts dd {
  min-width: 0;
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
  text-align: right;
  word-break: break-all;
}

.mqtt-workbench__quick-action {
  min-height: 32px;
  margin-bottom: 6px;
  padding: 8px;
  border-radius: var(--dc-radius-sm);
  font-size: 12px;
  font-weight: 700;
}

.mqtt-workbench__quick-action:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.mqtt-workbench__empty,
.mqtt-workbench__loading {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.mqtt-workbench__loading {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px;
}

.mqtt-workbench__loading svg {
  width: 14px;
  height: 14px;
  animation: mqtt-workbench-spin 0.9s linear infinite;
}

:deep(.mqtt-subscription-list),
:deep(.mqtt-message-viewer),
:deep(.mqtt-tag-list),
:deep(.mqtt-tag-monitor) {
  background: var(--dc-surface-raised);
}

:deep(.toolbar),
:deep(.mqtt-tag-list > div:first-child),
:deep(.mqtt-tag-monitor > div:first-child) {
  background: var(--dc-surface-subtle);
  border-color: var(--dc-border);
}

@keyframes mqtt-workbench-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1100px) {
  .mqtt-workbench {
    grid-template-columns: 240px minmax(0, 1fr);
  }

  .mqtt-workbench__inspector {
    display: none;
  }
}

@media (max-width: 760px) {
  .mqtt-workbench {
    grid-template-columns: 1fr;
  }

  .mqtt-workbench__explorer {
    max-height: 320px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }

  .mqtt-workbench__tag-panel {
    grid-template-columns: 1fr;
  }
}
</style>
