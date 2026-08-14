import { NextResponse } from "next/server";
import {
  getInduForgeAllowedParentOrigins,
  getInduForgeWorkspaceRoot,
} from "@/lib/induforge-config";

export const dynamic = "force-dynamic";

export async function GET() {
  return NextResponse.json({
    embedded: true,
    workspaceRoot: getInduForgeWorkspaceRoot(),
    allowedParentOrigins: getInduForgeAllowedParentOrigins(),
  });
}
