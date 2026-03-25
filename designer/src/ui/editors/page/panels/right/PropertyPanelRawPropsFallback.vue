<!--
  无 Manifest 时展示格式化后的原始 props
-->
<script setup lang="ts">
defineProps<{
  text: string;
}>();
</script>

<template>
  <div>
    <div class="text-xs text-gray-500 mb-2">Props</div>
    <pre class="text-xs bg-gray-50 dark:bg-gray-900 p-2 rounded overflow-auto max-h-60">{{
      text
    }}</pre>
  </div>
</template>
