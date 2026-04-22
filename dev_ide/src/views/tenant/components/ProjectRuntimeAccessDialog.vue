<template>
  <el-dialog
    v-model="dialogVisible"
    :title="t('projectManagement.runtimeAccess.dialogTitle', { name: project?.name || '' })"
    width="min(1120px, 96vw)"
    append-to-body
  >
    <div class="runtime-access-dialog">
      <el-alert
        type="info"
        :closable="false"
        class="mb-4"
        :title="t('projectManagement.runtimeAccess.publishTip')"
      />

      <div v-if="!project" class="py-8">
        <el-empty :description="t('common.noData')" />
      </div>

      <template v-else>
        <div class="runtime-access-toolbar">
          <div>
            <h3 class="runtime-access-toolbar__title">
              {{ project.name }}
            </h3>
            <p class="runtime-access-toolbar__description">
              {{ t('projectManagement.memberAndPermission') }}
            </p>
          </div>
          <el-button @click="refreshAll">
            {{ t('common.refresh') }}
          </el-button>
        </div>

        <el-tabs v-model="activeTab">
          <el-tab-pane
            :label="t('projectManagement.runtimeAccess.tabs.users')"
            :name="RUNTIME_ACCESS_TABS.USERS"
          >
            <div class="runtime-access-section-toolbar">
              <el-button type="primary" @click="openCreateUserEditor">
                {{ t('projectManagement.runtimeAccess.users.create') }}
              </el-button>
            </div>

            <el-table :data="runtimeUsers" v-loading="userLoading" height="420">
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
                :label="t('projectManagement.runtimeAccess.users.lastLoginAt')"
                min-width="180"
              >
                <template #default="{ row }">
                  {{ formatLastLogin(row.lastLoginAt) }}
                </template>
              </el-table-column>
              <el-table-column
                :label="t('projectManagement.runtimeAccess.users.actions')"
                min-width="300"
                fixed="right"
              >
                <template #default="{ row }">
                  <div class="runtime-access-actions">
                    <el-button size="small" @click="openEditUserEditor(row)">
                      {{ t('projectManagement.runtimeAccess.users.bindRoles') }}
                    </el-button>
                    <el-button size="small" @click="toggleRuntimeUserStatus(row)">
                      {{
                        row.status === 'active'
                          ? t('projectManagement.runtimeAccess.users.disable')
                          : t('projectManagement.runtimeAccess.users.enable')
                      }}
                    </el-button>
                    <el-button size="small" type="warning" @click="resetRuntimeUserPassword(row)">
                      {{ t('projectManagement.runtimeAccess.users.resetPassword') }}
                    </el-button>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <el-tab-pane
            :label="t('projectManagement.runtimeAccess.tabs.roles')"
            :name="RUNTIME_ACCESS_TABS.ROLES"
          >
            <div class="runtime-access-section-toolbar">
              <el-button type="primary" @click="openCreateRoleEditor">
                {{ t('projectManagement.runtimeAccess.roles.create') }}
              </el-button>
            </div>

            <el-table :data="runtimeRoles" v-loading="roleLoading" height="420">
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
                prop="grantCount"
                :label="t('projectManagement.runtimeAccess.roles.grantCount')"
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
          </el-tab-pane>
        </el-tabs>
      </template>
    </div>

    <template #footer>
      <el-button @click="dialogVisible = false">
        {{ t('common.cancel') }}
      </el-button>
    </template>
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
  >
    <el-form label-width="120px">
      <el-form-item v-if="userEditorMode === 'create'" :label="t('projectManagement.runtimeAccess.users.username')">
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
          <div class="runtime-access-readonly">{{ userForm.username || '--' }}</div>
        </el-form-item>
        <el-form-item :label="t('projectManagement.runtimeAccess.users.displayName')">
          <div class="runtime-access-readonly">{{ userForm.displayName || '--' }}</div>
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
  >
    <el-form label-width="120px">
      <el-form-item :label="t('projectManagement.runtimeAccess.roles.code')">
        <el-input
          v-model="roleForm.code"
          :disabled="roleEditorMode === 'edit' && roleForm.isSystem"
          :placeholder="t('projectManagement.runtimeAccess.roles.inputCode')"
        />
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
          <el-option
            :label="t('projectManagement.runtimeAccess.statusActive')"
            value="active"
          />
          <el-option
            :label="t('projectManagement.runtimeAccess.statusDisabled')"
            value="disabled"
          />
        </el-select>
      </el-form-item>
    </el-form>

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

<script setup>
import { computed, reactive, ref, toRef, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

import { projectAPI } from '../../../api/project.api.js'
import {
  RUNTIME_ACCESS_TABS,
  useProjectRuntimeAccessState,
} from './project-runtime-access-state.js'

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
  resetRuntimeUserPassword,
  openCreateRoleEditor,
  openEditRoleEditor,
  closeRoleEditor,
  submitRoleForm,
  deleteRuntimeRole,
  resolveUserRoleNames,
  resolveRuntimeAccessStatusMeta,
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

const formatLastLogin = (value) => value || '--'
</script>

<style scoped>
.runtime-access-dialog {
  min-height: 240px;
}

.runtime-access-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.runtime-access-toolbar__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #111827;
}

.runtime-access-toolbar__description {
  margin: 4px 0 0;
  font-size: 13px;
  color: #6b7280;
}

.runtime-access-section-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 12px;
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

.runtime-access-readonly {
  min-height: 32px;
  width: 100%;
  display: flex;
  align-items: center;
  color: #374151;
}

html.dark .runtime-access-toolbar__title,
[data-theme='dark'] .runtime-access-toolbar__title {
  color: #f3f4f6;
}

html.dark .runtime-access-toolbar__description,
[data-theme='dark'] .runtime-access-toolbar__description,
html.dark .runtime-access-empty-text,
[data-theme='dark'] .runtime-access-empty-text {
  color: #9ca3af;
}

html.dark .runtime-access-readonly,
[data-theme='dark'] .runtime-access-readonly {
  color: #e5e7eb;
}

@media (max-width: 768px) {
  .runtime-access-toolbar {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
