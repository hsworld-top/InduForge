import { getInduForgePlatformExtensionsDir, getProjectSkillsDir } from "./induforge-config";

export function getInduForgeResourceLoaderOptions() {
  const platformExtensionsDir = getInduForgePlatformExtensionsDir();
  return {
    noExtensions: true,
    noSkills: true,
    noPromptTemplates: true,
    noThemes: true,
    additionalSkillPaths: [getProjectSkillsDir()],
    systemPromptOverride: () => undefined,
    appendSystemPromptOverride: () => [],
    ...(platformExtensionsDir ? { additionalExtensionPaths: [platformExtensionsDir] } : {}),
  };
}
