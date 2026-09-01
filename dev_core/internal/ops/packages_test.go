package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilePackageStoreSharesBuildVersionAcrossPackages(t *testing.T) {
	directory := t.TempDir()
	writePackageVersion(t, directory, "1.2.3+build.20260831\n")
	writePackage(t, directory, "induforge-node-agent-linux-amd64.tar.gz")
	writePackage(t, directory, "induforge-node-agent-linux-arm64.tar.gz")
	writePackage(t, directory, "induforge-node-agent-windows.zip")

	items := NewFilePackageStore(directory).List()
	if len(items) != 3 {
		t.Fatalf("items=%d", len(items))
	}
	for _, item := range items {
		if !item.Available || item.Version != "1.2.3+build.20260831" {
			t.Fatalf("package must use shared build version: %#v", item)
		}
	}
	item, path, err := NewFilePackageStore(directory).Open(PackageLinuxARM64)
	if err != nil || item.Version != "1.2.3+build.20260831" || path != filepath.Join(directory, item.FileName) {
		t.Fatalf("open package=%#v path=%q err=%v", item, path, err)
	}
}

func TestFilePackageStoreFailsClosedWithoutValidVersionSidecar(t *testing.T) {
	for name, content := range map[string]string{
		"missing":       "",
		"multipleLines": "1.2.3\nextra\n",
		"invalidChars":  "1.2.3 / unsafe\n",
		"tooLong":       strings.Repeat("a", 129) + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			if name != "missing" {
				writePackageVersion(t, directory, content)
			}
			writePackage(t, directory, "induforge-node-agent-linux-amd64.tar.gz")
			writePackage(t, directory, "induforge-node-agent-linux-arm64.tar.gz")
			writePackage(t, directory, "induforge-node-agent-windows.zip")

			store := NewFilePackageStore(directory)
			for _, item := range store.List() {
				if item.Available || item.Version != "" {
					t.Fatalf("invalid sidecar must hide package: %#v", item)
				}
			}
			if _, _, err := store.Open(PlatformWindows); err == nil || !strings.Contains(err.Error(), "版本元数据") {
				t.Fatalf("invalid sidecar must reject download: %v", err)
			}
		})
	}
}

func TestFilePackageStoreRequiresEachPackageFile(t *testing.T) {
	directory := t.TempDir()
	writePackageVersion(t, directory, "1.2.3\n")
	writePackage(t, directory, "induforge-node-agent-linux-amd64.tar.gz")

	store := NewFilePackageStore(directory)
	items := store.List()
	if !items[0].Available || items[1].Available || items[2].Available || items[1].Version != "1.2.3" {
		t.Fatalf("unexpected package availability: %#v", items)
	}
	if _, _, err := store.Open(PlatformWindows); err == nil {
		t.Fatal("missing package file must not download")
	}
}

func TestFilePackageStoreRejectsSymlinkedPackage(t *testing.T) {
	directory := t.TempDir()
	writePackageVersion(t, directory, "1.2.3\n")
	target := filepath.Join(t.TempDir(), "package.tar.gz")
	if err := os.WriteFile(target, []byte("package"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "induforge-node-agent-linux-amd64.tar.gz")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symbolic link: %v", err)
	}

	store := NewFilePackageStore(directory)
	items := store.List()
	if items[0].Available {
		t.Fatalf("symbolic link must not be listed as a package: %#v", items[0])
	}
	if _, _, err := store.Open(PackageLinuxAMD64); err == nil {
		t.Fatal("symbolic link must not be downloadable")
	}
}

func writePackageVersion(t *testing.T, directory, value string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, packageVersionFile), []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}

func writePackage(t *testing.T, directory, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte("package"), 0600); err != nil {
		t.Fatal(err)
	}
}
