import type { ComputeFolder, ComputeUnit } from "@/api/schemas/compute.schema";

export type ComputeFolderTreeNode = {
  id: string;
  name: string;
  parentId: string | null;
  children: ComputeFolderTreeNode[];
  units: ComputeUnit[];
};

const toId = (value: unknown) =>
  value === null || value === undefined || value === "" ? null : String(value);

export function buildComputeFolderTree(
  folders: ComputeFolder[],
  units: ComputeUnit[],
) {
  const folderMap = new Map<string, ComputeFolderTreeNode>();

  const visitFolder = (folder: ComputeFolder, parentId?: string | null) => {
    const id = String(folder.id);
    const node = folderMap.get(id) ?? {
      id,
      name: folder.name,
      parentId: toId(folder.parentId) ?? parentId ?? null,
      children: [],
      units: [],
    };

    node.name = folder.name;
    node.parentId = toId(folder.parentId) ?? parentId ?? null;
    folderMap.set(id, node);

    folder.children?.forEach((child) => visitFolder(child, id));
  };

  folders.forEach((folder) => visitFolder(folder));

  const rootUnits: ComputeUnit[] = [];
  const childIdSet = new Set<string>();
  folderMap.forEach((node) => {
    node.children = [];
    node.units = [];
  });

  folderMap.forEach((node) => {
    if (!node.parentId || !folderMap.has(node.parentId)) return;
    folderMap.get(node.parentId)?.children.push(node);
    childIdSet.add(node.id);
  });

  units.forEach((unit) => {
    const folderId = toId(unit.folderId);
    if (folderId && folderMap.has(folderId)) {
      folderMap.get(folderId)?.units.push(unit);
      return;
    }
    rootUnits.push(unit);
  });

  const rootFolders = [...folderMap.values()].filter(
    (node) => !childIdSet.has(node.id),
  );

  const sortByName = <T extends { name?: string }>(items: T[]) =>
    [...items].sort((a, b) => (a.name || "").localeCompare(b.name || ""));

  const sortNode = (node: ComputeFolderTreeNode): ComputeFolderTreeNode => ({
    ...node,
    children: sortByName(node.children).map(sortNode),
    units: sortByName(node.units),
  });

  return {
    rootFolders: sortByName(rootFolders).map(sortNode),
    rootUnits: sortByName(rootUnits),
  };
}

export function filterComputeTree(
  folders: ComputeFolderTreeNode[],
  units: ComputeUnit[],
  keyword: string,
) {
  const lower = keyword.trim().toLowerCase();
  if (!lower) return { folders, units };

  const matchesUnit = (unit: ComputeUnit) =>
    [unit.name, unit.path, unit.outputPath, unit.description]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(lower));

  const matchesFolder = (folder: ComputeFolderTreeNode) =>
    folder.name.toLowerCase().includes(lower);

  const filterFolder = (
    folder: ComputeFolderTreeNode,
  ): ComputeFolderTreeNode | null => {
    const childFolders = folder.children
      .map(filterFolder)
      .filter((child): child is ComputeFolderTreeNode => Boolean(child));
    const childUnits = folder.units.filter(matchesUnit);

    if (matchesFolder(folder) || childFolders.length > 0 || childUnits.length > 0) {
      return {
        ...folder,
        children: matchesFolder(folder) ? folder.children : childFolders,
        units: matchesFolder(folder) ? folder.units : childUnits,
      };
    }
    return null;
  };

  return {
    folders: folders
      .map(filterFolder)
      .filter((folder): folder is ComputeFolderTreeNode => Boolean(folder)),
    units: units.filter(matchesUnit),
  };
}
