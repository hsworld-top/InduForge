/**
 * 选项卡属性配置工具。
 * 只处理可序列化的标签结构、默认激活值和子节点 tabKey 迁移。
 */

export interface TabsPanelItem extends Record<string, unknown> {
  name: string;
  label: string;
  disabled: boolean;
  content: string;
}

export interface TabsChildLike {
  id: string;
  props?: Record<string, unknown> | null;
}

export interface TabsChildKeyPatch {
  id: string;
  props: Record<string, unknown>;
}

const GENERATED_TAB_PREFIX = "tab";
const FALLBACK_TABS: TabsPanelItem[] = [
  { name: "tab1", label: "标签1", disabled: false, content: "" },
  { name: "tab2", label: "标签2", disabled: false, content: "" },
];

function normalizeText(value: unknown): string {
  return String(value ?? "").trim();
}

function getTabKey(item: Partial<TabsPanelItem> | Record<string, unknown>): string {
  return normalizeText(item.name ?? item.label ?? item.title);
}

function normalizeGeneratedTabLabel(label: string): string {
  const numericMatch = label.match(/^标签\s*(\d+)$/);
  if (numericMatch) return `标签${Number(numericMatch[1])}`;
  return label;
}

function getTabLabel(item: Partial<TabsPanelItem> | Record<string, unknown>, index: number): string {
  const label = normalizeText(item.label ?? item.title);
  return normalizeGeneratedTabLabel(label) || `标签${index + 1}`;
}

function getGeneratedIndex(value: string): number {
  const normalized = normalizeText(value);
  const match = normalized.match(/^tab(\d+)$/i);
  if (match) return Number(match[1]);
  const labelMatch = normalizeGeneratedTabLabel(normalized).match(/^标签(\d+)$/);
  return labelMatch ? Number(labelMatch[1]) : 0;
}

function getNextGeneratedIndex(items: unknown[]): number {
  let maxIndex = 0;
  let validCount = 0;
  for (const item of items) {
    if (!item || typeof item !== "object") continue;
    const record = item as Record<string, unknown>;
    const key = getTabKey(record);
    if (key) {
      validCount += 1;
      maxIndex = Math.max(maxIndex, getGeneratedIndex(key));
    }
    maxIndex = Math.max(maxIndex, getGeneratedIndex(normalizeText(record.label ?? record.title)));
  }
  return Math.max(maxIndex, validCount) + 1;
}

export function createUniqueTabsItemName(items: unknown, preferred?: string): string {
  const list = Array.isArray(items) ? items : [];
  const used = new Set(
    list
      .map((item) => (item && typeof item === "object" ? getTabKey(item as Record<string, unknown>) : ""))
      .filter(Boolean),
  );
  const normalizedPreferred = normalizeText(preferred);
  if (normalizedPreferred && !used.has(normalizedPreferred)) return normalizedPreferred;
  let index = getNextGeneratedIndex(list);
  while (used.has(`${GENERATED_TAB_PREFIX}${index}`)) {
    index += 1;
  }
  return `${GENERATED_TAB_PREFIX}${index}`;
}

export function normalizeTabsItems(input: unknown): TabsPanelItem[] {
  const source = Array.isArray(input) ? input : [];
  const used = new Set<string>();
  const normalized = source
    .map((item, index): TabsPanelItem | null => {
      if (!item || typeof item !== "object") return null;
      const record = item as Record<string, unknown>;
      const rawName = getTabKey(record);
      const name =
        rawName && !used.has(rawName)
          ? rawName
          : createUniqueTabsItemName([...used].map((value) => ({ name: value })));
      used.add(name);
      return {
        ...record,
        name,
        label: getTabLabel(record, index),
        disabled: Boolean(record.disabled),
        content: typeof record.content === "string" ? normalizeText(record.content) : "",
      };
    })
    .filter((item): item is TabsPanelItem => item !== null);

  if (normalized.length > 0) return normalized;
  return FALLBACK_TABS.map((item) => ({ ...item }));
}

export function createTabsItem(items: unknown, label?: string): TabsPanelItem {
  const normalizedItems = normalizeTabsItems(items);
  const name = createUniqueTabsItemName(normalizedItems);
  const suffix = Number(name.replace(GENERATED_TAB_PREFIX, ""));
  const fallbackIndex = Number.isFinite(suffix) && suffix > 0 ? suffix : normalizedItems.length + 1;
  return {
    name,
    label: normalizeText(label) || `标签${fallbackIndex}`,
    disabled: false,
    content: "",
  };
}

export function normalizeTabsModelValue(value: unknown, items: TabsPanelItem[]): string {
  const validKeys = new Set(items.map((item) => item.name).filter(Boolean));
  const normalized = normalizeText(value);
  if (normalized && validKeys.has(normalized)) return normalized;
  return items[0]?.name || "";
}

export function renameTabsItem(
  items: TabsPanelItem[],
  oldName: string,
  nextName: string,
):
  | { ok: false; reason: "emptyName" | "duplicateName" | "missingItem" }
  | { ok: true; oldName: string; newName: string; items: TabsPanelItem[] } {
  const normalizedOldName = normalizeText(oldName);
  const normalizedNextName = normalizeText(nextName);
  if (!normalizedNextName) return { ok: false, reason: "emptyName" };
  const targetIndex = items.findIndex((item) => item.name === normalizedOldName);
  if (targetIndex < 0) return { ok: false, reason: "missingItem" };
  if (items.some((item, index) => index !== targetIndex && item.name === normalizedNextName)) {
    return { ok: false, reason: "duplicateName" };
  }
  return {
    ok: true,
    oldName: normalizedOldName,
    newName: normalizedNextName,
    items: items.map((item, index) =>
      index === targetIndex ? { ...item, name: normalizedNextName } : item,
    ),
  };
}

export function buildTabsChildKeyPatches(
  children: TabsChildLike[],
  oldName: string,
  nextName: string,
): TabsChildKeyPatch[] {
  const normalizedOldName = normalizeText(oldName);
  const normalizedNextName = normalizeText(nextName);
  if (!normalizedOldName || !normalizedNextName || normalizedOldName === normalizedNextName) {
    return [];
  }
  return children
    .filter((child) => child?.props?.tabKey === normalizedOldName)
    .map((child) => ({
      id: child.id,
      props: { ...(child.props || {}), tabKey: normalizedNextName },
    }));
}

export function canRemoveTabsItem(
  name: string,
  children: TabsChildLike[],
): { ok: true } | { ok: false; reason: "hasChildren" } {
  const normalizedName = normalizeText(name);
  const hasChildren = children.some((child) => child?.props?.tabKey === normalizedName);
  return hasChildren ? { ok: false, reason: "hasChildren" } : { ok: true };
}
