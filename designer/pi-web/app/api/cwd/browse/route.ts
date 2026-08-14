import { NextResponse } from "next/server";

// GET /api/cwd/browse?path=...：列出文件系统中的可读子目录。
export async function GET() {
  return NextResponse.json({ error: "Workspace browsing is disabled" }, { status: 404 });
}
