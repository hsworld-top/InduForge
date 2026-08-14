"use client";

import { useSyncExternalStore } from "react";

export type ResolvedTheme = "light" | "dark";

const listeners = new Set<() => void>();
let theme: ResolvedTheme = "light";

function applyDomTheme(next: ResolvedTheme): void {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  root.classList.toggle("dark", next === "dark");
  root.dataset.theme = next;
  root.style.colorScheme = next;
}

export function applyHostTheme(next: ResolvedTheme): void {
  const changed = theme !== next;
  theme = next;
  applyDomTheme(next);
  if (changed) listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  applyDomTheme(theme);
  return () => listeners.delete(listener);
}

function getSnapshot(): ResolvedTheme {
  return theme;
}

export function useTheme() {
  const current = useSyncExternalStore(subscribe, getSnapshot, () => "light" as const);
  return {
    theme: current,
    isDark: current === "dark",
  };
}
