<template>
  <div class="design-center">
    <iframe
      id="ClientDesignerView"
      :src="webContainerUrl"
      frameborder="0"
      class="design-iframe"
      allow="cookies"
      sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-modals"
    >
      您的浏览器不支持 iframe。
    </iframe>
  </div>
</template>

<script setup>
import { computed } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();

// 从路由获取 project 信息
const project = computed(() => {
  // 优先从 URL 参数获取（支持 pid 和 id，兼容旧版本）
  const projectId = route.query.pid || route.query.id || route.meta?.project?.id;
  const tenantId = route.query.tenant || route.meta?.project?.tenantId;
  const pageId = route.query.pageid || route.meta?.project?.pageId || '1';
  const type = route.query.type || route.meta?.project?.type || 'app';
  
  if (projectId) {
    return { id: projectId, tenantId: tenantId, pageId: pageId, type: type };
  }
  return null;
});

const webContainerUrl = computed(() => {
  const params = [];
  const { id, tenantId: tenant, pageId, type } = project.value || {};
  if (id) params.push(`id=${id}`);
  if (tenant) params.push(`tenant=${tenant}`);
  if (pageId) params.push(`pageid=${pageId}`);
  if (type) params.push(`type=${type}`);
  
  console.log(import.meta.env.MODE);
  // 开发环境进入 localhost:8090
  if (import.meta.env.MODE?.includes('dev')) {
    return `http://localhost:8090/designer/?${params.join("&")}`;
  } else {
    return `/designer/?${params.join("&")}`; // 不写协议/端口
  }
});
</script>

<style scoped>
/* 父容器和 body/html 填满整个视口 */
html, body, #app {
  height: 100%;
  margin: 0;
  padding: 0;
}

.design-center {
  width: 100%;
  height: 100%;
  display: flex;
}

.design-iframe {
  flex: 1;       /* 自适应父容器高度和宽度 */
  width: 100%;
  border: none;
}
</style>

