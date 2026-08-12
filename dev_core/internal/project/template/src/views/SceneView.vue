<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const props = defineProps({ workspace: { type: String, required: true } })
const route = useRoute()
const resourceId = computed(() => route.params.resourceId || 'main')
</script>

<template>
  <main class="page-card">
    <p class="eyebrow">INDUSTRIAL SCENE ROUTE</p>
    <h1>{{ workspace }}</h1>
    <p>当前资源：{{ resourceId }}</p>
    <p>正式运行时由平台场景宿主根据该路由加载对应 HT 产物。</p>
  </main>
</template>
