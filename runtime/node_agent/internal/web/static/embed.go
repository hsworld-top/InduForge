package static

import (
	"embed"
	"io/fs"
	"net/http"
)

// embeddedFS 内置前端静态资源
//
//go:embed dist
var embeddedFS embed.FS

// DistFS 获取前端 dist 目录的 http.FileSystem
func DistFS() http.FileSystem {
	sub, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		// 正常情况下不会发生，返回空 FS 让上层 404
		return http.FS(osDirFS{})
	}
	return http.FS(sub)
}

// osDirFS 作为嵌入资源不可用时的占位实现
type osDirFS struct{}

func (osDirFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}
