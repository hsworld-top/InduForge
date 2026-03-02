<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark">NA</div>
        <div>
          <div class="brand-name">NodeAgent</div>
          <div class="brand-sub">运维代理控制台</div>
        </div>
      </div>

      <nav class="nav-list">
        <button
          v-for="item in navItems"
          :key="item.path"
          class="nav-item"
          :class="{ active: isActive(item.path) }"
          @click="go(item.path)"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <main class="main">
      <section class="content">
        <router-view />
      </section>
    </main>
  </div>
</template>

<script setup>
import { useRoute, useRouter } from 'vue-router'
import { Connection, Grid, Setting } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()

const navItems = [
  { path: '/projects/local', label: '本地运维', icon: Grid },
  { path: '/projects/online', label: '远程运维', icon: Connection },
  { path: '/init', label: '基础配置', icon: Setting },
]

const go = (path) => {
  if (route.path !== path) {
    router.push(path)
  }
}

const isActive = (path) => {
  return route.path === path
}
</script>

<style scoped>
.app-shell {
  display: grid;
  grid-template-columns: 248px 1fr;
  height: 100vh;
  overflow: hidden;
}

.sidebar {
  background: var(--bg-surface);
  border-right: 1px solid var(--border-subtle);
  padding: var(--space-5);
  overflow-y: auto;
}

.brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-6);
}

.brand-mark {
  width: 34px;
  height: 34px;
  border-radius: var(--radius-sm);
  background: var(--brand-500);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  display: grid;
  place-items: center;
}

.brand-name {
  font-size: 15px;
  font-weight: 700;
  color: var(--fg-primary);
}

.brand-sub {
  font-size: 12px;
  color: var(--fg-muted);
}

.nav-list {
  display: grid;
  gap: var(--space-2);
}

.nav-item {
  border: 1px solid transparent;
  background: transparent;
  border-radius: var(--radius-sm);
  padding: 10px 12px;
  color: var(--fg-secondary);
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  text-align: left;
}

.nav-item:hover {
  background: var(--bg-surface-2);
}

.nav-item.active {
  color: var(--brand-600);
  border-color: rgba(23, 105, 224, 0.24);
  background: rgba(23, 105, 224, 0.06);
}

.main {
  padding: var(--space-5);
  overflow-y: auto;
}

.content {
  min-height: 100%;
}

@media (max-width: 1000px) {
  .app-shell {
    grid-template-columns: 1fr;
  }

  .sidebar {
    display: none;
  }
}
</style>
