import { createRouter, createWebHistory } from 'vue-router'
import { ROLES, ROUTE_NAMES } from '@/constants'
import { Storage } from '@/utils/storage'
import { hasRole } from '@/permissions'

// 路由组件懒加载
const Login = () => import('@/views/auth/Login.vue')
const Dashboard = () => import('@/views/Dashboard.vue')

// 管理员路由
const AdminDashboard = () => import('@/views/admin/AdminDashboard.vue')
const TenantManagement = () => import('@/views/admin/TenantManagement.vue')

// 租户路由
const ProjectManagement = () => import('@/views/tenant/ProjectManagement.vue')
const TenantUserManagement = () => import('@/views/tenant/UserManagement.vue')

// 系统路由
const SystemLogs = () => import('@/views/tenant/SystemLogs.vue')
const SystemSettings = () => import('@/views/tenant/SystemSettings.vue')
const Profile = () => import('@/views/profile/Profile.vue')

// 路由配置
const routes = [
  {
    path: '/login',
    name: ROUTE_NAMES.LOGIN,
    component: Login,
    meta: {
      title: '登录',
      requiresAuth: false,
    },
  },
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/dashboard',
    name: ROUTE_NAMES.DASHBOARD,
    component: Dashboard,
    meta: {
      title: '仪表板',
      requiresAuth: true,
    },
  },

  // 超级管理员控制台
  {
    path: '/admin',
    component: AdminDashboard,
    meta: {
      title: '超级管理员控制台',
      requiresAuth: true,
      roles: [ROLES.SUPER_ADMIN],
    },
    children: [
      {
        path: '',
        redirect: '/admin/tenants',
      },
      {
        path: 'tenants',
        name: ROUTE_NAMES.TENANT_MANAGEMENT,
        component: TenantManagement,
        meta: {
          title: '租户管理',
          requiresAuth: true,
          roles: [ROLES.SUPER_ADMIN],
        },
      },
    ],
  },

  // 租户管理员路由
  {
    path: '/tenant/users',
    name: 'tenant-user-management',
    component: TenantUserManagement,
    meta: {
      title: '用户管理',
      requiresAuth: true,
      roles: [ROLES.SUPER_ADMIN, ROLES.SYSTEM_ADMIN, ROLES.USER_ADMIN],
    },
  },
  {
    path: '/tenant/projects',
    name: ROUTE_NAMES.PROJECT_MANAGEMENT,
    component: ProjectManagement,
    meta: {
      title: '工程管理',
      requiresAuth: true,
      roles: [ROLES.SUPER_ADMIN, ROLES.SYSTEM_ADMIN, ROLES.PROJECT_ADMIN],
    },
  },

  // 系统功能路由
  {
    path: '/system/logs',
    name: 'system-logs',
    component: SystemLogs,
    meta: {
      title: '系统日志',
      requiresAuth: true,
      roles: [ROLES.SYSTEM_ADMIN, ROLES.OPS_ADMIN],
    },
  },
  {
    path: '/system/settings',
    name: ROUTE_NAMES.SYSTEM_SETTINGS,
    component: SystemSettings,
    meta: {
      title: '系统设置',
      requiresAuth: true,
      roles: [ROLES.SYSTEM_ADMIN],
    },
  },

  // 个人资料路由
  {
    path: '/profile',
    name: ROUTE_NAMES.PROFILE,
    component: Profile,
    meta: {
      title: '个人资料',
      requiresAuth: true,
    },
  },

  // 404 页面
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFound.vue'),
    meta: {
      title: '页面未找到',
      requiresAuth: false,
    },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫
router.beforeEach((to, from, next) => {
  // 设置页面标题
  document.title = `${to.meta.title || '多租户管理系统'} - ProjectIDE`

  // 检查认证
  const token = Storage.getToken()
  const isAuthenticated = !!token

  // 如果需要认证但未登录，重定向到登录页
  if (to.meta.requiresAuth && !isAuthenticated) {
    next({ name: ROUTE_NAMES.LOGIN })
    return
  }

  // 如果已登录且访问登录页，根据角色重定向
  if (to.name === ROUTE_NAMES.LOGIN && isAuthenticated) {
    const userInfo = Storage.getUserInfo()
    if (userInfo?.role === ROLES.SUPER_ADMIN) {
      next('/admin')
    } else {
      next({ name: ROUTE_NAMES.DASHBOARD })
    }
    return
  }

  // 检查角色权限
  if (to.meta.roles && to.meta.roles.length > 0) {
    const userInfo = Storage.getUserInfo()
    const userRole = userInfo?.role

    if (!hasRole(userRole, to.meta.roles)) {
      // 权限不足，重定向到仪表板
      next({ name: ROUTE_NAMES.DASHBOARD })
      return
    }
  }

  // 超级管理员访问限制：不允许访问普通dashboard，必须访问管理员控制台
  if (isAuthenticated) {
    const userInfo = Storage.getUserInfo()
    const userRole = userInfo?.role

    if (userRole === ROLES.SUPER_ADMIN) {
      // 超级管理员不能访问普通dashboard
      if (to.name === ROUTE_NAMES.DASHBOARD) {
        next('/admin')
        return
      }

      // 超级管理员只能访问租户管理页面
      if (to.path.startsWith('/admin') && to.path !== '/admin/tenants' && to.path !== '/admin') {
        next('/admin/tenants')
        return
      }
    }
  }

  next()
})

export default router
