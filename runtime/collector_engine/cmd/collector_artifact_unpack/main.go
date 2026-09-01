package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/indu-forge/collector-engine/internal/artifact"
)

func main() {
	flags := flag.NewFlagSet("collector-artifact-unpack", flag.ContinueOnError)
	archive := flags.String("archive", "", "已验签 Release 中的 collector-artifact.tar.zst")
	target := flags.String("target", "", "emptyDir 解包目录")
	if flags.Parse(os.Args[1:]) != nil || *archive == "" || *target == "" {
		fmt.Fprintln(os.Stderr, "collector-artifact-unpack: argument invalid")
		os.Exit(2)
	}
	file, err := os.Open(*archive)
	if err != nil {
		fmt.Fprintln(os.Stderr, "collector-artifact-unpack: archive read failed")
		os.Exit(1)
	}
	defer file.Close()
	if err = artifact.Unpack(file, *target, artifact.Limits{MaxFiles: 10000, MaxFileBytes: 128 << 20, MaxTotalBytes: 512 << 20}); err != nil {
		fmt.Fprintln(os.Stderr, "collector-artifact-unpack: failed")
		os.Exit(1)
	}
}
