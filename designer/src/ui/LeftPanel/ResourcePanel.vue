<template>
  <div class="resource-panel" @contextmenu.prevent>
    <div class="resource-search">
      <el-input
        v-model="folderSearch"
        size="small"
        placeholder="搜索分组"
        clearable
      />
    </div>

    <div class="resource-layout">
      <div class="resource-folders">
        <div class="pane-title">
          <span>资源分组</span>
          <el-button size="small" text @click="handleCreateFolder"
            >新建文件夹</el-button
          >
        </div>
        <el-scrollbar class="folder-scroll">
          <el-tree
            ref="folderTreeRef"
            class="folder-tree"
            node-key="id"
            :data="filteredFolderTree"
            :expand-on-click-node="false"
            highlight-current
            default-expand-all
            :filter-node-method="filterFolderNode"
            @node-contextmenu="handleFolderContextMenu"
            @node-click="handleFolderClick"
          >
            <template #default="{ data }">
              <div
                class="folder-node"
                :class="{ 'is-root': data.type === 'root' }"
                @dragover.prevent
                @drop.prevent="handleDropToFolder(data)"
              >
                <span class="folder-icon">📁</span>
                <span class="folder-label" :title="data.label">{{
                  data.label
                }}</span>
              </div>
            </template>
          </el-tree>
        </el-scrollbar>
      </div>

      <div class="resource-assets">
        <div class="asset-header">
          <div class="asset-title">
            <span>{{ selectedFolderLabel }}</span>
          </div>
          <div class="asset-body">
            <el-input
              v-model="assetSearch"
              size="small"
              placeholder="搜索资源"
              clearable
              class="asset-search"
            />
            <span class="asset-count">{{ filteredAssets.length }} 项</span>
          </div>
        </div>

        <div
          class="asset-dropzone"
          @click="triggerFileSelect"
          @dragover.prevent
          @drop.prevent="handleUploadDrop"
        >
          拖拽文件到此区域上传
        </div>

        <el-scrollbar class="asset-scroll">
          <div v-if="viewMode === 'grid'" class="asset-grid">
            <div
              v-for="asset in filteredAssets"
              :key="asset.id"
              class="asset-card"
              draggable="true"
              @dragstart="handleAssetDragStart(asset, $event)"
              @dblclick="openPreview(asset)"
              @contextmenu.prevent.stop="openAssetContextMenu($event, asset)"
            >
              <div class="asset-thumb">
                <img
                  v-if="isImageAsset(asset)"
                  :src="asset.thumbnailUrl || asset.url"
                  :alt="asset.displayName"
                />
                <div v-else class="asset-thumb-placeholder">
                  {{ asset.ext || asset.type || "file" }}
                </div>
              </div>
              <div class="asset-name" :title="asset.displayName">
                {{ asset.displayName }}
              </div>
              <div class="asset-meta">
                <span>{{ asset.ext || asset.type || "-" }}</span>
                <span>{{ formatSize(asset.size) }}</span>
              </div>
            </div>
          </div>

          <div v-else class="asset-table">
            <div class="asset-table-header">
              <span class="col-name">名称</span>
              <span class="col-type">格式</span>
              <span class="col-size">大小</span>
            </div>
            <div
              v-for="asset in filteredAssets"
              :key="asset.id"
              class="asset-table-row"
              draggable="true"
              @dragstart="handleAssetDragStart(asset, $event)"
              @dblclick="openPreview(asset)"
              @contextmenu.prevent.stop="openAssetContextMenu($event, asset)"
            >
              <span class="col-name" :title="asset.displayName">{{
                asset.displayName
              }}</span>
              <span class="col-type">{{ asset.ext || asset.type || "-" }}</span>
              <span class="col-size">{{ formatSize(asset.size) }}</span>
            </div>
          </div>
        </el-scrollbar>

        <div class="view-toggle">
          <el-button
            circle
            size="small"
            :type="viewMode === 'grid' ? 'primary' : 'default'"
            @click="viewMode = 'grid'"
          >
            <el-icon><IconEpGrid /></el-icon>
          </el-button>
          <el-button
            circle
            size="small"
            :type="viewMode === 'list' ? 'primary' : 'default'"
            @click="viewMode = 'list'"
          >
            <el-icon><IconEpList /></el-icon>
          </el-button>
        </div>
      </div>
    </div>

    <input
      ref="fileInputRef"
      type="file"
      multiple
      class="hidden-input"
      @change="handleFileInputChange"
    />

    <div
      v-if="contextMenu.visible"
      class="context-menu"
      :style="contextMenuStyle"
    >
      <template v-if="contextMenu.type === 'asset'">
        <div class="context-item" @click="handlePreviewContext">预览</div>
        <div class="context-item" @click="handleDetailContext">查看详情</div>
        <div class="context-item" @click="handleRenameAsset">重命名</div>
        <div class="context-item" @click="handleMoveAsset">移动</div>
        <div class="context-item" @click="handleCopyAsset">复制</div>
        <div class="context-item" @click="handleCutAsset">剪切</div>
        <div
          class="context-item"
          @click="handlePasteAsset"
          :class="{ disabled: !clipboardAsset }"
        >
          粘贴
        </div>
        <div class="context-item danger" @click="handleDeleteAsset">删除</div>
      </template>
      <template v-else>
        <div class="context-item" @click="handleCreateFolder">新建文件夹</div>
        <div class="context-item" @click="handleRenameFolder">重命名</div>
        <div
          class="context-item"
          @click="handlePasteAsset"
          :class="{ disabled: !clipboardAsset }"
        >
          粘贴
        </div>
        <div class="context-item danger" @click="handleDeleteFolder">删除</div>
      </template>
    </div>

    <el-dialog
      v-model="previewVisible"
      title="资源预览"
      width="70vw"
      top="6vh"
      :close-on-click-modal="false"
      append-to-body
    >
      <div class="preview-body">
        <template v-if="previewAsset">
          <img
            v-if="isImageAsset(previewAsset)"
            :src="previewAsset.url"
            :alt="previewAsset.displayName"
          />
          <iframe
            v-else-if="isPdfAsset(previewAsset)"
            class="preview-pdf"
            :src="previewAsset.url"
          ></iframe>
          <div v-else class="preview-file">
            <div class="preview-file-name">{{ previewAsset.displayName }}</div>
            <div class="preview-file-meta">
              {{ previewAsset.ext || "-" }} |
              {{ formatSize(previewAsset.size) }}
            </div>
          </div>
        </template>
      </div>
    </el-dialog>

    <el-dialog
      v-model="detailVisible"
      title="资源详情"
      width="520px"
      top="30vh"
      append-to-body
    >
      <div v-if="detailAsset" class="detail-body">
        <div class="detail-row">
          <span>名称</span><span>{{ detailAsset.displayName }}</span>
        </div>
        <div class="detail-row">
          <span>格式</span><span>{{ detailAsset.ext || "-" }}</span>
        </div>
        <div class="detail-row">
          <span>大小</span><span>{{ formatSize(detailAsset.size) }}</span>
        </div>
        <div class="detail-row">
          <span>类型</span><span>{{ detailAsset.type || "-" }}</span>
        </div>
        <div class="detail-row">
          <span>链接</span>
          <span
            class="detail-link"
            :title="detailAsset.url"
            @click="handleCopyUrl(detailAsset.url)"
          >
            {{ detailAsset.url || "-" }}
          </span>
        </div>
      </div>
    </el-dialog>

    <el-dialog
      v-model="moveDialogVisible"
      title="移动到"
      width="420px"
      append-to-body
    >
      <el-tree
        ref="moveTreeRef"
        class="folder-tree"
        node-key="id"
        :data="folderTree"
        :expand-on-click-node="false"
        highlight-current
        default-expand-all
        @node-click="handleMoveFolderSelect"
      >
        <template #default="{ data }">
          <div class="folder-node">
            <span class="folder-icon">📁</span>
            <span class="folder-label" :title="data.label">{{
              data.label
            }}</span>
          </div>
        </template>
      </el-tree>
      <template #footer>
        <el-button @click="moveDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmMove">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import {
  ref,
  computed,
  onMounted,
  onBeforeUnmount,
  watch,
  nextTick,
} from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import IconEpGrid from "~icons/ep/grid";
import IconEpList from "~icons/ep/list";
import { useEditorStore } from "@/stores/editor-store";
import assetApi from "@/services/assetApi";

const editorStore = useEditorStore();
const projectId = computed(
  () => editorStore.projectId || editorStore.project?.id || "",
);

const folderSearch = ref("");
const assetSearch = ref("");
const folderTreeRef = ref(null);
const moveTreeRef = ref(null);
const fileInputRef = ref(null);

const folders = ref([]);
const assets = ref([]);
const selectedFolderId = ref("root");
const selectedFolderLabel = computed(() => {
  if (selectedFolderId.value === "root") return "全部资源";
  const found = folders.value.find(
    (item) => item.id === selectedFolderId.value,
  );
  return found?.name || "全部资源";
});

const viewMode = ref("grid");

const previewVisible = ref(false);
const previewAsset = ref(null);
const detailVisible = ref(false);
const detailAsset = ref(null);

const moveDialogVisible = ref(false);
const moveAssetTarget = ref(null);
const moveTargetFolderId = ref(null);

const contextMenu = ref({
  visible: false,
  x: 0,
  y: 0,
  type: "asset",
  asset: null,
  folder: null,
});

const clipboardAsset = ref(null);
const clipboardMode = ref(null);

const contextMenuStyle = computed(() => ({
  left: `${contextMenu.value.x}px`,
  top: `${contextMenu.value.y}px`,
}));

const adjustContextMenuPosition = async () => {
  await nextTick();
  const menuEl = contextMenuRef.value;
  if (!menuEl) return;
  const rect = menuEl.getBoundingClientRect();
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;
  let nextX = contextMenu.value.x;
  let nextY = contextMenu.value.y;
  const padding = 8;
  if (rect.right > viewportWidth - padding) {
    nextX = Math.max(padding, viewportWidth - rect.width - padding);
  }
  if (rect.bottom > viewportHeight - padding) {
    nextY = Math.max(padding, viewportHeight - rect.height - padding);
  }
  if (nextX !== contextMenu.value.x || nextY !== contextMenu.value.y) {
    contextMenu.value.x = nextX;
    contextMenu.value.y = nextY;
  }
};

const handleCopyUrl = async (value) => {
  if (!value) return;
  try {
    await navigator.clipboard.writeText(value);
    ElMessage.success("链接已复制");
  } catch (error) {
    ElMessage.error("复制失败");
  }
};
const unwrapApiData = (response) =>
  response?.data?.data ?? response?.data ?? response;

const decodeAssetName = (value) => {
  if (!value) return "";
  try {
    return decodeURIComponent(value);
  } catch (error) {
    try {
      return decodeURIComponent(escape(value));
    } catch (err) {
      return value;
    }
  }
};

const getAssetExt = (asset) => {
  const name = asset?.displayName || asset?.name || asset?.originalName || "";
  const index = name.lastIndexOf(".");
  if (index > -1 && index < name.length - 1) {
    return name.slice(index + 1).toLowerCase();
  }
  const mime = String(asset?.mimeType || "").toLowerCase();
  if (mime.includes("/")) {
    return mime.split("/").pop();
  }
  return asset?.type || "";
};

const formatSize = (size) => {
  if (size === null || size === undefined || size === "") return "-";
  const value = Number(size);
  if (Number.isNaN(value)) return "-";
  if (value === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let idx = 0;
  let num = value;
  while (num >= 1024 && idx < units.length - 1) {
    num /= 1024;
    idx += 1;
  }
  return `${num.toFixed(num >= 10 ? 0 : 1)} ${units[idx]}`;
};

const isImageAsset = (asset) => {
  const type = asset?.type || "";
  const ext = getAssetExt(asset);
  if (type === "image" || type === "svg") return true;
  return ["png", "jpg", "jpeg", "gif", "webp", "svg"].includes(ext);
};

const isPdfAsset = (asset) => {
  const ext = getAssetExt(asset);
  if (ext === "pdf") return true;
  const mime = String(asset?.mimeType || "").toLowerCase();
  return mime.includes("pdf");
};

const buildFolderTree = (items) => {
  const nodes = (items || []).map((item) => ({
    ...item,
    label: decodeAssetName(item.name || "未命名文件夹"),
    type: "folder",
    children: [],
  }));
  const map = new Map(nodes.map((item) => [item.id, item]));
  const root = {
    id: "root",
    label: "全部资源",
    type: "root",
    children: [],
  };
  nodes.forEach((node) => {
    if (node.parentId && map.has(node.parentId)) {
      map.get(node.parentId).children.push(node);
    } else {
      root.children.push(node);
    }
  });
  return [root];
};

const folderTree = computed(() => buildFolderTree(folders.value));

const filteredFolderTree = computed(() => folderTree.value);

const filterFolderNode = (value, data) => {
  if (!value) return true;
  return String(data?.label || "")
    .toLowerCase()
    .includes(String(value).toLowerCase());
};

watch(folderSearch, () => {
  folderTreeRef.value?.filter?.(folderSearch.value);
});

const normalizedAssets = computed(() =>
  (assets.value || []).map((asset) => {
    const displayName = decodeAssetName(asset.name || asset.originalName || "");
    return {
      ...asset,
      displayName,
      ext: getAssetExt({ ...asset, displayName }),
      size: Number(
        asset.size ??
          asset.fileSize ??
          asset.file_size ??
          asset.length ??
          asset.bytes ??
          asset.metadata?.size ??
          asset.metadata?.fileSize ??
          asset.metadata?.length ??
          0,
      ),
    };
  }),
);

const filteredAssets = computed(() => {
  let list = normalizedAssets.value;
  if (selectedFolderId.value !== "root") {
    list = list.filter((item) => item.folderId === selectedFolderId.value);
  }
  if (assetSearch.value) {
    const keyword = assetSearch.value.toLowerCase();
    list = list.filter((item) =>
      item.displayName.toLowerCase().includes(keyword),
    );
  }
  return list;
});

const handleFolderClick = (data) => {
  if (!data) return;
  selectedFolderId.value = data.id;
};

const handleFolderContextMenu = (event, data) => {
  if (!data) return;
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY,
    type: "folder",
    folder: data,
    asset: null,
  };
};

const openAssetContextMenu = (event, asset) => {
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY,
    type: "asset",
    asset,
    folder: null,
  };
};

const closeContextMenu = () => {
  contextMenu.value.visible = false;
};

const handlePreviewContext = () => {
  if (contextMenu.value.asset) openPreview(contextMenu.value.asset);
  closeContextMenu();
};

const handleDetailContext = () => {
  if (contextMenu.value.asset) {
    detailAsset.value = contextMenu.value.asset;
    detailVisible.value = true;
  }
  closeContextMenu();
};

const handleCreateFolder = async () => {
  closeContextMenu();
  if (!projectId.value) return;
  const result = await ElMessageBox.prompt("请输入文件夹名称", "新建文件夹", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    inputPlaceholder: "文件夹名称",
  }).catch(() => null);
  if (!result?.value) return;
  await assetApi.createFolder(projectId.value, {
    name: result.value,
    parentId:
      contextMenu.value.folder?.id && contextMenu.value.folder?.id !== "root"
        ? contextMenu.value.folder.id
        : null,
  });
  await loadFolders();
};

const handleRenameFolder = async () => {
  const folder = contextMenu.value.folder;
  closeContextMenu();
  if (!folder || folder.id === "root") return;
  const result = await ElMessageBox.prompt("请输入新的文件夹名称", "重命名", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    inputValue: folder.label,
  }).catch(() => null);
  if (!result?.value) return;
  await assetApi.renameFolder(projectId.value, folder.id, {
    name: result.value,
  });
  await loadFolders();
};

const handleDeleteFolder = async () => {
  const folder = contextMenu.value.folder;
  closeContextMenu();
  if (!folder || folder.id === "root") return;
  await ElMessageBox.confirm(
    "确认删除该文件夹吗？删除后无法恢复。",
    "删除确认",
    {
      type: "warning",
    },
  ).catch(() => null);
  await assetApi.deleteFolder(projectId.value, folder.id);
  await loadFolders();
  await loadAssets();
};

const handleRenameAsset = async () => {
  const asset = contextMenu.value.asset;
  closeContextMenu();
  if (!asset) return;
  const result = await ElMessageBox.prompt("请输入新的资源名称", "重命名", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    inputValue: asset.displayName,
  }).catch(() => null);
  if (!result?.value) return;
  await assetApi.updateAsset(projectId.value, asset.id, { name: result.value });
  await loadAssets();
};

const handleDeleteAsset = async () => {
  const asset = contextMenu.value.asset;
  closeContextMenu();
  if (!asset) return;
  await ElMessageBox.confirm("确认删除该资源吗？", "删除确认", {
    type: "warning",
  }).catch(() => null);
  await assetApi.deleteAsset(projectId.value, asset.id);
  await loadAssets();
};

const handleCopyAsset = () => {
  clipboardAsset.value = contextMenu.value.asset;
  clipboardMode.value = "copy";
  closeContextMenu();
};

const handleCutAsset = () => {
  clipboardAsset.value = contextMenu.value.asset;
  clipboardMode.value = "cut";
  closeContextMenu();
};

const handlePasteAsset = async () => {
  const targetFolderId = contextMenu.value.folder?.id || selectedFolderId.value;
  if (!clipboardAsset.value || !projectId.value) return;
  const folderId = targetFolderId === "root" ? null : targetFolderId;
  if (clipboardMode.value === "copy") {
    await assetApi.copyAsset(projectId.value, clipboardAsset.value.id, {
      folderId,
    });
  } else if (clipboardMode.value === "cut") {
    await assetApi.updateAsset(projectId.value, clipboardAsset.value.id, {
      folderId,
    });
    clipboardAsset.value = null;
    clipboardMode.value = null;
  }
  closeContextMenu();
  await loadAssets();
};

const handleMoveAsset = () => {
  moveTargetFolderId.value = selectedFolderId.value;
  moveAssetTarget.value = contextMenu.value.asset;
  moveDialogVisible.value = true;
  closeContextMenu();
};

const handleMoveFolderSelect = (data) => {
  moveTargetFolderId.value = data?.id;
};

const confirmMove = async () => {
  const asset = moveAssetTarget.value || clipboardAsset.value;
  if (!asset) return;
  const folderId =
    moveTargetFolderId.value === "root" ? null : moveTargetFolderId.value;
  await assetApi.updateAsset(projectId.value, asset.id, { folderId });
  moveDialogVisible.value = false;
  moveAssetTarget.value = null;
  await loadAssets();
};

const openPreview = (asset) => {
  previewAsset.value = asset;
  previewVisible.value = true;
};

const triggerFileSelect = () => {
  fileInputRef.value?.click?.();
};

const handleFileInputChange = async (event) => {
  const files = Array.from(event.target.files || []);
  event.target.value = "";
  if (!files.length) return;
  await uploadFiles(
    files,
    selectedFolderId.value === "root" ? null : selectedFolderId.value,
  );
};

const handleUploadDrop = async (event) => {
  const files = Array.from(event.dataTransfer?.files || []);
  if (!files.length) return;
  await uploadFiles(
    files,
    selectedFolderId.value === "root" ? null : selectedFolderId.value,
  );
};

const handleDropToFolder = (folder) => async (event) => {
  const files = Array.from(event.dataTransfer?.files || []);
  if (files.length) {
    await uploadFiles(files, folder.id === "root" ? null : folder.id);
    return;
  }
  const assetId = event.dataTransfer?.getData("asset-id");
  if (assetId) {
    await assetApi.updateAsset(projectId.value, assetId, {
      folderId: folder.id === "root" ? null : folder.id,
    });
    await loadAssets();
  }
};

const handleAssetDragStart = (asset, event) => {
  if (!asset) return;
  const dataTransfer = event?.dataTransfer;
  if (dataTransfer) {
    dataTransfer.setData("asset-id", asset.id);
  }
};

const uploadFiles = async (files, folderId) => {
  if (!projectId.value || !files.length) return;
  const duplicated = files.some((file) =>
    normalizedAssets.value.some(
      (asset) =>
        asset.displayName === file.name &&
        (asset.folderId || null) === (folderId || null),
    ),
  );
  let conflictStrategy = "rename";
  if (duplicated) {
    const result = await ElMessageBox.confirm(
      "存在同名资源，是否替换？",
      "上传冲突",
      {
        confirmButtonText: "替换",
        cancelButtonText: "重命名",
        type: "warning",
      },
    ).catch(() => null);
    conflictStrategy = result ? "replace" : "rename";
  }
  await assetApi.uploadAssets(projectId.value, files, folderId, {
    conflictStrategy,
  });
  ElMessage.success("上传成功");
  await loadAssets();
};

const loadFolders = async () => {
  if (!projectId.value) return;
  const response = await assetApi.getFolders(projectId.value);
  const data = unwrapApiData(response);
  const list = data?.folders || data?.items || data || [];
  folders.value = Array.isArray(list) ? list : [];
};

const loadAssets = async () => {
  if (!projectId.value) return;
  const response = await assetApi.getAssets(projectId.value);
  const data = unwrapApiData(response);
  const list = data?.assets || data?.items || data || [];
  assets.value = Array.isArray(list) ? list : [];
};

const handleGlobalClick = (event) => {
  const menu = document.querySelector(".context-menu");
  if (menu && !menu.contains(event.target)) {
    closeContextMenu();
  }
};

onMounted(() => {
  document.addEventListener("click", handleGlobalClick);
  if (projectId.value) {
    loadFolders();
    loadAssets();
  }
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleGlobalClick);
});

watch(projectId, (value) => {
  if (value) {
    loadFolders();
    loadAssets();
  }
});
</script>

<style scoped>
.resource-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 10px 12px;
  height: 100%;
}

.resource-search {
  margin-bottom: 4px;
}

.resource-layout {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1;
  min-height: 0;
}

.resource-folders {
  width: 100%;
  display: flex;
  flex-direction: column;
  border-bottom: 1px solid #edf0f5;
  padding-bottom: 10px;
  min-height: 0;
  flex: 0 0 42%;
}

.resource-assets {
  width: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
  flex: 1;
}

.pane-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
  margin-bottom: 6px;
}

.folder-scroll,
.asset-scroll {
  flex: 1;
  min-height: 0;
}

.folder-node {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 2px;
}

.folder-label {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
  gap: 8px;
}

.asset-title {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.asset-count {
  font-size: 12px;
  color: #8b95a6;
}

.asset-dropzone {
  border: 1px dashed #d8dfea;
  border-radius: 8px;
  padding: 8px 10px;
  color: #8b95a6;
  font-size: 12px;
  text-align: center;
  margin-bottom: 8px;
  cursor: pointer;
}

.asset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
  gap: 10px;
  padding-bottom: 12px;
}

.asset-card {
  border: 1px solid #edf0f5;
  border-radius: 10px;
  padding: 8px;
  background: #fff;
  cursor: pointer;
}

.asset-thumb {
  width: 100%;
  height: 100px;
  border-radius: 8px;
  overflow: hidden;
  background: #f5f7fb;
  display: flex;
  align-items: center;
  justify-content: center;
}

.asset-thumb img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.asset-thumb-placeholder {
  font-size: 12px;
  color: #8b95a6;
}

.asset-name {
  margin-top: 6px;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.asset-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: #8b95a6;
}
.asset-search {
  width: 80%;
}
.asset-body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
  width: 100%;
}
.asset-table {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.asset-table-header,
.asset-table-row {
  display: grid;
  grid-template-columns: 1fr 60px 70px;
  gap: 8px;
  align-items: center;
}

.asset-table-header {
  font-size: 12px;
  color: #8b95a6;
  padding: 6px 8px;
}

.asset-table-row {
  padding: 6px 8px;
  border-radius: 6px;
}

.asset-table-row:hover {
  background: #f5f7fb;
}

.col-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.view-toggle {
  display: flex;
  gap: 6px;
  justify-content: flex-end;
  padding-top: 8px;
}

.hidden-input {
  display: none;
}

.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e6ebf2;
  box-shadow: 0 6px 18px rgba(16, 24, 40, 0.12);
  border-radius: 8px;
  padding: 6px;
  z-index: 9999;
  min-width: 140px;
}

.context-item {
  padding: 6px 10px;
  font-size: 12px;
  color: #1f2d3d;
  cursor: pointer;
  border-radius: 6px;
}

.context-item:hover {
  background: #f5f7fb;
}

.context-item.danger {
  color: #f56c6c;
}

.context-item.disabled {
  color: #c0c4cc;
  cursor: not-allowed;
}

.preview-body {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 240px;
  max-height: 70vh;
  overflow: auto;
  padding: 10px;
}

.preview-body img {
  max-width: 100%;
  max-height: 60vh;
  object-fit: contain;
}

.preview-pdf {
  width: 100%;
  height: 65vh;
  border: none;
}

.preview-file {
  text-align: center;
}

.preview-file-name {
  font-size: 14px;
  margin-bottom: 6px;
}

.detail-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #1f2d3d;
}

:deep(.el-scrollbar__bar) {
  opacity: 0.35;
}

:deep(.el-scrollbar__bar.is-vertical) {
  width: 6px;
}

:deep(.el-scrollbar__bar.is-horizontal) {
  height: 6px;
  display: none;
}

:deep(.el-scrollbar__thumb) {
  background-color: rgba(96, 98, 102, 0.35);
  border-radius: 6px;
}
.asset-header {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
}

.asset-title-line {
  width: 100%;
}

.asset-title-text {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
  color: #1f2d3d;
}

.asset-subline {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.detail-link {
  color: #2f6fe4;
  cursor: pointer;
  word-break: break-all;
  width: 90%;
}

.detail-link:hover {
  text-decoration: underline;
}

.preview-pdf-wrap {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.preview-link {
  align-self: flex-end;
  margin-top: 8px;
}

.asset-table :deep(.el-scrollbar__wrap) {
  overflow-x: hidden;
}
</style>
