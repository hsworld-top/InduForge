<script setup lang="ts">
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useEditorStore } from "@/stores/editor-store";

interface MenuItem {
  id: string;
  label: string;
  icon?: string;
  type: "builtin" | "script";
  builtinAction?: "profile" | "locale" | "theme" | "logout" | string;
  script?: string;
  danger?: boolean;
  hidden?: boolean;
}

interface RuntimeUserLike {
  id?: string;
  username?: string;
  displayName?: string;
  email?: string;
  avatar?: string;
  avatarUrl?: string;
  roles?: Array<{ name?: string; code?: string } | string>;
  status?: string;
}

const fallbackMenuItems: MenuItem[] = [
  { id: "profile", label: "个人资料", icon: "User", type: "script", script: "" },
  { id: "locale", label: "语言切换", icon: "Switch", type: "builtin", builtinAction: "locale" },
  { id: "theme", label: "主题切换", icon: "Moon", type: "builtin", builtinAction: "theme" },
  {
    id: "logout",
    label: "退出登录",
    icon: "SwitchButton",
    type: "builtin",
    builtinAction: "logout",
    danger: true,
  },
];

const props = defineProps<{
  resolvedProps?: Record<string, unknown>;
}>();

const editorStore = useEditorStore();
const {
  runtimeUsers,
  selectedPreviewRuntimeUserId,
  projectI18n,
  projectRuntimeLocale,
  projectRuntimeTheme,
  entryConfig,
  pages,
} = storeToRefs(editorStore);

const showName = computed(() => props.resolvedProps?.showName !== false);
const showSubtitle = computed(() => props.resolvedProps?.showSubtitle !== false);
const avatarSize = computed(() => {
  const value = Number(props.resolvedProps?.avatarSize ?? 32);
  if (!Number.isFinite(value)) return 32;
  return Math.min(56, Math.max(24, Math.round(value)));
});
const variant = computed(() => String(props.resolvedProps?.variant || "standard"));
const isCompact = computed(() => variant.value === "compact");
const enabledLocales = computed(() => projectI18n.value.locales.filter((item) => item.enabled));
const normalizedMenuItems = computed<MenuItem[]>(() => {
  const raw = props.resolvedProps?.menuItems;
  const source = Array.isArray(raw) && raw.length > 0 ? raw : fallbackMenuItems;
  const items: MenuItem[] = [];
  source.forEach((item, index) => {
    const normalized = normalizeMenuItem(item, index);
    if (normalized && normalized.hidden !== true) {
      items.push(normalized);
    }
  });
  return items;
});

const currentUser = computed<RuntimeUserLike>(() => {
  const selectedId = selectedPreviewRuntimeUserId.value;
  const users = runtimeUsers.value || [];
  const matched = selectedId ? users.find((user) => user.id === selectedId) : null;
  const active = matched || users.find((user) => user.status !== "disabled") || users[0];
  return (
    active || {
      id: "builtin-admin",
      username: "admin",
      displayName: "管理员",
      roles: [{ name: "管理员" }],
      status: "active",
    }
  );
});

const displayName = computed(() => {
  return currentUser.value.displayName || currentUser.value.username || "管理员";
});
const subtitle = computed(() => {
  const roles = currentUser.value.roles || [];
  const roleText = roles
    .map((role) => (typeof role === "string" ? role : role.name || role.code || ""))
    .filter(Boolean)
    .join("、");
  return roleText || currentUser.value.email || currentUser.value.username || "admin";
});
const avatarUrl = computed(() => currentUser.value.avatarUrl || currentUser.value.avatar || "");
const avatarText = computed(() => displayName.value.trim().slice(0, 1).toUpperCase() || "A");

function normalizeMenuItem(value: unknown, index: number): MenuItem | null {
  if (!value || typeof value !== "object") return null;
  const record = value as Record<string, unknown>;
  const id = String(record.id || record.builtinAction || `item-${index}`).trim();
  const label = String(record.label || id).trim();
  if (!id || !label) return null;
  const type = record.type === "builtin" ? "builtin" : "script";
  return {
    id,
    label,
    icon: String(record.icon || ""),
    type,
    builtinAction: String(record.builtinAction || id),
    script: String(record.script || ""),
    danger: Boolean(record.danger),
    hidden: Boolean(record.hidden),
  };
}

function handleCommand(command: string): void {
  const [kind, value] = command.split(":");
  if (kind === "locale" && value) {
    editorStore.setProjectRuntimeLocale(value);
    return;
  }
  if (kind === "theme" && value) {
    editorStore.setProjectRuntimeTheme(value);
    return;
  }
  if (kind === "logout") {
    handleLogout();
    return;
  }
  if (kind === "script" && value) {
    const item = normalizedMenuItems.value.find((menuItem) => menuItem.id === value);
    void runMenuScript(item);
  }
}

function handleLogout(): void {
  editorStore.setSelectedPreviewRuntimeUserId("");
  const logoutPageId = String((entryConfig.value as Record<string, unknown>)?.logoutPageId || "");
  const loginPageId = String((entryConfig.value as Record<string, unknown>)?.loginPageId || "");
  const targetPageId =
    findExistingPageId(logoutPageId) ||
    findPageIdByPath("/logout") ||
    findExistingPageId(loginPageId) ||
    findPageIdByPath("/login");
  if (targetPageId) {
    void editorStore.setCurrentPage(targetPageId);
  }
}

function findExistingPageId(pageId: string): string {
  if (!pageId) return "";
  return pages.value.some((page) => page.id === pageId) ? pageId : "";
}

function findPageIdByPath(path: string): string {
  return pages.value.find((page) => String((page as Record<string, unknown>).path || "") === path)
    ?.id || "";
}

function itemCommand(item: MenuItem): string {
  const action = item.builtinAction || item.id;
  if (item.type === "builtin" && action === "logout") return "logout";
  return `script:${item.id}`;
}

async function runMenuScript(item: MenuItem | undefined): Promise<void> {
  const code = String(item?.script || "").trim();
  if (!code) return;
  const currentPage = pages.value.find((page) => page.id === editorStore.currentPageId) || null;
  const context = {
    $event: { type: "user-avatar-menu", item, user: currentUser.value },
    $user: currentUser.value,
    $store: editorStore,
    pages: pages.value,
    currentPage,
    console,
  };
  try {
    // eslint-disable-next-line no-new-func
    const runner = new Function(
      ...Object.keys(context),
      `"use strict";\nreturn (async function() {\n${code}\n}).call(this);`,
    );
    await runner.call(null, ...Object.values(context));
  } catch (error) {
    console.error("[UserAvatarMenu] menu script error:", error);
  }
}
</script>

<template>
  <el-dropdown
    class="user-avatar-menu"
    trigger="click"
    popper-class="user-avatar-menu-popper"
    @command="handleCommand"
  >
    <button
      class="avatar-trigger"
      :class="{ 'is-compact': isCompact }"
      type="button"
      :style="{ minHeight: `${avatarSize + 8}px` }"
    >
      <el-avatar :size="avatarSize" :src="avatarUrl || undefined" class="avatar-trigger__avatar">
        {{ avatarText }}
      </el-avatar>
      <span v-if="!isCompact && (showName || showSubtitle)" class="avatar-trigger__text">
        <span v-if="showName" class="avatar-trigger__name">{{ displayName }}</span>
        <span v-if="showSubtitle" class="avatar-trigger__subtitle">{{ subtitle }}</span>
      </span>
      <span class="avatar-trigger__arrow" aria-hidden="true"></span>
    </button>
    <template #dropdown>
      <el-dropdown-menu class="avatar-dropdown">
        <template v-for="item in normalizedMenuItems" :key="item.id">
          <template
            v-if="item.type === 'builtin' && item.builtinAction === 'locale' && projectI18n.enabled"
          >
            <el-dropdown-item disabled class="avatar-dropdown__group">
              {{ item.label }}
            </el-dropdown-item>
            <el-dropdown-item
              v-for="localeItem in enabledLocales"
              :key="`${item.id}-${localeItem.code}`"
              :command="`locale:${localeItem.code}`"
              :class="{ 'is-selected': projectRuntimeLocale === localeItem.code }"
            >
              <span>{{ localeItem.name }}</span>
              <span class="avatar-dropdown__meta">{{ localeItem.code }}</span>
            </el-dropdown-item>
          </template>

          <template v-else-if="item.type === 'builtin' && item.builtinAction === 'theme'">
            <el-dropdown-item disabled class="avatar-dropdown__group">
              {{ item.label }}
            </el-dropdown-item>
            <el-dropdown-item
              command="theme:light"
              :class="{ 'is-selected': projectRuntimeTheme === 'light' }"
            >
              浅色主题
            </el-dropdown-item>
            <el-dropdown-item
              command="theme:dark"
              :class="{ 'is-selected': projectRuntimeTheme === 'dark' }"
            >
              深色主题
            </el-dropdown-item>
          </template>

          <el-dropdown-item
            v-else
            :command="itemCommand(item)"
            :class="{ 'is-danger': item.danger }"
          >
            {{ item.label }}
          </el-dropdown-item>
        </template>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<style scoped>
.user-avatar-menu {
  display: inline-flex;
  width: 100%;
}

.avatar-trigger {
  display: inline-flex;
  width: 100%;
  min-width: 44px;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--runtime-border-color, #d0d5dd);
  border-radius: 999px;
  background: var(--runtime-surface-color, #fff);
  color: var(--runtime-text-color, #1f2937);
  cursor: pointer;
  padding: 4px 10px 4px 4px;
  text-align: left;
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease,
    background-color 0.16s ease;
}

.avatar-trigger:hover {
  border-color: var(--runtime-primary-color, #1677ff);
  box-shadow: 0 4px 14px rgba(15, 23, 42, 0.12);
}

.avatar-trigger.is-compact {
  width: auto;
  padding-right: 6px;
}

.avatar-trigger__avatar {
  flex: 0 0 auto;
  background: var(--runtime-primary-color, #1677ff);
  color: #fff;
  font-weight: 700;
}

.avatar-trigger__text {
  display: inline-flex;
  min-width: 0;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 1px;
}

.avatar-trigger__name,
.avatar-trigger__subtitle {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.avatar-trigger__name {
  font-size: 13px;
  font-weight: 600;
  line-height: 1.2;
}

.avatar-trigger__subtitle {
  color: var(--runtime-text-muted-color, #667085);
  font-size: 11px;
  line-height: 1.2;
}

.avatar-trigger__arrow {
  width: 0;
  height: 0;
  flex: 0 0 auto;
  border-top: 5px solid var(--runtime-text-muted-color, #667085);
  border-right: 4px solid transparent;
  border-left: 4px solid transparent;
}

.avatar-dropdown__group {
  margin-top: 2px;
  color: var(--el-text-color-secondary);
  cursor: default;
  font-size: 12px;
  font-weight: 600;
}

.avatar-dropdown__meta {
  margin-left: 8px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.avatar-dropdown :deep(.is-selected) {
  color: var(--el-color-primary);
  font-weight: 600;
}

.avatar-dropdown :deep(.is-danger) {
  color: var(--el-color-danger);
}
</style>
