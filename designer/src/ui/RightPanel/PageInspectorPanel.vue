<template>
  <div class="page-inspector-panel">
    <!-- 页面标题 -->
    <div class="panel-title">
      <span>页面设置</span>
      <span class="page-name">{{ form.name || '未命名' }}</span>
    </div>

    <el-divider />

    <el-form label-width="72px" size="small">
      <!-- 基础信息 -->
      <div class="section-title">基础信息</div>

      <el-form-item label="页面名称">
        <el-input v-model="form.name" @blur="handleNameUpdate" />
      </el-form-item>

      <el-form-item label="路由路径">
        <el-input v-model="form.path" disabled />
      </el-form-item>

      <el-divider />

      <!-- 画布设置 -->
      <div class="section-title">画布设置</div>

      <el-form-item label="画布预设">
        <el-select
          v-model="form.presetKey"
          clearable
          placeholder="自定义尺寸"
          @change="handlePresetChange"
        >
          <el-option
            v-for="preset in VIEW_PRESETS"
            :key="preset.key"
            :label="`${preset.label} (${preset.width}×${preset.height})`"
            :value="preset.key"
          />
        </el-select>
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

      <el-form-item label="显示网格">
        <el-switch
          v-model="form.showGrid"
          @change="handleConfigUpdate"
        />
      </el-form-item>

      <el-form-item label="启用吸附">
        <el-switch
          v-model="form.enableSnap"
          @change="handleConfigUpdate"
        />
      </el-form-item>

      <el-divider />

      <!-- 背景设置 -->
      <div class="section-title">背景设置</div>

      <el-form-item label="背景类型">
        <el-select v-model="form.backgroundKind" @change="handleBackgroundUpdate">
          <el-option label="颜色" value="color" />
          <el-option label="图片" value="image" />
          <el-option label="渐变" value="gradient" />
        </el-select>
      </el-form-item>

      <el-form-item v-if="form.backgroundKind === 'color'" label="背景色">
        <el-color-picker
          v-model="form.backgroundValue"
          @change="handleBackgroundUpdate"
        />
      </el-form-item>

      <el-form-item v-else label="背景值">
        <el-input
          v-model="form.backgroundValue"
          @blur="handleBackgroundUpdate"
        />
      </el-form-item>

      <el-divider />

      <!-- 诊断信息 -->
      <div class="section-title">诊断信息</div>

      <el-form-item label="数据点">
        <div class="diagnostic-info">
          <el-tag type="success" size="small">
            正常 {{ diagnostic.active }}
          </el-tag>
          <el-tag v-if="diagnostic.invalid > 0" type="danger" size="small">
            失效 {{ diagnostic.invalid }}
          </el-tag>
          <el-tag type="info" size="small">
            共 {{ diagnostic.total }}
          </el-tag>
        </div>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup>
/**
 * 页面设置面板
 * 显示和编辑当前页面的配置信息
 */

import { computed, reactive, watch } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { ElMessage } from "element-plus";
import IconEpHomeFilled from "~icons/ep/home-filled";
import { VIEW_PRESETS } from "@/constants";

const editorStore = useEditorStore();
const { currentPage, currentPageId, pages, doc } = storeToRefs(editorStore);

const form = reactive({
  name: "",
  path: "",
  pageType: "business",
  presetKey: "",
  width: 1920,
  height: 1080,
  fitMode: "contain",
  showGrid: false,
  enableSnap: true,
  backgroundKind: "color",
  backgroundValue: "#ffffff",
});

// 诊断信息
const diagnostic = reactive({
  total: 0,
  active: 0,
  invalid: 0,
});

/**
 * 判断当前页面是否为首页
 */
const isHomePage = computed(() => {
  if (!currentPageId.value || !doc.value) return false;
  return doc.value.entry?.homePageId === currentPageId.value;
});

/**
 * 路由路径由名称生成
 * @param {string} name - 名称
 * @returns {string}
 */
const toRoutePath = (name) => {
  const normalized = name.trim().replace(/\s+/g, "-");
  const sanitized = normalized.replace(/[/?#\\]+/g, "-");
  return `/${sanitized || "page"}`;
};

/**
 * 根据尺寸匹配预设
 * @param {number} width - 宽度
 * @param {number} height - 高度
 * @returns {string}
 */
const resolvePresetKey = (width, height) => {
  const preset = VIEW_PRESETS.find(
    (item) => item.width === Number(width) && item.height === Number(height)
  );
  return preset?.key || "";
};

/**
 * 格式化路由路径展示
 * @param {string} path - 路由路径
 * @returns {string}
 */
const formatPathForDisplay = (path) => {
  if (!path) return "/";
  try {
    return decodeURIComponent(path);
  } catch (error) {
    return path;
  }
};

/**
 * 获取页面类型
 * @param {Object} page - 页面对象
 * @returns {'business' | 'login' | 'logout'}
 */
const getPageType = (page) => {
  if (!page || !doc.value) return "business";
  if (doc.value.entry?.loginPageId === page.id) return "login";
  if (doc.value.entry?.logoutPageId === page.id) return "logout";
  if (page.path === "/login") return "login";
  if (page.path === "/logout") return "logout";
  return "business";
};

/**
 * 获取页面固定路径
 * @param {Object} page - 页面
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
 * @param {Object | null} page - 页面
 */
const syncForm = (page) => {
  // 优先从 pages 列表获取页面信息（更准确）
  const pageFromList = currentPageId.value
    ? pages.value.find((p) => p.id === currentPageId.value)
    : null;

  if (!page && !pageFromList) {
    return;
  }

  // 使用 pages 列表的数据优先
  const nextName = pageFromList?.name || page?.name || "";
  form.name = nextName;

  // 路由路径优先从 pages 列表获取
  const path = pageFromList?.path || page?.path || resolvePath(page, nextName);
  form.path = formatPathForDisplay(path);

  form.pageType = getPageType(page || pageFromList);
  form.width = page?.config?.width ?? 1920;
  form.height = page?.config?.height ?? 1080;
  form.presetKey = resolvePresetKey(form.width, form.height);
  form.fitMode = page?.config?.fitMode || "contain";
  form.showGrid = page?.config?.showGrid ?? false;
  form.enableSnap = page?.config?.enableSnap ?? true;
  form.backgroundKind = page?.config?.background?.kind || "color";
  form.backgroundValue = page?.config?.background?.value || "#ffffff";

  // 更新诊断信息（模拟数据，实际应从 DataService 获取）
  updateDiagnostic();
};

/**
 * 更新诊断信息
 */
const updateDiagnostic = () => {
  // TODO: 从 DataService 获取实际数据点状态
  diagnostic.total = 0;
  diagnostic.active = 0;
  diagnostic.invalid = 0;
};

// 监听 currentPage 和 currentPageId 变化，同步表单
watch(
  [() => currentPage.value, () => currentPageId.value, () => pages.value],
  () => {
    syncForm(currentPage.value);
  },
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
    const path = currentPage.value.path || resolvePath(currentPage.value, name);
    form.path = formatPathForDisplay(path);
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
 * 处理页面类型变化
 * @param {string} type - 页面类型
 */
const handlePageTypeChange = (type) => {
  if (!currentPage.value || !doc.value) return;

  const pageId = currentPage.value.id;
  let path = form.path;

  if (type === "login") {
    path = "/login";
    editorStore.updateEntry({ loginPageId: pageId });
    void editorStore.persistEntry();
  } else if (type === "logout") {
    path = "/logout";
    editorStore.updateEntry({ logoutPageId: pageId });
    void editorStore.persistEntry();
  } else {
    // 业务页，清除特殊入口
    const entry = doc.value.entry || {};
    if (entry.loginPageId === pageId) {
      editorStore.updateEntry({ loginPageId: null });
      void editorStore.persistEntry();
    }
    if (entry.logoutPageId === pageId) {
      editorStore.updateEntry({ logoutPageId: null });
      void editorStore.persistEntry();
    }
    path = toRoutePath(form.name);
  }

  form.path = path;
  editorStore.updateCurrentPage({ path });
};

/**
 * 设为首页
 */
const handleSetAsHome = () => {
  if (!currentPageId.value) return;
  editorStore.updateEntry({ homePageId: currentPageId.value });
  void editorStore.persistEntry();
  ElMessage.success("已设为首页");
};

/**
 * 选择画布预设
 * @param {string} key - 预设键
 */
const handlePresetChange = (key) => {
  const preset = VIEW_PRESETS.find((item) => item.key === key);
  if (!preset) return;
  form.width = preset.width;
  form.height = preset.height;
  handleConfigUpdate();
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
    showGrid: form.showGrid,
    enableSnap: form.enableSnap,
  };
  editorStore.updateCurrentPage({ config: nextConfig });
  form.presetKey = resolvePresetKey(form.width, form.height);
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

<style scoped>
.page-inspector-panel {
  padding: 12px;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
}

.page-name {
  color: var(--el-color-primary);
  font-weight: 600;
}

.section-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin: 12px 0 8px;
}

.diagnostic-info {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
