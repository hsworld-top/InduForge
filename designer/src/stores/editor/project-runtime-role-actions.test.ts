import { describe, expect, it, vi } from "vitest";
import { ref } from "vue";
import { createBaseSchema } from "./normalize-schema";
import {
  applyRuntimeRoleCodesToSchema,
  loadProjectRuntimeRoleCodesForStore,
  normalizeProjectRuntimeRoleCodes,
} from "./project-runtime-role-actions";

describe("project-runtime-role-actions", () => {
  it("normalizes runtime role codes from api payload", () => {
    expect(
      normalizeProjectRuntimeRoleCodes({
        runtimeRoles: [
          { code: " runtime_admin " },
          { code: "operator" },
          { code: "runtime_admin" },
          { code: "" },
        ],
      }),
    ).toEqual(["runtime_admin", "operator"]);
  });

  it("overrides schema security roles with runtime role master data", () => {
    const schema = createBaseSchema("project-1");

    expect(schema.securityDecl.roles).toEqual(["admin", "operator", "viewer"]);

    applyRuntimeRoleCodesToSchema(schema, []);
    expect(schema.securityDecl.roles).toEqual([]);

    applyRuntimeRoleCodesToSchema(schema, ["runtime_admin", "auditor"]);
    expect(schema.securityDecl.roles).toEqual(["runtime_admin", "auditor"]);
  });

  it("falls back to empty roles when runtime role api fails", async () => {
    const runtimeRoleCodes = ref<string[]>(["stale"]);
    const runtimeRoleLoadState = ref<"idle" | "loading" | "ready" | "error">("idle");
    const runtimeRoleLoadError = ref("");
    const activeProjectId = ref("project-1");
    const runtimeRoleRequestSerial = ref(0);

    const result = await loadProjectRuntimeRoleCodesForStore(
      "project-1",
      {
        getRuntimeRoles: vi.fn().mockRejectedValue(new Error("403")),
      },
      {
        activeProjectId,
        runtimeRoleCodes,
        runtimeRoleLoadState,
        runtimeRoleLoadError,
        runtimeRoleRequestSerial,
      },
    );

    expect(result).toEqual({ accepted: true, runtimeRoleCodes: [] });
    expect(runtimeRoleCodes.value).toEqual([]);
    expect(runtimeRoleLoadState.value).toBe("error");
    expect(runtimeRoleLoadError.value).toContain("403");
  });

  it("keeps empty role list when api succeeds but project has no runtime roles", async () => {
    const runtimeRoleCodes = ref<string[]>(["stale"]);
    const runtimeRoleLoadState = ref<"idle" | "loading" | "ready" | "error">("idle");
    const runtimeRoleLoadError = ref("");
    const activeProjectId = ref("project-1");
    const runtimeRoleRequestSerial = ref(0);

    const result = await loadProjectRuntimeRoleCodesForStore(
      "project-1",
      {
        getRuntimeRoles: vi.fn().mockResolvedValue({
          runtimeRoles: [],
        }),
      },
      {
        activeProjectId,
        runtimeRoleCodes,
        runtimeRoleLoadState,
        runtimeRoleLoadError,
        runtimeRoleRequestSerial,
      },
    );

    expect(result).toEqual({ accepted: true, runtimeRoleCodes: [] });
    expect(runtimeRoleCodes.value).toEqual([]);
    expect(runtimeRoleLoadState.value).toBe("ready");
    expect(runtimeRoleLoadError.value).toBe("");
  });

  it("ignores late response from previous project and keeps latest project roles", async () => {
    let resolveProjectA!: (value: unknown) => void;
    let resolveProjectB!: (value: unknown) => void;
    const runtimeRoleCodes = ref<string[]>([]);
    const runtimeRoleLoadState = ref<"idle" | "loading" | "ready" | "error">("idle");
    const runtimeRoleLoadError = ref("");
    const activeProjectId = ref("project-a");
    const runtimeRoleRequestSerial = ref(0);
    const api = {
      getRuntimeRoles: vi.fn((projectId: string) => {
        return new Promise((resolve) => {
          if (projectId === "project-a") {
            resolveProjectA = resolve;
            return;
          }
          resolveProjectB = resolve;
        });
      }),
    };

    const projectAPromise = loadProjectRuntimeRoleCodesForStore("project-a", api, {
      activeProjectId,
      runtimeRoleCodes,
      runtimeRoleLoadState,
      runtimeRoleLoadError,
      runtimeRoleRequestSerial,
    });

    activeProjectId.value = "project-b";

    const projectBPromise = loadProjectRuntimeRoleCodesForStore("project-b", api, {
      activeProjectId,
      runtimeRoleCodes,
      runtimeRoleLoadState,
      runtimeRoleLoadError,
      runtimeRoleRequestSerial,
    });

    if (!resolveProjectB) {
      throw new Error("project-b resolver should be ready");
    }
    resolveProjectB({ runtimeRoles: [{ code: "role_b" }] });
    const projectBResult = await projectBPromise;

    if (!resolveProjectA) {
      throw new Error("project-a resolver should be ready");
    }
    resolveProjectA({ runtimeRoles: [{ code: "role_a" }] });
    const projectAResult = await projectAPromise;

    expect(projectBResult).toEqual({ accepted: true, runtimeRoleCodes: ["role_b"] });
    expect(projectAResult).toEqual({ accepted: false, runtimeRoleCodes: ["role_a"] });
    expect(runtimeRoleCodes.value).toEqual(["role_b"]);
    expect(runtimeRoleLoadState.value).toBe("ready");
    expect(runtimeRoleLoadError.value).toBe("");
  });
});
