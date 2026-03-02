import { createRouter, createWebHistory } from 'vue-router'
import InitWizard from '@/views/InitWizard.vue'
import ProjectsLocal from '@/views/ProjectsLocal.vue'
import ProjectsOnline from '@/views/ProjectsOnline.vue'
import Profile from '@/views/Profile.vue'
import Logs from '@/views/Logs.vue'

const routes = [
  {
    path: '/init',
    name: 'InitWizard',
    component: InitWizard,
    meta: { noAuthRequired: true, title: '基础配置' },
  },
  {
    path: '/',
    redirect: '/projects/local',
  },
  {
    path: '/projects/local',
    name: 'ProjectsLocal',
    component: ProjectsLocal,
    meta: { title: '本地运维' },
  },
  {
    path: '/projects/online',
    name: 'ProjectsOnline',
    component: ProjectsOnline,
    meta: { title: '在线状态' },
  },
  {
    path: '/projects',
    redirect: '/projects/local',
  },
  {
    path: '/profile',
    name: 'Profile',
    component: Profile,
    meta: { title: '节点设置（含运维中心绑定）' },
  },
  {
    path: '/logs/:projectId',
    name: 'Logs',
    component: Logs,
    props: true,
    meta: { title: '项目日志' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.afterEach((to) => {
  const title = to.meta?.title
  if (title) {
    document.title = `NodeAgent - ${title}`
  }
})

export default router
