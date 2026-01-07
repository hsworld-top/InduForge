<template>
  <div class="flex flex-col gap-3">
    <el-form label-width="72px" size="small">
      <el-form-item label="页面名称">
        <el-input v-model="form.name" @blur="handleNameUpdate" />
      </el-form-item>
      <el-form-item label="路由路径">
        <el-input v-model="form.path" disabled />
      </el-form-item>
      <el-form-item label="宽度">
        <el-input-number
          v-model="form.width"
          :min="320"
          :max="99999"
          controls-position="right"
          @change="handleConfigUpdate"
        />
      </el-form-item>
      <el-form-item label="高度">
        <el-input-number
          v-model="form.height"
          :min="240"
          :max="99999"
          controls-position="right"
          @change="handleConfigUpdate"
        />
      </el-form-item>
      <el-form-item label="适配模式">
        <el-select v-model="form.fitMode" @change="handleConfigUpdate">
          <el-option label="contain" value="contain" />
          <el-option label="cover" value="cover" />
          <el-option label="fill" value="fill" />
          <el-option label="none" value="none" />
        </el-select>
      </el-form-item>
      <el-form-item label="背景类型">
        <el-select v-model="form.backgroundKind" @change="handleBackgroundUpdate">
          <el-option label="color" value="color" />
          <el-option label="image" value="image" />
          <el-option label="gradient" value="gradient" />
        </el-select>
      </el-form-item>
      <el-form-item label="背景值">
        <el-input
          v-model="form.backgroundValue"
          @blur="handleBackgroundUpdate"
        />
      </el-form-item>
      <el-form-item v-if="form.backgroundKind === 'color'" label="背景色">
        <el-color-picker
          v-model="form.backgroundValue"
          @change="handleBackgroundUpdate"
        />
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
import { reactive, watch } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { ElMessage } from "element-plus";

const editorStore = useEditorStore();
const { currentPage, pages, doc } = storeToRefs(editorStore);

const form = reactive({
  name: "",
  path: "",
  width: 1920,
  height: 1080,
  fitMode: "contain",
  backgroundKind: "color",
  backgroundValue: "#ffffff",
});

/**
 * 路由路径由名称生成
 * @param {string} name - 名称
 * @returns {string}
 */
const toRoutePath = (name) => {
  const normalized = name.trim().replace(/\s+/g, "-");
  return `/${encodeURIComponent(normalized)}`;
};

/**
 * 获取页面固定路径
 * @param {import('@/editor-core').PageNode} page - 页面
 * @param {string} name - 页面名称
 * @returns {string}
 */
const resolvePath = (page, name) => {
  if (!page) return "/";
  if (doc.value?.entry?.loginPageId === page.id) return "/login";
  if (page.path === "/login") return "/login";
  if (page.path === "/logout") return "/logout";
  if (page.name?.includes("登录")) return "/login";
  if (page.name?.includes("登出")) return "/logout";
  return toRoutePath(name);
};

/**
 * 校验名称唯一
 * @param {string} name - 名称
 * @param {string} excludeId - 排除的页面 ID
 * @returns {boolean}
 */
const isNameUnique = (name, excludeId) => {
  const lowerName = name.trim().toLowerCase();
  return !pages.value.some(
    (page) =>
      page.id !== excludeId &&
      (page.name || "").trim().toLowerCase() === lowerName
  );
};

/**
 * 同步页面配置到表单
 * @param {import('@/editor-core').PageNode | null} page - 页面
 */
const syncForm = (page) => {
  if (!page) return;
  const nextName = page.name || "";
  form.name = nextName;
  form.path = resolvePath(page, nextName);
  form.width = page.config?.width ?? 1920;
  form.height = page.config?.height ?? 1080;
  form.fitMode = page.config?.fitMode || "contain";
  form.backgroundKind = page.config?.background?.kind || "color";
  form.backgroundValue = page.config?.background?.value || "#ffffff";
};

watch(
  () => currentPage.value,
  (page) => syncForm(page),
  { immediate: true }
);

/**
 * 更新页面名称
 */
const handleNameUpdate = async () => {
  if (!currentPage.value) return;
  const name = form.name.trim();
  if (!name) {
    ElMessage.warning("名称不能为空");
    syncForm(currentPage.value);
    return;
  }
  if (!isNameUnique(name, currentPage.value.id)) {
    ElMessage.warning("页面名称已存在");
    syncForm(currentPage.value);
    return;
  }
  if (name === (currentPage.value.name || "")) {
    form.path = resolvePath(currentPage.value, name);
    return;
  }
  const path = resolvePath(currentPage.value, name);
  try {
    await editorStore.renamePage(currentPage.value.id, name, path);
    syncForm(currentPage.value);
  } catch (error) {
    ElMessage.error("更新页面名称失败");
    syncForm(currentPage.value);
  }
};

/**
 * 更新页面配置
 */
const handleConfigUpdate = () => {
  if (!currentPage.value) return;
  const nextConfig = {
    ...currentPage.value.config,
    width: Number(form.width) || currentPage.value.config?.width || 1920,
    height: Number(form.height) || currentPage.value.config?.height || 1080,
    fitMode: form.fitMode,
  };
  editorStore.updateCurrentPage({ config: nextConfig });
};

/**
 * 更新背景配置
 */
const handleBackgroundUpdate = () => {
  if (!currentPage.value) return;
  const nextConfig = {
    ...currentPage.value.config,
    background: {
      kind: form.backgroundKind,
      value: form.backgroundValue,
    },
  };
  editorStore.updateCurrentPage({ config: nextConfig });
};
</script>
