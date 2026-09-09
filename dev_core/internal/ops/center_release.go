package ops

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/indu-forge/dev_core/internal/objectstore"
	"github.com/klauspost/compress/zstd"
)

type centerReleaseObjects interface {
	Open(context.Context, string) (objectstore.ObjectReader, error)
}

type centerReleaseStore struct {
	objects             centerReleaseObjects
	localRoot, hostRoot string
}

func (r *KubernetesProjectReconciler) SetCenterReleaseStore(objects centerReleaseObjects, localRoot, hostRoot string) {
	r.centerReleases = &centerReleaseStore{objects: objects, localRoot: localRoot, hostRoot: hostRoot}
}

// prepare 只接受数据库中已验证的不可变制品描述。摘要校验先于解包，临时目录
// 与最终目录位于同一文件系统，完整落盘后原子发布；失败不会留下可运行目录。
func (s *centerReleaseStore) prepare(ctx context.Context, deploymentID string, release releaseMetadata) (string, error) {
	if s == nil || s.objects == nil || !filepath.IsAbs(s.localRoot) || !filepath.IsAbs(s.hostRoot) {
		return "", fmt.Errorf("中心制品存储未配置")
	}
	if !validUUID(deploymentID) || !validSHA256Hex(release.ArtifactHash) || release.ArtifactSize <= 0 || release.ArtifactSize > 512<<20 {
		return "", fmt.Errorf("中心制品描述无效")
	}
	relative := filepath.Join(deploymentID, "sha256-"+strings.ToLower(release.ArtifactHash))
	target, hostPath := filepath.Join(s.localRoot, relative), filepath.Join(s.hostRoot, relative)
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return hostPath, nil
	}
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(parent, ".preparing-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	archive, err := os.CreateTemp(parent, ".archive-")
	if err != nil {
		return "", err
	}
	defer os.Remove(archive.Name())
	defer archive.Close()
	object, err := s.objects.Open(ctx, release.ArtifactKey)
	if err != nil {
		return "", err
	}
	defer object.Reader.Close()
	if object.Size != release.ArtifactSize {
		return "", fmt.Errorf("中心制品大小不匹配")
	}
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(archive, hash), io.LimitReader(object.Reader, release.ArtifactSize+1))
	if err != nil {
		return "", err
	}
	if size != release.ArtifactSize || hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(release.ArtifactHash) {
		return "", fmt.Errorf("中心制品摘要不匹配")
	}
	if _, err = archive.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if err = unpackCenterRelease(ctx, archive, stage); err != nil {
		return "", err
	}
	if err = os.Chmod(stage, 0755); err != nil {
		return "", err
	}
	if err = os.Rename(stage, target); err != nil {
		return "", err
	}
	return hostPath, nil
}

func unpackCenterRelease(ctx context.Context, source io.Reader, target string) error {
	decoder, err := zstd.NewReader(source, zstd.WithDecoderMaxMemory(64<<20), zstd.WithDecoderConcurrency(1))
	if err != nil {
		return err
	}
	defer decoder.Close()
	reader := tar.NewReader(decoder)
	var total int64
	for count := 0; ; count++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := path.Clean(header.Name)
		if count >= 128 || name != header.Name || name == "." || strings.Contains(name, "\\") || path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return fmt.Errorf("中心制品包含非法路径或过多文件")
		}
		if header.Typeflag != tar.TypeReg || header.Size < 0 || header.Size > 512<<20 || total+header.Size > 512<<20 {
			return fmt.Errorf("中心制品包含非法文件类型或超出解包限制")
		}
		total += header.Size
		filename := filepath.Join(target, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		_, copyErr := io.CopyN(file, reader, header.Size)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
}
