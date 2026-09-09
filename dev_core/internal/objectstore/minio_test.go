package objectstore

import "testing"

func TestImageUploadUsesSmallSerialPartsOnlyForImages(t *testing.T) {
	for _, key := range []string{"node-images/sha256/abc.tar", "node-images/catalog.json"} {
		o := putOptions(key, "application/octet-stream")
		if o.NumThreads != 1 || o.PartSize != 8<<20 {
			t.Fatalf("%s: threads=%d part=%d", key, o.NumThreads, o.PartSize)
		}
	}
	o := putOptions("releases/abc.ifp", "application/octet-stream")
	if o.NumThreads != 0 || o.PartSize != 0 {
		t.Fatal("non-image upload behavior changed")
	}
}
