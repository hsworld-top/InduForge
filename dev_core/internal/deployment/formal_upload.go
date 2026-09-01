package deployment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/indu-forge/dev_core/internal/releasebuilder"
	"path"
)

func uploadFormalRelease(ctx context.Context, store ArtifactStore, project Project, version Version, result releasebuilder.Result) (VersionReadyInput, error) {
	key := path.Join("releases", project.TenantID, project.ID, version.Version, result.OuterSHA256+".tar.zst")
	ref, err := store.Put(ctx, key, bytes.NewReader(result.Bundle), result.Size, "application/zstd")
	if err != nil {
		return VersionReadyInput{}, err
	}
	if ref.Key != key || ref.Size != result.Size {
		return VersionReadyInput{}, fmt.Errorf("正式 Release 对象存储确认不匹配")
	}
	var manifest map[string]any
	if err = json.Unmarshal(result.Manifest, &manifest); err != nil {
		return VersionReadyInput{}, err
	}
	return VersionReadyInput{Bucket: ref.Bucket, ArtifactKey: key, ArtifactHash: result.OuterSHA256, ArtifactSize: result.Size, Manifest: manifest, ManifestHash: result.ManifestSHA256, ChecksumsHash: result.ChecksumsSHA256}, nil
}
