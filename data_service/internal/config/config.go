package config

import (
	"fmt"
	"net"
	"os"
	"strings"
)

const defaultAddr = ":9095"

// Config 定义 data_service 的基础运行配置。
type Config struct {
	Addr string
}

// Load 从环境变量读取服务配置，并在缺省时使用内置默认值。
func Load() (Config, error) {
	addr := strings.TrimSpace(os.Getenv("DATA_SERVICE_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}

	if err := validateAddr(addr); err != nil {
		return Config{}, err
	}

	return Config{
		Addr: addr,
	}, nil
}

// validateAddr 校验监听地址是否为合法的 TCP 地址。
func validateAddr(addr string) error {
	if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
		return fmt.Errorf("无效的 DATA_SERVICE_ADDR %q: %w", addr, err)
	}
	return nil
}
