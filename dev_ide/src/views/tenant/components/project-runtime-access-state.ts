export const RUNTIME_ACCESS_TABS = {
  USERS: 'users',
  ROLES: 'roles',
}

export const RUNTIME_ROLE_CODE_PREFIX = 'PROJECT_'

type AnyRecord = Record<string, any>
type RuntimeStatus = 'active' | 'disabled'

interface RuntimeRole extends AnyRecord {
  id?: string
  code?: string
  name?: string
  description?: string
  status?: RuntimeStatus | string
  isSystem?: boolean
  bindingCount?: number
  userCount?: number
  binding_count?: number
  grantCount?: number
  grants?: unknown[]
}

interface RuntimeUser extends AnyRecord {
  id?: string
  username?: string
  displayName?: string
  status?: RuntimeStatus | string
  roleIds?: string[]
  roles?: RuntimeRole[]
}

interface RuntimeUserForm {
  id: string
  username: string
  displayName: string
  initialPassword: string
  roleIds: string[]
  status: RuntimeStatus
}

interface RuntimeRoleForm {
  id: string
  code: string
  name: string
  description: string
  status: RuntimeStatus
  isSystem: boolean
}

type RuntimeUserPayload = Record<string, unknown> & {
  username?: string
  displayName?: string
  initialPassword?: string
  roleIds?: string[]
  status?: RuntimeStatus
}

type RuntimeRolePayload = Record<string, unknown> & {
  name: string
  code: string
  description: string
  status?: RuntimeStatus
}

type Translator = (key: string, params?: Record<string, unknown>) => string

interface RuntimeAccessApi {
  listRuntimeUsers: (projectId: string) => Promise<unknown>
  createRuntimeUser: (projectId: string, payload: Record<string, unknown>) => Promise<unknown>
  updateRuntimeUserStatus: (
    projectId: string,
    userId: string,
    payload: Record<string, unknown>,
  ) => Promise<unknown>
  deleteRuntimeUser: (projectId: string, userId: string) => Promise<unknown>
  updateRuntimeUserRoles: (
    projectId: string,
    userId: string,
    payload: Record<string, unknown>,
  ) => Promise<unknown>
  resetRuntimeUserPassword: (
    projectId: string,
    userId: string,
    payload: Record<string, unknown>,
  ) => Promise<unknown>
  listRuntimeRoles: (projectId: string) => Promise<unknown>
  createRuntimeRole: (projectId: string, payload: Record<string, unknown>) => Promise<unknown>
  updateRuntimeRole: (
    projectId: string,
    roleId: string,
    payload: Record<string, unknown>,
  ) => Promise<unknown>
  deleteRuntimeRole: (projectId: string, roleId: string) => Promise<unknown>
}

interface RuntimeAccessDependencies {
  visibleRef: { value: boolean }
  projectRef: { value: { id?: string } | null | undefined }
  t: Translator
  api: RuntimeAccessApi
  ref: (...args: any[]) => any
  reactive: (...args: any[]) => any
  computed: (...args: any[]) => any
  watch: (...args: any[]) => any
  message: {
    error: (message: string) => unknown
    warning: (message: string) => unknown
    success: (message: string) => unknown
  }
  messageBox: {
    confirm: (...args: any[]) => Promise<unknown>
    prompt: (...args: any[]) => Promise<{ value: string }>
  }
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

const trimText = (value: unknown) => (typeof value === 'string' ? value.trim() : '')

const normalizeCode = (value: unknown) => trimText(value).replace(/\s+/g, '_').toUpperCase()

export const isDefaultRuntimeAdminUser = (user?: RuntimeUser | null) =>
  trimText(user?.username).toLowerCase() === 'admin'

export const stripRuntimeRoleCodePrefix = (value: unknown) => {
  const normalizedCode = normalizeCode(value)
  return normalizedCode.startsWith(RUNTIME_ROLE_CODE_PREFIX)
    ? normalizedCode.slice(RUNTIME_ROLE_CODE_PREFIX.length)
    : normalizedCode
}

export const normalizeRuntimeRoleCode = (value: unknown) => {
  const suffix = stripRuntimeRoleCodePrefix(value)
  return suffix ? `${RUNTIME_ROLE_CODE_PREFIX}${suffix}` : ''
}

const normalizeIdList = (values: unknown): string[] => {
  if (!Array.isArray(values)) {
    return []
  }

  return [...new Set(values.map((item) => trimText(item)).filter(Boolean))]
}

const normalizeCount = (value: unknown) => {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0
}

const createEmptyUserForm = (): RuntimeUserForm => ({
  id: '',
  username: '',
  displayName: '',
  initialPassword: '',
  roleIds: [],
  status: 'active',
})

const createEmptyRoleForm = (): RuntimeRoleForm => ({
  id: '',
  code: '',
  name: '',
  description: '',
  status: 'active',
  isSystem: false,
})

export const resolveRuntimeAccessStatusMeta = (status: unknown) =>
  RUNTIME_STATUS_META[status as RuntimeStatus] || RUNTIME_STATUS_META.disabled

const resolveOptionalRuntimeStatus = (status: unknown): RuntimeStatus | '' => {
  const normalized = trimText(status)
  return normalized && normalized in RUNTIME_STATUS_META ? (normalized as RuntimeStatus) : ''
}

export const buildRuntimeRolePayload = (
  form: Partial<RuntimeRoleForm> = {},
  options: { includeStatus?: boolean } = {},
): RuntimeRolePayload => {
  const { includeStatus = true } = options
  const payload: RuntimeRolePayload = {
    name: trimText(form.name),
    code: normalizeRuntimeRoleCode(form.code),
    description: trimText(form.description),
  }

  if (includeStatus) {
    payload.status = resolveRuntimeAccessStatusMeta(form.status).value as RuntimeStatus
  }

  return payload
}

export const buildRuntimeUserPayload = (
  form: Partial<RuntimeUserForm> = {},
  options: { includeStatus?: boolean } = {},
): RuntimeUserPayload => {
  const { includeStatus = true } = options
  const payload: RuntimeUserPayload = {}

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

export const buildRuntimeUserRoleBindingPayload = (form: Partial<RuntimeUserForm> = {}) => ({
  roleIds: normalizeIdList(form.roleIds),
})

export const resolveRuntimeUserBindingPlan = ({
  selectedRoleIds = [],
  createdUserId = '',
  fallbackUserId = '',
}: {
  selectedRoleIds?: unknown
  createdUserId?: unknown
  fallbackUserId?: unknown
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
}: {
  requestProjectId?: unknown
  activeProjectId?: unknown
  requestToken?: number
  activeToken?: number
  visible?: boolean
}) =>
  Boolean(
    visible &&
    trimText(requestProjectId) &&
    trimText(requestProjectId) === trimText(activeProjectId) &&
    requestToken === activeToken,
  )

export const isRuntimeAccessDialogCancelled = (error: any) => {
  if (error === 'cancel' || error === 'close') {
    return true
  }

  const action = trimText(error?.action)
  return action === 'cancel' || action === 'close'
}

export const summarizeRuntimeGrantCount = (roleLike: unknown) => {
  if (!roleLike || typeof roleLike !== 'object') {
    return 0
  }

  const roleRecord = roleLike as AnyRecord
  const directCount = normalizeCount(roleRecord.grantCount)
  if (directCount > 0) {
    return directCount
  }

  if (!Array.isArray(roleRecord.grants)) {
    return 0
  }

  return roleRecord.grants.reduce((total: number, item: unknown) => {
    if (item && typeof item === 'object' && 'count' in item) {
      return total + normalizeCount(item.count)
    }

    return total + 1
  }, 0)
}

const unwrapPayload = (response: any): any => response?.data ?? response ?? {}

const unwrapData = (response: any): any => {
  const payload = unwrapPayload(response)
  return payload?.data ?? payload
}

const extractList = (response: any, candidateKeys: string[] = []): AnyRecord[] => {
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

const extractSingle = (response: any, candidateKeys: string[] = []): AnyRecord | null => {
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

const normalizeRuntimeRoleRecord = (role: RuntimeRole = {}): RuntimeRole => ({
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

const normalizeRuntimeUserRecord = (user: RuntimeUser = {}): RuntimeUser => {
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

const getErrorMessage = (error: any, fallback: string): string => {
  const responseMessage =
    error?.response?.data?.msg || error?.response?.data?.error || error?.message

  return responseMessage || fallback
}

const applyUserForm = (target: RuntimeUserForm, user: RuntimeUser | null = null) => {
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

const applyRoleForm = (target: RuntimeRoleForm, role: RuntimeRole | null = null) => {
  Object.assign(target, createEmptyRoleForm(), {
    id: role?.id || '',
    code: stripRuntimeRoleCodePrefix(role?.code || ''),
    name: role?.name || '',
    description: role?.description || '',
    status: resolveRuntimeAccessStatusMeta(role?.status).value,
    isSystem: Boolean(role?.isSystem),
  })
}

const validateUserPayload = (
  payload: RuntimeUserPayload,
  t: Translator,
  { requirePassword }: { requirePassword: boolean },
) => {
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

const validateRolePayload = (payload: RuntimeRolePayload, t: Translator) => {
  if (!payload.name) {
    throw new Error(t('projectManagement.runtimeAccess.roles.nameRequired'))
  }
  if (!payload.code) {
    throw new Error(t('projectManagement.runtimeAccess.roles.codeRequired'))
  }
}

const createRuntimeUserPartialSuccessError = (type: string, cause: unknown = null) => ({
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
}: RuntimeAccessDependencies) => {
  const activeTab = ref(RUNTIME_ACCESS_TABS.USERS)
  const userLoading = ref(false)
  const roleLoading = ref(false)
  const userSubmitting = ref(false)
  const roleSubmitting = ref(false)
  const runtimeUsers = ref([] as RuntimeUser[])
  const runtimeRoles = ref([] as RuntimeRole[])
  const userEditorVisible = ref(false)
  const roleEditorVisible = ref(false)
  const userEditorMode = ref('create')
  const roleEditorMode = ref('create')
  const userForm = reactive(createEmptyUserForm())
  const roleForm = reactive(createEmptyRoleForm())
  const runtimeUsersRequestToken = ref(0)
  const runtimeRolesRequestToken = ref(0)

  const roleOptions = computed(() =>
    runtimeRoles.value.map((role: RuntimeRole) => ({
      value: role.id,
      label: role.name || role.code || role.id,
      disabled: role.status === 'disabled',
    })),
  )

  const roleNameMap = computed(() => {
    const map = new Map()
    runtimeRoles.value.forEach((role: RuntimeRole) => {
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
    ([visible, id]: [boolean, string], previousValue: [boolean, string] | undefined) => {
      const [previousVisible, previousProjectId] = previousValue ?? [false, '']
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

  const openEditUserEditor = (user: RuntimeUser) => {
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
    } catch (error: any) {
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
            refreshedUsers.find((user: RuntimeUser) => user.username === payload.username)?.id ||
            runtimeUsers.value.find((user: RuntimeUser) => user.username === payload.username)
              ?.id ||
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
          } catch (error: any) {
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
    } catch (error: any) {
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

  const toggleRuntimeUserStatus = async (user: RuntimeUser) => {
    if (!projectId.value || !user?.id) {
      return
    }

    const nextStatus = user.status === 'active' ? 'disabled' : 'active'
    if (nextStatus === 'disabled' && isDefaultRuntimeAdminUser(user)) {
      message.warning(t('projectManagement.runtimeAccess.users.adminStatusBlocked'))
      return
    }

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
    } catch (error: any) {
      message.error(
        t('projectManagement.runtimeAccess.users.toggleStatusFailed', {
          message: getErrorMessage(error, t('common.error')),
        }),
      )
    }
  }

  const deleteRuntimeUser = async (user: RuntimeUser) => {
    if (!projectId.value || !user?.id) {
      return
    }

    if (isDefaultRuntimeAdminUser(user)) {
      message.warning(t('projectManagement.runtimeAccess.users.adminDeleteBlocked'))
      return
    }

    try {
      await messageBox.confirm(
        t('projectManagement.runtimeAccess.users.deleteConfirmText', {
          username: user.username || user.displayName || user.id,
        }),
        t('projectManagement.runtimeAccess.users.deleteConfirmTitle'),
        { type: 'warning' },
      )
      await api.deleteRuntimeUser(projectId.value, user.id)
      message.success(t('projectManagement.runtimeAccess.users.deleteSuccess'))
      await Promise.all([loadRuntimeUsers(), loadRuntimeRoles()])
    } catch (error: any) {
      if (isRuntimeAccessDialogCancelled(error)) {
        return
      }

      message.error(
        t('projectManagement.runtimeAccess.users.deleteFailed', {
          message: getErrorMessage(error, t('common.error')),
        }),
      )
    }
  }

  const resetRuntimeUserPassword = async (user: RuntimeUser) => {
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
          inputValidator(inputValue: string) {
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
    } catch (error: any) {
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

  const openEditRoleEditor = (role: RuntimeRole) => {
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
    } catch (error: any) {
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
    } catch (error: any) {
      message.error(
        t('projectManagement.runtimeAccess.roles.saveFailed', {
          message: getErrorMessage(error, t('common.error')),
        }),
      )
    } finally {
      roleSubmitting.value = false
    }
  }

  const deleteRuntimeRole = async (role: RuntimeRole) => {
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
    } catch (error: any) {
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

  const resolveUserRoleNames = (user: RuntimeUser) => {
    const ids = normalizeIdList(user?.roleIds)
    if (ids.length > 0) {
      return ids.map((id) => roleNameMap.value.get(id) || id)
    }

    if (Array.isArray(user?.roles) && user.roles.length > 0) {
      return user.roles.map((role) => role?.name || role?.code || role?.id).filter(Boolean)
    }

    return []
  }

  const upsertRuntimeUser = (response: unknown) => {
    const user = extractSingle(response, ['runtimeUser', 'user'])
    return user ? normalizeRuntimeUserRecord(user) : null
  }

  const upsertRuntimeRole = (response: unknown) => {
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
    normalizeRuntimeRoleRecord,
    normalizeRuntimeUserRecord,
    upsertRuntimeUser,
    upsertRuntimeRole,
  }
}
