/**
 * 项目 API：页面与工程设置
 */

import type { PagesListPayload } from "@/types/api";
import request from "@/utils/request";

export interface CreatePageBody {
  name: string;
  type: string;
  parentId?: string | null;
  path?: string;
  schemaContent?: unknown;
}

export interface ProjectSettingsPayload {
  globalVariables?: unknown;
  globalScripts?: unknown;
}

export interface ProjectRuntimeRoleItem {
  id?: string;
  code?: string;
  name?: string;
  description?: string | null;
  status?: string;
}

export interface ProjectRuntimeRolesPayload {
  runtimeRoles?: ProjectRuntimeRoleItem[];
}

export interface ProjectRuntimeUserRoleItem extends ProjectRuntimeRoleItem {
  isSystem?: boolean;
}

export interface ProjectRuntimeUserItem {
  id?: string;
  username?: string;
  displayName?: string | null;
  status?: string;
  roleIds?: string[];
  roles?: ProjectRuntimeUserRoleItem[];
}

export interface ProjectRuntimeUsersPayload {
  runtimeUsers?: ProjectRuntimeUserItem[];
}

export const projectApi = {
  getPages(projectId: string) {
    return request.get<PagesListPayload>(`/design/projects/${projectId}/pages`);
  },

  getPage(projectId: string, pageId: string) {
    return request.get(`/design/projects/${projectId}/pages/${pageId}`);
  },

  createPage(projectId: string, data: CreatePageBody) {
    return request.post(`/design/projects/${projectId}/pages`, data);
  },

  updatePage(projectId: string, pageId: string, schema: unknown) {
    return request.put(`/design/projects/${projectId}/pages/${pageId}`, {
      schema,
    });
  },

  deletePage(projectId: string, pageId: string, mode?: "single" | "folder-only" | "cascade") {
    const query = mode ? `?mode=${encodeURIComponent(mode)}` : "";
    return request.delete(`/design/projects/${projectId}/pages/${pageId}${query}`);
  },

  renamePage(projectId: string, pageId: string, name: string, path?: string) {
    return request.patch(`/design/projects/${projectId}/pages/${pageId}/rename`, { name, path });
  },

  movePageToGroup(projectId: string, pageId: string, targetGroupId: string | null, path?: string) {
    return request.patch(`/design/projects/${projectId}/pages/${pageId}/move`, {
      parentId: targetGroupId,
      path,
    });
  },

  getProjectVariables(projectId: string) {
    return request.get<unknown>(`/design/projects/${projectId}/variables`);
  },

  updateProjectVariables(projectId: string, variables: unknown) {
    return request.put(`/design/projects/${projectId}/variables`, {
      variables,
    });
  },

  updateEntryConfig(projectId: string, entryConfig: unknown) {
    return request.put(`/design/projects/${projectId}/entry`, entryConfig);
  },

  getProjectSettings(projectId: string) {
    return request.get<ProjectSettingsPayload>(`/design/projects/${projectId}/settings`);
  },

  getRuntimeRoles(projectId: string) {
    return request.get<ProjectRuntimeRolesPayload>(`/projects/${projectId}/runtime-roles`);
  },

  getRuntimeUsers(projectId: string) {
    return request.get<ProjectRuntimeUsersPayload>(`/projects/${projectId}/runtime-users`);
  },

  updateProjectSettings(projectId: string, settings: unknown) {
    return request.put(`/design/projects/${projectId}/settings`, settings);
  },
};

export default projectApi;
