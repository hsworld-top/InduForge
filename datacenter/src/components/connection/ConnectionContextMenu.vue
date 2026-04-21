<template>
  <div
    v-show="visible"
    :style="{
      position: 'fixed',
      left: position.x + 'px',
      top: position.y + 'px',
      zIndex: 9999,
    }"
    class="bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 min-w-[150px]"
  >
    <div
      @click="handleOpen"
      class="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center"
    >
      <IconTablerPlugConnected class="mr-2 w-4 h-4" />
      {{ t("actions.openConnection") }}
    </div>
    <div
      @click="handleDisconnect"
      class="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center"
    >
      <IconTablerPlugConnectedX class="mr-2 w-4 h-4" />
      {{ t("actions.disconnectConnection") }}
    </div>
    <div class="border-t border-gray-200 dark:border-gray-700 my-1"></div>
    <div
      @click="handleViewDetails"
      class="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center"
    >
      <IconTablerEye class="mr-2 w-4 h-4" />
      {{ t("actions.viewDetails") }}
    </div>
    <div
      @click="handleEdit"
      class="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center"
    >
      <IconTablerEdit class="mr-2 w-4 h-4" />
      {{ t("actions.edit") }}
    </div>
    <div class="border-t border-gray-200 dark:border-gray-700 my-1"></div>
    <div
      @click="handleDelete"
      class="px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center"
    >
      <IconTablerTrash class="mr-2 w-4 h-4" />
      {{ t("actions.delete") }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { watch } from "vue";
import IconTablerPlugConnected from "~icons/tabler/plug-connected";
import IconTablerPlugConnectedX from "~icons/tabler/plug-connected-x";
import IconTablerEye from "~icons/tabler/eye";
import IconTablerEdit from "~icons/tabler/edit";
import IconTablerTrash from "~icons/tabler/trash";
import { t } from "@/i18n/runtime";

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  position: {
    type: Object,
    default: () => ({ x: 0, y: 0 }),
  },
  connection: {
    type: Object,
    default: null,
  },
});

const emit = defineEmits([
  "update:visible",
  "open",
  "disconnect",
  "view-details",
  "edit",
  "delete",
]);

// 点击菜单项后关闭菜单
const closeMenu = () => {
  emit("update:visible", false);
};

const handleOpen = () => {
  emit("open", props.connection);
  closeMenu();
};

const handleDisconnect = () => {
  emit("disconnect", props.connection);
  closeMenu();
};

const handleViewDetails = () => {
  emit("view-details", props.connection);
  closeMenu();
};

const handleEdit = () => {
  emit("edit", props.connection);
  closeMenu();
};

const handleDelete = () => {
  emit("delete", props.connection);
  closeMenu();
};

// 监听 visible 变化，添加/移除全局点击事件
watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      // 延迟添加事件监听，避免立即触发
      setTimeout(() => {
        document.addEventListener("click", handleClickOutside);
      }, 100);
    } else {
      document.removeEventListener("click", handleClickOutside);
    }
  },
);

const handleClickOutside = () => {
  closeMenu();
};
</script>
