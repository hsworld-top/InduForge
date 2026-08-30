package loader

import "testing"

func TestMountReadOnlyChoosesMostSpecificMount(t *testing.T) {
	mountInfo := "36 25 0:30 / / ro,relatime - overlay overlay ro\n40 36 0:31 / /release ro,relatime - tmpfs tmpfs ro\n41 40 0:32 / /release/work rw,relatime - tmpfs tmpfs rw\n"
	if err := mountReadOnly("/release/artifact.json", mountInfo); err != nil {
		t.Fatal(err)
	}
	if err := mountReadOnly("/release/work/artifact.json", mountInfo); err == nil {
		t.Fatal("expected nested rw mount rejection")
	}
}

func TestMountReadOnlyRejectsMissingCoverageAndEscapedPaths(t *testing.T) {
	mountInfo := "36 25 0:30 / /release ro,relatime - overlay overlay ro\n"
	if err := mountReadOnly("/other/a.json", mountInfo); err == nil {
		t.Fatal("expected missing mount")
	}
	if pathWithin("/release", "/release-other/a.json") {
		t.Fatal("path boundary escaped")
	}
}
