<template>
  <div class="embedded-app">
    <iframe
      :src="appUrl"
      frameborder="0"
      class="embedded-iframe"
      allow="cookies"
      sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-modals allow-top-navigation"
    >
      您的浏览器不支持 iframe。
    </iframe>
  </div>
</template>

<script setup>
import { computed } from "vue";

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

const appUrl = computed(() => {
  const params = new URLSearchParams();
  if (props.project.id) params.set("pid", props.project.id);
  if (props.project.tenantId) params.set("tenant", props.project.tenantId);

  if (props.appType === "designer") {
    // params.set("pageid", "1");
    params.set("type", "app");
    return `/designer/?${params.toString()}`;
  } else {
    return `/datacenter/?${params.toString()}`;
  }
});
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
