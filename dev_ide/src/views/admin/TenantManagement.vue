<template>
  <div class="h-full flex flex-col p-4 md:p-6 max-w-[1600px] mx-auto w-full font-sans antialiased">
    <!-- 页头 -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-5 md:mb-6 lg:mb-8">
      <div>
        <h1 class="text-2xl md:text-3xl font-extrabold text-slate-900 dark:text-white tracking-tight">{{ t("tenantManagement.title") }}</h1>
        <p class="text-xs md:text-sm font-medium text-slate-500 dark:text-slate-400 mt-1 md:mt-2">
          {{ t("tenantManagement.list") }} · {{ t("tenantManagement.companyInfo") }} · {{ t("tenantManagement.brandAssets") }}
        </p>
      </div>
      <button
        class="group inline-flex items-center justify-center gap-2 px-4 py-2 md:px-5 md:py-2.5 text-sm font-semibold text-white bg-blue-600 rounded-xl hover:bg-blue-700 focus:ring-4 focus:ring-blue-500/20 dark:focus:ring-blue-900/40 transition-all shadow-md hover:shadow-lg active:scale-[0.98]"
        type="button"
        @click="openAddDialog"
      >
        <svg class="w-4 h-4 md:w-5 md:h-5 transition-transform group-hover:scale-110" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19" />
          <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
        {{ t("tenantManagement.addTenant") }}
      </button>
    </div>



    <!-- 工具栏和表格卡片 -->
    <div class="flex-1 flex flex-col bg-white dark:bg-slate-900 border border-slate-200/60 dark:border-slate-800 rounded-2xl shadow-sm overflow-hidden min-h-0">
      
      <!-- 工具栏 -->
      <div class="flex flex-col sm:flex-row gap-4 p-5 border-b border-slate-200/60 dark:border-slate-800 bg-slate-50/30 dark:bg-slate-900/50">
        <div class="relative flex-1 max-w-md">
          <div class="absolute inset-y-0 left-0 flex items-center pl-4 pointer-events-none">
            <svg class="w-5 h-5 text-slate-400 dark:text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </div>
          <input
            v-model="filters.search"
            @input="debouncedSearch"
            type="text"
            class="block w-full py-2.5 pl-11 pr-4 text-sm font-medium text-slate-900 border border-slate-300 rounded-xl bg-white focus:ring-4 focus:ring-blue-500/10 focus:border-blue-500 dark:bg-slate-950 dark:border-slate-700 dark:placeholder-slate-500 dark:text-white dark:focus:ring-blue-900/20 dark:focus:border-blue-500 outline-none transition-all shadow-sm"
            :placeholder="t('tenantManagement.searchPlaceholder')"
          />
        </div>
        <select
          v-model="filters.status"
          @change="handleStatusChange"
          class="block w-full sm:w-48 py-2.5 px-4 text-sm font-medium text-slate-900 border border-slate-300 rounded-xl bg-white focus:ring-4 focus:ring-blue-500/10 focus:border-blue-500 dark:bg-slate-950 dark:border-slate-700 dark:text-white dark:focus:ring-blue-900/20 dark:focus:border-blue-500 outline-none cursor-pointer transition-all shadow-sm"
        >
          <option value="">{{ t("tenantManagement.allStatus") }}</option>
          <option value="active">{{ t("tenantManagement.statusActive") }}</option>
          <option value="inactive">{{ t("tenantManagement.statusInactive") }}</option>
          <option value="suspended">{{ t("tenantManagement.statusSuspended") }}</option>
        </select>
      </div>

      <!-- 表格内容 -->
      <div class="flex-1 overflow-auto custom-scrollbar">
        <table class="w-full text-sm text-left text-slate-600 dark:text-slate-400">
          <thead class="text-xs text-slate-500 uppercase bg-slate-50/90 dark:bg-slate-900/90 dark:text-slate-400 sticky top-0 z-10 backdrop-blur-md border-b border-slate-200 dark:border-slate-800">
            <tr>
              <th scope="col" class="px-6 py-4 font-bold tracking-wider">{{ t("tenantManagement.tenantInfo") }}</th>
              <th scope="col" class="px-6 py-4 font-bold tracking-wider">{{ t("tenantManagement.contact") }}</th>
              <th scope="col" class="px-6 py-4 font-bold tracking-wider">{{ t("tenantManagement.status") }}</th>
              <th scope="col" class="px-6 py-4 font-bold tracking-wider">{{ t("tenantManagement.createdAt") }}</th>
              <th scope="col" class="px-6 py-4 font-bold tracking-wider text-right">{{ t("tenantManagement.actions") }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100 dark:divide-slate-800/60 bg-white dark:bg-slate-900">
            <tr v-if="loading">
              <td colspan="5" class="px-6 py-12 text-center">
                <div class="flex items-center justify-center space-x-2">
                  <div class="w-2.5 h-2.5 bg-blue-600 rounded-full animate-bounce"></div>
                  <div class="w-2.5 h-2.5 bg-blue-600 rounded-full animate-bounce" style="animation-delay: 0.15s"></div>
                  <div class="w-2.5 h-2.5 bg-blue-600 rounded-full animate-bounce" style="animation-delay: 0.3s"></div>
                </div>
              </td>
            </tr>
            <tr v-else-if="!tenants.length">
              <td colspan="5" class="px-6 py-20 text-center text-slate-500 dark:text-slate-400">
                <div class="flex flex-col items-center justify-center">
                  <svg class="w-16 h-16 mb-4 text-slate-300 dark:text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                  </svg>
                  <span class="font-medium text-base">{{ t("common.noData") }}</span>
                </div>
              </td>
            </tr>
            <tr v-for="tenant in tenants" :key="tenant.id" class="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors group">
              <td class="px-6 py-4">
                <div class="flex items-center gap-4">
                  <img
                    class="w-12 h-12 rounded-xl object-cover border border-slate-200 dark:border-slate-700 bg-white shadow-sm"
                    :src="tenant.logoUrl || defaultLogoUrl"
                    :alt="tenant.name"
                  />
                  <div>
                    <div class="font-bold text-slate-900 dark:text-white">{{ tenant.name }}</div>
                    <div class="text-xs font-medium text-slate-500 dark:text-slate-400 mt-1">{{ tenant.code }}</div>
                  </div>
                </div>
              </td>
              <td class="px-6 py-4">
                <div class="font-medium text-slate-900 dark:text-slate-300">{{ tenant.contactEmail || "—" }}</div>
                <div class="text-xs font-medium text-slate-500 dark:text-slate-400 mt-1">{{ tenant.contactPhone || "" }}</div>
              </td>
              <td class="px-6 py-4">
                <span
                  :class="[
                    'inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-bold border',
                    tenant.status === 'active' ? 'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20' : 
                    tenant.status === 'inactive' ? 'bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20' : 
                    'bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-500/10 dark:text-rose-400 dark:border-rose-500/20'
                  ]"
                >
                  <span class="w-1.5 h-1.5 rounded-full mr-2" :class="[
                    tenant.status === 'active' ? 'bg-emerald-500' : 
                    tenant.status === 'inactive' ? 'bg-amber-500' : 'bg-rose-500'
                  ]"></span>
                  {{
                    tenant.status === "active"
                      ? t("tenantManagement.statusActive")
                      : tenant.status === "inactive"
                        ? t("tenantManagement.statusInactive")
                        : t("tenantManagement.statusSuspended")
                  }}
                </span>
              </td>
              <td class="px-6 py-4 font-medium text-slate-500 dark:text-slate-400 whitespace-nowrap">
                {{ formatDate(tenant.createdAt) }}
              </td>
              <td class="px-6 py-4 text-right">
                <div class="flex items-center justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    class="p-2 text-slate-500 bg-white border border-slate-200 rounded-xl hover:text-blue-600 hover:bg-blue-50 hover:border-blue-200 dark:bg-slate-800 dark:border-slate-700 dark:text-slate-400 dark:hover:text-blue-400 dark:hover:bg-blue-900/20 dark:hover:border-blue-800 transition-all shadow-sm hover:shadow"
                    @click="editTenant(tenant)"
                    :title="t('tenantManagement.edit')"
                  >
                    <svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7" stroke-linecap="round" stroke-linejoin="round" />
                      <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z" stroke-linecap="round" stroke-linejoin="round" />
                    </svg>
                  </button>
                  <button
                    class="p-2 text-slate-500 bg-white border border-slate-200 rounded-xl hover:text-rose-600 hover:bg-rose-50 hover:border-rose-200 dark:bg-slate-800 dark:border-slate-700 dark:text-slate-400 dark:hover:text-rose-400 dark:hover:bg-rose-900/20 dark:hover:border-rose-800 transition-all shadow-sm hover:shadow"
                    @click="deleteTenant(tenant)"
                    :title="t('tenantManagement.delete')"
                  >
                    <svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div class="flex items-center justify-between p-4 border-t border-slate-200/60 dark:border-slate-800 bg-white dark:bg-slate-900" v-if="pagination.total > 0">
        <span class="text-sm font-medium text-slate-500 dark:text-slate-400">
          {{
            t("tenantManagement.pageSummary", {
              start: (pagination.page - 1) * pagination.limit + 1,
              end: Math.min(pagination.page * pagination.limit, pagination.total),
              total: pagination.total,
            })
          }}
        </span>
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="p-1.5 text-slate-500 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-slate-900 dark:border-slate-700 dark:text-slate-400 dark:hover:bg-slate-800 transition-colors shadow-sm"
            :disabled="pagination.page <= 1"
            @click="changePage(pagination.page - 1)"
          >
            <svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <span class="text-sm font-bold text-slate-700 dark:text-slate-300 min-w-[3rem] text-center">
            {{ pagination.page }} / {{ pagination.totalPages }}
          </span>
          <button
            type="button"
            class="p-1.5 text-slate-500 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-slate-900 dark:border-slate-700 dark:text-slate-400 dark:hover:bg-slate-800 transition-colors shadow-sm"
            :disabled="pagination.page >= pagination.totalPages"
            @click="changePage(pagination.page + 1)"
          >
            <svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <TenantEditDrawer 
      v-model:visible="drawerVisible"
      :mode="drawerMode"
      :initial-data="editingTenant"
      @saved="loadTenants"
      @updateListBrand="syncTenantBrandingInList"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { ElMessage, ElMessageBox } from "element-plus";
import dayjs from "dayjs";
import { tenantAPI } from "@/api";
import { TIME_FORMAT } from "@/constants";
import defaultLogoUrl from "@/assets/images/default-logo.svg";

import TenantEditDrawer from './components/TenantEditDrawer.vue'

const { t } = useI18n();
const tenants = ref<any[]>([]);
const loading = ref(false);

const drawerVisible = ref(false);
const drawerMode = ref('add');
const editingTenant = ref<any>({});

const pagination = reactive({
  page: 1,
  limit: 10,
  total: 0,
  totalPages: 0,
});

const filters = reactive({ status: "", search: "" });

let searchTimer: number | null = null;
const debouncedSearch = () => {
  if (searchTimer) {
    window.clearTimeout(searchTimer);
  }
  searchTimer = window.setTimeout(() => {
    pagination.page = 1;
    loadTenants();
  }, 400);
};

const handleStatusChange = () => {
  pagination.page = 1;
  loadTenants();
};

const loadTenants = async () => {
  loading.value = true;
  try {
    const res = (await tenantAPI.getTenants({
      page: pagination.page,
      limit: Number(pagination.limit),
      keyword: filters.search?.trim() || undefined,
      status: filters.status || undefined,
    })) as any;
    tenants.value = res.data.tenants;
    pagination.total = res.pagination.total;
    pagination.totalPages = res.pagination.totalPages;
  } catch (e) {
    console.error("加载租户列表失败:", e);
    ElMessage.error(t("tenantManagement.loadFailed"));
  } finally {
    loading.value = false;
  }
};

const formatDate = (d: string) => {
  const p = dayjs(d);
  return p.isValid() ? p.format(TIME_FORMAT) : "";
};

const changePage = (p: number) => {
  if (p >= 1 && p <= pagination.totalPages) {
    pagination.page = p;
    loadTenants();
  }
};

const openAddDialog = () => {
  drawerMode.value = 'add';
  editingTenant.value = {};
  drawerVisible.value = true;
};

const editTenant = (tenant: any) => {
  drawerMode.value = 'edit';
  editingTenant.value = tenant;
  drawerVisible.value = true;
};

const deleteTenant = async (tenant: any) => {
  try {
    await ElMessageBox.confirm(
      t("tenantManagement.deleteConfirmText", { name: tenant.name }),
      t("tenantManagement.deleteConfirmTitle"),
      {
        confirmButtonText: t("tenantManagement.deleteConfirmButton"),
        cancelButtonText: t("tenantManagement.cancel"),
        type: "warning",
      },
    );
    await tenantAPI.deleteTenant(tenant.id);
    ElMessage.success(t("tenantManagement.deleteSuccess"));
    loadTenants();
  } catch (e) {
    if (e !== "cancel") {
      console.error(e);
      ElMessage.error(t("tenantManagement.deleteFailed"));
    }
  }
};

const syncTenantBrandingInList = (updates: any) => {
  const idx = tenants.value.findIndex((i) => i.code === updates.code);
  if (idx < 0) return;
  tenants.value[idx] = {
    ...tenants.value[idx],
    ...updates,
  };
};

onMounted(loadTenants);
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: #cbd5e1;
  border-radius: 10px;
}
html.dark .custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: #475569;
}
</style>
