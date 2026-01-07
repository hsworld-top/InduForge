<template>
  <div class="flex flex-col gap-3">
    <el-input v-model="keyword" size="small" placeholder="搜索组件" clearable />
    <div v-if="filteredCategories.length" class="flex flex-col gap-2">
      <el-collapse>
        <el-collapse-item
          v-for="category in filteredCategories"
          :key="category.key"
          :name="category.key"
        >
          <template #title>
            <span class="text-sm font-medium">{{ category.label }}</span>
          </template>
          <div class="flex flex-col gap-1">
            <div
              v-for="item in category.items"
              :key="item.type"
              class="flex items-center justify-between px-2 py-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <div>
                <div class="text-sm text-gray-800 dark:text-gray-200">
                  {{ item.name }}
                </div>
                <div class="text-xs text-gray-500">{{ item.type }}</div>
              </div>
              <el-tag size="small" type="info">
                {{ category.label }}
              </el-tag>
            </div>
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>
    <div v-else class="text-sm text-gray-400 text-center py-6">
      暂无可用组件
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from "vue";
import { componentRegistry } from "@/editor-core";

/**
 * 搜索关键字
 */
const keyword = ref("");

/**
 * 组件分类映射
 */
const categoryLabels = {
  basic: "基础",
  container: "容器",
  form: "表单",
  data: "数据",
  chart: "图表",
  navigation: "导航",
  layout: "布局",
  media: "媒体",
  custom: "自定义",
};

/**
 * 过滤后的组件分类列表
 */
const filteredCategories = computed(() => {
  const results = [];
  const keywordValue = keyword.value.trim().toLowerCase();
  const categories = componentRegistry.getCategories();

  for (const category of categories) {
    const items = componentRegistry
      .getByCategory(category)
      .filter((item) => {
        if (!keywordValue) return true;
        return (
          item.type.toLowerCase().includes(keywordValue) ||
          item.name.toLowerCase().includes(keywordValue)
        );
      });

    if (items.length) {
      results.push({
        key: category,
        label: categoryLabels[category] || category,
        items,
      });
    }
  }

  return results;
});
</script>
