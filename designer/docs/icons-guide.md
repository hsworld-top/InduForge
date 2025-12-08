# 图标使用指南

## 已集成 unplugin-icons

项目已集成 `unplugin-icons` + `Iconify`，支持 200,000+ 图标。

## 使用方法

### 1. 导入图标

```vue
<script setup>
// 格式：~icons/[图标集]/[图标名]
import IconTablerHome from '~icons/tabler/home';
import IconTablerEdit from '~icons/tabler/edit';
import IconMdiAccount from '~icons/mdi/account';
</script>
```

### 2. 使用图标

```vue
<template>
  <!-- 作为组件使用 -->
  <IconTablerHome />
  
  <!-- 设置大小和颜色 -->
  <IconTablerEdit style="font-size: 20px; color: #409eff" />
  
  <!-- 动态组件 -->
  <component :is="iconComponent" />
</template>
```

## 推荐的图标集

### Tabler Icons (主要使用)
- **前缀**: `~icons/tabler/`
- **数量**: 4,000+
- **风格**: 简洁、线条、现代
- **官网**: https://tabler-icons.io/

**常用图标**:
```javascript
import IconTablerCopy from '~icons/tabler/copy';           // 复制
import IconTablerTrash from '~icons/tabler/trash';         // 删除
import IconTablerEdit from '~icons/tabler/edit';           // 编辑
import IconTablerPlus from '~icons/tabler/plus';           // 添加
import IconTablerSettings from '~icons/tabler/settings';   // 设置
import IconTablerSearch from '~icons/tabler/search';       // 搜索
import IconTablerHome from '~icons/tabler/home';           // 首页
import IconTablerUser from '~icons/tabler/user';           // 用户
import IconTablerFile from '~icons/tabler/file';           // 文件
import IconTablerFolder from '~icons/tabler/folder';       // 文件夹
import IconTablerDownload from '~icons/tabler/download';   // 下载
import IconTablerUpload from '~icons/tabler/upload';       // 上传
import IconTablerEye from '~icons/tabler/eye';             // 查看
import IconTablerEyeOff from '~icons/tabler/eye-off';      // 隐藏
import IconTablerCheck from '~icons/tabler/check';         // 确认
import IconTablerX from '~icons/tabler/x';                 // 关闭
import IconTablerArrowUp from '~icons/tabler/arrow-up';    // 上箭头
import IconTablerArrowDown from '~icons/tabler/arrow-down'; // 下箭头
```

### Material Design Icons (备选)
- **前缀**: `~icons/mdi/`
- **数量**: 7,000+
- **风格**: 全面、标准
- **官网**: https://materialdesignicons.com/

```javascript
import IconMdiContentCopy from '~icons/mdi/content-copy';
import IconMdiDelete from '~icons/mdi/delete';
import IconMdiPencil from '~icons/mdi/pencil';
```

### Heroicons (备选)
- **前缀**: `~icons/heroicons/` 或 `~icons/heroicons-outline/` 或 `~icons/heroicons-solid/`
- **数量**: 200+
- **风格**: Tailwind CSS 官方，两种风格
- **官网**: https://heroicons.com/

```javascript
import IconHeroiconsHome from '~icons/heroicons/home';
import IconHeroiconsOutlineHome from '~icons/heroicons-outline/home';
import IconHeroiconsSolidHome from '~icons/heroicons-solid/home';
```

## 搜索图标

访问 **Icônes** 网站搜索和预览所有图标：
- 🔗 https://icones.js.org/

使用方法：
1. 搜索图标名称（如 "home"）
2. 选择图标集（Tabler、MDI 等）
3. 点击图标，复制导入代码
4. 粘贴到代码中使用

## 已更新的组件

- ✅ `ContextMenu.vue` - 右键菜单（已使用 Tabler Icons）
  - IconTablerCopy - 复制
  - IconTablerClipboard - 粘贴
  - IconTablerCopyPlus - 复制并粘贴
  - IconTablerTrash - 删除
  - IconTablerArrowBigUpLines - 置顶
  - IconTablerArrowBigDownLines - 置底
  - IconTablerArrowUp - 上移
  - IconTablerArrowDown - 下移

## 性能优化

- ✅ **按需导入**: 只打包使用的图标
- ✅ **自动安装**: Vite 插件自动下载需要的图标集
- ✅ **树摇优化**: 未使用的图标不会打包
- ✅ **体积小**: 每个图标仅 1-2KB

## 注意事项

1. 图标名称使用 **kebab-case**（如 `arrow-up`）
2. 组件名称使用 **PascalCase**（如 `IconTablerArrowUp`）
3. 首次使用新图标集时，会自动下载（需要网络）
4. 推荐优先使用 Tabler Icons，风格统一
5. 图标颜色默认继承父元素的 `color` 属性

## 示例

```vue
<template>
  <div>
    <!-- 按钮中使用 -->
    <el-button>
      <IconTablerPlus style="margin-right: 4px" />
      添加组件
    </el-button>

    <!-- 菜单项中使用 -->
    <div class="menu-item">
      <IconTablerEdit class="icon" />
      <span>编辑</span>
    </div>

    <!-- 工具栏中使用 -->
    <div class="toolbar">
      <IconTablerUndo @click="undo" />
      <IconTablerRedo @click="redo" />
      <IconTablerSave @click="save" />
    </div>
  </div>
</template>

<script setup>
import IconTablerPlus from '~icons/tabler/plus';
import IconTablerEdit from '~icons/tabler/edit';
import IconTablerUndo from '~icons/tabler/arrow-back-up';
import IconTablerRedo from '~icons/tabler/arrow-forward-up';
import IconTablerSave from '~icons/tabler/device-floppy';
</script>

<style scoped>
.icon {
  width: 18px;
  height: 18px;
  color: #606266;
}

.toolbar {
  display: flex;
  gap: 8px;
}

.toolbar svg {
  width: 20px;
  height: 20px;
  cursor: pointer;
  color: #606266;
  transition: color 0.3s;
}

.toolbar svg:hover {
  color: #409eff;
}
</style>
```

