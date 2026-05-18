const HANDOFF_QUERY_KEYS = new Set(["handoff", "handoffId"]);

export function stripHandoffQuery(query: Record<string, unknown> = {}) {
  const nextQuery: Record<string, unknown> = {};
  Object.entries(query).forEach(([key, value]) => {
    if (HANDOFF_QUERY_KEYS.has(key)) return;
    nextQuery[key] = value;
  });
  return nextQuery;
}

export function hasHandoffQuery(query: Record<string, unknown> = {}) {
  return Object.keys(query).some((key) => HANDOFF_QUERY_KEYS.has(key));
}
