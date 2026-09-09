package main

import (
	"context"
	"flag"
	"github.com/indu-forge/dev_core/internal/imagecatalog"
	"github.com/indu-forge/dev_core/internal/objectstore"
	"log"
	"net"
	"os"
)

func main() {
	dir := flag.String("dir", "", "包含 manifest.json 和镜像归档的目录")
	flag.Parse()
	if *dir == "" {
		log.Fatal("必须指定 --dir")
	}
	ctx := context.Background()
	store, e := objectstore.NewMinIO(ctx, objectstore.Config{Endpoint: net.JoinHostPort(env("IF_OBJECT_STORE_ENDPOINT", "127.0.0.1"), env("IF_OBJECT_STORE_PORT", "18500")), AccessKey: os.Getenv("IF_OBJECT_STORE_ACCESS_KEY"), SecretKey: os.Getenv("IF_OBJECT_STORE_SECRET_KEY"), Bucket: env("IF_OBJECT_STORE_BUCKET_IFP", "ifp-artifacts"), Region: env("IF_OBJECT_STORE_REGION", "us-east-1"), UseSSL: os.Getenv("IF_OBJECT_STORE_USE_SSL") == "true"})
	if e != nil {
		log.Fatal(e)
	}
	if e = (&imagecatalog.Catalog{Store: store}).Import(ctx, *dir); e != nil {
		log.Fatal(e)
	}
	log.Print("中心镜像资源导入完成")
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
