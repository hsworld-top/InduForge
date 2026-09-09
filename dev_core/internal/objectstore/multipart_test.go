package objectstore

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestImageMultipartUsesSerialEightMiBRequests(t *testing.T) {
	var mu sync.Mutex
	active, maxActive := 0, 0
	var sizes []int64
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Query().Has("uploads") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<InitiateMultipartUploadResult><Bucket>images</Bucket><Key>node-images/image.tar</Key><UploadId>one</UploadId></InitiateMultipartUploadResult>`)
			return
		}
		if r.Method == "PUT" && r.URL.Query().Get("uploadId") == "one" {
			mu.Lock()
			active++
			if active > maxActive {
				maxActive = active
			}
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			n, e := io.Copy(io.Discard, r.Body)
			if e != nil {
				t.Error(e)
			}
			mu.Lock()
			sizes = append(sizes, n)
			active--
			mu.Unlock()
			w.Header().Set("ETag", `"00000000000000000000000000000000"`)
			return
		}
		if r.Method == "POST" && r.URL.Query().Get("uploadId") == "one" {
			_, _ = io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<CompleteMultipartUploadResult><Bucket>images</Bucket><Key>node-images/image.tar</Key><ETag>00000000000000000000000000000000-3</ETag></CompleteMultipartUploadResult>`)
			return
		}
		t.Errorf("unexpected S3 request %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	}))
	defer s.Close()
	client, e := minio.New(strings.TrimPrefix(s.URL, "http://"), &minio.Options{Creds: credentials.NewStaticV4("", "", ""), Region: "us-east-1", BucketLookup: minio.BucketLookupPath})
	if e != nil {
		t.Fatal(e)
	}
	store := &MinIO{client: client, bucket: "images"}
	data := bytes.Repeat([]byte("x"), 17<<20)
	if _, e = store.Put(context.Background(), "node-images/image.tar", bytes.NewReader(data), int64(len(data)), "application/octet-stream"); e != nil {
		t.Fatal(e)
	}
	mu.Lock()
	defer mu.Unlock()
	if maxActive != 1 || len(sizes) != 3 {
		t.Fatalf("concurrency=%d sizes=%v", maxActive, sizes)
	}
	if fmt.Sprint(sizes) != fmt.Sprint([]int64{8 << 20, 8 << 20, 1 << 20}) {
		t.Fatal(sizes)
	}
}
