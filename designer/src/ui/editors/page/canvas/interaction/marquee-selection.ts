export interface MarqueeRectLike {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

export interface MarqueeNodeLike {
  id: string;
  type?: string;
  children?: string[];
}

/**
 * 判断两个矩形是否相交。
 * @param {MarqueeRectLike} a - 矩形 A
 * @param {MarqueeRectLike} b - 矩形 B
 * @returns {boolean}
 */
export function rectsIntersect(a: MarqueeRectLike, b: MarqueeRectLike): boolean {
  return !(a.right < b.left || a.left > b.right || a.bottom < b.top || a.top > b.bottom);
}

interface CollectMarqueeNodeIdsOptions {
  rootId: string;
  marqueeRect: MarqueeRectLike;
  getNode: (id: string) => MarqueeNodeLike | null | undefined;
  getRect: (id: string) => MarqueeRectLike | null;
  isContainer: (type: string) => boolean;
}

/**
 * 收集与框选矩形相交的节点 ID。
 * 规则：
 * - 命中容器时优先向子节点下钻；若子节点无命中，则回退选中该容器。
 * - 支持“同级多命中 + 容器深钻”混合场景，避免只在单命中时才下钻。
 * @param {CollectMarqueeNodeIdsOptions} options - 收集参数
 * @returns {string[]}
 */
export function collectMarqueeNodeIds(options: CollectMarqueeNodeIdsOptions): string[] {
  const { rootId, marqueeRect, getNode, getRect, isContainer } = options;
  if (!rootId) return [];

  const collectFromParent = (parentId: string): string[] => {
    const parentNode = getNode(parentId);
    if (!parentNode) return [];
    const childIds = parentNode.children || [];
    const hits: string[] = [];

    for (const childId of childIds) {
      const childRect = getRect(childId);
      if (!childRect || !rectsIntersect(childRect, marqueeRect)) {
        continue;
      }
      const childNode = getNode(childId);
      const childType = String(childNode?.type || "");
      const canDive =
        Boolean(childNode) && isContainer(childType) && Boolean(childNode?.children?.length);
      if (canDive) {
        const nestedHits = collectFromParent(childId);
        if (nestedHits.length) {
          nestedHits.forEach((id) => {
            if (!hits.includes(id)) {
              hits.push(id);
            }
          });
          continue;
        }
      }
      if (!hits.includes(childId)) {
        hits.push(childId);
      }
    }

    return hits;
  };

  return collectFromParent(rootId);
}

