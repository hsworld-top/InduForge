export type CollectorBrowsePriority = 'user' | 'prefetch'

type BrowseLoader<T> = () => Promise<T[]>

type BrowseJob<T> = {
  key: string
  contextKey: string
  epoch: number
  loader: BrowseLoader<T>
  promise: Promise<T[]>
  resolve: (value: T[]) => void
  reject: (reason?: unknown) => void
  started: boolean
  countedAsPrefetch: boolean
}

export class CollectorBrowseCache<T> {
  private readonly cache = new Map<string, T[]>()
  private readonly epochs = new Map<string, number>()
  private readonly jobs = new Map<string, BrowseJob<T>>()
  private readonly prefetchQueue: BrowseJob<T>[] = []
  private activePrefetchCount = 0

  constructor(private readonly maxPrefetchConcurrency = 3) {}

  get(
    contextKey: string,
    parentNodeId: string,
    priority: CollectorBrowsePriority,
    loader: BrowseLoader<T>,
  ): Promise<T[]> {
    const epoch = this.currentEpoch(contextKey)
    const key = this.buildKey(contextKey, epoch, parentNodeId)
    const cached = this.cache.get(key)
    if (cached) return Promise.resolve(cached)

    const existing = this.jobs.get(key)
    if (existing) {
      // 用户真实展开节点时直接启动排队中的预取任务，不等待后台并发槽位。
      if (priority === 'user' && !existing.started) this.start(existing, false)
      return existing.promise
    }

    const job = this.createJob(key, contextKey, epoch, loader)
    this.jobs.set(key, job)
    if (priority === 'user') this.start(job, false)
    else {
      this.prefetchQueue.push(job)
      this.drainPrefetchQueue()
    }
    return job.promise
  }

  peek(contextKey: string, parentNodeId: string) {
    const key = this.buildKey(contextKey, this.currentEpoch(contextKey), parentNodeId)
    return this.cache.get(key)
  }

  epoch(contextKey: string) {
    return this.currentEpoch(contextKey)
  }

  set(contextKey: string, parentNodeId: string, nodes: T[], epoch = this.currentEpoch(contextKey)) {
    if (epoch !== this.currentEpoch(contextKey)) return
    const key = this.buildKey(contextKey, epoch, parentNodeId)
    this.cache.set(key, nodes)
  }

  prefetch(contextKey: string, parentNodeIds: string[], loader: (nodeId: string) => Promise<T[]>) {
    return parentNodeIds.map((nodeId) =>
      this.get(contextKey, nodeId, 'prefetch', () => loader(nodeId)).catch(() => []),
    )
  }

  refresh(contextKey: string) {
    const nextEpoch = this.currentEpoch(contextKey) + 1
    this.epochs.set(contextKey, nextEpoch)
    const prefix = `${contextKey}\u0000`
    for (const key of this.cache.keys()) {
      if (key.startsWith(prefix)) this.cache.delete(key)
    }

    // 已经发出的请求无法安全取消；通过 epoch 阻止旧结果重新写入缓存。
    for (let index = this.prefetchQueue.length - 1; index >= 0; index--) {
      const job = this.prefetchQueue[index]
      if (job.contextKey !== contextKey) continue
      this.prefetchQueue.splice(index, 1)
      this.jobs.delete(job.key)
      job.resolve([])
    }
  }

  private createJob(
    key: string,
    contextKey: string,
    epoch: number,
    loader: BrowseLoader<T>,
  ): BrowseJob<T> {
    let resolve!: (value: T[]) => void
    let reject!: (reason?: unknown) => void
    const promise = new Promise<T[]>((nextResolve, nextReject) => {
      resolve = nextResolve
      reject = nextReject
    })
    return {
      key,
      contextKey,
      epoch,
      loader,
      promise,
      resolve,
      reject,
      started: false,
      countedAsPrefetch: false,
    }
  }

  private start(job: BrowseJob<T>, countedAsPrefetch: boolean) {
    if (job.started) return
    job.started = true
    job.countedAsPrefetch = countedAsPrefetch
    const queuedIndex = this.prefetchQueue.indexOf(job)
    if (queuedIndex >= 0) this.prefetchQueue.splice(queuedIndex, 1)
    if (countedAsPrefetch) this.activePrefetchCount += 1

    void job
      .loader()
      .then((nodes) => {
        if (job.epoch === this.currentEpoch(job.contextKey)) this.cache.set(job.key, nodes)
        job.resolve(nodes)
      })
      .catch(job.reject)
      .finally(() => {
        this.jobs.delete(job.key)
        if (job.countedAsPrefetch) this.activePrefetchCount -= 1
        this.drainPrefetchQueue()
      })
  }

  private drainPrefetchQueue() {
    while (
      this.activePrefetchCount < this.maxPrefetchConcurrency &&
      this.prefetchQueue.length > 0
    ) {
      const job = this.prefetchQueue.shift()
      if (job) this.start(job, true)
    }
  }

  private currentEpoch(contextKey: string) {
    return this.epochs.get(contextKey) || 0
  }

  private buildKey(contextKey: string, epoch: number, parentNodeId: string) {
    return `${contextKey}\u0000${epoch}\u0000${parentNodeId}`
  }
}
