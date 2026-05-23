import { describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { loadProjectForStore } from './project-load-actions'
import { createBaseSchema } from './normalize-schema'

describe('project-load-actions', () => {
  function buildProjectLoadContext(overrides: Record<string, unknown> = {}) {
    return {
      projectId: ref(''),
      projectLoadRequestSerial: ref(0),
      projectLoadInvalidatorProjectIds: ref<Record<number, string>>({}),
      isLoading: ref(false),
      error: ref(''),
      doc: shallowRef(null),
      currentPageId: ref(''),
      loadProjectSettings: vi.fn().mockResolvedValue(undefined),
      loadProjectRuntimeRoles: vi.fn(),
      releasePageLock: vi.fn().mockResolvedValue(undefined),
      refreshPages: vi.fn().mockResolvedValue({
        pages: [{ id: 'page-1', name: '首页', path: '/' }],
        entryConfig: { homePageId: 'page-1' },
      }),
      createHomePage: vi.fn(),
      initEditor: vi.fn(),
      createBaseSchema,
      resolveLandingPageId: vi.fn().mockReturnValue('page-1'),
      projectApi: {
        getPage: vi.fn().mockResolvedValue({
          page: {
            id: 'page-1',
            name: '首页',
            path: '/',
            rootNodeId: 'node-root',
            config: { width: 1920, height: 1080 },
          },
          nodesById: {
            'node-root': {
              id: 'node-root',
              type: 'FreeContainer',
              props: {},
              style: {},
              bindings: {},
              permissions: {},
              events: {},
              children: [],
              layoutItem: null,
            },
          },
          graphicsById: {},
        }),
      },
      resolveProjectSchema: vi.fn().mockImplementation((payload, projectId, pageId) => {
        return {
          ...createBaseSchema(projectId),
          pagesById: {
            [pageId]: {
              id: pageId,
              name: '首页',
              path: '/',
              target: 'pc',
              logicalId: 'logic-1',
              isDefaultTarget: true,
              rootNodeId: 'node-root',
              graphicsIds: [],
              config: {
                width: 1920,
                height: 1080,
                fitMode: 'contain',
                showGrid: false,
                enableSnap: true,
                autoFit: true,
                background: { kind: 'color', value: '#ffffff' },
              },
            },
          },
          nodesById: {
            'node-root': {
              id: 'node-root',
              type: 'FreeContainer',
              label: '画布',
              props: {},
              style: {},
              bindings: {},
              permissions: {},
              events: {},
              children: [],
              layoutItem: null,
            },
          },
          graphicsById: {},
          entry: {
            homePageId: pageId,
          },
        }
      }),
      applyRuntimeRoleCodesToSchema: vi.fn((schema) => schema),
      ...overrides,
    }
  }

  it('does not block project load when runtime role request is still pending', async () => {
    const pendingRuntimeRoles = new Promise(() => {})
    const loadProjectRuntimeRoles = vi.fn().mockReturnValue(pendingRuntimeRoles)
    const ctx = buildProjectLoadContext({
      loadProjectRuntimeRoles,
    })

    const resultPromise = loadProjectForStore('project-1', ctx)

    const result = await Promise.race([
      resultPromise,
      new Promise<'timeout'>((resolve) => setTimeout(() => resolve('timeout'), 50)),
    ])

    expect(result).toEqual({ ok: true })
    expect(loadProjectRuntimeRoles).toHaveBeenCalledWith('project-1')
    expect(ctx.initEditor).toHaveBeenCalledTimes(1)
  })

  it('ignores stale project load result when two projects resolve out of order', async () => {
    let resolveProjectA!: (value: unknown) => void
    let resolveProjectB!: (value: unknown) => void
    const ctx = buildProjectLoadContext({
      refreshPages: vi
        .fn()
        .mockResolvedValueOnce({
          pages: [{ id: 'page-a', name: '工程A首页', path: '/a' }],
          entryConfig: { homePageId: 'page-a' },
        })
        .mockResolvedValueOnce({
          pages: [{ id: 'page-b', name: '工程B首页', path: '/b' }],
          entryConfig: { homePageId: 'page-b' },
        }),
      resolveLandingPageId: vi.fn().mockReturnValueOnce('page-a').mockReturnValueOnce('page-b'),
      projectApi: {
        getPage: vi.fn((projectId: string) => {
          return new Promise((resolve) => {
            if (projectId === 'project-a') {
              resolveProjectA = resolve
              return
            }
            resolveProjectB = resolve
          })
        }),
      },
    })

    const projectAPromise = loadProjectForStore('project-a', ctx)
    await new Promise((resolve) => setTimeout(resolve, 0))

    const projectBPromise = loadProjectForStore('project-b', ctx)
    await new Promise((resolve) => setTimeout(resolve, 0))

    resolveProjectB({
      page: {
        id: 'page-b',
        name: '工程B首页',
        path: '/b',
        rootNodeId: 'node-b',
        config: { width: 1920, height: 1080 },
      },
      nodesById: {
        'node-b': {
          id: 'node-b',
          type: 'FreeContainer',
          props: {},
          style: {},
          bindings: {},
          permissions: {},
          events: {},
          children: [],
          layoutItem: null,
        },
      },
      graphicsById: {},
    })
    await projectBPromise

    resolveProjectA({
      page: {
        id: 'page-a',
        name: '工程A首页',
        path: '/a',
        rootNodeId: 'node-a',
        config: { width: 1920, height: 1080 },
      },
      nodesById: {
        'node-a': {
          id: 'node-a',
          type: 'FreeContainer',
          props: {},
          style: {},
          bindings: {},
          permissions: {},
          events: {},
          children: [],
          layoutItem: null,
        },
      },
      graphicsById: {},
    })
    await projectAPromise

    expect(ctx.projectId.value).toBe('project-b')
    expect(ctx.currentPageId.value).toBe('page-b')
    expect(ctx.initEditor).toHaveBeenCalledTimes(1)
    expect((ctx.projectApi.getPage as ReturnType<typeof vi.fn>).mock.calls).toEqual([
      ['project-a', 'page-a'],
      ['project-b', 'page-b'],
    ])
    const initializedSchema = (ctx.initEditor as ReturnType<typeof vi.fn>).mock.calls[0]?.[0]
    expect(initializedSchema?.project.projectId).toBe('project-b')
  })

  it('同一工程重入后再切换工程时，旧请求的首页回滚策略仍冻结为不回滚远端', async () => {
    let resolveFirstHomePage!: (value: { ok: boolean; pageId?: string }) => void
    const createHomePage = vi
      .fn()
      .mockImplementationOnce(() => {
        return new Promise<{ ok: boolean; pageId?: string }>((resolve) => {
          resolveFirstHomePage = resolve
        })
      })
      .mockResolvedValue({ ok: true, pageId: 'page-a' })
    const projectApi = {
      getPage: vi.fn().mockImplementation(async (projectId: string, pageId: string) => ({
        page: {
          id: pageId,
          name: `${projectId}-${pageId}`,
          path: `/${pageId}`,
          rootNodeId: `node-${pageId}`,
          config: { width: 1920, height: 1080 },
        },
        nodesById: {
          [`node-${pageId}`]: {
            id: `node-${pageId}`,
            type: 'FreeContainer',
            props: {},
            style: {},
            bindings: {},
            permissions: {},
            events: {},
            children: [],
            layoutItem: null,
          },
        },
        graphicsById: {},
      })),
    }
    const ctx = buildProjectLoadContext({
      refreshPages: vi
        .fn()
        .mockResolvedValueOnce({
          pages: [],
          entryConfig: {},
        })
        .mockResolvedValueOnce({
          pages: [{ id: 'page-a', name: '工程A首页', path: '/a' }],
          entryConfig: { homePageId: 'page-a' },
        })
        .mockResolvedValueOnce({
          pages: [{ id: 'page-b', name: '工程B首页', path: '/b' }],
          entryConfig: { homePageId: 'page-b' },
        }),
      resolveLandingPageId: vi.fn().mockReturnValueOnce('page-a').mockReturnValueOnce('page-b'),
      createHomePage,
      projectApi,
    })

    const firstLoad = loadProjectForStore('project-a', ctx)
    await new Promise((resolve) => setTimeout(resolve, 0))

    const firstRollbackPolicy = (createHomePage as ReturnType<typeof vi.fn>).mock.calls[0]?.[2]

    const secondLoad = loadProjectForStore('project-a', ctx)
    await new Promise((resolve) => setTimeout(resolve, 0))

    const thirdLoad = loadProjectForStore('project-b', ctx)
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(typeof firstRollbackPolicy).toBe('function')
    expect(firstRollbackPolicy()).toBe(false)

    resolveFirstHomePage({ ok: true, pageId: 'page-a' })
    await Promise.all([firstLoad, secondLoad, thirdLoad])
  })
})
