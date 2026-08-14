import { NextResponse } from "next/server";
import { getInduForgeWorkspaceRoot } from "@/lib/induforge-config";

export async function GET() {
  return NextResponse.json({ home: getInduForgeWorkspaceRoot() });
}
