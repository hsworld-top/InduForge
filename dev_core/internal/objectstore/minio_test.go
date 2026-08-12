package objectstore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestMinIOPresignGetUsesSignatureWithoutSecret(t *testing.T) {
	const secretKey = "secret-key-must-not-leak"
	client, err := minio.New("objects.example.com", &minio.Options{
		Creds:  credentials.NewStaticV4("access-key", secretKey, ""),
		Secure: true,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	store := &MinIO{client: client, bucket: "private-bucket"}
	url, err := store.PresignGet(context.Background(), "tenant/logo.png", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "X-Amz-Signature=") || !strings.Contains(url, "X-Amz-Expires=3600") {
		t.Fatalf("签名 URL 参数不完整: %s", url)
	}
	if strings.Contains(url, secretKey) {
		t.Fatalf("签名 URL 泄露 Secret Key: %s", url)
	}
}
