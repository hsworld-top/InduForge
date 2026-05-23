import { ref, shallowRef } from 'vue'

// 服务端分页通用 composable
// loader：接受 { page, pageSize, sort?, filter? } 返回 { list, total }

export interface PaginationParams {
  page: number
  pageSize: number
  sort?: { field: string; order: 'asc' | 'desc' } | null
  filter?: Record<string, unknown>
}

export interface PaginationResult<T> {
  list: T[]
  total: number
}

export function useServerPagination<T>(
  loader: (params: PaginationParams) => Promise<PaginationResult<T>>,
  initial: Partial<PaginationParams> = {},
) {
  const list = shallowRef<T[]>([])
  const total = ref(0)
  const page = ref(initial.page ?? 1)
  const pageSize = ref(initial.pageSize ?? 20)
  const sort = ref<PaginationParams['sort']>(initial.sort ?? null)
  const filter = ref<Record<string, unknown>>(initial.filter ?? {})
  const loading = ref(false)

  async function refresh() {
    loading.value = true
    try {
      const res = await loader({
        page: page.value,
        pageSize: pageSize.value,
        sort: sort.value,
        filter: filter.value,
      })
      list.value = res.list
      total.value = res.total
    } finally {
      loading.value = false
    }
  }

  function setPage(val: number) {
    page.value = val
    refresh()
  }

  function setPageSize(val: number) {
    pageSize.value = val
    page.value = 1
    refresh()
  }

  function setSort(val: PaginationParams['sort']) {
    sort.value = val
    page.value = 1
    refresh()
  }

  function setFilter(val: Record<string, unknown>) {
    filter.value = val
    page.value = 1
    refresh()
  }

  return {
    list,
    total,
    page,
    pageSize,
    loading,
    refresh,
    setPage,
    setPageSize,
    setSort,
    setFilter,
  }
}
