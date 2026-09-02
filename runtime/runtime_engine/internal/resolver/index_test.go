package resolver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRejectsSymlinkInAnyParent(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "index.json"), []byte(`{"schemaVersion":"collector-runtime-index.v1","resources":{},"secrets":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(filepath.Join(link, "index.json"), false); err == nil {
		t.Fatal("parent symlink must fail closed")
	}
}

func TestSecretRejectsParentSymlink(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	secrets := filepath.Join(root, "secrets")
	if err := os.Mkdir(secrets, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secrets, "pg.json"), []byte(`{"schemaVersion":"postgres-dsn.v1","dsn":"postgres://safe"}`), 0600); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(root, "index.json")
	if err := os.WriteFile(index, []byte(`{"schemaVersion":"collector-runtime-index.v1","resources":{},"secrets":{"secret://site/db":"secrets/pg.json"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := Open(index, false)
	if err != nil {
		t.Fatal(err)
	}
	if dsn, err := r.ResolvePostgres(t.Context(), "secret://site/db"); err != nil || dsn != "postgres://safe" {
		t.Fatalf("nested secret must resolve before replacement: dsn=%q err=%v", dsn, err)
	}
	if err := os.RemoveAll(secrets); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, secrets); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolvePostgres(t.Context(), "secret://site/db"); err == nil {
		t.Fatal("secret parent symlink must fail closed")
	}
}

func TestSecretReferenceRejectsEscapesAndBackslashes(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"../pg.json", "..", `secrets\\pg.json`, "secrets/../pg.json"} {
		t.Run(relative, func(t *testing.T) {
			indexPath := filepath.Join(root, "index-"+strings.NewReplacer("/", "-", "\\", "-", ".", "dot").Replace(relative)+".json")
			body := `{"schemaVersion":"collector-runtime-index.v1","resources":{},"secrets":{"secret://site/db":"` + relative + `"}}`
			if err := os.WriteFile(indexPath, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(indexPath, false); err == nil {
				t.Fatalf("unsafe secret relative path accepted: %q", relative)
			}
		})
	}
}

func TestOpenRejectsMalformedReferenceKeysAndAcceptsSchemaMaximumMultiSegmentKey(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, reference := range []string{
		"site-resource://foo?x", "site-resource://foo#bar", "site-resource://", "secret://foo?x", "secret://foo#bar", "secret://", "site-resource://-bad", "secret://_bad",
	} {
		t.Run(reference, func(t *testing.T) {
			indexPath := filepath.Join(root, strings.NewReplacer(":", "-", "/", "-", "?", "-", "#", "-", "_", "-").Replace(reference)+".json")
			body := `{"schemaVersion":"collector-runtime-index.v1","resources":{"` + reference + `":{}},"secrets":{}}`
			if strings.HasPrefix(reference, "secret://") {
				body = `{"schemaVersion":"collector-runtime-index.v1","resources":{},"secrets":{"` + reference + `":"pg.json"}}`
			}
			if err := os.WriteFile(indexPath, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(indexPath, false); err == nil {
				t.Fatalf("malformed reference accepted: %q", reference)
			}
		})
	}
	validTail := "a/segment:with@schema.allowed-characters_" + strings.Repeat("z", 256-len("a/segment:with@schema.allowed-characters_"))
	valid := "site-resource://" + validTail
	indexPath := filepath.Join(root, "valid-multisegment.json")
	if err := os.WriteFile(indexPath, []byte(`{"schemaVersion":"collector-runtime-index.v1","resources":{"`+valid+`":{}},"secrets":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(indexPath, false); err != nil {
		t.Fatalf("schema-maximum multi-segment reference rejected: %v", err)
	}
}

func TestMostSpecificReadOnlyMount(t *testing.T) {
	mountInfo := []byte("36 25 0:32 / /work ro,relatime - tmpfs tmpfs rw\n" +
		"37 36 0:33 / /work/nested rw,relatime - tmpfs tmpfs rw\n")
	readOnly, err := mostSpecificReadOnlyMount("/work/index.json", mountInfo)
	if err != nil || !readOnly {
		t.Fatalf("parent ro mount = %v, %v", readOnly, err)
	}
	readOnly, err = mostSpecificReadOnlyMount("/work/nested/index.json", mountInfo)
	if err != nil || readOnly {
		t.Fatalf("nested rw mount must override parent ro: %v, %v", readOnly, err)
	}
	if _, err := mostSpecificReadOnlyMount("/uncovered/index.json", mountInfo); err == nil {
		t.Fatal("uncovered path must fail closed")
	}
}

func TestProductionIndexAndSecretUseSameReadOnlyMountProof(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(root, "index.json")
	secretPath := filepath.Join(root, "pg.json")
	if err := os.WriteFile(indexPath, []byte(`{"schemaVersion":"collector-runtime-index.v1","resources":{},"secrets":{"secret://site/db":"pg.json"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretPath, []byte(`{"schemaVersion":"postgres-dsn.v1","dsn":"postgres://safe"}`), 0600); err != nil {
		t.Fatal(err)
	}
	withMountProof(t, root, true)
	if _, _, err := secureRead(indexPath, true); err != nil {
		t.Fatalf("injected ro proof rejected index: %v", err)
	}
	index, err := Open(indexPath, true)
	if err != nil {
		t.Fatalf("ro index must open: %v", err)
	}
	if dsn, err := index.ResolvePostgres(context.Background(), "secret://site/db"); err != nil || dsn != "postgres://safe" {
		t.Fatalf("ro secret must resolve: dsn=%q err=%v", dsn, err)
	}
	// Switch the injected most-specific proof to rw after the index has opened:
	// secret loading must apply the same fail-closed rule as index loading.
	withMountProof(t, root, false)
	if _, err := index.ResolvePostgres(context.Background(), "secret://site/db"); err == nil {
		t.Fatal("rw secret mount must be rejected")
	}
}

func TestProductionResolvesProjectedSecretWithinIndexRoot(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	secrets := filepath.Join(root, "secrets")
	version := filepath.Join(secrets, "..2026_09_02")
	if err = os.MkdirAll(version, 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(version, "pg.json"), []byte(`{"schemaVersion":"postgres-dsn.v1","dsn":"postgres://safe"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("..2026_09_02", filepath.Join(secrets, "..data")); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("..data/pg.json", filepath.Join(secrets, "pg.json")); err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(root, "index.json")
	if err = os.WriteFile(indexPath, []byte(`{"schemaVersion":"collector-runtime-index.v1","resources":{},"secrets":{"secret://site/db":"secrets/pg.json"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	withMountProof(t, root, true)
	index, err := Open(indexPath, true)
	if err != nil {
		t.Fatal(err)
	}
	if dsn, err := index.ResolvePostgres(context.Background(), "secret://site/db"); err != nil || dsn != "postgres://safe" {
		t.Fatalf("projected secret must resolve: dsn=%q err=%v", dsn, err)
	}
	outside := filepath.Join(t.TempDir(), "pg.json")
	if err = os.WriteFile(outside, []byte(`{"schemaVersion":"postgres-dsn.v1","dsn":"postgres://outside"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(secrets, "pg.json")); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside, filepath.Join(secrets, "pg.json")); err != nil {
		t.Fatal(err)
	}
	if _, err = index.ResolvePostgres(context.Background(), "secret://site/db"); err == nil {
		t.Fatal("escaped projected secret must be rejected")
	}
}

func TestProductionRejectsWritableFileOnRWMount(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "index.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":"collector-runtime-index.v1","resources":{},"secrets":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	withMountProof(t, root, false)
	if _, err := Open(path, true); err == nil {
		t.Fatal("ordinary rw 0644 index must be rejected")
	}
}

func withMountProof(t *testing.T, root string, readOnly bool) {
	t.Helper()
	oldMountInfo, oldFDLink := mountInfoReader, fdLinkReader
	t.Cleanup(func() { mountInfoReader, fdLinkReader = oldMountInfo, oldFDLink })
	mode := "rw"
	if readOnly {
		mode = "ro"
	}
	mountInfoReader = func() ([]byte, error) {
		return []byte("36 25 0:32 / " + root + " " + mode + ",relatime - tmpfs tmpfs rw\n"), nil
	}
	fdLinkReader = func(file *os.File) (string, error) {
		// The production reader resolves /proc/self/fd.  The injected reader
		// returns the opened file name so the test stays portable on macOS while
		// still exercising the same inode and mount decision.
		return file.Name(), nil
	}
}
