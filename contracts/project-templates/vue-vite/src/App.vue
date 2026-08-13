<script setup>
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <strong>INDUFORGE</strong>
      <nav>
        <RouterLink to="/">页面</RouterLink>
        <RouterLink to="/2d/main">2D</RouterLink>
        <RouterLink to="/3d/main">3D</RouterLink>
      </nav>
    </header>
    <RouterView />
  </div>
</template>
