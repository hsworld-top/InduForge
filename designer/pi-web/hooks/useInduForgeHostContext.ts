"use client";

import { useEffect } from "react";
import { applyHostTheme } from "./useTheme";
import type { Locale } from "@/lib/i18n/types";
import { isInduForgeHostContext } from "@/lib/induforge-host-contract";

interface EmbeddedConfig {
  embedded: true;
  workspaceRoot: string;
  allowedParentOrigins: string[];
}

export function useInduForgeHostContext(setLocaleFromHost: (locale: Locale) => void): void {
  useEffect(() => {
    const projectId = new URL(window.location.href).searchParams.get("induforgeProjectId")?.trim();
    if (!projectId || window.parent === window) return;

    const controller = new AbortController();
    let allowedOrigins = new Set<string>();

    const handleMessage = (event: MessageEvent) => {
      if (event.source !== window.parent || !allowedOrigins.has(event.origin)) return;
      if (!isInduForgeHostContext(event.data) || event.data.projectId !== projectId) return;
      applyHostTheme(event.data.theme);
      setLocaleFromHost(event.data.locale === "en" ? "en" : "zh-CN");
    };

    window.addEventListener("message", handleMessage);
    void fetch("/api/induforge/config", { cache: "no-store", signal: controller.signal })
      .then((response) => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json() as Promise<EmbeddedConfig>;
      })
      .then((config) => {
        allowedOrigins = new Set(config.allowedParentOrigins);
        for (const origin of allowedOrigins) {
          window.parent.postMessage(
            { type: "INDUFORGE_PI_READY", version: 1, projectId },
            origin,
          );
        }
      })
      .catch((error) => {
        if (!controller.signal.aborted) console.error("Failed to initialize InduForge host context:", error);
      });

    return () => {
      controller.abort();
      window.removeEventListener("message", handleMessage);
    };
  }, [setLocaleFromHost]);
}
