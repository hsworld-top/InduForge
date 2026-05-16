import { Storage } from "@/utils/storage";

// 草稿读写，key 由 projectId + module + objectId 拼接，存 localStorage
// token / refreshToken 不进草稿，只存业务内容
// 用法：const draft = useDraft('project-1', 'alarm', 'rule-42')

export function useDraft<T>(projectId: string, module: string, objectId: string) {
  // 组合 key，格式：draft_v2:<projectId>:<module>:<objectId>
  const storageKey = `draft_v2:${projectId}:${module}:${objectId}`;

  function read(): T | null {
    return Storage.get(storageKey, null) as T | null;
  }

  function write(value: T) {
    Storage.set(storageKey, value);
  }

  function clear() {
    Storage.remove(storageKey);
  }

  /** 判断草稿是否存在（非 null/undefined） */
  function hasDraft(): boolean {
    return read() != null;
  }

  return { read, write, clear, hasDraft };
}
