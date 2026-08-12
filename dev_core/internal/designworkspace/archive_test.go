package designworkspace

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
)

func TestUploadArchiveImportsWithoutConflicts(t *testing.T) {
	service, actor, workspace := newArchiveTestService(t)
	content := makeArchive(t, map[string]string{
		"assets/pump.png":    "image",
		"displays/main.json": `{"image":"assets/pump.png"}`,
	})

	result, err := service.Execute(context.Background(), actor, testProjectID, mustAction(t, "upload", map[string]any{
		"path":    "import.zip",
		"content": archiveDataURL(content),
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertArchiveResult(t, result, archiveImportResult{
		Conflicts: []string{},
		Roots:     []string{"assets", "displays"},
	})
	assertFileContent(t, filepath.Join(workspace, "displays", "main.json"), `{"image":"assets/pump.png"}`)
	assertFileContent(t, filepath.Join(workspace, "assets", "pump.png"), "image")
}

func TestUploadArchiveConflictCanCancelOrOverwrite(t *testing.T) {
	service, actor, workspace := newArchiveTestService(t)
	target := filepath.Join(workspace, "scenes", "main.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	content := makeArchive(t, map[string]string{"scenes/main.json": "new"})

	pending := uploadArchiveForTest(t, service, actor, content)
	if !pending.ImportPending || pending.Path == "" {
		t.Fatalf("冲突导入应进入待确认状态: %#v", pending)
	}
	if len(pending.Conflicts) != 1 || pending.Conflicts[0] != "scenes/main.json" {
		t.Fatalf("冲突文件不正确: %#v", pending.Conflicts)
	}

	cancelResult, err := service.Execute(context.Background(), actor, testProjectID, mustAction(t, "import", map[string]any{
		"path": pending.Path,
		"move": false,
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertArchiveResult(t, cancelResult, archiveImportResult{Conflicts: []string{}, Roots: []string{"scenes"}})
	assertFileContent(t, target, "old")

	pending = uploadArchiveForTest(t, service, actor, content)
	commitResult, err := service.Execute(context.Background(), actor, testProjectID, mustAction(t, "import", map[string]any{
		"path": pending.Path,
		"move": true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	assertArchiveResult(t, commitResult, archiveImportResult{Conflicts: []string{}, Roots: []string{"scenes"}})
	assertFileContent(t, target, "new")
}

func TestUploadArchiveRejectsUnsafeEntries(t *testing.T) {
	service, actor, _ := newArchiveTestService(t)
	tests := []struct {
		name    string
		content []byte
		wantErr error
	}{
		{name: "路径越界", content: makeArchive(t, map[string]string{"../secret.txt": "x"}), wantErr: ErrInvalidPath},
		{name: "绝对路径", content: makeArchive(t, map[string]string{"/assets/file.txt": "x"}), wantErr: ErrInvalidPath},
		{name: "未知根目录", content: makeArchive(t, map[string]string{"other/file.txt": "x"}), wantErr: ErrInvalidPath},
		{name: "符号链接", content: makeSymlinkArchive(t, "assets/link"), wantErr: ErrSymbolicLink},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.Execute(context.Background(), actor, testProjectID, mustAction(t, "upload", map[string]any{
				"path":    "unsafe.zip",
				"content": archiveDataURL(test.content),
			}))
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("期望错误 %v，实际 %v", test.wantErr, err)
			}
		})
	}
}

func TestExtractArchiveRejectsFileCountAndSizeLimits(t *testing.T) {
	tooMany := makeArchiveWithCount(t, maxArchiveFiles+1)
	if _, _, err := extractArchive(tooMany, t.TempDir()); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("超出文件数量限制应被拒绝: %v", err)
	}

	tooLarge := makeSizedArchive(t, []int64{maxArchiveFileSize + 1})
	if _, _, err := extractArchive(tooLarge, t.TempDir()); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("超出单文件限制应被拒绝: %v", err)
	}

	totalTooLarge := makeSizedArchive(t, []int64{
		maxArchiveFileSize, maxArchiveFileSize, maxArchiveFileSize,
		maxArchiveFileSize, maxArchiveFileSize, maxArchiveFileSize,
		maxArchiveFileSize, maxArchiveFileSize, 1,
	})
	if _, _, err := extractArchive(totalTooLarge, t.TempDir()); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("超出总解压限制应被拒绝: %v", err)
	}
}

func TestExportArchiveCollectsHTDependencies(t *testing.T) {
	service, actor, workspace := newArchiveTestService(t)
	files := map[string]string{
		"displays/Main.json":      `{"model":"models/pump/pump.obj","nested":{"image":"assets/pump.png"}}`,
		"displays/Main.png":       "preview",
		"models/pump/pump.obj":    "mtllib pump.mtl\no pump",
		"models/pump/pump.mtl":    "newmtl pump\nmap_Kd ../../assets/pump-texture.png",
		"assets/pump.png":         "image",
		"assets/pump-texture.png": "texture",
	}
	for relative, content := range files {
		path := filepath.Join(workspace, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	archive, err := service.ExportArchive(context.Background(), actor, testProjectID, []string{"displays/Main.json"})
	if err != nil {
		t.Fatal(err)
	}
	if archive.Filename == "" || len(archive.Content) == 0 {
		t.Fatalf("导出结果不完整: %#v", archive)
	}
	entries := archiveEntries(t, archive.Content)
	want := []string{
		"assets/pump-texture.png",
		"assets/pump.png",
		"displays/Main.json",
		"displays/Main.png",
		"models/pump/pump.mtl",
		"models/pump/pump.obj",
	}
	if len(entries) != len(want) {
		t.Fatalf("导出文件数量不正确: got=%v want=%v", entries, want)
	}
	for index := range want {
		if entries[index] != want[index] {
			t.Fatalf("导出文件不正确: got=%v want=%v", entries, want)
		}
	}
}

func TestCollectArchiveFilesRejectsTotalSizeLimit(t *testing.T) {
	workspace := t.TempDir()
	for name, content := range map[string]string{"assets/a.bin": "1234", "assets/b.bin": "5678"} {
		path := filepath.Join(workspace, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := collectArchiveFilesWithLimit(workspace, []string{"assets"}, 7); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("导出总大小超限应被拒绝: %v", err)
	}
}

func TestCommitArchiveRejectsDirectoryConflict(t *testing.T) {
	workspace := t.TempDir()
	filesRoot := filepath.Join(workspace, ".induforge", "design-imports", "token", "files")
	source := filepath.Join(filesRoot, "assets", "pump.png")
	target := filepath.Join(workspace, "assets", "pump.png")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("image"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := commitArchive(workspace, filesRoot, []string{"assets/pump.png"}, true); !errors.Is(err, ErrDestinationBusy) {
		t.Fatalf("目录冲突不应被文件覆盖: %v", err)
	}
}

func newArchiveTestService(t *testing.T) (*Service, auth.User, string) {
	t.Helper()
	workspace := t.TempDir()
	service := NewService(fakeProjects{item: project.Project{
		ID: testProjectID, TenantID: "tenant", CreatedBy: "owner", Visibility: "private", WorkspacePath: workspace,
	}})
	return service, auth.User{ID: "owner", TenantID: "tenant", Role: "DEVELOPER"}, workspace
}

func uploadArchiveForTest(t *testing.T, service *Service, actor auth.User, content []byte) archiveImportResult {
	t.Helper()
	result, err := service.Execute(context.Background(), actor, testProjectID, mustAction(t, "upload", map[string]any{
		"path": "import.zip", "content": archiveDataURL(content),
	}))
	if err != nil {
		t.Fatal(err)
	}
	value, ok := result.(archiveImportResult)
	if !ok {
		t.Fatalf("ZIP upload 返回类型不正确: %T", result)
	}
	return value
}

func assertArchiveResult(t *testing.T, actual any, expected archiveImportResult) {
	t.Helper()
	value, ok := actual.(archiveImportResult)
	if !ok {
		t.Fatalf("归档返回类型不正确: %T", actual)
	}
	if value.ImportPending != expected.ImportPending || value.Path != expected.Path {
		t.Fatalf("归档状态不正确: got=%#v want=%#v", value, expected)
	}
	if !equalStrings(value.Conflicts, expected.Conflicts) || !equalStrings(value.Roots, expected.Roots) {
		t.Fatalf("归档列表不正确: got=%#v want=%#v", value, expected)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil || string(content) != expected {
		t.Fatalf("文件内容不正确: path=%s content=%q err=%v", path, content, err)
	}
}

func archiveDataURL(content []byte) string {
	return "data:application/zip;base64," + base64.StdEncoding.EncodeToString(content)
}

func makeArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for name, content := range files {
		destination, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(destination, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func makeArchiveWithCount(t *testing.T, count int) []byte {
	t.Helper()
	files := make(map[string]string, count)
	for index := 0; index < count; index++ {
		files["assets/file-"+strconv.Itoa(index)+".txt"] = ""
	}
	return makeArchive(t, files)
}

func makeSizedArchive(t *testing.T, sizes []int64) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for index, size := range sizes {
		destination, err := writer.Create("assets/file-" + strconv.Itoa(index) + ".bin")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.CopyN(destination, zeroReader{}, size); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

type zeroReader struct{}

func (zeroReader) Read(buffer []byte) (int, error) {
	clear(buffer)
	return len(buffer), nil
}

func makeSymlinkArchive(t *testing.T, name string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	header := &zip.FileHeader{Name: name}
	header.SetMode(os.ModeSymlink | 0o777)
	destination, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(destination, "target"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func archiveEntries(t *testing.T, content []byte) []string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatal(err)
	}
	entries := make([]string, 0, len(reader.File))
	for _, item := range reader.File {
		entries = append(entries, item.Name)
	}
	sort.Strings(entries)
	return entries
}
