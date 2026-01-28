package handler

import (
	"io"
	"net/http"
	"path"
	"strings"
)

// spaHandler 提供前端 SPA 路由的回退处理
type spaHandler struct {
	fs        http.FileSystem
	indexPath string
}

// newSPAHandler 创建 SPA 路由处理器
func newSPAHandler(fs http.FileSystem, indexPath string) http.Handler {
	return &spaHandler{
		fs:        fs,
		indexPath: indexPath,
	}
}

// ServeHTTP 处理静态资源与 SPA 回退逻辑
func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		h.serveIndex(w, r)
		return
	}

	filePath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if filePath == "" || filePath == "." {
		h.serveIndex(w, r)
		return
	}

	exists, isDir := h.fileExists(filePath)
	if exists && !isDir {
		http.FileServer(h.fs).ServeHTTP(w, r)
		return
	}

	if isDir {
		dirIndex := path.Join(filePath, "index.html")
		if ok, _ := h.fileExists(dirIndex); ok {
			http.FileServer(h.fs).ServeHTTP(w, r)
			return
		}
	}

	h.serveIndex(w, r)
}

// fileExists 判断文件是否存在并返回是否为目录
func (h *spaHandler) fileExists(name string) (bool, bool) {
	f, err := h.fs.Open(name)
	if err != nil {
		return false, false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false, false
	}

	return true, info.IsDir()
}

// serveIndex 输出 SPA 的入口文件
func (h *spaHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	f, err := h.fs.Open(h.indexPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
