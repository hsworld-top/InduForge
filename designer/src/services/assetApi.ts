/**
 * 资源 API：文件夹与素材
 */

import type { AxiosProgressEvent } from "axios";
import request from "@/utils/request";
import type {
  AssetFoldersPayload,
  AssetsPayload,
  AssetItem,
  AssetFolder,
} from "@/types/api";

export type AssetMutationPayload = Record<string, unknown>;

export const assetApi = {
  getFolders(projectId: string) {
    return request.get<AssetFoldersPayload>(
      `/design/projects/${projectId}/asset-folders`,
    );
  },

  createFolder(projectId: string, data: AssetMutationPayload) {
    return request.post<AssetFolder>(
      `/design/projects/${projectId}/asset-folders`,
      data,
    );
  },

  renameFolder(
    projectId: string,
    folderId: string,
    payload: AssetMutationPayload = {},
  ) {
    return request.patch<AssetFolder>(
      `/design/projects/${projectId}/asset-folders/${folderId}`,
      payload,
    );
  },

  deleteFolder(projectId: string, folderId: string) {
    return request.delete(
      `/design/projects/${projectId}/asset-folders/${folderId}`,
    );
  },

  updateAsset(
    projectId: string,
    assetId: string,
    payload: AssetMutationPayload = {},
  ) {
    return request.patch<AssetItem>(
      `/design/projects/${projectId}/assets/${assetId}`,
      payload,
    );
  },

  copyAsset(
    projectId: string,
    assetId: string,
    payload: AssetMutationPayload = {},
  ) {
    return request.post<AssetItem>(
      `/design/projects/${projectId}/assets/${assetId}/copy`,
      payload,
    );
  },

  getAssets(projectId: string, params: Record<string, unknown> = {}) {
    return request.get<AssetsPayload>(`/design/projects/${projectId}/assets`, {
      params,
    });
  },

  uploadAssets(
    projectId: string,
    files: File[],
    folderId?: string,
    options: { conflictStrategy?: string } = {},
    onProgress?: (e: AxiosProgressEvent) => void,
  ) {
    const form = new FormData();
    (files || []).forEach((file) => form.append("files", file));
    if (folderId) {
      form.append("folderId", folderId);
    }
    if (options?.conflictStrategy) {
      form.append("conflictStrategy", options.conflictStrategy);
    }
    return request.post<AssetsPayload>(`/design/projects/${projectId}/assets`, form, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
      ...(onProgress ? { onUploadProgress: onProgress } : {}),
    });
  },

  deleteAsset(projectId: string, assetId: string) {
    return request.delete(`/design/projects/${projectId}/assets/${assetId}`);
  },
};

export default assetApi;
