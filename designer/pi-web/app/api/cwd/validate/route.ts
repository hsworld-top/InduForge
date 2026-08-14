import { NextResponse } from "next/server";
import { allowFileRoot } from "@/lib/file-access";
import { getInduForgeWorkspaceRoot, isInduForgeWorkspaceRoot } from "@/lib/induforge-config";

// POST /api/cwd/validate  body: { cwd: string }
// Validates a candidate workspace before the UI selects it.
export async function POST(req: Request) {
  try {
    const body = await req.json() as { cwd?: unknown };
    const cwd = typeof body.cwd === "string" ? body.cwd.trim() : "";

    if (!cwd) {
      return NextResponse.json({ error: "Path is required" }, { status: 400 });
    }

    if (!isInduForgeWorkspaceRoot(cwd)) {
      return NextResponse.json({ error: "Only the configured workspace is available" }, { status: 403 });
    }

    const workspaceRoot = getInduForgeWorkspaceRoot();
    allowFileRoot(workspaceRoot);
    return NextResponse.json({ success: true, cwd: workspaceRoot });
  } catch (error) {
    return NextResponse.json({ error: String(error) }, { status: 500 });
  }
}
