package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/indu-forge/runtime-engine/internal/provisioner"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "prepare" {
		fmt.Fprintln(os.Stderr, "usage: if-runtime-provisioner prepare --input FILE")
		os.Exit(2)
	}
	flags := flag.NewFlagSet("prepare", flag.ContinueOnError)
	inputPath := flags.String("input", "", "runtime-binding.input.v1 文件")
	if err := flags.Parse(os.Args[2:]); err != nil || *inputPath == "" {
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
