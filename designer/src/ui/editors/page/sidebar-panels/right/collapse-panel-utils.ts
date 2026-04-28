/**
 * 折叠面板属性配置工具。
 * 这些函数只处理可序列化数据，便于属性面板、画布拖拽和单元测试共同复用。
 */

export interface CollapsePanelItem extends Record<string, unknown> {
  name: string;
  title: string;
  disabled: boolean;
  content: string;
}

export interface CollapseChildLike {
  id: string;
  props?: Record<string, unknown> | null;
}

export interface CollapseChildKeyPatch {
  id: string;
  props: Record<string, unknown>;
}

export type CollapseModelValue = string | string[];

const GENERATED_ITEM_PREFIX = "panel";
const FALLBACK_COLLAPSE_ITEMS: CollapsePanelItem[] = [
  { name: "1", title: "面板一", disabled: false, content: "内容一" },
  { name: "2", title: "面板二", disabled: false, content: "内容二" },
];

function normalizeText(value: unknown): string {
  return String(value ?? "").trim();
}

function getItemKey(item: Partial<CollapsePanelItem> | Record<string, unknown>): string {
  return normalizeText(item.name ?? item.title ?? item.label);
}

function getItemTitle(item: Partial<CollapsePanelItem> | Record<string, unknown>, index: number) {
  const title = normalizeText(item.title ?? item.label);
  return title || `面板 ${index + 1}`;
}

export function createUniqueCollapseItemName(items: unknown, preferred?: string): string {
  const list = Array.isArray(items) ? items : [];
  const used = new Set(
    list
      .map((item) => (item && typeof item === "object" ? getItemKey(item as Record<string, unknown>) : ""))
      .filter(Boolean),
  );
  const normalizedPreferred = normalizeText(preferred);
  if (normalizedPreferred && !used.has(normalizedPreferred)) {
    return normalizedPreferred;
  }
  let index = 1;
  while (used.has(`${GENERATED_ITEM_PREFIX}${index}`)) {
    index += 1;
  }
  return `${GENERATED_ITEM_PREFIX}${index}`;
}

export function normalizeCollapseItems(input: unknown): CollapsePanelItem[] {
  const source = Array.isArray(input) ? input : [];
  const used = new Set<string>();
  const normalized = source
    .map((item, index): CollapsePanelItem | null => {
      if (!item || typeof item !== "object") return null;
      const record = item as Record<string, unknown>;
      const rawName = getItemKey(record);
      const name = rawName && !used.has(rawName)
        ? rawName
        : createUniqueCollapseItemName([...used].map((value) => ({ name: value })));
      used.add(name);
      return {
        ...record,
        name,
        title: getItemTitle(record, index),
        disabled: Boolean(record.disabled),
        content: normalizeText(record.content),
      };
    })
    .filter((item): item is CollapsePanelItem => item !== null);

  if (normalized.length > 0) return normalized;
  return FALLBACK_COLLAPSE_ITEMS.map((item) => ({ ...item }));
}

export function createCollapseItem(items: unknown, title?: string): CollapsePanelItem {
  const normalizedItems = normalizeCollapseItems(items);
  const name = createUniqueCollapseItemName(normalizedItems);
  const suffix = name.replace(GENERATED_ITEM_PREFIX, "");
  return {
    name,
    title: normalizeText(title) || `面板 ${suffix || normalizedItems.length + 1}`,
    disabled: false,
    content: "",
  };
}

function getValidKeys(items: CollapsePanelItem[]): Set<string> {
  return new Set(items.map((item) => item.name).filter(Boolean));
}

function normalizeModelValueList(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map((item) => normalizeText(item)).filter(Boolean);
  }
  const normalized = normalizeText(value);
  return normalized ? [normalized] : [];
}

export function normalizeCollapseModelValue(
  value: unknown,
  items: CollapsePanelItem[],
  accordion: boolean,
): CollapseModelValue {
  const validKeys = getValidKeys(items);
  const values = normalizeModelValueList(value).filter((item) => validKeys.has(item));
  if (accordion) {
    return values[0] ?? "";
  }
  return Array.from(new Set(values));
}

export function renameCollapseItem(
  items: CollapsePanelItem[],
  oldName: string,
  nextName: string,
):
  | { ok: false; reason: "emptyName" | "duplicateName" | "missingItem" }
  | { ok: true; oldName: string; newName: string; items: CollapsePanelItem[] } {
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

export function buildCollapseChildKeyPatches(
  children: CollapseChildLike[],
  oldName: string,
  nextName: string,
): CollapseChildKeyPatch[] {
  const normalizedOldName = normalizeText(oldName);
  const normalizedNextName = normalizeText(nextName);
  if (!normalizedOldName || !normalizedNextName || normalizedOldName === normalizedNextName) {
    return [];
  }
  return children
    .filter((child) => child?.props?.collapseKey === normalizedOldName)
    .map((child) => ({
      id: child.id,
      props: { ...(child.props || {}), collapseKey: normalizedNextName },
    }));
}

export function canRemoveCollapseItem(
  name: string,
  children: CollapseChildLike[],
): { ok: true } | { ok: false; reason: "hasChildren" } {
  const normalizedName = normalizeText(name);
  const hasChildren = children.some((child) => child?.props?.collapseKey === normalizedName);
  return hasChildren ? { ok: false, reason: "hasChildren" } : { ok: true };
}
