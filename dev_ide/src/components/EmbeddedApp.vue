<template>
  <div class="embedded-app">
    <iframe
      :src="appUrl"
      frameborder="0"
      class="embedded-iframe"
      allow="cookies"
      sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-modals allow-top-navigation allow-downloads"
    >
      您的浏览器不支持 iframe。
    </iframe>
  </div>
</template>

<script setup>
import { computed } from "vue";
import { buildAppUrl } from "@/utils/appUrl";

const props = defineProps({
  appType: {
    type: String,
    required: true,
    validator: (value) => ["datacenter", "designer"].includes(value),
  },
  project: {
    type: Object,
    required: true,
  },
});

const appUrl = computed(() => buildAppUrl(props.appType, props.project));
</script>

<style scoped>
.embedded-app {
  width: 100%;
  height: 100%;
  display: flex;
  overflow: hidden;
}

.embedded-iframe {
  flex: 1;
  width: 100%;
  height: 100%;
  border: none;
}
</style>
