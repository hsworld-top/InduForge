//go:build !linux

package sandbox

import "fmt"

func RunVerifiedRuntime(_ []string) error { return fmt.Errorf("隔离运行仅支持 Linux") }
