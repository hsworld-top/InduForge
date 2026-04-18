package config

import (
	"os"
	"strings"
)

const defaultAddr = ":9095"

// Config 定义 data_service 的基础运行配置。
type Config struct {
	Addr string
}

// Load 从环境变量读取服务配置，并在缺省时使用内置默认值。
func Load() Config {
	addr := strings.TrimSpace(os.Getenv("DATA_SERVICE_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}

	return Config{
		Addr: addr,
	}
}
