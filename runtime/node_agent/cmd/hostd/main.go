package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/indu-forge/node_agent/internal/hostd"
)

const defaultSocketPath = "/run/induforge/hostd.sock"

func main() {
	if os.Geteuid() != 0 {
		log.Fatal("induforge-node-hostd 必须以 root 运行")
	}
	config := hostd.DefaultManagerConfig()
	config.AssetsDir = envOr("INDUFORGE_HOSTD_ASSETS_DIR", config.AssetsDir)
	config.BinaryPath = envOr("INDUFORGE_HOSTD_K3S_BINARY", config.BinaryPath)
	config.ConfigDir = envOr("INDUFORGE_HOSTD_K3S_CONFIG_DIR", config.ConfigDir)
	config.StateDir = envOr("INDUFORGE_HOSTD_STATE_DIR", config.StateDir)
	manager, err := hostd.NewManager(config)
	if err != nil {
		log.Fatal(err)
	}
	api, err := hostd.NewAPI(manager)
	if err != nil {
		log.Fatal(err)
	}
	socketPath := os.Getenv("INDUFORGE_HOSTD_SOCKET")
	if socketPath == "" {
		socketPath = defaultSocketPath
	}
	listener, err := listenUnix(socketPath, envOr("INDUFORGE_HOSTD_GROUP", "induforge"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(socketPath)
	}()
	handler := api.Handler()
	if os.Getenv("INDUFORGE_HOSTD_IMAGES_ONLY") == "true" {
		handler = api.ImagesHandler()
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Minute,
		IdleTimeout:       30 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	log.Printf("InduForge Hostd listening on unix://%s", socketPath)
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func listenUnix(socketPath, groupName string) (net.Listener, error) {
	if !filepath.IsAbs(socketPath) || filepath.Clean(socketPath) != socketPath {
		return nil, fmt.Errorf("hostd socket 必须是规范化绝对路径")
	}
	group, err := user.LookupGroup(groupName)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.Atoi(group.Gid)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0750); err != nil {
		return nil, err
	}
	// socket 权限只有在父目录允许 Agent 组穿过时才真正生效。安装升级可能
	// 留下 root:root 目录，因此 Hostd 每次启动都收敛目录所有权和权限。
	if err := os.Chown(filepath.Dir(socketPath), 0, gid); err != nil {
		return nil, err
	}
	if err := os.Chmod(filepath.Dir(socketPath), 0750); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(socketPath); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("拒绝覆盖非 socket 文件: %s", socketPath)
		}
		if err := os.Remove(socketPath); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	if err := os.Chown(socketPath, 0, gid); err != nil {
		listener.Close()
		return nil, err
	}
	if err := os.Chmod(socketPath, 0660); err != nil {
		listener.Close()
		return nil, err
	}
	return listener, nil
}
