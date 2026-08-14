import { DefaultResourceLoader, getAgentDir } from "@earendil-works/pi-coding-agent";
import type { SkillInfo, SkillsResponse } from "@/lib/api-types";
import { annotateSkillsWithInstallInfo } from "@/lib/skill-lock";
import { getProjectSkillsDir, isInduForgeWorkspaceRoot } from "@/lib/induforge-config";

export async function loadSkillsWithInstallInfo(cwd: string): Promise<SkillsResponse> {
  if (!isInduForgeWorkspaceRoot(cwd)) throw new Error("Access denied");
  const agentDir = getAgentDir();
  const loader = new DefaultResourceLoader({
    cwd,
    agentDir,
    noExtensions: true,
    noSkills: true,
    noPromptTemplates: true,
    noThemes: true,
    additionalSkillPaths: [getProjectSkillsDir()],
    systemPromptOverride: () => undefined,
    appendSystemPromptOverride: () => [],
  });
  await loader.reload();
  const { skills, diagnostics } = loader.getSkills();
  const projectSkills = (skills as SkillInfo[]).filter(
    (skill) => skill.sourceInfo?.scope === "project" || skill.filePath.startsWith(getProjectSkillsDir()),
  );
  return {
    skills: annotateSkillsWithInstallInfo(projectSkills, { cwd, agentDir }),
    diagnostics,
    projectResourcesLoaded: true,
  };
}
