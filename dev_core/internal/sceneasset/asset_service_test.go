package sceneasset

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"testing"
)

type archiveEntry struct {
	name    string
	content string
	mode    fs.FileMode
}

func makeAssetArchive(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		if entry.mode != 0 {
			header.SetMode(entry.mode)
		}
		writer, err := archive.CreateHeader(header)
		if err != nil {
			t.Fatalf("创建 ZIP 条目失败: %v", err)
		}
		if _, err := writer.Write([]byte(entry.content)); err != nil {
			t.Fatalf("写入 ZIP 条目失败: %v", err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("关闭 ZIP 失败: %v", err)
	}
	return output.Bytes()
}

func TestUnpackAssetCanonicalizesJSON(t *testing.T) {
	content := makeAssetArchive(t, archiveEntry{name: "symbols/pump.json", content: "{\n  \"name\": \"pump\"\n}"})
	files, err := unpackAsset(AssetImportInput{Filename: "pump.zip", ContentType: "application/zip", Content: content})
	if err != nil {
		t.Fatalf("解压资源失败: %v", err)
	}
	if got := string(files["symbols/pump.json"].Content); got != `{"name":"pump"}` {
		t.Fatalf("JSON 未规范化: got %q", got)
	}
}

func TestUnpackAssetRejectsUnsafeEntries(t *testing.T) {
	tests := []struct {
		name    string
		entries []archiveEntry
		want    error
	}{
		{name: "路径逃逸", entries: []archiveEntry{{name: "../pump.json", content: "{}"}}, want: ErrInvalidPath},
		{name: "重复路径", entries: []archiveEntry{{name: "pump.json", content: "{}"}, {name: "pump.json", content: "{}"}}, want: ErrDuplicateFile},
		{name: "文件祖先冲突", entries: []archiveEntry{{name: "textures", content: "file"}, {name: "textures/pump.png", content: "png"}}, want: ErrDuplicateFile},
		{name: "符号链接", entries: []archiveEntry{{name: "pump.json", content: "target", mode: fs.ModeSymlink | 0o777}}, want: ErrInvalidArchive},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content := makeAssetArchive(t, test.entries...)
			_, err := unpackAsset(AssetImportInput{Filename: "pump.zip", ContentType: "application/zip", Content: content})
			if !errors.Is(err, test.want) {
				t.Fatalf("错误类型不符: got %v, want %v", err, test.want)
			}
		})
	}
}

func TestInferAssetTypeFromFiles(t *testing.T) {
	tests := []struct {
		name string
		file string
		body string
		want AssetType
	}{
		{name: "image", file: "cover.png", want: AssetImage},
		{name: "font", file: "font.woff2", want: AssetFont},
		{name: "model", file: "scene.obj", want: AssetModel},
		{name: "material hint", file: "surface.json", body: `{"type":"material"}`, want: AssetMaterial},
		{name: "selection hint", file: "selection.json", body: `{"induforgeResourceType":"selection"}`, want: AssetSymbol},
		{name: "legacy json", file: "shape.json", body: `{"datas":[]}`, want: AssetSymbol},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := inferAssetType(map[string]ProviderFile{test.file: canonicalProviderFile(test.file, []byte(test.body), "")})
			if err != nil || got != test.want {
				t.Fatalf("inferAssetType() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}
