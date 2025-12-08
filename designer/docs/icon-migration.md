# Element Plus 图标迁移指南

## 已迁移的文件 ✅

- ✅ `ContextMenu.vue` - 右键菜单
- ✅ `TextComponentEditor.vue` - 文本编辑器
- ✅ `SpacingEditor.vue` - 间距编辑器
- ✅ `FlexEditor.vue` - Flex布局编辑器
- ✅ `GridEditor.vue` - Grid布局编辑器
- ✅ `DesignCenter.vue` - 主视图
- ✅ `main.js` - 已注释全局注册

## 待迁移的文件 📝

需要手动迁移以下文件中的 Element Plus 图标：

1. `components/panels/ComponentLibrary.vue`
2. `components/panels/ComponentTree.vue`
3. `components/panels/DataSourcePanel.vue`
4. `components/panels/DataBindingPanel.vue`
5. `components/panels/PageTree.vue`

## 迁移步骤

### 1. 替换导入语句

**旧写法**:
```javascript
import { Plus, Delete, Edit } from '@element-plus/icons-vue';
```

**新写法**:
```javascript
import IconTablerPlus from '~icons/tabler/plus';
import IconTablerTrash from '~icons/tabler/trash';
import IconTablerEdit from '~icons/tabler/edit';
```

### 2. 替换模板中的使用

**旧写法**:
```vue
<el-icon><Plus /></el-icon>
<el-icon><Delete /></el-icon>
```

**新写法**:
```vue
<IconTablerPlus />
<IconTablerTrash />
```

或在 el-icon 中使用：
```vue
<el-icon><IconTablerPlus /></el-icon>
```

## Element Plus → Tabler Icons 映射表

| Element Plus | Tabler Icons | 导入语句 |
|-------------|--------------|----------|
| `Plus` | `IconTablerPlus` | `import IconTablerPlus from '~icons/tabler/plus';` |
| `Delete` | `IconTablerTrash` | `import IconTablerTrash from '~icons/tabler/trash';` |
| `Edit` | `IconTablerEdit` | `import IconTablerEdit from '~icons/tabler/edit';` |
| `Search` | `IconTablerSearch` | `import IconTablerSearch from '~icons/tabler/search';` |
| `Folder` | `IconTablerFolder` | `import IconTablerFolder from '~icons/tabler/folder';` |
| `FolderOpened` | `IconTablerFolderOpen` | `import IconTablerFolderOpen from '~icons/tabler/folder-open';` |
| `Document` | `IconTablerFile` | `import IconTablerFile from '~icons/tabler/file';` |
| `Pointer` | `IconTablerPointer` | `import IconTablerPointer from '~icons/tabler/pointer';` |
| `Picture` | `IconTablerPhoto` | `import IconTablerPhoto from '~icons/tabler/photo';` |
| `EditPen` | `IconTablerPencil` | `import IconTablerPencil from '~icons/tabler/pencil';` |
| `Grid` | `IconTablerLayoutGrid` | `import IconTablerLayoutGrid from '~icons/tabler/layout-grid';` |
| `Menu` | `IconTablerMenu2` | `import IconTablerMenu2 from '~icons/tabler/menu-2';` |
| `List` | `IconTablerList` | `import IconTablerList from '~icons/tabler/list';` |
| `Operation` | `IconTablerSettings` | `import IconTablerSettings from '~icons/tabler/settings';` |
| `Lock` | `IconTablerLock` | `import IconTablerLock from '~icons/tabler/lock';` |
| `Unlock` | `IconTablerLockOpen` | `import IconTablerLockOpen from '~icons/tabler/lock-open';` |
| `Refresh` | `IconTablerRefresh` | `import IconTablerRefresh from '~icons/tabler/refresh';` |
| `Database` | `IconTablerDatabase` | `import IconTablerDatabase from '~icons/tabler/database';` |
| `Link` | `IconTablerLink` | `import IconTablerLink from '~icons/tabler/link';` |
| `Calculator` | `IconTablerCalculator` | `import IconTablerCalculator from '~icons/tabler/calculator';` |
| `ArrowRight` | `IconTablerArrowRight` | `import IconTablerArrowRight from '~icons/tabler/arrow-right';` |
| `HomeFilled` | `IconTablerHome` | `import IconTablerHome from '~icons/tabler/home';` |
| `RefreshLeft` | `IconTablerArrowBackUp` | `import IconTablerArrowBackUp from '~icons/tabler/arrow-back-up';` |
| `RefreshRight` | `IconTablerArrowForwardUp` | `import IconTablerArrowForwardUp from '~icons/tabler/arrow-forward-up';` |
| `Check` | `IconTablerCheck` | `import IconTablerCheck from '~icons/tabler/check';` |
| `ZoomIn` | `IconTablerZoomIn` | `import IconTablerZoomIn from '~icons/tabler/zoom-in';` |
| `ZoomOut` | `IconTablerZoomOut` | `import IconTablerZoomOut from '~icons/tabler/zoom-out';` |
| `FullScreen` | `IconTablerMaximize` | `import IconTablerMaximize from '~icons/tabler/maximize';` |
| `Monitor` | `IconTablerDeviceDesktop` | `import IconTablerDeviceDesktop from '~icons/tabler/device-desktop';` |
| `Iphone` | `IconTablerDeviceMobile` | `import IconTablerDeviceMobile from '~icons/tabler/device-mobile';` |
| `Rank` | `IconTablerStack` | `import IconTablerStack from '~icons/tabler/stack';` |
| `Right` | `IconTablerArrowRight` | `import IconTablerArrowRight from '~icons/tabler/arrow-right';` |
| `Back` | `IconTablerArrowLeft` | `import IconTablerArrowLeft from '~icons/tabler/arrow-left';` |
| `Bottom` | `IconTablerArrowDown` | `import IconTablerArrowDown from '~icons/tabler/arrow-down';` |
| `Top` | `IconTablerArrowUp` | `import IconTablerArrowUp from '~icons/tabler/arrow-up';` |

## 批量迁移示例

### ComponentLibrary.vue

**旧代码**:
```vue
<script setup>
import { Search, Folder, Document, Pointer, Picture, EditPen, Grid, Menu, List, Operation } from '@element-plus/icons-vue';
</script>

<template>
  <el-icon><Search /></el-icon>
  <el-icon><Folder /></el-icon>
</template>
```

**新代码**:
```vue
<script setup>
import IconTablerSearch from '~icons/tabler/search';
import IconTablerFolder from '~icons/tabler/folder';
import IconTablerFile from '~icons/tabler/file';
import IconTablerPointer from '~icons/tabler/pointer';
import IconTablerPhoto from '~icons/tabler/photo';
import IconTablerPencil from '~icons/tabler/pencil';
import IconTablerLayoutGrid from '~icons/tabler/layout-grid';
import IconTablerMenu2 from '~icons/tabler/menu-2';
import IconTablerList from '~icons/tabler/list';
import IconTablerSettings from '~icons/tabler/settings';
</script>

<template>
  <IconTablerSearch />
  <IconTablerFolder />
</template>
```

## 快速搜索替换

在 VS Code 中使用正则表达式批量替换：

### 1. 替换导入语句
查找：`import \{ ([^}]+) \} from '@element-plus/icons-vue';`
替换：手动根据上面的映射表替换

### 2. 替换模板使用
查找：`<el-icon><(\w+) /></el-icon>`
替换：`<IconTabler$1 />`（需要根据映射表调整）

## 注意事项

1. ⚠️ 某些 Element Plus 图标没有完全对应的 Tabler 图标，可能需要选择相似的
2. ⚠️ 图标大小和颜色继承父元素样式
3. ⚠️ 如果在 `<el-icon>` 中使用，需要保持 `<el-icon>` 包装器
4. ✅ 所有新图标都支持 SVG 属性（如 `style`, `class`）
5. ✅ 图标会自动按需加载，无需担心包体积

## 验证迁移

迁移完成后：

1. 运行 `npm run dev` 启动项目
2. 检查控制台是否有图标相关错误
3. 检查页面上图标是否正常显示
4. 确认所有交互功能正常

## 完全移除 @element-plus/icons-vue

当所有文件迁移完成后，可以移除依赖：

```bash
npm uninstall @element-plus/icons-vue
```

然后从 `vite.config.js` 的 `manualChunks` 中移除：
```javascript
manualChunks: {
  vendor: ['vue', 'vue-router', 'pinia'],
  ui: ['element-plus'], // 移除 '@element-plus/icons-vue'
},
```

