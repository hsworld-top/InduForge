<template>
  <div class="tm-page">
    <!-- 页头 -->
    <div class="tm-header">
      <div>
        <h1 class="tm-title">{{ t("tenantManagement.title") }}</h1>
        <p class="tm-desc">
          {{ t("tenantManagement.list") }} ·
          {{ t("tenantManagement.companyInfo") }} ·
          {{ t("tenantManagement.brandAssets") }}
        </p>
      </div>
      <button
        class="tm-btn tm-btn-primary"
        type="button"
        @click="openAddDialog"
      >
        <svg
          width="14"
          height="14"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2.5"
        >
          <line x1="12" y1="5" x2="12" y2="19" />
          <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
        {{ t("tenantManagement.addTenant") }}
      </button>
    </div>

    <!-- 统计 -->
    <div class="tm-stats-bar">
      <div class="tm-stat">
        <span class="tm-stat-val">{{ tenantStats.total }}</span>
        <span class="tm-stat-txt">{{ t("tenantManagement.allStatus") }}</span>
      </div>
      <i class="tm-stat-sep"></i>
      <div class="tm-stat">
        <span class="tm-stat-val c-green">{{ tenantStats.active }}</span>
        <span class="tm-stat-txt">{{
          t("tenantManagement.statusActive")
        }}</span>
      </div>
      <i class="tm-stat-sep"></i>
      <div class="tm-stat">
        <span class="tm-stat-val c-orange">{{ tenantStats.inactive }}</span>
        <span class="tm-stat-txt">{{
          t("tenantManagement.statusInactive")
        }}</span>
      </div>
      <i class="tm-stat-sep"></i>
      <div class="tm-stat">
        <span class="tm-stat-val c-red">{{ tenantStats.suspended }}</span>
        <span class="tm-stat-txt">{{
          t("tenantManagement.statusSuspended")
        }}</span>
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="tm-toolbar">
      <div class="tm-search">
        <svg
          class="tm-search-ico"
          width="15"
          height="15"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <circle cx="11" cy="11" r="8" />
          <line x1="21" y1="21" x2="16.65" y2="16.65" />
        </svg>
        <input
          v-model="filters.search"
          @input="debouncedSearch"
          type="text"
          :placeholder="t('tenantManagement.searchPlaceholder')"
        />
      </div>
      <select
        v-model="filters.status"
        @change="handleStatusChange"
        class="tm-filter-select"
      >
        <option value="">{{ t("tenantManagement.allStatus") }}</option>
        <option value="active">{{ t("tenantManagement.statusActive") }}</option>
        <option value="inactive">
          {{ t("tenantManagement.statusInactive") }}
        </option>
        <option value="suspended">
          {{ t("tenantManagement.statusSuspended") }}
        </option>
      </select>
    </div>

    <!-- 表格 -->
    <div class="tm-card">
      <table class="tm-table">
        <colgroup>
          <col style="min-width: 200px" />
          <col style="min-width: 180px" />
          <col style="width: 100px" />
          <col style="width: 160px" />
          <col style="width: 160px" />
        </colgroup>
        <thead>
          <tr>
            <th>{{ t("tenantManagement.tenantInfo") }}</th>
            <th>{{ t("tenantManagement.contact") }}</th>
            <th>{{ t("tenantManagement.status") }}</th>
            <th>{{ t("tenantManagement.createdAt") }}</th>
            <th style="text-align: right">
              {{ t("tenantManagement.actions") }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="5" class="tm-table-empty">
              <span class="tm-dots"><i /><i /><i /></span>
            </td>
          </tr>
          <tr v-else-if="!tenants.length">
            <td colspan="5" class="tm-table-empty">{{ t("common.noData") }}</td>
          </tr>
          <template v-else>
            <tr v-for="tenant in tenants" :key="tenant.id">
              <td>
                <div class="tm-cell-tenant">
                  <img
                    class="tm-avatar"
                    :src="tenant.logoUrl || defaultLogoUrl"
                    :alt="tenant.name"
                  />
                  <div>
                    <p class="tm-name">{{ tenant.name }}</p>
                    <p class="tm-code">{{ tenant.code }}</p>
                  </div>
                </div>
              </td>
              <td>
                <p class="tm-contact-email">{{ tenant.contactEmail || "—" }}</p>
                <p class="tm-contact-phone">{{ tenant.contactPhone || "" }}</p>
              </td>
              <td>
                <span :class="['tm-tag', 'tm-tag-' + tenant.status]">
                  {{
                    tenant.status === "active"
                      ? t("tenantManagement.statusActive")
                      : tenant.status === "inactive"
                        ? t("tenantManagement.statusInactive")
                        : t("tenantManagement.statusSuspended")
                  }}
                </span>
              </td>
              <td class="tm-cell-date">{{ formatDate(tenant.createdAt) }}</td>
              <td style="text-align: right">
                <div class="tm-cell-ops">
                  <button
                    class="tm-op"
                    @click="editTenant(tenant)"
                    :title="t('tenantManagement.edit')"
                  >
                    <svg
                      width="14"
                      height="14"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <path
                        d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"
                      />
                      <path
                        d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"
                      />
                    </svg>
                  </button>
                  <button
                    class="tm-op tm-op-del"
                    @click="deleteTenant(tenant)"
                    :title="t('tenantManagement.delete')"
                  >
                    <svg
                      width="14"
                      height="14"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <polyline points="3 6 5 6 21 6" />
                      <path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6" />
                      <path d="M10 11v6" />
                      <path d="M14 11v6" />
                      <path d="M9 6V4a1 1 0 011-1h4a1 1 0 011 1v2" />
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <!-- 分页 -->
    <div class="tm-pager" v-if="pagination.total > 0">
      <span class="tm-pager-text">
        {{
          t("tenantManagement.pageSummary", {
            start: (pagination.page - 1) * pagination.limit + 1,
            end: Math.min(pagination.page * pagination.limit, pagination.total),
            total: pagination.total,
          })
        }}
      </span>
      <div class="tm-pager-nav">
        <button
          type="button"
          :disabled="pagination.page <= 1"
          @click="changePage(pagination.page - 1)"
        >
          <svg
            width="14"
            height="14"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <polyline points="15 18 9 12 15 6" />
          </svg>
        </button>
        <span>{{ pagination.page }}/{{ pagination.totalPages }}</span>
        <button
          type="button"
          :disabled="pagination.page >= pagination.totalPages"
          @click="changePage(pagination.page + 1)"
        >
          <svg
            width="14"
            height="14"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <polyline points="9 18 15 12 9 6" />
          </svg>
        </button>
      </div>
    </div>

    <!-- 遮罩 -->
    <transition name="fade">
      <div v-if="drawerVisible" class="tm-mask" @click.self="closeDialog"></div>
    </transition>

    <!-- 抽屉 -->
    <transition name="slide">
      <aside v-if="drawerVisible" class="tm-drawer">
        <header class="tm-drawer-head">
          <div>
            <h2>
              {{
                showAddDialog
                  ? t("tenantManagement.addDialogTitle")
                  : t("tenantManagement.editDialogTitle")
              }}
            </h2>
            <p>
              {{ t("tenantManagement.tenantCode") }} /
              {{ t("tenantManagement.companyInfo") }} /
              {{ t("tenantManagement.brandAssets") }}
            </p>
          </div>
          <button class="tm-drawer-x" type="button" @click="closeDialog">
            <svg
              width="18"
              height="18"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </header>

        <nav class="tm-tabs">
          <button
            type="button"
            v-for="t2 in tabs"
            :key="t2.k"
            :class="{ active: curTab === t2.k }"
            @click="curTab = t2.k"
          >
            {{ t2.l }}
          </button>
        </nav>

        <div class="tm-drawer-body">
          <!-- ====== 基本信息 ====== -->
          <section v-show="curTab === 'basic'">
            <div class="fm-row">
              <div class="fm-col">
                <label class="fm-lbl"
                  >{{ t("tenantManagement.tenantName") }} <em>*</em></label
                >
                <input v-model="tenantForm.name" required class="fm-input" />
              </div>
              <div class="fm-col">
                <label class="fm-lbl"
                  >{{ t("tenantManagement.tenantCode") }} <em>*</em></label
                >
                <input
                  v-model="tenantForm.code"
                  required
                  :disabled="showEditDialog"
                  class="fm-input"
                />
              </div>
            </div>
            <label class="fm-lbl mt16">{{
              t("tenantManagement.description")
            }}</label>
            <textarea
              v-model="tenantForm.description"
              rows="3"
              class="fm-input fm-ta"
            ></textarea>

            <div class="fm-row mt16">
              <div class="fm-col">
                <label class="fm-lbl">{{
                  t("tenantManagement.contactEmail")
                }}</label>
                <input
                  v-model="tenantForm.contactEmail"
                  type="email"
                  class="fm-input"
                  placeholder="name@example.com"
                />
              </div>
              <div class="fm-col">
                <label class="fm-lbl">{{
                  t("tenantManagement.contactPhone")
                }}</label>
                <input v-model="tenantForm.contactPhone" class="fm-input" />
              </div>
            </div>

            <div class="fm-row mt16">
              <div class="fm-col">
                <label class="fm-lbl">{{
                  t("tenantManagement.maxUsers")
                }}</label>
                <input
                  v-model.number="tenantForm.maxUsers"
                  type="number"
                  min="1"
                  class="fm-input"
                />
              </div>
              <div class="fm-col">
                <label class="fm-lbl">{{
                  t("tenantManagement.maxProjects")
                }}</label>
                <input
                  v-model.number="tenantForm.maxProjects"
                  type="number"
                  min="1"
                  class="fm-input"
                />
              </div>
            </div>

            <div v-if="showEditDialog" class="mt16">
              <label class="fm-lbl">{{ t("tenantManagement.status") }}</label>
              <select v-model="tenantForm.status" class="fm-input fm-sel">
                <option value="active">
                  {{ t("tenantManagement.statusActive") }}
                </option>
                <option value="inactive">
                  {{ t("tenantManagement.statusInactive") }}
                </option>
                <option value="suspended">
                  {{ t("tenantManagement.statusSuspended") }}
                </option>
              </select>
            </div>
          </section>

          <!-- ====== 公司信息（含备案号） ====== -->
          <section v-show="curTab === 'company'">
            <div class="fm-row">
              <div class="fm-col">
                <label class="fm-lbl">{{
                  t("tenantManagement.companyName")
                }}</label>
                <input v-model="tenantForm.companyName" class="fm-input" />
              </div>
              <div class="fm-col">
                <label class="fm-lbl">{{
                  t("tenantManagement.companyPhone")
                }}</label>
                <input v-model="tenantForm.companyPhone" class="fm-input" />
              </div>
            </div>

            <label class="fm-lbl mt16">{{
              t("tenantManagement.companyAddress")
            }}</label>
            <textarea
              v-model="tenantForm.companyAddress"
              rows="2"
              class="fm-input fm-ta"
            ></textarea>

            <div class="fm-row mt16">
              <div class="fm-col">
                <label class="fm-lbl">{{
                  t("tenantManagement.companyWebsite")
                }}</label>
                <input
                  v-model="tenantForm.companyWebsite"
                  type="url"
                  class="fm-input"
                  placeholder="https://"
                />
              </div>
              <div class="fm-col">
                <label class="fm-lbl">{{
                  t("tenantManagement.icpNumber")
                }}</label>
                <input
                  v-model="tenantForm.icpNumber"
                  class="fm-input"
                  :placeholder="t('tenantManagement.icpPlaceholder')"
                />
              </div>
            </div>

            <div class="fm-sep"></div>
            <p class="fm-group-title">
              {{ t("tenantManagement.loginDisplaySettings") }}
            </p>
            <div class="fm-checks">
              <label
                ><input
                  v-model="tenantForm.showCompanyName"
                  type="checkbox"
                /><span>{{
                  t("tenantManagement.showCompanyName")
                }}</span></label
              >
              <label
                ><input
                  v-model="tenantForm.showCompanyPhone"
                  type="checkbox"
                /><span>{{
                  t("tenantManagement.showCompanyPhone")
                }}</span></label
              >
              <label
                ><input
                  v-model="tenantForm.showCompanyAddress"
                  type="checkbox"
                /><span>{{
                  t("tenantManagement.showCompanyAddress")
                }}</span></label
              >
              <label
                ><input
                  v-model="tenantForm.showCompanyWebsite"
                  type="checkbox"
                /><span>{{
                  t("tenantManagement.showCompanyWebsite")
                }}</span></label
              >
              <label
                ><input v-model="tenantForm.showIcp" type="checkbox" /><span>{{
                  t("tenantManagement.showIcp")
                }}</span></label
              >
            </div>
          </section>

          <!-- ====== 品牌资产 ====== -->
          <section v-show="curTab === 'brand'">
            <div class="fm-upload-item">
              <p class="fm-upload-name">{{ t("tenantManagement.logo") }}</p>
              <p class="fm-upload-hint">推荐 200×200，PNG / JPG / SVG</p>
              <el-upload
                class="tm-upload-card"
                list-type="picture-card"
                :auto-upload="true"
                accept="image/*"
                :file-list="logoFileList"
                :before-upload="beforeImageUpload"
                :http-request="uploadLogoRequest"
                :on-preview="handleUploadPreview"
                :on-remove="removeLogo"
              >
                <span class="tm-upload-plus">+</span>
              </el-upload>
            </div>

            <div class="fm-sep"></div>

            <div class="fm-upload-item">
              <p class="fm-upload-name">
                {{ t("tenantManagement.loginBackground") }}
              </p>
              <p class="fm-upload-hint">推荐 1920×1080，JPG 或 PNG</p>
              <el-upload
                class="tm-upload-card tm-upload-card-bg"
                list-type="picture-card"
                :auto-upload="true"
                accept="image/*"
                :file-list="backgroundFileList"
                :before-upload="beforeImageUpload"
                :http-request="uploadBackgroundRequest"
                :on-preview="handleUploadPreview"
                :on-remove="removeBackground"
              >
                <span class="tm-upload-plus">+</span>
              </el-upload>
            </div>

            <div class="fm-preview-card">
              <img :src="logoPreviewUrl" alt="" class="fm-preview-logo" />
              <div>
                <p class="fm-preview-name">
                  {{ tenantForm.name || t("tenantManagement.tenantName") }}
                </p>
                <p class="fm-preview-code">
                  {{ tenantForm.code || "tenant-code" }}
                </p>
              </div>
            </div>
          </section>
        </div>

        <footer class="tm-drawer-foot">
          <button type="button" class="tm-btn" @click="closeDialog">
            {{ t("tenantManagement.cancel") }}
          </button>
          <button
            type="button"
            class="tm-btn tm-btn-primary"
            :disabled="saving"
            @click="saveTenant"
          >
            {{
              saving ? t("tenantManagement.saving") : t("tenantManagement.save")
            }}
          </button>
        </footer>
      </aside>
    </transition>

    <el-dialog
      v-model="previewVisible"
      width="680px"
      :title="t('tenantManagement.preview')"
      append-to-body
    >
      <div class="tm-preview-dialog">
        <img :src="previewImageUrl" alt="preview" />
      </div>
    </el-dialog>
  </div>
</template>

<script lang="ts">
// @ts-nocheck
import { ref, reactive, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { ElMessage, ElMessageBox } from "element-plus";
import dayjs from "dayjs";
import { tenantAPI } from "@/api";
import { getApiErrorMessage } from "@/utils/request";
import { TIME_FORMAT } from "@/constants";
import defaultLogoUrl from "@/assets/images/default-logo.svg";
import defaultLoginBgUrl from "@/assets/images/default-login-bg.svg";

export default {
  name: "TenantManagement",
  setup() {
    const { t } = useI18n();
    const tenants = ref([]);
    const loading = ref(false);
    const saving = ref(false);
    const curTab = ref("basic");

    const tabs = computed(() => [
      { k: "basic", l: t("tenantManagement.tenantInfo") },
      { k: "company", l: t("tenantManagement.companyInfo") },
      { k: "brand", l: t("tenantManagement.brandAssets") },
    ]);

    const showAddDialog = ref(false);
    const showEditDialog = ref(false);
    const drawerVisible = computed(
      () => showAddDialog.value || showEditDialog.value,
    );

    const pagination = reactive({
      page: 1,
      limit: 10,
      total: 0,
      totalPages: 0,
    });
    const filters = reactive({ status: "", search: "" });

    const tenantForm = reactive({
      name: "",
      code: "",
      description: "",
      contactEmail: "",
      contactPhone: "",
      maxUsers: 100,
      maxProjects: 50,
      status: "active",
      logoUrl: "",
      loginBackgroundUrl: "",
      companyName: "",
      companyAddress: "",
      companyPhone: "",
      companyWebsite: "",
      showCompanyName: true,
      showCompanyPhone: false,
      showCompanyAddress: false,
      showCompanyWebsite: false,
      showIcp: false,
      icpNumber: "",
    });

    const previewVisible = ref(false);
    const previewImageUrl = ref("");
    const logoPreviewUrl = computed(() => tenantForm.logoUrl || defaultLogoUrl);
    const backgroundPreviewUrl = computed(
      () => tenantForm.loginBackgroundUrl || defaultLoginBgUrl,
    );
    const logoFileList = computed(() =>
      tenantForm.logoUrl
        ? [{ name: "logo", url: tenantForm.logoUrl, status: "success" }]
        : [],
    );
    const backgroundFileList = computed(() =>
      tenantForm.loginBackgroundUrl
        ? [
            {
              name: "background",
              url: tenantForm.loginBackgroundUrl,
              status: "success",
            },
          ]
        : [],
    );

    const tenantStats = computed(() => {
      const l = tenants.value || [];
      return {
        total: l.length,
        active: l.filter((i) => i.status === "active").length,
        inactive: l.filter((i) => i.status === "inactive").length,
        suspended: l.filter((i) => i.status === "suspended").length,
      };
    });

    let searchTimer = null;
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
        const res = await tenantAPI.getTenants({
          page: pagination.page,
          limit: Number(pagination.limit),
          keyword: filters.search?.trim() || undefined,
          status: filters.status || undefined,
        });
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

    const formatDate = (d) => {
      const p = dayjs(d);
      return p.isValid() ? p.format(TIME_FORMAT) : "";
    };
    const changePage = (p) => {
      if (p >= 1 && p <= pagination.totalPages) {
        pagination.page = p;
        loadTenants();
      }
    };

    const openAddDialog = () => {
      curTab.value = "basic";
      showAddDialog.value = true;
    };

    const editTenant = (tenant) => {
      const ld = tenant.settings?.loginDisplay || {};
      Object.assign(tenantForm, {
        name: tenant.name,
        code: tenant.code,
        description: tenant.description || "",
        contactEmail: tenant.contactEmail || "",
        contactPhone: tenant.contactPhone || "",
        maxUsers: tenant.maxUsers,
        maxProjects: tenant.maxProjects,
        status: tenant.status,
        logoUrl: tenant.logoUrl || "",
        loginBackgroundUrl: tenant.loginBackgroundUrl || "",
        companyName: tenant.companyName || "",
        companyAddress: tenant.companyAddress || "",
        companyPhone: tenant.companyPhone || "",
        companyWebsite: tenant.companyWebsite || "",
        showCompanyName: ld.showCompanyName !== false,
        showCompanyPhone: Boolean(ld.showCompanyPhone),
        showCompanyAddress: Boolean(ld.showCompanyAddress),
        showCompanyWebsite: Boolean(ld.showCompanyWebsite),
        showIcp: Boolean(ld.showIcp),
        icpNumber: ld.icpNumber || "",
      });
      curTab.value = "basic";
      showEditDialog.value = true;
    };

    const deleteTenant = async (tenant) => {
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

    const saveTenant = async () => {
      saving.value = true;
      try {
        const payload = {
          ...tenantForm,
          settings: {
            loginDisplay: {
              showCompanyName: Boolean(tenantForm.showCompanyName),
              showCompanyPhone: Boolean(tenantForm.showCompanyPhone),
              showCompanyAddress: Boolean(tenantForm.showCompanyAddress),
              showCompanyWebsite: Boolean(tenantForm.showCompanyWebsite),
              showIcp: Boolean(tenantForm.showIcp),
              icpNumber: tenantForm.icpNumber || "",
            },
          },
        };
        if (showAddDialog.value) {
          await tenantAPI.createTenant(payload);
          ElMessage.success(t("tenantManagement.createSuccess"));
        } else {
          await tenantAPI.updateTenant(tenantForm.code, payload);
          ElMessage.success(t("tenantManagement.updateSuccess"));
        }
        closeDialog();
        loadTenants();
      } catch (e) {
        console.error(e);
        ElMessage.error(
          getApiErrorMessage(e, t("tenantManagement.saveFailed")),
        );
      } finally {
        saving.value = false;
      }
    };

    const resetForm = () => {
      Object.assign(tenantForm, {
        name: "",
        code: "",
        description: "",
        contactEmail: "",
        contactPhone: "",
        maxUsers: 100,
        maxProjects: 50,
        status: "active",
        logoUrl: "",
        loginBackgroundUrl: "",
        companyName: "",
        companyAddress: "",
        companyPhone: "",
        companyWebsite: "",
        showCompanyName: true,
        showCompanyPhone: false,
        showCompanyAddress: false,
        showCompanyWebsite: false,
        showIcp: false,
        icpNumber: "",
      });
    };

    const closeDialog = () => {
      showAddDialog.value = false;
      showEditDialog.value = false;
      resetForm();
      curTab.value = "basic";
    };

    const syncTenantBrandingInList = (updates) => {
      const idx = tenants.value.findIndex((i) => i.code === tenantForm.code);
      if (idx < 0) return;
      tenants.value[idx] = {
        ...tenants.value[idx],
        ...updates,
      };
    };

    const fileToDataUrl = (file) =>
      new Promise((resolve, reject) => {
        const r = new window.FileReader();
        r.onload = () => resolve(r.result);
        r.onerror = () => reject(new Error("读取失败"));
        r.readAsDataURL(file);
      });

    const beforeImageUpload = (file) => {
      const isImage = file.type.startsWith("image/");
      if (!isImage) {
        ElMessage.error(t("tenantManagement.uploadImageOnly"));
        return false;
      }
      const isLt5MB = file.size / 1024 / 1024 < 5;
      if (!isLt5MB) {
        ElMessage.error(t("tenantManagement.uploadSizeLimit"));
        return false;
      }
      return true;
    };

    const uploadLogoRequest = async (options) => {
      const f = options.file;
      try {
        if (showAddDialog.value) {
          tenantForm.logoUrl = await fileToDataUrl(f);
          ElMessage.success(t("tenantManagement.logoUploadSuccess"));
          options.onSuccess?.({}, f);
          return true;
        }
        const fd = new window.FormData();
        fd.append("file", f);
        const res = await tenantAPI.uploadFile(tenantForm.code, "logo", fd);
        tenantForm.logoUrl = res.data.fileUrl;
        syncTenantBrandingInList({ logoUrl: res.data.fileUrl });
        ElMessage.success(t("tenantManagement.logoUploadSuccess"));
        options.onSuccess?.(res, f);
        return true;
      } catch (err) {
        ElMessage.error(
          getApiErrorMessage(err, t("tenantManagement.logoUploadFailed")),
        );
        options.onError?.(err);
        return false;
      }
    };

    const uploadBackgroundRequest = async (options) => {
      const f = options.file;
      try {
        if (showAddDialog.value) {
          tenantForm.loginBackgroundUrl = await fileToDataUrl(f);
          ElMessage.success(t("tenantManagement.backgroundUploadSuccess"));
          options.onSuccess?.({}, f);
          return true;
        }
        const fd = new window.FormData();
        fd.append("file", f);
        const res = await tenantAPI.uploadFile(
          tenantForm.code,
          "background",
          fd,
        );
        tenantForm.loginBackgroundUrl = res.data.fileUrl;
        syncTenantBrandingInList({ loginBackgroundUrl: res.data.fileUrl });
        ElMessage.success(t("tenantManagement.backgroundUploadSuccess"));
        options.onSuccess?.(res, f);
        return true;
      } catch (err) {
        ElMessage.error(
          getApiErrorMessage(err, t("tenantManagement.backgroundUploadFailed")),
        );
        options.onError?.(err);
        return false;
      }
    };

    const removeLogo = () => {
      tenantForm.logoUrl = "";
    };

    const removeBackground = () => {
      tenantForm.loginBackgroundUrl = "";
    };

    const handleUploadPreview = (file) => {
      previewImageUrl.value = file.url || file.response?.data?.fileUrl || "";
      if (!previewImageUrl.value) return;
      previewVisible.value = true;
    };

    onMounted(loadTenants);

    return {
      t,
      tenants,
      loading,
      saving,
      pagination,
      filters,
      tenantForm,
      showAddDialog,
      showEditDialog,
      drawerVisible,
      defaultLogoUrl,
      logoPreviewUrl,
      backgroundPreviewUrl,
      logoFileList,
      backgroundFileList,
      previewVisible,
      previewImageUrl,
      tenantStats,
      curTab,
      tabs,
      loadTenants,
      handleStatusChange,
      formatDate,
      changePage,
      openAddDialog,
      editTenant,
      deleteTenant,
      saveTenant,
      closeDialog,
      beforeImageUpload,
      uploadLogoRequest,
      uploadBackgroundRequest,
      removeLogo,
      removeBackground,
      handleUploadPreview,
      debouncedSearch,
    };
  },
};
</script>

<style scoped>
/* === Reset & Page === */
.tm-page {
  padding: 20px 24px;
  max-width: 1600px;
  margin: 0 auto;
}

/* === Header === */
.tm-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 18px;
}

.tm-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.tm-desc {
  margin: 3px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

/* === Stats === */
.tm-stats-bar {
  display: inline-flex;
  align-items: center;
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  background: var(--el-bg-color);
  margin-bottom: 14px;
  padding: 8px 0;
  overflow-x: auto;
  max-width: 100%;
}

.tm-stat {
  padding: 0 18px;
  text-align: center;
}

.tm-stat-val {
  display: block;
  font-size: 20px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.1;
}

.tm-stat-txt {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
  display: block;
}

.tm-stat-sep {
  display: block;
  width: 1px;
  height: 28px;
  background: var(--el-border-color-lighter);
}

.c-green {
  color: #16a34a;
}

.c-orange {
  color: #d97706;
}

.c-red {
  color: #dc2626;
}

/* === Toolbar === */
.tm-toolbar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}

.tm-search {
  position: relative;
}

.tm-search-ico {
  position: absolute;
  left: 9px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--el-text-color-placeholder);
  pointer-events: none;
}

.tm-search input {
  height: 32px;
  width: clamp(220px, 32vw, 320px);
  padding: 0 10px 0 30px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  font-size: 13px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  outline: none;
}

.tm-search input:focus {
  border-color: var(--el-color-primary);
}

.tm-filter-select {
  height: 32px;
  min-width: 130px;
  padding: 0 8px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  font-size: 13px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  outline: none;
  cursor: pointer;
}

/* === Card / Table === */
.tm-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  background: var(--el-bg-color);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  overflow-x: auto;
}

.tm-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.tm-table thead {
  background: var(--el-fill-color-light);
  position: sticky;
  top: 0;
  z-index: 1;
}

.tm-table th {
  padding: 9px 14px;
  text-align: left;
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  text-transform: uppercase;
  letter-spacing: 0.03em;
  white-space: nowrap;
  border-bottom: 1px solid var(--el-border-color-light);
}

.tm-table tbody tr {
  border-bottom: 1px solid var(--el-border-color-extra-light);
  transition: background 0.1s;
}

.tm-table tbody tr:last-child {
  border-bottom: none;
}

.tm-table tbody tr:hover {
  background: var(--el-fill-color-lighter);
}

.tm-table td {
  padding: 10px 14px;
  vertical-align: middle;
}

.tm-cell-tenant {
  display: flex;
  align-items: center;
  gap: 10px;
}

.tm-avatar {
  width: 34px;
  height: 34px;
  border-radius: 4px;
  object-fit: cover;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color);
}

.tm-name {
  margin: 0;
  font-weight: 500;
  color: var(--el-text-color-primary);
  line-height: 1.3;
}

.tm-code {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.tm-contact-email {
  margin: 0;
  color: var(--el-text-color-primary);
}

.tm-contact-phone {
  margin: 1px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.tm-cell-date {
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

/* Tag */
.tm-tag {
  display: inline-block;
  padding: 1px 8px;
  border-radius: 3px;
  font-size: 12px;
  font-weight: 500;
  line-height: 20px;
  white-space: nowrap;
}

.tm-tag-active {
  background: #f0fdf4;
  color: #15803d;
}

.tm-tag-inactive {
  background: #fffbeb;
  color: #b45309;
}

.tm-tag-suspended {
  background: #fef2f2;
  color: #b91c1c;
}

/* Ops */
.tm-cell-ops {
  display: flex;
  justify-content: flex-end;
  gap: 4px;
}

.tm-op {
  width: 30px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: none;
  color: var(--el-text-color-regular);
  cursor: pointer;
  transition: all 0.12s;
  backdrop-filter: blur(2px);
}

.tm-op:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.tm-op-del:hover {
  border-color: #f56c6c;
  color: #f56c6c;
  background: #fef0f0;
}

/* Empty / Loading */
.tm-table-empty {
  padding: 36px 0 !important;
  text-align: center;
  color: var(--el-text-color-placeholder);
}

.tm-dots {
  display: inline-flex;
  gap: 4px;
}

.tm-dots i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--el-color-primary);
  animation: dot-b 1s infinite ease-in-out;
}

.tm-dots i:nth-child(2) {
  animation-delay: 0.15s;
}

.tm-dots i:nth-child(3) {
  animation-delay: 0.3s;
}

@keyframes dot-b {
  0%,
  80%,
  100% {
    transform: scale(0.6);
    opacity: 0.4;
  }

  40% {
    transform: scale(1);
    opacity: 1;
  }
}

/* === Pager === */
.tm-pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 2px 0;
}

.tm-pager-text {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.tm-pager-nav {
  display: flex;
  align-items: center;
  gap: 6px;
}

.tm-pager-nav span {
  font-size: 13px;
  color: var(--el-text-color-regular);
  min-width: 50px;
  text-align: center;
}

.tm-pager-nav button {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  cursor: pointer;
}

.tm-pager-nav button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.tm-pager-nav button:not(:disabled):hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

/* === Buttons === */
.tm-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 32px;
  padding: 0 14px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  font-size: 13px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  cursor: pointer;
  transition: all 0.12s;
  white-space: nowrap;
}

.tm-btn:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

.tm-btn-primary {
  background: linear-gradient(
    135deg,
    var(--el-color-primary),
    var(--el-color-primary-dark-2)
  );
  color: #fff;
  border-color: transparent;
  box-shadow: 0 6px 16px rgba(59, 130, 246, 0.24);
}

.tm-btn-primary:hover {
  background: var(--el-color-primary-dark-2);
  border-color: var(--el-color-primary-dark-2);
  color: #fff;
}

.tm-btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.tm-btn-sm {
  height: 28px;
  padding: 0 10px;
  font-size: 12px;
}

/* === Mask & Drawer === */
.tm-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.22);
  z-index: 1000;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.tm-drawer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  width: min(560px, 100vw);
  z-index: 1001;
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color);
  display: flex;
  flex-direction: column;
  box-shadow: -2px 0 12px rgba(0, 0, 0, 0.06);
}

.slide-enter-active,
.slide-leave-active {
  transition: transform 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
}

.tm-drawer-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 16px 20px 12px;
  border-bottom: 1px solid var(--el-border-color-light);
}

.tm-drawer-head h2 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.tm-drawer-head p {
  margin: 3px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.tm-drawer-x {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: none;
  color: var(--el-text-color-secondary);
  border-radius: 4px;
  cursor: pointer;
}

.tm-drawer-x:hover {
  background: var(--el-fill-color);
  color: var(--el-text-color-primary);
}

/* Tabs */
.tm-tabs {
  display: flex;
  padding: 0 20px;
  border-bottom: 1px solid var(--el-border-color-light);
}

.tm-tabs button {
  padding: 10px 14px;
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  cursor: pointer;
  transition: color 0.12s;
}

.tm-tabs button:hover {
  color: var(--el-text-color-primary);
}

.tm-tabs button.active {
  color: var(--el-color-primary);
  border-bottom-color: var(--el-color-primary);
}

/* Drawer body */
.tm-drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  overscroll-behavior: contain;
}

.tm-drawer-body::-webkit-scrollbar {
  width: 4px;
}

.tm-drawer-body::-webkit-scrollbar-thumb {
  background: var(--el-border-color);
  border-radius: 2px;
}

/* Footer */
.tm-drawer-foot {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--el-border-color-light);
}

/* === Form inside drawer === */
.fm-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.fm-col {
  display: flex;
  flex-direction: column;
}

.fm-lbl {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-regular);
  margin-bottom: 5px;
}

.fm-lbl em {
  color: #f56c6c;
  font-style: normal;
}

.fm-input {
  height: 32px;
  padding: 0 9px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  font-size: 13px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  outline: none;
  transition: border-color 0.15s;
}

.fm-input:focus {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.14);
}

.fm-input:disabled {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-placeholder);
  cursor: not-allowed;
}

.fm-ta {
  height: auto;
  padding: 7px 9px;
  resize: vertical;
  line-height: 1.5;
}

.fm-sel {
  cursor: pointer;
}

.mt16 {
  margin-top: 16px;
}

.fm-sep {
  height: 1px;
  background: var(--el-border-color-extra-light);
  margin: 18px 0;
}

.fm-group-title {
  margin: 0 0 10px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

/* Checkboxes */
.fm-checks {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
}

.fm-checks label {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  padding: 5px 6px;
  border-radius: 6px;
  cursor: pointer;
}

.fm-checks label:hover {
  background: var(--el-fill-color-lighter);
}

.fm-checks input {
  width: 14px;
  height: 14px;
  accent-color: var(--el-color-primary);
  cursor: pointer;
}

/* Upload */
.fm-upload {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.fm-upload-item {
  min-width: 0;
}

:deep(.tm-upload-card .el-upload--picture-card),
:deep(.tm-upload-card .el-upload-list__item) {
  width: 112px;
  height: 112px;
  border-radius: 8px;
}

:deep(.tm-upload-card-bg .el-upload--picture-card),
:deep(.tm-upload-card-bg .el-upload-list__item) {
  width: 170px;
  height: 112px;
}

:deep(.tm-upload-card .el-upload--picture-card) {
  border-color: var(--el-border-color);
  background: var(--el-fill-color-lighter);
}

:deep(.tm-upload-card .el-upload--picture-card:hover) {
  border-color: var(--el-color-primary);
}

.tm-upload-plus {
  font-size: 28px;
  line-height: 1;
  color: var(--el-text-color-placeholder);
}

.tm-preview-dialog {
  display: flex;
  justify-content: center;
}

.tm-preview-dialog img {
  max-width: 100%;
  max-height: 70vh;
  border-radius: 8px;
}

.fm-upload-thumb {
  flex-shrink: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  overflow: hidden;
  background: var(--el-fill-color);
}

.fm-upload-thumb.sq {
  width: 64px;
  height: 64px;
}

.fm-upload-thumb.wide {
  width: 100px;
  height: 64px;
}

.fm-upload-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.fm-upload-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.fm-upload-name {
  margin: 0;
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.fm-upload-hint {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.fm-upload-body .tm-btn-sm {
  margin-top: 4px;
}

/* Brand preview */
.fm-preview-card {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 18px;
  padding: 12px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.fm-preview-logo {
  width: 32px;
  height: 32px;
  border-radius: 4px;
  object-fit: cover;
  border: 1px solid var(--el-border-color-lighter);
}

.fm-preview-name {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.fm-preview-code {
  margin: 1px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

@media (max-width: 1024px) {
  .tm-pager {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .tm-drawer {
    width: min(100vw, 720px);
  }
}

@media (max-width: 768px) {
  .tm-page {
    padding: 14px 12px;
  }

  .tm-title {
    font-size: 17px;
  }

  .tm-desc {
    font-size: 12px;
  }

  .tm-stats-bar {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    padding: 6px;
    gap: 6px;
  }

  .tm-stat-sep {
    display: none;
  }

  .tm-stat {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    padding: 8px 10px;
  }

  .tm-search {
    flex: 1 1 100%;
  }

  .tm-search input {
    width: 100%;
  }

  .tm-filter-select {
    width: 100%;
  }

  .tm-drawer {
    width: 100vw;
  }

  .tm-drawer-head,
  .tm-drawer-foot,
  .tm-tabs,
  .tm-drawer-body {
    padding-left: 14px;
    padding-right: 14px;
  }

  .fm-row {
    grid-template-columns: 1fr;
  }

  .fm-checks {
    grid-template-columns: 1fr;
  }
}
</style>

<style>
html.dark .tm-page .tm-stats-bar,
html.dark .tm-page .tm-card,
html.dark .tm-page .tm-drawer,
html.dark .tm-page .tm-search input,
html.dark .tm-page .tm-filter-select,
html.dark .tm-page .fm-input,
html.dark .tm-page .fm-preview-card,
[data-theme="dark"] .tm-page .tm-stats-bar,
[data-theme="dark"] .tm-page .tm-card,
[data-theme="dark"] .tm-page .tm-drawer,
[data-theme="dark"] .tm-page .tm-search input,
[data-theme="dark"] .tm-page .tm-filter-select,
[data-theme="dark"] .tm-page .fm-input,
[data-theme="dark"] .tm-page .fm-preview-card {
  background: #111827 !important;
  border-color: #374151 !important;
  color: #e5e7eb !important;
}

html.dark .tm-page .tm-table thead,
[data-theme="dark"] .tm-page .tm-table thead {
  background: #1f2937 !important;
}

html.dark .tm-page .tm-table th,
[data-theme="dark"] .tm-page .tm-table th {
  color: #9ca3af !important;
  border-bottom-color: #374151 !important;
}

html.dark .tm-page .tm-table tbody tr,
[data-theme="dark"] .tm-page .tm-table tbody tr {
  border-bottom-color: #2d3748 !important;
}

html.dark .tm-page .tm-table tbody tr:hover,
[data-theme="dark"] .tm-page .tm-table tbody tr:hover {
  background: #1b2535 !important;
}

html.dark .tm-page .tm-drawer-head,
html.dark .tm-page .tm-tabs,
html.dark .tm-page .tm-drawer-foot,
[data-theme="dark"] .tm-page .tm-drawer-head,
[data-theme="dark"] .tm-page .tm-tabs,
[data-theme="dark"] .tm-page .tm-drawer-foot {
  border-color: #374151 !important;
}

html.dark .tm-page .tm-title,
html.dark .tm-page .tm-name,
html.dark .tm-page .fm-upload-name,
html.dark .tm-page .fm-preview-name,
[data-theme="dark"] .tm-page .tm-title,
[data-theme="dark"] .tm-page .tm-name,
[data-theme="dark"] .tm-page .fm-upload-name,
[data-theme="dark"] .tm-page .fm-preview-name {
  color: #f3f4f6 !important;
}

html.dark .tm-page .tm-desc,
html.dark .tm-page .tm-code,
html.dark .tm-page .tm-contact-phone,
html.dark .tm-page .fm-upload-hint,
html.dark .tm-page .fm-preview-code,
[data-theme="dark"] .tm-page .tm-desc,
[data-theme="dark"] .tm-page .tm-code,
[data-theme="dark"] .tm-page .tm-contact-phone,
[data-theme="dark"] .tm-page .fm-upload-hint,
[data-theme="dark"] .tm-page .fm-preview-code {
  color: #9ca3af !important;
}

html.dark .tm-page .tm-mask,
[data-theme="dark"] .tm-page .tm-mask {
  background: rgba(2, 6, 23, 0.62) !important;
}
</style>



