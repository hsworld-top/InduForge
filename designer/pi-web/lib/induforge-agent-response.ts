export function hideSystemPrompt<T>(value: T): T {
  if (!value || typeof value !== "object" || Array.isArray(value)) return value;
  const record = { ...(value as Record<string, unknown>) };
  delete record.systemPrompt;
  return record as T;
}
