package gateway

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

const (
	clientArchiveName       = "client-assets.tar.zst"
	clientDigestMarkerName  = ".client-assets.sha256"
	maxClientArchiveBytes   = 256 << 20
	maxClientUnpackedBytes  = 512 << 20
	maxClientIndividualSize = 128 << 20
	maxClientFiles          = 10000
)

type releaseManifest struct {
	Artifacts struct {
		Client struct {
			File     string `json:"file"`
			Checksum string `json:"checksum"`
		} `json:"client"`
	} `json:"artifacts"`
}

type releaseChecksums struct {
	SchemaVersion string `json:"schemaVersion"`
	Files         []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
	} `json:"files"`
}

// PrepareClientAssets 只接受 NodeAgent 已验签并以只读卷挂载的 Release。它会再次
// 校验 client 子制品摘要，再以严格 tar 规则解到 emptyDir，避免网页入口解释任意路径。
func PrepareClientAssets(releaseRoot, clientRoot string) error {
	releaseRoot, err := safeDirectory(releaseRoot, "Release 根目录")
	if err != nil {
		return err
	}
	clientRoot, err = prepareTargetDirectory(clientRoot)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(releaseRoot, clientArchiveName)
	archive, err := regularFile(archivePath, maxClientArchiveBytes)
	if err != nil {
		return fmt.Errorf("读取 client 子制品: %w", err)
	}
	expected, err := declaredClientChecksum(releaseRoot, archive.Size())
	if err != nil {
		return err
	}
	actual, err := sha256File(archivePath)
	if err != nil {
		return fmt.Errorf("计算 client 子制品摘要: %w", err)
	}
	if actual != expected {
		return errors.New("client 子制品摘要与 Release 清单不一致")
	}
	if reusableClientRoot(clientRoot, expected) {
		return nil
	}
	staging, err := os.MkdirTemp(filepath.Dir(clientRoot), ".client-assets-")
	if err != nil {
		return fmt.Errorf("创建 client 临时目录: %w", err)
	}
	defer os.RemoveAll(staging)
	if err := unpackClientArchive(archivePath, staging); err != nil {
		return err
	}
	if err := validateClientRoot(staging); err != nil {
		return fmt.Errorf("client 子制品不完整: %w", err)
	}
	if err := os.WriteFile(filepath.Join(staging, clientDigestMarkerName), []byte(expected+"\n"), 0o444); err != nil {
		return fmt.Errorf("写入 client 完整标记: %w", err)
	}
	if err := os.RemoveAll(clientRoot); err != nil {
		return fmt.Errorf("替换 client 解包目录: %w", err)
	}
	if err := os.Rename(staging, clientRoot); err != nil {
		return fmt.Errorf("提交 client 解包目录: %w", err)
	}
	return nil
}

func reusableClientRoot(root, expected string) bool {
	marker, err := os.ReadFile(filepath.Join(root, clientDigestMarkerName))
	return err == nil && strings.TrimSpace(string(marker)) == expected && validateClientRoot(root) == nil
}

func safeDirectory(value, label string) (string, error) {
	root, err := filepath.Abs(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("解析%s: %w", label, err)
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%s必须是普通目录", label)
	}
	return root, nil
}

func prepareTargetDirectory(value string) (string, error) {
	root, err := filepath.Abs(strings.TrimSpace(value))
	if err != nil || root == string(filepath.Separator) {
		return "", errors.New("client 解包目录无效")
	}
	if info, statErr := os.Lstat(root); statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("client 解包目录必须是普通目录")
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", fmt.Errorf("读取 client 解包目录: %w", statErr)
	} else if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("创建 client 解包目录: %w", err)
	}
	return root, nil
}

func regularFile(path string, limit int64) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 0 || info.Size() > limit {
		return nil, errors.New("必须是受限大小的普通文件")
	}
	return info, nil
}

func declaredClientChecksum(root string, actualSize int64) (string, error) {
	manifestRaw, err := os.ReadFile(filepath.Join(root, "release-manifest.json"))
	if err != nil || len(manifestRaw) > 1<<20 {
		return "", errors.New("读取 Release manifest 失败")
	}
	var manifest releaseManifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil || manifest.Artifacts.Client.File != clientArchiveName {
		return "", errors.New("Release manifest 的 client 子制品无效")
	}
	checksum, ok := normalizedChecksum(manifest.Artifacts.Client.Checksum)
	if !ok {
		return "", errors.New("Release manifest 的 client 摘要无效")
	}
	checksumsRaw, err := os.ReadFile(filepath.Join(root, "checksums.json"))
	if err != nil || len(checksumsRaw) > 1<<20 {
		return "", errors.New("读取 Release checksums 失败")
	}
	var checksums releaseChecksums
	if err := json.Unmarshal(checksumsRaw, &checksums); err != nil || checksums.SchemaVersion != "release-checksums.v1" {
		return "", errors.New("Release checksums 无效")
	}
	for _, entry := range checksums.Files {
		if entry.Path != clientArchiveName {
			continue
		}
		declared, valid := normalizedChecksum(entry.SHA256)
		if !valid || entry.Size != actualSize || declared != checksum {
			return "", errors.New("Release checksums 的 client 条目无效")
		}
		return checksum, nil
	}
	return "", errors.New("Release checksums 缺少 client 子制品")
}

func normalizedChecksum(value string) (string, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "sha256:")
	if len(value) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(value); err != nil || value != strings.ToLower(value) {
		return "", false
	}
	return value, true
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func unpackClientArchive(archivePath, target string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder, err := zstd.NewReader(file)
	if err != nil {
		return errors.New("client 子制品 zstd 无效")
	}
	defer decoder.Close()
	reader := tar.NewReader(decoder)
	var total int64
	files := 0
	for {
		header, nextErr := reader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return errors.New("读取 client 子制品 tar 失败")
		}
		name, err := safeArchivePath(header.Name)
		if err != nil || header.Size < 0 || header.Size > maxClientIndividualSize {
			return errors.New("client 子制品包含非法文件")
		}
		destination := filepath.Join(target, filepath.FromSlash(name))
		if relative, err := filepath.Rel(target, destination); err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.New("client 子制品路径越界")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return fmt.Errorf("创建 client 目录: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			files++
			total += header.Size
			if files > maxClientFiles || total > maxClientUnpackedBytes {
				return errors.New("client 子制品解包超过限制")
			}
			if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
				return fmt.Errorf("创建 client 父目录: %w", err)
			}
			output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o444)
			if err != nil {
				return errors.New("client 子制品包含重复或不可写文件")
			}
			_, copyErr := io.Copy(output, io.LimitReader(reader, header.Size+1))
			closeErr := output.Close()
			if copyErr != nil || closeErr != nil {
				return errors.New("解开 client 文件失败")
			}
		default:
			return errors.New("client 子制品不允许链接或特殊文件")
		}
	}
	return nil
}

func safeArchivePath(value string) (string, error) {
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || strings.HasPrefix(value, "./") {
		return "", errors.New("非法路径")
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != value {
		return "", errors.New("非法路径")
	}
	return clean, nil
}
