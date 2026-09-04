package objectstore

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

type MinIO struct {
	client *minio.Client
	bucket string
}

type ObjectRef struct {
	Bucket      string
	Key         string
	Size        int64
	ContentType string
}

type ObjectReader struct {
	Bucket      string
	Reader      io.ReadCloser
	Size        int64
	ContentType string
}

func NewMinIO(ctx context.Context, config Config) (*MinIO, error) {
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
		Region: config.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("创建对象存储客户端失败: %w", err)
	}
	exists, err := client.BucketExists(ctx, config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("检查对象存储桶失败: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, config.Bucket, minio.MakeBucketOptions{Region: config.Region}); err != nil {
			return nil, fmt.Errorf("创建对象存储桶失败: %w", err)
		}
	}
	return &MinIO{client: client, bucket: config.Bucket}, nil
}

func (store *MinIO) Put(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) (ObjectRef, error) {
	info, err := store.client.PutObject(ctx, store.bucket, objectKey, reader, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return ObjectRef{}, fmt.Errorf("上传对象失败: %w", err)
	}
	return ObjectRef{Bucket: store.bucket, Key: objectKey, Size: info.Size, ContentType: contentType}, nil
}

func (store *MinIO) PresignGet(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		return "", fmt.Errorf("签名有效期必须大于 0")
	}
	signed, err := store.client.PresignedGetObject(ctx, store.bucket, objectKey, ttl, nil)
	if err != nil {
		return "", fmt.Errorf("生成对象下载地址失败: %w", err)
	}
	return signed.String(), nil
}

// Open 仅供服务端受控流式读取，避免浏览器直接持有 MinIO 地址或长期凭证。
func (store *MinIO) Open(ctx context.Context, objectKey string) (ObjectReader, error) {
	info, err := store.client.StatObject(ctx, store.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return ObjectReader{}, fmt.Errorf("读取对象元数据失败: %w", err)
	}
	object, err := store.client.GetObject(ctx, store.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return ObjectReader{}, fmt.Errorf("打开对象失败: %w", err)
	}
	return ObjectReader{Bucket: store.bucket, Reader: object, Size: info.Size, ContentType: info.ContentType}, nil
}

func (store *MinIO) Delete(ctx context.Context, objectKey string) error {
	if err := store.client.RemoveObject(ctx, store.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("删除对象失败: %w", err)
	}
	return nil
}
