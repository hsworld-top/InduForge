import request from "@/utils/request";

export const assetApi = {
  getFolders(projectId) {
    return request.get(`/design/projects/${projectId}/asset-folders`);
  },
  createFolder(projectId, data) {
    return request.post(`/design/projects/${projectId}/asset-folders`, data);
  },
  renameFolder(projectId, folderId, payload = {}) {
    return request.patch(
      `/design/projects/${projectId}/asset-folders/${folderId}`,
      payload
    );
  },
  deleteFolder(projectId, folderId) {
    return request.delete(
      `/design/projects/${projectId}/asset-folders/${folderId}`
    );
  },
  updateAsset(projectId, assetId, payload = {}) {
    return request.patch(
      `/design/projects/${projectId}/assets/${assetId}`,
      payload
    );
  },
  copyAsset(projectId, assetId, payload = {}) {
    return request.post(
      `/design/projects/${projectId}/assets/${assetId}/copy`,
      payload
    );
  },
  getAssets(projectId, params = {}) {
    return request.get(`/design/projects/${projectId}/assets`, { params });
  },
  uploadAssets(projectId, files, folderId, options = {}, onProgress) {
    const form = new FormData();
    (files || []).forEach((file) => form.append("files", file));
    if (folderId) {
      form.append("folderId", folderId);
    }
    if (options?.conflictStrategy) {
      form.append("conflictStrategy", options.conflictStrategy);
    }
    return request.post(`/design/projects/${projectId}/assets`, form, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
      onUploadProgress: onProgress,
    });
  },
  deleteAsset(projectId, assetId) {
    return request.delete(`/design/projects/${projectId}/assets/${assetId}`);
  },
};

export default assetApi;