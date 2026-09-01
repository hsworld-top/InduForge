package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/indu-forge/node_agent/internal/hostd"
)

func main() {
	if len(os.Args) < 2 {
		fatal("用法: induforge-node-hostctl status|apply|uninstall")
	}
	socket := os.Getenv("INDUFORGE_HOSTD_SOCKET")
	if socket == "" {
		socket = "/run/induforge/hostd.sock"
	}
	client, err := hostd.NewUnixClient(socket)
	if err != nil {
		fatal(err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var state hostd.ClusterState
	switch os.Args[1] {
	case "status":
		state, err = client.Status(ctx)
	case "apply":
		flags := flag.NewFlagSet("apply", flag.ExitOnError)
		planPath := flags.String("plan", "", "root-owned cluster plan JSON")
		_ = flags.Parse(os.Args[2:])
		if *planPath == "" {
			fatal("apply 必须指定 --plan")
		}
		var plan hostd.ClusterPlan
		if err = decodeFile(*planPath, &plan); err == nil {
			state, err = client.Apply(ctx, plan)
		}
	case "uninstall":
		flags := flag.NewFlagSet("uninstall", flag.ExitOnError)
		clusterID := flags.String("cluster-id", "", "recorded runtime cluster id")
		nodeID := flags.String("node-id", "", "recorded node id")
		purge := flags.Bool("purge-data", false, "兼容参数；受管 K3s 数据在卸载时始终清理")
		_ = flags.Parse(os.Args[2:])
		state, err = client.Uninstall(ctx, hostd.UninstallRequest{ClusterID: *clusterID, NodeID: *nodeID, PurgeData: *purge})
	case "uninstall-current":
		flags := flag.NewFlagSet("uninstall-current", flag.ExitOnError)
		purge := flags.Bool("purge-data", false, "兼容参数；受管 K3s 数据在卸载时始终清理")
		_ = flags.Parse(os.Args[2:])
		if state, err = client.Status(ctx); err == nil && state.ObservedState != "not-installed" {
			state, err = client.Uninstall(ctx, hostd.UninstallRequest{ClusterID: state.ClusterID, NodeID: state.NodeID, PurgeData: *purge})
		}
	default:
		fatal("不支持的 hostctl 操作")
	}
	if err != nil {
		fatal(err.Error())
	}
	output, err := json.Marshal(state)
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println(string(output))
}

func decodeFile(path string, destination any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
