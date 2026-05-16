<template>
  <div class="datapoint-workspace">
    <DataPointList
      :project-id="projectId"
      mode="management"
      :filter-q="filterQ"
      :filter-status="filterStatus"
      :filter-source="filterSource"
      :filter-tags="filterTags"
      :sort-field="sortField"
      :sort-order="sortOrder"
      :page="page"
      @update:filter="handleFilterUpdate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import DataPointList from "@/components/datapoint/DataPointList.vue";

defineProps<{
  projectId: string;
}>();

const route = useRoute();
const router = useRouter();

/* 从 URL query 读取筛选参数 */
const filterQ = computed(() => String(route.query.q || ""));
const filterStatus = computed(() => String(route.query.status || ""));
const filterSource = computed(() => String(route.query.source || ""));
const filterTags = computed(() => {
  const raw = route.query.tags;
  if (!raw) return [];
  return (Array.isArray(raw) ? raw : [raw]).map(String);
});
const sortField = computed(() => String(route.query.sort || "updatedAt"));
const sortOrder = computed(() => String(route.query.order || "desc"));
const page = computed(() => {
  const n = Number(route.query.page);
  return n > 0 ? n : 1;
});

/* 列表发出筛选变化时同步到 URL */
function handleFilterUpdate(params: Record<string, unknown>) {
  const query: Record<string, string | string[]> = {};
  if (params.q) query.q = String(params.q);
  if (params.status) query.status = String(params.status);
  if (params.source) query.source = String(params.source);
  if (Array.isArray(params.tags) && params.tags.length > 0) {
    query.tags = (params.tags as string[]);
  }
  if (params.sort && params.sort !== "updatedAt") query.sort = String(params.sort);
  if (params.order && params.order !== "desc") query.order = String(params.order);
  if (params.page && Number(params.page) > 1) query.page = String(params.page);
  void router.replace({ query });
}
</script>

<style scoped>
.datapoint-workspace {
  height: calc(100vh - 32px);
  min-height: 0;
  overflow: hidden;
}
</style>
