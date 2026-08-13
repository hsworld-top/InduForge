<script setup>
import { points } from '@induforge/runtime-sdk'

async function loadExample() {
  await points.example.get({})
}
</script>

<template>
  <main class="page-card">
    <p class="eyebrow">VUE 3 + JAVASCRIPT</p>
    <h1>工业应用页面工程</h1>
    <p>路由和页面源码是应用的唯一事实。数据点、角色和场景说明位于 `.induforge/context/`。</p>
    <button type="button" @click="loadExample">调用示例数据点</button>
  </main>
</template>
