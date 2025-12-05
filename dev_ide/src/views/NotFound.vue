<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
    <div class="text-center">
      <div class="text-6xl font-bold text-gray-300 dark:text-gray-600 mb-4">404</div>
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white mb-4">页面未找到</h1>
      <p class="text-gray-600 dark:text-gray-400 mb-8">抱歉，您访问的页面不存在或已被移除。</p>
      <router-link to="/dashboard" class="btn-primary inline-block"> 返回仪表板 </router-link>
    </div>
  </div>
</template>

<script>
export default {
  name: 'NotFound',
}
</script>
