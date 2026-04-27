<template>
  <el-dialog
    v-model="dialogVisible"
    :title="
      t('projectManagement.runtimeAccess.dialogTitle', {
        name: project?.name || '',
      })
    "
    width="min(1120px, 96vw)"
    append-to-body
    class="runtime-access-modal"
  >
    <div class="runtime-access-dialog">
      <div class="runtime-access-shell">
        <div class="runtime-access-notice">
          <el-alert
            type="info"
            :closable="false"
            class="runtime-access-alert"
            :title="t('projectManagement.runtimeAccess.publishTip')"
          />
        </div>

        <div v-if="!project" class="runtime-access-empty-state">
          <el-empty :description="t('common.noData')" />
        </div>

        <template v-else>
          <section class="runtime-access-tabs-card">
            <div class="runtime-access-tabs-toolbar">
              <div class="runtime-access-tab-switcher">
                <button
                  type="button"
                  class="runtime-access-tab-button"
                  :class="{ 'is-active': activeTab === RUNTIME_ACCESS_TABS.USERS }"
                  @click="activeTab = RUNTIME_ACCESS_TABS.USERS"
                >
                  {{ t('projectManagement.runtimeAccess.tabs.users') }}
                </button>
                <button
                  type="button"
                  class="runtime-access-tab-button"
                  :class="{ 'is-active': activeTab === RUNTIME_ACCESS_TABS.ROLES }"
                  @click="activeTab = RUNTIME_ACCESS_TABS.ROLES"
                >
                  {{ t('projectManagement.runtimeAccess.tabs.roles') }}
                </button>
              </div>
              <div class="runtime-access-tabs-actions">
                <el-button
                  v-if="activeTab === RUNTIME_ACCESS_TABS.USERS"
                  type="primary"
                  round
                  @click="openCreateUserEditor"
                >
                  {{ t('projectManagement.runtimeAccess.users.create') }}
                </el-button>
                <el-button v-else type="primary" round @click="openCreateRoleEditor">
                  {{ t('projectManagement.runtimeAccess.roles.create') }}
                </el-button>
                <el-button round @click="refreshAll">
                  {{ t('common.refresh') }}
                </el-button>
              </div>
            </div>

            <section v-if="activeTab === RUNTIME_ACCESS_TABS.USERS" class="runtime-access-section">
              <div class="runtime-access-table-wrap">
                <el-table
                  :data="runtimeUsers"
                  v-loading="userLoading"
                  height="420"
                  class="runtime-access-table"
                >
                  <el-table-column
                    prop="username"
                    :label="t('projectManagement.runtimeAccess.users.username')"
                    min-width="180"
                  />
                  <el-table-column
                    prop="displayName"
                    :label="t('projectManagement.runtimeAccess.users.displayName')"
                    min-width="160"
                  />
                  <el-table-column
                    :label="t('projectManagement.runtimeAccess.users.roles')"
                    min-width="220"
                  >
                    <template #default="{ row }">
                      <div v-if="resolveUserRoleNames(row).length > 0" class="runtime-access-tags">
                        <el-tag
                          v-for="roleName in resolveUserRoleNames(row)"
                          :key="roleName"
                          size="small"
                        >
                          {{ roleName }}
                        </el-tag>
                      </div>
                      <span v-else class="runtime-access-empty-text">
                        {{ t('projectManagement.runtimeAccess.users.noRoles') }}
                      </span>
                    </template>
                  </el-table-column>
                  <el-table-column
                    :label="t('projectManagement.runtimeAccess.users.status')"
                    width="120"
                  >
                    <template #default="{ row }">
                      <el-tag :type="resolveRuntimeAccessStatusMeta(row.status).tagType">
                        {{ t(resolveRuntimeAccessStatusMeta(row.status).labelKey) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column
                    :label="t('projectManagement.runtimeAccess.users.actions')"
                    min-width="360"
                    fixed="right"
                  >
                    <template #default="{ row }">
                      <div class="runtime-access-actions">
                        <el-button size="small" @click="openEditUserEditor(row)">
                          {{ t('projectManagement.runtimeAccess.users.bindRoles') }}
                        </el-button>
                        <el-button
                          size="small"
                          :disabled="isDefaultRuntimeAdminUser(row)"
                          @click="toggleRuntimeUserStatus(row)"
                        >
                          {{
                            row.status === 'active'
                              ? t('projectManagement.runtimeAccess.users.disable')
                              : t('projectManagement.runtimeAccess.users.enable')
                          }}
                        </el-button>
                        <el-button
                          size="small"
                          type="warning"
                          @click="resetRuntimeUserPassword(row)"
                        >
                          {{ t('projectManagement.runtimeAccess.users.resetPassword') }}
                        </el-button>
                        <el-button
                          size="small"
                          type="danger"
                          :disabled="isDefaultRuntimeAdminUser(row)"
                          @click="deleteRuntimeUser(row)"
                        >
                          {{ t('projectManagement.runtimeAccess.users.delete') }}
                        </el-button>
                      </div>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </section>

            <section v-else class="runtime-access-section">
              <div class="runtime-access-table-wrap">
                <el-table
                  :data="runtimeRoles"
                  v-loading="roleLoading"
                  height="420"
                  class="runtime-access-table"
                >
                  <el-table-column
                    prop="code"
                    :label="t('projectManagement.runtimeAccess.roles.code')"
                    min-width="180"
                  />
                  <el-table-column
                    prop="name"
                    :label="t('projectManagement.runtimeAccess.roles.name')"
                    min-width="180"
                  >
                    <template #default="{ row }">
                      <div class="runtime-role-name">
                        <span>{{ row.name }}</span>
                        <el-tag v-if="row.isSystem" size="small" type="warning">
                          {{ t('projectManagement.runtimeAccess.roles.systemRole') }}
                        </el-tag>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column
                    prop="description"
                    :label="t('projectManagement.runtimeAccess.roles.description')"
                    min-width="220"
                  />
                  <el-table-column
                    :label="t('projectManagement.runtimeAccess.roles.status')"
                    width="120"
                  >
                    <template #default="{ row }">
                      <el-tag :type="resolveRuntimeAccessStatusMeta(row.status).tagType">
                        {{ t(resolveRuntimeAccessStatusMeta(row.status).labelKey) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column
                    prop="bindingCount"
                    :label="t('projectManagement.runtimeAccess.roles.bindingCount')"
                    width="110"
                  />
                  <el-table-column
                    :label="t('projectManagement.runtimeAccess.roles.actions')"
                    min-width="220"
                    fixed="right"
                  >
                    <template #default="{ row }">
                      <div class="runtime-access-actions">
                        <el-button size="small" @click="openEditRoleEditor(row)">
                          {{ t('common.edit') }}
                        </el-button>
                        <el-button
                          size="small"
                          type="danger"
                          :disabled="row.isSystem"
                          @click="deleteRuntimeRole(row)"
                        >
                          {{ t('common.delete') }}
                        </el-button>
                      </div>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </section>
          </section>
        </template>
      </div>
    </div>
  </el-dialog>

  <el-dialog
    v-model="userEditorVisible"
    :title="
      userEditorMode === 'create'
        ? t('projectManagement.runtimeAccess.users.create')
        : t('projectManagement.runtimeAccess.users.bindRoles')
    "
    width="560px"
    append-to-body
    class="runtime-access-editor-dialog"
  >
    <div class="runtime-access-editor__body">
      <el-form label-width="120px" class="runtime-access-form">
        <el-form-item
          v-if="userEditorMode === 'create'"
          :label="t('projectManagement.runtimeAccess.users.username')"
        >
          <el-input
            v-model="userForm.username"
            :placeholder="t('projectManagement.runtimeAccess.users.inputUsername')"
          />
        </el-form-item>
        <el-form-item
          v-if="userEditorMode === 'create'"
          :label="t('projectManagement.runtimeAccess.users.displayName')"
        >
          <el-input
            v-model="userForm.displayName"
            :placeholder="t('projectManagement.runtimeAccess.users.inputDisplayName')"
          />
        </el-form-item>
        <el-form-item
          v-if="userEditorMode === 'create'"
          :label="t('projectManagement.runtimeAccess.users.initialPassword')"
        >
          <el-input
            v-model="userForm.initialPassword"
            type="password"
            show-password
            :placeholder="t('projectManagement.runtimeAccess.users.inputInitialPassword')"
          />
        </el-form-item>
        <template v-else>
          <el-form-item :label="t('projectManagement.runtimeAccess.users.username')">
            <div class="runtime-access-readonly">
              {{ userForm.username || '--' }}
            </div>
          </el-form-item>
          <el-form-item :label="t('projectManagement.runtimeAccess.users.displayName')">
            <div class="runtime-access-readonly">
              {{ userForm.displayName || '--' }}
            </div>
          </el-form-item>
        </template>
        <el-form-item :label="t('projectManagement.runtimeAccess.users.roles')">
          <el-select
            v-model="userForm.roleIds"
            multiple
            collapse-tags
            collapse-tags-tooltip
            style="width: 100%"
            :placeholder="t('projectManagement.runtimeAccess.users.selectRoles')"
          >
            <el-option
              v-for="option in roleOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
              :disabled="option.disabled"
            />
          </el-select>
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="closeUserEditor">
        {{ t('common.cancel') }}
      </el-button>
      <el-button type="primary" :loading="userSubmitting" @click="submitUserForm">
        {{ t('common.save') }}
      </el-button>
    </template>
  </el-dialog>

  <el-dialog
    v-model="roleEditorVisible"
    :title="
      roleEditorMode === 'create'
        ? t('projectManagement.runtimeAccess.roles.create')
        : t('projectManagement.runtimeAccess.roles.edit')
    "
    width="560px"
    append-to-body
    class="runtime-access-editor-dialog"
  >
    <div class="runtime-access-editor__body">
      <el-form label-width="120px" class="runtime-access-form">
        <el-form-item :label="t('projectManagement.runtimeAccess.roles.code')">
          <el-input
            v-model="roleForm.code"
            :disabled="roleEditorMode === 'edit' && roleForm.isSystem"
            :placeholder="t('projectManagement.runtimeAccess.roles.inputCode')"
          >
            <template #prepend>
              {{ RUNTIME_ROLE_CODE_PREFIX }}
            </template>
          </el-input>
        </el-form-item>
        <el-form-item :label="t('projectManagement.runtimeAccess.roles.name')">
          <el-input
            v-model="roleForm.name"
            :placeholder="t('projectManagement.runtimeAccess.roles.inputName')"
          />
        </el-form-item>
        <el-form-item :label="t('projectManagement.runtimeAccess.roles.description')">
          <el-input
            v-model="roleForm.description"
            type="textarea"
            :rows="3"
            :placeholder="t('projectManagement.runtimeAccess.roles.inputDescription')"
          />
        </el-form-item>
        <el-form-item
          v-if="roleEditorMode === 'edit'"
          :label="t('projectManagement.runtimeAccess.roles.status')"
        >
          <el-select v-model="roleForm.status" style="width: 100%">
            <el-option :label="t('projectManagement.runtimeAccess.statusActive')" value="active" />
            <el-option
              :label="t('projectManagement.runtimeAccess.statusDisabled')"
              value="disabled"
            />
          </el-select>
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="closeRoleEditor">
        {{ t('common.cancel') }}
      </el-button>
      <el-button type="primary" :loading="roleSubmitting" @click="submitRoleForm">
        {{ t('common.save') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, toRef, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

import { projectAPI } from '../../../api/project.api.js'
import {
  RUNTIME_ACCESS_TABS,
  RUNTIME_ROLE_CODE_PREFIX,
  useProjectRuntimeAccessState,
} from './project-runtime-access-state'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  project: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['update:visible'])
const { t } = useI18n()

const dialogVisible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value),
})

const {
  activeTab,
  userLoading,
  roleLoading,
  userSubmitting,
  roleSubmitting,
  runtimeUsers,
  runtimeRoles,
  roleOptions,
  userEditorVisible,
  roleEditorVisible,
  userEditorMode,
  roleEditorMode,
  userForm,
  roleForm,
  refreshAll,
  openCreateUserEditor,
  openEditUserEditor,
  closeUserEditor,
  submitUserForm,
  toggleRuntimeUserStatus,
  deleteRuntimeUser,
  resetRuntimeUserPassword,
  openCreateRoleEditor,
  openEditRoleEditor,
  closeRoleEditor,
  submitRoleForm,
  deleteRuntimeRole,
  resolveUserRoleNames,
  resolveRuntimeAccessStatusMeta,
  isDefaultRuntimeAdminUser,
} = useProjectRuntimeAccessState({
  visibleRef: dialogVisible,
  projectRef: toRef(props, 'project'),
  t,
  api: projectAPI,
  ref,
  reactive,
  computed,
  watch,
  message: ElMessage,
  messageBox: ElMessageBox,
})
</script>

<style scoped>
.runtime-access-dialog {
  min-height: 240px;
}

.runtime-access-shell {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.runtime-access-notice {
  margin-bottom: 2px;
}

.runtime-access-empty-state {
  padding: 28px 0;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.78);
  box-shadow: inset 0 0 0 1px rgba(226, 232, 240, 0.55);
  backdrop-filter: blur(10px);
}

.runtime-access-tabs-card {
  background: transparent;
  box-shadow: none;
  padding: 0;
  backdrop-filter: none;
}

.runtime-access-tabs-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.runtime-access-tabs-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.runtime-access-tab-switcher {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px;
  border-radius: 999px;
  background: rgba(241, 245, 249, 0.86);
  box-shadow: inset 0 0 0 1px rgba(226, 232, 240, 0.7);
}

.runtime-access-tab-button {
  height: 36px;
  padding: 0 16px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: #64748b;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition:
    color 0.2s ease,
    background-color 0.2s ease,
    box-shadow 0.2s ease;
}

.runtime-access-tab-button.is-active {
  color: #2563eb;
  background: #ffffff;
  box-shadow: 0 6px 14px rgba(37, 99, 235, 0.1);
}

.runtime-access-section {
  padding: 2px 2px 0;
}

.runtime-access-section-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.runtime-access-section-meta {
  min-width: 0;
}

.runtime-access-section-label {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  margin-bottom: 8px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.12);
  color: #334155;
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
}

.runtime-access-section-description {
  margin: 0;
  font-size: 13px;
  color: #64748b;
}

.runtime-access-table-wrap {
  overflow: hidden;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow:
    inset 0 0 0 1px rgba(226, 232, 240, 0.7),
    0 8px 18px rgba(15, 23, 42, 0.03);
}

.runtime-access-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.runtime-access-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.runtime-access-empty-text {
  color: #9ca3af;
}

.runtime-role-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.runtime-access-editor__body {
  padding: 4px 0;
}

.runtime-access-readonly {
  min-height: 40px;
  width: 100%;
  display: flex;
  align-items: center;
  padding: 0 12px;
  border: 1px solid #dbe3ee;
  border-radius: 12px;
  background: #f8fafc;
  color: #334155;
}

:deep(.runtime-access-modal .el-dialog) {
  overflow: hidden;
  border-radius: 24px;
  background: #f8fafc;
  margin: 0 !important;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 28px 80px rgba(15, 23, 42, 0.2);
}

:deep(.runtime-access-modal .el-overlay-dialog) {
  overflow: hidden;
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}

:deep(.runtime-access-modal .el-dialog__header) {
  margin: 0;
  padding: 20px 24px;
  background: #f8fafc;
}

:deep(.runtime-access-modal .el-dialog__title) {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
}

:deep(.runtime-access-modal .el-dialog__body) {
  padding: 24px;
  background: #f8fafc;
  overflow: auto;
}

:deep(.runtime-access-modal .el-dialog__footer) {
  display: none;
}

:deep(.runtime-access-editor-dialog .el-dialog) {
  overflow: hidden;
  border-radius: 20px;
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.18);
}

:deep(.runtime-access-editor-dialog .el-dialog__header) {
  margin: 0;
  padding: 18px 24px;
  background: #f8fafc;
}

:deep(.runtime-access-editor-dialog .el-dialog__title) {
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
}

:deep(.runtime-access-editor-dialog .el-dialog__body) {
  padding: 20px 24px;
}

:deep(.runtime-access-editor-dialog .el-dialog__footer) {
  padding: 0 24px 20px;
}

:deep(.runtime-access-alert) {
  border-radius: 16px;
  border: none;
  background: linear-gradient(180deg, rgba(219, 234, 254, 0.88) 0%, rgba(239, 246, 255, 0.72) 100%);
  box-shadow: inset 0 0 0 1px rgba(147, 197, 253, 0.34);
}

:deep(.runtime-access-table .el-table__inner-wrapper::before) {
  display: none;
}

:deep(.runtime-access-table .el-table__header-wrapper th) {
  background: #f8fafc !important;
  color: #475569 !important;
  font-weight: 600;
  height: 48px;
  border-bottom: none;
}

:deep(.runtime-access-table .el-table__body-wrapper td) {
  padding-top: 14px;
  padding-bottom: 14px;
}

:deep(.runtime-access-table .el-table__cell) {
  border-bottom-color: rgba(226, 232, 240, 0.7);
}

:deep(.runtime-access-section-toolbar .el-button),
:deep(.runtime-access-actions .el-button),
:deep(.runtime-access-toolbar__actions .el-button) {
  border-radius: 999px;
}

:deep(.runtime-access-form .el-form-item__label) {
  font-weight: 600;
  color: #334155;
}

:deep(.runtime-access-form .el-input__wrapper),
:deep(.runtime-access-form .el-textarea__inner),
:deep(.runtime-access-form .el-select__wrapper) {
  border-radius: 12px;
}

html.dark .runtime-access-empty-state,
[data-theme='dark'] .runtime-access-empty-state,
html.dark .runtime-access-tabs-card,
[data-theme='dark'] .runtime-access-tabs-card,
html.dark .runtime-access-table-wrap,
[data-theme='dark'] .runtime-access-table-wrap {
  background: rgba(15, 23, 42, 0.88);
  box-shadow:
    inset 0 0 0 1px rgba(51, 65, 85, 0.75),
    0 20px 42px rgba(2, 6, 23, 0.34);
}

html.dark .runtime-access-section-description,
[data-theme='dark'] .runtime-access-section-description,
html.dark .runtime-access-empty-text,
[data-theme='dark'] .runtime-access-empty-text {
  color: #9ca3af;
}

html.dark .runtime-access-section-label,
[data-theme='dark'] .runtime-access-section-label {
  background: rgba(148, 163, 184, 0.14);
  color: #cbd5e1;
}

html.dark .runtime-access-readonly,
[data-theme='dark'] .runtime-access-readonly {
  border-color: transparent;
  background: #0f172a;
  color: #e5e7eb;
  box-shadow: inset 0 0 0 1px rgba(51, 65, 85, 0.78);
}

html.dark :deep(.runtime-access-modal .el-dialog),
[data-theme='dark'] :deep(.runtime-access-modal .el-dialog),
html.dark :deep(.runtime-access-modal .el-dialog__body),
[data-theme='dark'] :deep(.runtime-access-modal .el-dialog__body),
html.dark :deep(.runtime-access-modal .el-dialog__header),
[data-theme='dark'] :deep(.runtime-access-modal .el-dialog__header),
html.dark :deep(.runtime-access-modal .el-dialog__footer),
[data-theme='dark'] :deep(.runtime-access-modal .el-dialog__footer),
html.dark :deep(.runtime-access-editor-dialog .el-dialog__header),
[data-theme='dark'] :deep(.runtime-access-editor-dialog .el-dialog__header),
html.dark :deep(.runtime-access-editor-dialog .el-dialog__body),
[data-theme='dark'] :deep(.runtime-access-editor-dialog .el-dialog__body),
html.dark :deep(.runtime-access-editor-dialog .el-dialog__footer),
[data-theme='dark'] :deep(.runtime-access-editor-dialog .el-dialog__footer) {
  background: #020617;
}

html.dark :deep(.runtime-access-modal .el-dialog__title),
[data-theme='dark'] :deep(.runtime-access-modal .el-dialog__title),
html.dark :deep(.runtime-access-editor-dialog .el-dialog__title),
[data-theme='dark'] :deep(.runtime-access-editor-dialog .el-dialog__title) {
  color: #f8fafc;
}

html.dark :deep(.runtime-access-alert),
[data-theme='dark'] :deep(.runtime-access-alert) {
  background: linear-gradient(180deg, rgba(30, 41, 59, 0.9) 0%, rgba(15, 23, 42, 0.96) 100%);
  box-shadow: inset 0 0 0 1px rgba(59, 130, 246, 0.22);
}

html.dark .runtime-access-tab-switcher,
[data-theme='dark'] .runtime-access-tab-switcher {
  background: #0f172a;
  box-shadow: inset 0 0 0 1px rgba(51, 65, 85, 0.82);
}

html.dark .runtime-access-tab-button,
[data-theme='dark'] .runtime-access-tab-button {
  color: #94a3b8;
}

html.dark .runtime-access-tab-button.is-active,
[data-theme='dark'] .runtime-access-tab-button.is-active {
  color: #93c5fd;
  background: #1e293b;
  box-shadow: none;
}

html.dark :deep(.runtime-access-table .el-table__header-wrapper th),
[data-theme='dark'] :deep(.runtime-access-table .el-table__header-wrapper th) {
  background: #0f172a !important;
  color: #cbd5e1 !important;
}

html.dark :deep(.runtime-access-table .el-table__cell),
[data-theme='dark'] :deep(.runtime-access-table .el-table__cell) {
  border-bottom-color: rgba(51, 65, 85, 0.72);
}

html.dark :deep(.runtime-access-form .el-form-item__label),
[data-theme='dark'] :deep(.runtime-access-form .el-form-item__label) {
  color: #cbd5e1;
}

@media (max-width: 768px) {
  .runtime-access-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .runtime-access-tabs-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .runtime-access-tabs-actions {
    justify-content: flex-end;
    flex-wrap: wrap;
  }

  .runtime-access-section-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .runtime-access-tabs-card {
    padding: 16px;
  }

  :deep(.runtime-access-modal .el-dialog) {
    max-height: 94vh;
  }

  :deep(.runtime-access-modal .el-overlay-dialog) {
    padding: 12px;
    align-items: flex-start;
  }

  :deep(.runtime-access-modal .el-dialog__body) {
    padding: 18px;
  }
}
</style>
