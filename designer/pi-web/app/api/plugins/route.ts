import { NextResponse } from "next/server";

function unavailable() {
  return NextResponse.json({ error: "Plugin management is disabled" }, { status: 404 });
}

export const GET = unavailable;
export const POST = unavailable;
