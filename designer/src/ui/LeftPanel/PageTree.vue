<template>
  <div class="flex flex-col gap-4 page-tree-root">
    <div class="page-tree-actions">
      <el-tooltip content="新建" placement="right">
        <el-button size="small" text @click="openCreateDialog">
          <IconEpPlus />
        </el-button>
      </el-tooltip>
    </div>

    <section>
      <div class="flex items-center gap-2 text-xs text-gray-500 mb-2">
        <span>系统页</span>
      </div>
      <el-tree
        v-if="systemTreeData.length"
        :data="systemTreeData"
        node-key="id"
        highlight-current
        :current-node-key="currentPageId"
        @node-click="handleNodeClick"
      >
        <template #default="{ data }">
          <el-dropdown trigger="contextmenu">
            <span class="page-tree-node" @contextmenu.prevent>
              <component
                :is="data.type === 'folder' ? IconEpFolder : IconEpDocument"
                class="page-tree-node__icon"
              />
              <span class="page-tree-node__label">{{ data.label }}</span>
              <span v-if="data.badge" class="page-tree-node__badge">
                {{ data.badge }}
              </span>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleCommand('delete', data)">
                  删除
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-tree>
      <div v-else class="text-xs text-gray-400 px-2">暂无系统页</div>
    </section>

    <section>
      <div class="text-xs text-gray-500 mb-2">业务页</div>
      <el-tree
        v-if="appTreeData.length"
        :data="appTreeData"
        node-key="id"
        highlight-current
        :current-node-key="currentPageId"
        @node-click="handleNodeClick"
      >
        <template #default="{ data }">
          <el-dropdown trigger="contextmenu">
            <span class="page-tree-node" @contextmenu.prevent>
              <component
                :is="data.type === 'folder' ? IconEpFolder : IconEpDocument"
                class="page-tree-node__icon"
              />
              <span class="page-tree-node__label">{{ data.label }}</span>
              <span v-if="data.badge" class="page-tree-node__badge">
                {{ data.badge }}
              </span>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleCommand('rename', data)">
                  重命名
                </el-dropdown-item>
                <el-dropdown-item
                  v-if="data.type !== 'folder'"
                  @click="handleCommand('setHome', data)"
                >
                  设为首页
                </el-dropdown-item>
                <el-dropdown-item
                  v-if="data.type !== 'folder'"
                  class="page-tree-move"
                >
                  <el-popover
                    trigger="hover"
                    placement="right-start"
                    width="160"
                    popper-class="page-tree-move-popover"
                  >
                    <div class="page-tree-move-list">
                      <el-button
                        v-for="target in getMoveTargets(data)"
                        :key="target.id ?? 'root'"
                        size="small"
                        text
                        @click="handleMove(data.id, target.id)"
                      >
                        {{ target.label }}
                      </el-button>
                      <span
                        v-if="!getMoveTargets(data).length"
                        class="text-xs text-gray-400 px-2 py-1"
                      >
                        无可选分组
                      </span>
                    </div>
                    <template #reference>
                      <span class="page-tree-move-label">移动到分组</span>
                    </template>
                  </el-popover>
                </el-dropdown-item>
                <el-dropdown-item @click="handleCommand('delete', data)" divided>
                  删除
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-tree>
      <div v-else class="text-xs text-gray-400 px-2">暂无业务页</div>
    </section>
  </div>
  <el-dialog v-model="createDialogVisible" title="新建" width="360px">
    <el-form label-width="72px">
      <el-form-item label="类型">
        <el-select v-model="createForm.type">
          <el-option label="页面" value="page" />
          <el-option label="分组" value="folder" />
          <el-option label="登录页" value="login" />
          <el-option label="登出页" value="logout" />
        </el-select>
      </el-form-item>
      <el-form-item label="名称">
        <el-input v-model="createForm.name" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="createDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="handleCreateConfirm">创建</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, ref } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { ElMessage, ElMessageBox } from "element-plus";
import IconEpPlus from "~icons/ep/plus";
import IconEpFolder from "~icons/ep/folder";
import IconEpDocument from "~icons/ep/document";

const editorStore = useEditorStore();
const { pages, currentPageId, doc, canUndo } = storeToRefs(editorStore);
const selectedNode = ref(null);
const createDialogVisible = ref(false);
const createForm = ref({ type: "page", name: "" });

/**
 * 获取页面显示名称
 * @param {import('@/editor-core').PageNode} page - 页面
 * @returns {string}
 */
const getPageLabel = (page) => {
  return page.name || page.title || page.id;
};

/**
 * 页面标识
 * @param {import('@/editor-core').PageNode} page - 页面
 * @returns {string}
 */
const getPageBadge = (page) => {
  if (doc.value?.entry?.homePageId === page.id) {
    return "首页";
  }
  if (doc.value?.entry?.loginPageId === page.id) {
    return "登录";
  }
  if (page.path?.includes("logout") || page.name?.includes("登出")) {
    return "登出";
  }
  return "";
};

/**
 * 系统页
 */
const systemPages = computed(() =>
  pages.value.filter(
    (page) =>
      page.type === "page" &&
      (page.path?.includes("login") ||
        page.path?.includes("logout") ||
        page.name?.includes("登录") ||
        page.name?.includes("登出"))
  )
);

/**
 * 业务页
 */
const appPages = computed(() =>
  pages.value.filter(
    (page) => !systemPages.value.includes(page) && page.type !== "dialog"
  )
);

/**
 * 页面树数据
 */
const systemTreeData = computed(() =>
  systemPages.value.map((page) => ({
    id: page.id,
    label: getPageLabel(page),
    type: page.type,
    badge: getPageBadge(page),
  }))
);

const buildAppTree = (pageList) => {
  const nodeMap = new Map();
  const roots = [];

  for (const page of pageList) {
    nodeMap.set(page.id, {
      id: page.id,
      label: getPageLabel(page),
      type: page.type,
      parentId: page.parentId || null,
      badge: getPageBadge(page),
      children: [],
    });
  }

  for (const node of nodeMap.values()) {
    if (node.parentId && nodeMap.has(node.parentId)) {
      nodeMap.get(node.parentId).children.push(node);
    } else {
      roots.push(node);
    }
  }

  return roots;
};

const appTreeData = computed(() => buildAppTree(appPages.value));

/**
 * 选中页面
 * @param {{ id: string }} node - 点击的节点
 */
const handleNodeClick = async (node) => {
  selectedNode.value = node;
  if (node.type === "folder") return;
  if (node.id === currentPageId.value) return;

  if (canUndo.value) {
    try {
      const targetPage = pages.value.find((page) => page.id === node.id);
      const targetName = targetPage?.name || targetPage?.id || "目标页面";
      const action = await ElMessageBox.confirm(
        `当前页面未保存，是否切换到 ${targetName}？`,
        "切换页面",
        {
          confirmButtonText: "保存并切换",
          cancelButtonText: "不保存切换",
          distinguishCancelAndClose: true,
          closeOnClickModal: false,
        }
      );
      if (action === "confirm") {
        await editorStore.saveCurrentPage();
      }
      editorStore.setCurrentPage(node.id);
      return;
    } catch (error) {
      if (error === "cancel") {
        editorStore.setCurrentPage(node.id);
        return;
      }
      if (error && error !== "close") {
        ElMessage.error("切换页面失败");
      }
      return;
    }
  }

  editorStore.setCurrentPage(node.id);
};

/**
 * 右键菜单
 * @param {string} command - 命令
 * @param {{ id: string, type: string, label: string }} data - 节点
 */
const handleCommand = (command, data) => {
  selectedNode.value = data;
  if (command === "rename") {
    void handleRename();
    return;
  }
  if (command === "delete") {
    void handleDelete();
    return;
  }
  if (command === "setHome") {
    handleSetHome(data.id);
  }
};

/**
 * 打开新建弹窗
 */
const openCreateDialog = () => {
  createForm.value = { type: "page", name: "" };
  createDialogVisible.value = true;
};

/**
 * 移动页面到分组
 * @param {string} pageId - 页面 ID
 * @param {string | null} parentId - 分组 ID
 */
const handleMove = async (pageId, parentId) => {
  try {
    await editorStore.movePageToGroup(pageId, parentId);
    ElMessage.success("移动成功");
  } catch (error) {
    ElMessage.error("移动失败");
  }
};

/**
 * 设置首页
 * @param {string} pageId - 页面 ID
 */
const handleSetHome = (pageId) => {
  editorStore.updateEntry({ homePageId: pageId });
  ElMessage.success("已设置为首页");
};

/**
 * 校验名称唯一
 * @param {string} name - 名称
 * @param {string} [excludeId] - 排除的页面 ID
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
 * 路由路径由名称生成
 * @param {string} name - 名称
 * @returns {string}
 */
const toRoutePath = (name) => {
  const normalized = name.trim().replace(/\s+/g, "-");
  return `/${encodeURIComponent(normalized)}`;
};

/**
 * 获取登录/登出固定路径
 * @param {import('@/editor-core').PageNode | undefined} page - 页面
 * @returns {string | null}
 */
const getFixedSystemPath = (page) => {
  if (!page) return null;
  if (doc.value?.entry?.loginPageId === page.id) return "/login";
  if (page.path === "/login") return "/login";
  if (page.path === "/logout") return "/logout";
  if (page.name?.includes("登录")) return "/login";
  if (page.name?.includes("登出")) return "/logout";
  return null;
};

/**
 * 创建页面/分组
 */
const handleCreateConfirm = async () => {
  const name = createForm.value.name?.trim();
  if (!name) {
    ElMessage.warning("名称不能为空");
    return;
  }
  if (!isNameUnique(name)) {
    ElMessage.warning("页面或分组名称已存在");
    return;
  }

  const type = createForm.value.type;
  const parentId =
    selectedNode.value?.type === "folder" ? selectedNode.value.id : null;

  try {
    if (type === "folder") {
      await editorStore.createPage({ name, type: "folder", parentId });
      ElMessage.success("分组创建成功");
      createDialogVisible.value = false;
      return;
    }

    if (type === "login") {
      if (
        doc.value?.entry?.loginPageId ||
        pages.value.some((page) => page.path === "/login")
      ) {
        ElMessage.warning("登录页已存在");
        return;
      }
      const result = await editorStore.createPage({
        name,
        type: "page",
        parentId: null,
      });
      const pageId = result?.data?.id || result?.id;
      if (!pageId) throw new Error("登录页创建失败");
      const payload = editorStore.createPageSchemaPayload({
        id: pageId,
        name,
        path: "/login",
      });
      await editorStore.updatePageSchema(pageId, payload);
      editorStore.updateEntry({ loginPageId: pageId });
      ElMessage.success("登录页创建成功");
      createDialogVisible.value = false;
      return;
    }

    if (type === "logout") {
      if (pages.value.some((page) => page.path === "/logout")) {
        ElMessage.warning("登出页已存在");
        return;
      }
      const result = await editorStore.createPage({
        name,
        type: "page",
        parentId: null,
      });
      const pageId = result?.data?.id || result?.id;
      if (!pageId) throw new Error("登出页创建失败");
      const payload = editorStore.createPageSchemaPayload({
        id: pageId,
        name,
        path: "/logout",
      });
      await editorStore.updatePageSchema(pageId, payload);
      ElMessage.success("登出页创建成功");
      createDialogVisible.value = false;
      return;
    }

    const result = await editorStore.createPage({
      name,
      type: "page",
      parentId,
    });
    const pageId = result?.data?.id || result?.id;
    if (!pageId) throw new Error("页面创建失败");
    const payload = editorStore.createPageSchemaPayload({
      id: pageId,
      name,
      path: toRoutePath(name),
    });
    await editorStore.updatePageSchema(pageId, payload);
    ElMessage.success("页面创建成功");
    createDialogVisible.value = false;
  } catch (error) {
    ElMessage.error("创建失败");
  }
};

/**
 * 重命名页面/分组
 */
const handleRename = async () => {
  const target = selectedNode.value;
  if (!target) {
    ElMessage.warning("请先选择要重命名的页面或分组");
    return;
  }
  try {
    const result = await ElMessageBox.prompt("请输入新名称", "重命名", {
      inputValue: target.label || "",
      confirmButtonText: "确定",
      cancelButtonText: "取消",
      closeOnClickModal: false,
    });
    const name = result?.value?.trim();
    if (!name) {
      ElMessage.warning("名称不能为空");
      return;
    }
    if (!isNameUnique(name, target.id)) {
      ElMessage.warning("页面或分组名称已存在");
      return;
    }
    const page = pages.value.find((item) => item.id === target.id);
    const fixedPath = getFixedSystemPath(page);
    const path =
      page?.type === "page" ? fixedPath || toRoutePath(name) : undefined;
    await editorStore.renamePage(target.id, name, path);
    ElMessage.success("重命名成功");
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error("重命名失败");
    }
  }
};

/**
 * 删除页面/分组
 */
const handleDelete = async () => {
  const target = selectedNode.value;
  if (!target) {
    ElMessage.warning("请先选择要删除的页面或分组");
    return;
  }
  if (target.type === "page" && target.id === currentPageId.value) {
    ElMessage.warning("当前页面不可删除");
    return;
  }
  try {
    await ElMessageBox.confirm(
      `确定删除 ${target.label} 吗？`,
      "删除确认",
      {
        confirmButtonText: "删除",
        cancelButtonText: "取消",
        type: "warning",
      }
    );
    await editorStore.deletePage(target.id);
    selectedNode.value = null;
    ElMessage.success("删除成功");
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error("删除失败");
    }
  }
};

/**
 * 分组选项
 */
const folderOptions = computed(() =>
  appPages.value
    .filter((page) => page.type === "folder")
    .map((page) => ({ id: page.id, name: page.name }))
);

/**
 * 可移动目标列表
 * @param {{ parentId?: string | null }} node - 节点
 * @returns {Array<{ id: string | null, label: string }>}
 */
const getMoveTargets = (node) => {
  const targets = [];
  if (node?.parentId) {
    targets.push({ id: null, label: "根目录" });
  }
  const groups = folderOptions.value.filter((group) => group.id !== node.parentId);
  for (const group of groups) {
    targets.push({ id: group.id, label: group.name });
  }
  return targets;
};
</script>
