import { NextResponse } from "next/server";
import { allowFileRoot } from "@/lib/file-access";
import { getInduForgeWorkspaceRoot } from "@/lib/induforge-config";

// POST /api/default-cwd
// Creates ~/pi-cwd-<YYYYMMDD> if it doesn't exist and returns the path.
export async function POST() {
  try {
    const workspaceRoot = getInduForgeWorkspaceRoot();
    allowFileRoot(workspaceRoot);
    return NextResponse.json({ cwd: workspaceRoot });
  } catch (error) {
    return NextResponse.json({ error: String(error) }, { status: 500 });
  }
}
