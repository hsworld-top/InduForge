import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import SceneView from '../views/SceneView.vue'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/2d/:resourceId?', name: '2d', component: SceneView, props: { workspace: '2d' } },
    { path: '/3d/:resourceId?', name: '3d-scene', component: SceneView, props: { workspace: '3d-scene' } },
    { path: '/3d-model/:resourceId?', name: '3d-model', component: SceneView, props: { workspace: '3d-model' } },
  ],
})
