export const RUNTIME_ACCESS_TABS = {
  USERS: 'users',
  ROLES: 'roles',
}

const RUNTIME_STATUS_META = {
  active: {
    value: 'active',
    tagType: 'success',
    labelKey: 'projectManagement.runtimeAccess.statusActive',
  },
  disabled: {
    value: 'disabled',
    tagType: 'info',
    labelKey: 'projectManagement.runtimeAccess.statusDisabled',
  },
}

const trimText = (value) => (typeof value === 'string' ? value.trim() : '')

const normalizeCode = (value) => trimText(value).replace(/\s+/g, '_').toUpperCase()

const normalizeIdList = (values) => {
  if (!Array.isArray(values)) {
    return []
  }

  return [...new Set(values.map((item) => trimText(item)).filter(Boolean))]
}

const normalizeCount = (value) => {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0
}

const createEmptyUserForm = () => ({
  id: '',
  username: '',
  displayName: '',
  initialPassword: '',
  roleIds: [],
  status: 'active',
})

const createEmptyRoleForm = () => ({
  id: '',
  code: '',
  name: '',
  description: '',
  status: 'active',
  isSystem: false,
})

export const resolveRuntimeAccessStatusMeta = (status) =>
  RUNTIME_STATUS_META[status] || RUNTIME_STATUS_META.disabled

const resolveOptionalRuntimeStatus = (status) => {
  const normalized = trimText(status)
  return normalized && RUNTIME_STATUS_META[normalized] ? normalized : ''
}

export const buildRuntimeRolePayload = (form = {}, options = {}) => {
  const { includeStatus = true } = options
  const payload = {
    name: trimText(form.name),
    code: normalizeCode(form.code),
    description: trimText(form.description),
  }

  if (includeStatus) {
    payload.status = resolveRuntimeAccessStatusMeta(form.status).value
  }

  return payload
}

export const buildRuntimeUserPayload = (form = {}, options = {}) => {
  const { includeStatus = true } = options
  const payload = {}

  const username = trimText(form.username)
  if (username) {
    payload.username = username
  }

  const displayName = trimText(form.displayName)
  if (displayName) {
    payload.displayName = displayName
  }

  const initialPassword = trimText(form.initialPassword)
  if (initialPassword) {
    payload.initialPassword = initialPassword
  }

  const roleIds = normalizeIdList(form.roleIds)
  if (roleIds.length > 0) {
    payload.roleIds = roleIds
  }

  const status = resolveOptionalRuntimeStatus(form.status)
  if (includeStatus && status) {
    payload.status = status
  }

  return payload
}

export const buildRuntimeUserRoleBindingPayload = (form = {}) => ({
  roleIds: normalizeIdList(form.roleIds),
})

export const resolveRuntimeUserBindingPlan = ({
  selectedRoleIds = [],
  createdUserId = '',
  fallbackUserId = '',
}) => {
  const normalizedRoleIds = normalizeIdList(selectedRoleIds)
  if (normalizedRoleIds.length === 0) {
    return {
      type: 'skip',
      userId: '',
    }
  }

  const resolvedUserId = trimText(createdUserId) || trimText(fallbackUserId)
  if (resolvedUserId) {
    return {
      type: 'bind',
      userId: resolvedUserId,
    }
  }

  return {
    type: 'partial_success_missing_user_id',
    userId: '',
  }
}

export const shouldApplyRuntimeAccessLoadResult = ({
  requestProjectId = '',
  activeProjectId = '',
  requestToken = 0,
  activeToken = 0,
  visible = false,
}) =>
  Boolean(
    visible &&
    trimText(requestProjectId) &&
    trimText(requestProjectId) === trimText(activeProjectId) &&
    requestToken === activeToken,
  )

export const isRuntimeAccessDialogCancelled = (error) => {
  if (error === 'cancel' || error === 'close') {
    return true
  }

  const action = trimText(error?.action)
  return action === 'cancel' || action === 'close'
}

export const summarizeRuntimeGrantCount = (roleLike) => {
  if (!roleLike || typeof roleLike !== 'object') {
    return 0
  }

  const directCount = normalizeCount(roleLike.grantCount)
  if (directCount > 0) {
    return directCount
  }

  if (!Array.isArray(roleLike.grants)) {
    return 0
  }

  return roleLike.grants.reduce((total, item) => {
    if (item && typeof item === 'object' && 'count' in item) {
      return total + normalizeCount(item.count)
    }

    return total + 1
  }, 0)
}

const unwrapPayload = (response) => response?.data ?? response ?? {}

const unwrapData = (response) => {
  const payload = unwrapPayload(response)
  return payload?.data ?? payload
}

const extractList = (response, candidateKeys = []) => {
  const payload = unwrapPayload(response)
  const data = unwrapData(response)

  if (Array.isArray(data)) {
    return data
  }

  for (const key of candidateKeys) {
    if (Array.isArray(data?.[key])) {
      return data[key]
    }
    if (Array.isArray(payload?.[key])) {
      return payload[key]
    }
  }

  return []
}

const extractSingle = (response, candidateKeys = []) => {
  const payload = unwrapPayload(response)
  const data = unwrapData(response)

  if (data && typeof data === 'object' && !Array.isArray(data)) {
    for (const key of candidateKeys) {
      if (data[key] && typeof data[key] === 'object') {
        return data[key]
      }
    }
    return data
  }

  if (payload && typeof payload === 'object' && !Array.isArray(payload)) {
    for (const key of candidateKeys) {
      if (payload[key] && typeof payload[key] === 'object') {
        return payload[key]
      }
    }
  }

  return null
}

const normalizeRuntimeRoleRecord = (role = {}) => ({
  ...role,
  id: role.id,
  code: trimText(role.code),
  name: trimText(role.name),
  description: trimText(role.description),
  status: resolveRuntimeAccessStatusMeta(role.status).value,
  isSystem: Boolean(role.isSystem),
  bindingCount: normalizeCount(role.bindingCount ?? role.userCount ?? role.binding_count),
  grantCount: summarizeRuntimeGrantCount(role),
})

const normalizeRuntimeUserRecord = (user = {}) => {
  const roles = Array.isArray(user.roles) ? user.roles.filter(Boolean) : []
  const roleIds =
    normalizeIdList(user.roleIds) || normalizeIdList(roles.map((role) => role?.id || role?.roleId))

  return {
    ...user,
    id: user.id,
    username: trimText(user.username),
    displayName: trimText(user.displayName),
    status: resolveRuntimeAccessStatusMeta(user.status).value,
    roleIds:
      roleIds.length > 0 ? roleIds : normalizeIdList(roles.map((role) => role?.id || role?.roleId)),
    roles,
  }
}

const getErrorMessage = (error, fallback) => {
  const responseMessage =
    error?.response?.data?.message || error?.response?.data?.error || error?.message

  return responseMessage || fallback
}

const applyUserForm = (target, user) => {
  Object.assign(target, createEmptyUserForm(), {
    id: user?.id || '',
    username: user?.username || '',
    displayName: user?.displayName || '',
    roleIds: normalizeIdList(
      user?.roleIds?.length ? user.roleIds : user?.roles?.map((role) => role?.id || role?.roleId),
    ),
    status: resolveRuntimeAccessStatusMeta(user?.status).value,
  })
}

const applyRoleForm = (target, role) => {
  Object.assign(target, createEmptyRoleForm(), {
    id: role?.id || '',
    code: role?.code || '',
    name: role?.name || '',
    description: role?.description || '',
    status: resolveRuntimeAccessStatusMeta(role?.status).value,
    isSystem: Boolean(role?.isSystem),
  })
}

const validateUserPayload = (payload, t, { requirePassword }) => {
  if (!payload.username) {
    throw new Error(t('projectManagement.runtimeAccess.users.usernameRequired'))
  }
  if (!payload.displayName) {
    throw new Error(t('projectManagement.runtimeAccess.users.displayNameRequired'))
  }
  if (requirePassword && !payload.initialPassword) {
    throw new Error(t('projectManagement.runtimeAccess.users.passwordRequired'))
  }
}

const validateRolePayload = (payload, t) => {
  if (!payload.name) {
    throw new Error(t('projectManagement.runtimeAccess.roles.nameRequired'))
  }
  if (!payload.code) {
    throw new Error(t('projectManagement.runtimeAccess.roles.codeRequired'))
  }
}

const createRuntimeUserPartialSuccessError = (type, cause = null) => ({
  kind: 'runtime_user_partial_success',
  type,
  cause,
})

export const useProjectRuntimeAccessState = ({
  visibleRef,
  projectRef,
  t,
  api,
  ref,
  reactive,
  computed,
  watch,
  message,
  messageBox,
}) => {
  const activeTab = ref(RUNTIME_ACCESS_TABS.USERS)
  const userLoading = ref(false)
  const roleLoading = ref(false)
  const userSubmitting = ref(false)
  const roleSubmitting = ref(false)
  const runtimeUsers = ref([])
  const runtimeRoles = ref([])
  const userEditorVisible = ref(false)
  const roleEditorVisible = ref(false)
  const userEditorMode = ref('create')
  const roleEditorMode = ref('create')
  const userForm = reactive(createEmptyUserForm())
  const roleForm = reactive(createEmptyRoleForm())
  const runtimeUsersRequestToken = ref(0)
  const runtimeRolesRequestToken = ref(0)

  const roleOptions = computed(() =>
    runtimeRoles.value.map((role) => ({
      value: role.id,
      label: role.name || role.code || role.id,
      disabled: role.status === 'disabled',
    })),
  )

  const roleNameMap = computed(() => {
    const map = new Map()
    runtimeRoles.value.forEach((role) => {
      map.set(role.id, role.name || role.code || role.id)
    })
    return map
  })

  const projectId = computed(() => projectRef.value?.id || '')

  const clearRuntimeAccessData = () => {
    runtimeUsers.value = []
    runtimeRoles.value = []
    userLoading.value = false
    roleLoading.value = false
  }

  const invalidateRuntimeAccessRequests = () => {
    runtimeUsersRequestToken.value += 1
    runtimeRolesRequestToken.value += 1
  }

  const resetRuntimeAccessState = () => {
    invalidateRuntimeAccessRequests()
    clearRuntimeAccessData()
    userEditorVisible.value = false
    roleEditorVisible.value = false
    applyUserForm(userForm)
    applyRoleForm(roleForm)
  }

  const loadRuntimeUsers = async () => {
    if (!projectId.value || !visibleRef.value) {
      runtimeUsers.value = []
      return []
    }

    const requestProjectId = projectId.value
    const requestToken = runtimeUsersRequestToken.value + 1
    runtimeUsersRequestToken.value = requestToken
    userLoading.value = true
    try {
      const response = await api.listRuntimeUsers(requestProjectId)
      if (
        !shouldApplyRuntimeAccessLoadResult({
          requestProjectId,
          activeProjectId: projectId.value,
          requestToken,
          activeToken: runtimeUsersRequestToken.value,
          visible: visibleRef.value,
        })
      ) {
        return []
      }

      runtimeUsers.value = extractList(response, ['runtimeUsers', 'items', 'list']).map(
        normalizeRuntimeUserRecord,
      )
      return runtimeUsers.value
    } catch (error) {
      if (
        !shouldApplyRuntimeAccessLoadResult({
          requestProjectId,
          activeProjectId: projectId.value,
          requestToken,
          activeToken: runtimeUsersRequestToken.value,
          visible: visibleRef.value,
        })
      ) {
        return []
      }

      runtimeUsers.value = []
      message.error(
        t('projectManagement.runtimeAccess.users.loadFailed', {
          message: getErrorMessage(error, t('common.error')),
        }),
      )
      return []
    } finally {
      if (
        shouldApplyRuntimeAccessLoadResult({
          requestProjectId,
          activeProjectId: projectId.value,
          requestToken,
          activeToken: runtimeUsersRequestToken.value,
          visible: visibleRef.value,
        })
      ) {
        userLoading.value = false
      }
    }
  }

  const loadRuntimeRoles = async () => {
    if (!projectId.value || !visibleRef.value) {
      runtimeRoles.value = []
      return []
    }

    const requestProjectId = projectId.value
    const requestToken = runtimeRolesRequestToken.value + 1
    runtimeRolesRequestToken.value = requestToken
    roleLoading.value = true
    try {
      const response = await api.listRuntimeRoles(requestProjectId)
      if (
        !shouldApplyRuntimeAccessLoadResult({
          requestProjectId,
          activeProjectId: projectId.value,
          requestToken,
          activeToken: runtimeRolesRequestToken.value,
          visible: visibleRef.value,
        })
      ) {
        return []
      }

      runtimeRoles.value = extractList(response, ['runtimeRoles', 'items', 'list']).map(
        normalizeRuntimeRoleRecord,
      )
      return runtimeRoles.value
    } catch (error) {
      if (
        !shouldApplyRuntimeAccessLoadResult({
          requestProjectId,
          activeProjectId: projectId.value,
          requestToken,
          activeToken: runtimeRolesRequestToken.value,
          visible: visibleRef.value,
        })
      ) {
        return []
      }

      runtimeRoles.value = []
      message.error(
        t('projectManagement.runtimeAccess.roles.loadFailed', {
          message: getErrorMessage(error, t('common.error')),
        }),
      )
      return []
    } finally {
      if (
        shouldApplyRuntimeAccessLoadResult({
          requestProjectId,
          activeProjectId: projectId.value,
          requestToken,
          activeToken: runtimeRolesRequestToken.value,
          visible: visibleRef.value,
        })
      ) {
        roleLoading.value = false
      }
    }
  }

  const refreshAll = async () => {
    await Promise.all([loadRuntimeUsers(), loadRuntimeRoles()])
  }

  watch(
    () => [visibleRef.value, projectId.value],
    ([visible, id], [previousVisible, previousProjectId] = []) => {
      if (!visible || !id) {
        resetRuntimeAccessState()
        activeTab.value = RUNTIME_ACCESS_TABS.USERS
        return
      }

      if (!previousVisible || previousProjectId !== id) {
        resetRuntimeAccessState()
      }

      if (visible && id) {
        refreshAll()
      }
    },
    { immediate: true },
  )

  const openCreateUserEditor = () => {
    userEditorMode.value = 'create'
    applyUserForm(userForm)
    userEditorVisible.value = true
  }

  const openEditUserEditor = (user) => {
    userEditorMode.value = 'edit'
    applyUserForm(userForm, user)
    userEditorVisible.value = true
  }

  const closeUserEditor = () => {
    userEditorVisible.value = false
  }

  const submitUserForm = async () => {
    if (!projectId.value) {
      return
    }

    const payload = buildRuntimeUserPayload(userForm, {
      includeStatus: userEditorMode.value !== 'create',
    })

    try {
      if (userEditorMode.value === 'create') {
        validateUserPayload(payload, t, { requirePassword: true })
      }
    } catch (error) {
      message.warning(error.message)
      return
    }

    userSubmitting.value = true
    try {
      if (userEditorMode.value === 'create') {
        const createPayload = {
          username: payload.username,
          displayName: payload.displayName,
          initialPassword: payload.initialPassword,
        }

        const createResponse = await api.createRuntimeUser(projectId.value, createPayload)
        const createdUser =
          upsertRuntimeUser(createResponse) ||
          extractSingle(createResponse, ['runtimeUser', 'user']) ||
          null

        const roleBindingPayload = buildRuntimeUserRoleBindingPayload(userForm)
        let fallbackUserId = ''

        if (roleBindingPayload.roleIds.length > 0 && !createdUser?.id) {
          const refreshedUsers = await loadRuntimeUsers()
          fallbackUserId =
            refreshedUsers.find((user) => user.username === payload.username)?.id ||
            runtimeUsers.value.find((user) => user.username === payload.username)?.id ||
            ''
        }

        const bindingPlan = resolveRuntimeUserBindingPlan({
          selectedRoleIds: roleBindingPayload.roleIds,
          createdUserId: createdUser?.id,
          fallbackUserId,
        })

        if (bindingPlan.type === 'bind') {
          try {
            await api.updateRuntimeUserRoles(
              projectId.value,
              bindingPlan.userId,
              roleBindingPayload,
            )
          } catch (error) {
            throw createRuntimeUserPartialSuccessError('role_binding_failed', error)
          }
        } else if (bindingPlan.type === 'partial_success_missing_user_id') {
          throw createRuntimeUserPartialSuccessError('missing_user_id')
        }

        message.success(t('projectManagement.runtimeAccess.users.createSuccess'))
      } else {
        await api.updateRuntimeUserRoles(
          projectId.value,
          userForm.id,
          buildRuntimeUserRoleBindingPayload(userForm),
        )
        message.success(t('projectManagement.runtimeAccess.users.rolesUpdateSuccess'))
      }

      userEditorVisible.value = false
      await Promise.all([loadRuntimeUsers(), loadRuntimeRoles()])
    } catch (error) {
      if (error?.kind === 'runtime_user_partial_success') {
        userEditorVisible.value = false
        await Promise.all([loadRuntimeUsers(), loadRuntimeRoles()])
        const messageKey =
          error.type === 'role_binding_failed'
            ? 'projectManagement.runtimeAccess.users.partialSuccessRoleBindingFailed'
            : 'projectManagement.runtimeAccess.users.partialSuccessMissingUserId'
        message.warning(
          t(messageKey, {
            message: getErrorMessage(error.cause, t('common.error')),
          }),
        )
        return
      }

      message.error(
        t('projectManagement.runtimeAccess.users.saveFailed', {
          message: getErrorMessage(error, t('common.error')),
        }),
      )
    } finally {
      userSubmitting.value = false
    }
  }

  const toggleRuntimeUserStatus = async (user) => {
    if (!projectId.value || !user?.id) {
      return
    }

    const nextStatus = user.status === 'active' ? 'disabled' : 'active'

    try {
      await api.updateRuntimeUserStatus(projectId.value, user.id, {
        status: nextStatus,
      })
      message.success(
        t('projectManagement.runtimeAccess.users.toggleStatusSuccess', {
          status: t(resolveRuntimeAccessStatusMeta(nextStatus).labelKey),
        }),
      )
      await loadRuntimeUsers()
    } catch (error) {
      message.error(
        t('projectManagement.runtimeAccess.users.toggleStatusFailed', {
          message: getErrorMessage(error, t('common.error')),
        }),
      )
    }
  }

  const resetRuntimeUserPassword = async (user) => {
    if (!projectId.value || !user?.id) {
      return
    }

    try {
      const { value } = await messageBox.prompt(
        t('projectManagement.runtimeAccess.users.resetPasswordInput', {
          username: user.username || user.displayName || user.id,
        }),
        t('projectManagement.runtimeAccess.users.resetPasswordTitle'),
        {
          inputType: 'password',
          inputPlaceholder: t('projectManagement.runtimeAccess.users.inputInitialPassword'),
          inputValidator(inputValue) {
            return trimText(inputValue)
              ? true
              : t('projectManagement.runtimeAccess.users.passwordRequired')
          },
        },
      )

      await api.resetRuntimeUserPassword(projectId.value, user.id, {
        newPassword: trimText(value),
      })
      message.success(t('projectManagement.runtimeAccess.users.resetPasswordSuccess'))
    } catch (error) {
      if (isRuntimeAccessDialogCancelled(error)) {
        return
      }

      message.error(
        t('projectManagement.runtimeAccess.users.resetPasswordFailed', {
          message: getErrorMessage(error, t('common.error')),
        }),
      )
    }
  }

  const openCreateRoleEditor = () => {
    roleEditorMode.value = 'create'
    applyRoleForm(roleForm)
    roleEditorVisible.value = true
  }

  const openEditRoleEditor = (role) => {
    roleEditorMode.value = 'edit'
    applyRoleForm(roleForm, role)
    roleEditorVisible.value = true
  }

  const closeRoleEditor = () => {
    roleEditorVisible.value = false
  }

  const submitRoleForm = async () => {
    if (!projectId.value) {
      return
    }

    const payload = buildRuntimeRolePayload(roleForm, {
      includeStatus: roleEditorMode.value !== 'create',
    })

    try {
      validateRolePayload(payload, t)
    } catch (error) {
      message.warning(error.message)
      return
    }

    roleSubmitting.value = true
    try {
      if (roleEditorMode.value === 'create') {
        await api.createRuntimeRole(projectId.value, payload)
        message.success(t('projectManagement.runtimeAccess.roles.createSuccess'))
      } else {
        await api.updateRuntimeRole(projectId.value, roleForm.id, payload)
        message.success(t('projectManagement.runtimeAccess.roles.updateSuccess'))
      }

      roleEditorVisible.value = false
      await Promise.all([loadRuntimeRoles(), loadRuntimeUsers()])
    } catch (error) {
      message.error(
        t('projectManagement.runtimeAccess.roles.saveFailed', {
          message: getErrorMessage(error, t('common.error')),
        }),
      )
    } finally {
      roleSubmitting.value = false
    }
  }

  const deleteRuntimeRole = async (role) => {
    if (!projectId.value || !role?.id) {
      return
    }

    if (role.isSystem) {
      message.warning(t('projectManagement.runtimeAccess.roles.systemRoleDeleteBlocked'))
      return
    }

    try {
      await messageBox.confirm(
        t('projectManagement.runtimeAccess.roles.deleteConfirmText', {
          name: role.name || role.code || role.id,
        }),
        t('projectManagement.runtimeAccess.roles.deleteConfirmTitle'),
        { type: 'warning' },
      )
      await api.deleteRuntimeRole(projectId.value, role.id)
      message.success(t('projectManagement.runtimeAccess.roles.deleteSuccess'))
      await Promise.all([loadRuntimeRoles(), loadRuntimeUsers()])
    } catch (error) {
      if (isRuntimeAccessDialogCancelled(error)) {
        return
      }

      message.error(
        t('projectManagement.runtimeAccess.roles.deleteFailed', {
          message: getErrorMessage(error, t('common.error')),
        }),
      )
    }
  }

  const resolveUserRoleNames = (user) => {
    const ids = normalizeIdList(user?.roleIds)
    if (ids.length > 0) {
      return ids.map((id) => roleNameMap.value.get(id) || id)
    }

    if (Array.isArray(user?.roles) && user.roles.length > 0) {
      return user.roles.map((role) => role?.name || role?.code || role?.id).filter(Boolean)
    }

    return []
  }

  const upsertRuntimeUser = (response) => {
    const user = extractSingle(response, ['runtimeUser', 'user'])
    return user ? normalizeRuntimeUserRecord(user) : null
  }

  const upsertRuntimeRole = (response) => {
    const role = extractSingle(response, ['runtimeRole', 'role'])
    return role ? normalizeRuntimeRoleRecord(role) : null
  }

  return {
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
    loadRuntimeUsers,
    loadRuntimeRoles,
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
    normalizeRuntimeRoleRecord,
    normalizeRuntimeUserRecord,
    upsertRuntimeUser,
    upsertRuntimeRole,
  }
}
