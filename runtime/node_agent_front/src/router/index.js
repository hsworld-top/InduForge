import { createRouter, createWebHistory } from 'vue-router'
import { isInitialized } from '@/utils/initCheck'
import InitWizard from '@/views/InitWizard.vue'
import Dashboard from '@/views/Dashboard.vue'
import Projects from '@/views/Projects.vue'
import Profile from '@/views/Profile.vue'
import Logs from '@/views/Logs.vue'

const routes = [
  {
    path: '/init',
    name: 'InitWizard',
    component: InitWizard,
    meta: { noAuthRequired: true },
  },
  {
    path: '/',
    name: 'Dashboard',
    component: Dashboard,
  },
  {
    path: '/projects',
    name: 'Projects',
    component: Projects,
  },
  {
    path: '/profile',
    name: 'Profile',
    component: Profile,
  },
  {
    path: '/logs/:projectId',
    name: 'Logs',
    component: Logs,
    props: true,
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 全局路由守卫 - 检查初始化状态
router.beforeEach(async (to, from, next) => {
  const initialized = await isInitialized()

  // 如果未初始化且不是访问初始化页面，重定向到初始化页面
  if (!initialized && to.path !== '/init') {
    next('/init')
  }
  // 如果已初始化但访问初始化页面，重定向到首页
  else if (initialized && to.path === '/init') {
    next('/')
  }
  // 其他情况正常导航
  else {
    next()
  }
})

export default router
