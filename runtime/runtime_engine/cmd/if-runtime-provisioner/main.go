package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/indu-forge/runtime-engine/internal/provision"
	"github.com/indu-forge/runtime-engine/internal/provisioner"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: if-runtime-provisioner <prepare|provision-nats|provision-state|unpack-runtime> --input FILE")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "prepare":
		runPrepare(os.Args[2:])
	case "provision-nats":
		runProvisionNATS(os.Args[2:])
	case "provision-state":
		runProvisionState(os.Args[2:])
	case "unpack-runtime":
		runUnpackRuntime(os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, "usage: if-runtime-provisioner <prepare|provision-nats|provision-state|unpack-runtime> --input FILE")
		os.Exit(2)
	}
}

func runUnpackRuntime(args []string) {
	flags := flag.NewFlagSet("unpack-runtime", flag.ContinueOnError)
	archive := flags.String("archive", "", "已验签 Release 中的 runtime-artifact.tar.zst")
	target := flags.String("target", "", "emptyDir 解包目录")
	if err := flags.Parse(args); err != nil || *archive == "" || *target == "" {
		fmt.Fprintln(os.Stderr, "unpack-runtime: argument invalid")
		os.Exit(2)
	}
	file, err := os.Open(*archive)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unpack-runtime: archive read failed")
		os.Exit(1)
	}
	defer file.Close()
	if err := provision.UnpackRuntimeArtifact(file, *target, provision.Limits{MaxFiles: 10000, MaxFileBytes: 128 << 20, MaxTotalBytes: 512 << 20}); err != nil {
		fmt.Fprintln(os.Stderr, "unpack-runtime: failed")
		os.Exit(1)
	}
}

func runPrepare(args []string) {
	flags := flag.NewFlagSet("prepare", flag.ContinueOnError)
	inputPath := flags.String("input", "", "runtime-binding.input.v1 文件")
	if err := flags.Parse(args); err != nil || *inputPath == "" {
		fmt.Fprintln(os.Stderr, "prepare: input invalid")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "prepare: input read failed")
		os.Exit(1)
	}
	in, err := provisioner.Decode(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "prepare: input invalid")
		os.Exit(1)
	}
	if err := provisioner.Prepare(in); err != nil {
		fmt.Fprintln(os.Stderr, "prepare: failed")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "prepare: complete")
}

func runProvisionNATS(args []string) {
	flags := flag.NewFlagSet("provision-nats", flag.ContinueOnError)
	inputPath := flags.String("input", "", "runtime-binding.input.v1 文件")
	credentialsPath := flags.String("credentials", "", "NATS bootstrap 凭据文件")
	if err := flags.Parse(args); err != nil || *inputPath == "" || *credentialsPath == "" {
		fmt.Fprintln(os.Stderr, "provision-nats: argument invalid")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "provision-nats: input read failed")
		os.Exit(1)
	}
	in, err := provisioner.Decode(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "provision-nats: input invalid")
		os.Exit(1)
	}
	credentials, err := os.ReadFile(*credentialsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "provision-nats: credentials read failed")
		os.Exit(1)
	}
	if err := provisioner.ProvisionNATS(context.Background(), in, credentials); err != nil {
		// provisioner 只返回受控阶段与资源名，不包含 token、endpoint 或 broker 原始错误。
		fmt.Fprintf(os.Stderr, "provision-nats: failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "provision-nats: complete")
}

func runProvisionState(args []string) {
	flags := flag.NewFlagSet("provision-state", flag.ContinueOnError)
	inputPath := flags.String("input", "", "runtime-binding.input.v1 文件")
	credentialsPath := flags.String("credentials", "", "PostgreSQL bootstrap 凭据文件")
	if err := flags.Parse(args); err != nil || *inputPath == "" || *credentialsPath == "" {
		fmt.Fprintln(os.Stderr, "provision-state: argument invalid")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "provision-state: input read failed")
		os.Exit(1)
	}
	in, err := provisioner.Decode(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "provision-state: input invalid")
		os.Exit(1)
	}
	credentials, err := os.ReadFile(*credentialsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "provision-state: credentials read failed")
		os.Exit(1)
	}
	if err := provisioner.ProvisionState(context.Background(), in, credentials); err != nil {
		// provisioner 只返回受控阶段与资源名，不包含 DSN、密码或数据库原始错误。
		fmt.Fprintf(os.Stderr, "provision-state: failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "provision-state: complete")
}
