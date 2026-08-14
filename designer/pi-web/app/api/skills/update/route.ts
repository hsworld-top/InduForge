import { NextResponse } from "next/server";
import { runNpx } from "@/lib/npx";
import { buildSkillUpdateArgs } from "@/lib/skill-updates";
import { loadSkillsWithInstallInfo } from "@/lib/skills-service";
import { getAllowedFileRoots, isExistingFilePathAllowed } from "@/lib/file-access";

export const dynamic = "force-dynamic";

export async function POST(req: Request) {
  try {
    const body = await req.json() as {
      cwd?: unknown;
      package?: unknown;
      scope?: unknown;
    };
    const cwd = typeof body.cwd === "string" ? body.cwd : "";
    const pkg = typeof body.package === "string" ? body.package : "";
    const scope = body.scope === "project" ? "project" : undefined;
    if (!cwd || !pkg || !scope) {
      return NextResponse.json({ error: "cwd, package, and scope are required" }, { status: 400 });
    }
    if (scope !== "project") {
      return NextResponse.json({ error: "Only project skills are supported" }, { status: 403 });
    }
    const allowedRoots = await getAllowedFileRoots();
    if (!isExistingFilePathAllowed(cwd, allowedRoots)) {
      return NextResponse.json({ error: "Access denied" }, { status: 403 });
    }

    const { skills } = await loadSkillsWithInstallInfo(cwd);
    const skill = skills.find(
      (item) => item.install?.package === pkg && item.install.scope === scope,
    );
    if (!skill?.install) {
      return NextResponse.json({ error: "Installed skill not found" }, { status: 404 });
    }
    if (!skill.install.canCheckForUpdates) {
      return NextResponse.json({ error: "This skill cannot be updated automatically" }, { status: 400 });
    }

    const { stdout, stderr } = await runNpx(buildSkillUpdateArgs(skill.install), {
      timeout: 60_000,
      cwd,
      env: { ...process.env, FORCE_COLOR: "0" },
    });

    const refreshed = await loadSkillsWithInstallInfo(cwd);
    const updatedSkill = refreshed.skills.find(
      (item) => item.install?.package === pkg && item.install.scope === scope,
    );
    return NextResponse.json({
      success: true,
      skill: updatedSkill,
      output: `${stdout}${stderr}`.slice(-500),
    });
  } catch (error: unknown) {
    const detail = error as { stdout?: string; stderr?: string; message?: string };
    const output = `${detail.stdout ?? ""}${detail.stderr ?? ""}`;
    return NextResponse.json(
      { error: output || detail.message || String(error) },
      { status: 500 },
    );
  }
}
