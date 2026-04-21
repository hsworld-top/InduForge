<template>
  <Teleport to="body">
    <div
      v-show="visible"
      :style="{
        position: 'fixed',
        left: position.x + 'px',
        top: position.y + 'px',
        zIndex: 9999,
      }"
      class="bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 min-w-[150px]"
      @click="handleClose"
    >
      <div
        @click="handleViewDetails"
        class="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center"
      >
        <IconTablerInfoCircle class="mr-2 w-4 h-4" />
        {{ t("actions.viewDetails") }}
      </div>
      <div
        @click="handleOpen"
        class="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center"
      >
        <IconTablerFileCode class="mr-2 w-4 h-4" />
        {{ t("actions.openQuery") }}
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
  </Teleport>
</template>

<script setup>
import { watch, onMounted, onBeforeUnmount } from "vue";
import IconTablerInfoCircle from "~icons/tabler/info-circle";
import IconTablerFileCode from "~icons/tabler/file-code";
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
  query: {
    type: Object,
    default: null,
  },
});

const emit = defineEmits(["update:visible", "view-details", "open", "delete"]);

const handleClose = () => {
  emit("update:visible", false);
};

const handleViewDetails = () => {
  emit("view-details", props.connection, props.query);
  handleClose();
};

const handleOpen = () => {
  emit("open", props.connection, props.query);
  handleClose();
};

const handleDelete = () => {
  emit("delete", props.connection, props.query);
  handleClose();
};

// 点击外部关闭菜单
const handleClickOutside = () => {
  if (props.visible) {
    handleClose();
  }
};

watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      setTimeout(() => {
        document.addEventListener("click", handleClickOutside);
      }, 0);
    } else {
      document.removeEventListener("click", handleClickOutside);
    }
  },
);

onMounted(() => {
  if (props.visible) {
    document.addEventListener("click", handleClickOutside);
  }
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleClickOutside);
});
</script>
