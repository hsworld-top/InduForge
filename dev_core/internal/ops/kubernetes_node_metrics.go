package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (r *KubernetesProjectReconciler) collectNodeResourceSummaries(ctx context.Context, nodes []KubernetesNode) error {
	metrics, metricsErr := r.loadNodeMetrics(ctx)
	var errs []error
	if metricsErr != nil {
		errs = append(errs, metricsErr)
	}
	for i := range nodes {
		if nodes[i].Labels["induforge.io/center-node"] != "true" {
			continue
		}
		summary := map[string]any{"system": map[string]any{
			"distribution":  nodes[i].OSImage,
			"kernelVersion": nodes[i].KernelVersion,
			"dataPath":      r.centerDataPath,
		}}
		if usage, ok := metrics[nodes[i].Name]; ok {
			cpuUsed, cpuOK := parseCPUQuantity(usage.CPU)
			cpuTotal, totalOK := parseCPUQuantity(nodes[i].CPUCapacity)
			if cpuOK && totalOK && cpuTotal > 0 {
				summary["cpu"] = map[string]any{"count": cpuTotal, "usedPercent": boundedPercent(cpuUsed / cpuTotal * 100)}
			}
			memoryUsed, memoryOK := parseByteQuantity(usage.Memory)
			memoryTotal, totalOK := parseByteQuantity(nodes[i].MemoryCapacity)
			if memoryOK && totalOK && memoryTotal > 0 {
				summary["memory"] = map[string]any{"totalBytes": uint64(memoryTotal), "usedPercent": boundedPercent(memoryUsed / memoryTotal * 100)}
			}
		}
		disk, err := r.loadNodeDiskUsage(ctx, nodes[i].Name)
		if err != nil {
			errs = append(errs, err)
		} else if disk.CapacityBytes > 0 {
			summary["disk"] = map[string]any{"totalBytes": disk.CapacityBytes, "usedPercent": boundedPercent(float64(disk.UsedBytes) / float64(disk.CapacityBytes) * 100)}
		}
		nodes[i].ResourceSummary = summary
	}
	if len(errs) > 0 {
		return fmt.Errorf("读取中心节点资源指标失败: %v", errs)
	}
	return nil
}

type kubernetesNodeUsage struct{ CPU, Memory string }

func (r *KubernetesProjectReconciler) loadNodeMetrics(ctx context.Context) (map[string]kubernetesNodeUsage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+"/apis/metrics.k8s.io/v1beta1/nodes", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Metrics API 返回 HTTP %d", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Usage struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			} `json:"usage"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	result := make(map[string]kubernetesNodeUsage, len(body.Items))
	for _, item := range body.Items {
		result[item.Metadata.Name] = kubernetesNodeUsage{CPU: item.Usage.CPU, Memory: item.Usage.Memory}
	}
	return result, nil
}

type nodeDiskUsage struct{ CapacityBytes, UsedBytes uint64 }

func (r *KubernetesProjectReconciler) loadNodeDiskUsage(ctx context.Context, nodeName string) (nodeDiskUsage, error) {
	path := "/api/v1/nodes/" + url.PathEscape(nodeName) + "/proxy/stats/summary"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+path, nil)
	if err != nil {
		return nodeDiskUsage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		return nodeDiskUsage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nodeDiskUsage{}, fmt.Errorf("节点 %s Kubelet summary 返回 HTTP %d", nodeName, resp.StatusCode)
	}
	var body struct {
		Node struct {
			FS struct {
				CapacityBytes, UsedBytes uint64
			} `json:"fs"`
		} `json:"node"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nodeDiskUsage{}, err
	}
	return nodeDiskUsage{CapacityBytes: body.Node.FS.CapacityBytes, UsedBytes: body.Node.FS.UsedBytes}, nil
}

func parseCPUQuantity(value string) (float64, bool) {
	return parseQuantity(value, map[string]float64{"n": 1e-9, "u": 1e-6, "m": 1e-3, "": 1})
}

func parseByteQuantity(value string) (float64, bool) {
	return parseQuantity(value, map[string]float64{
		"Ki": 1 << 10, "Mi": 1 << 20, "Gi": 1 << 30, "Ti": 1 << 40,
		"K": 1e3, "M": 1e6, "G": 1e9, "T": 1e12, "": 1,
	})
}

func parseQuantity(value string, suffixes map[string]float64) (float64, bool) {
	for _, suffix := range []string{"Ki", "Mi", "Gi", "Ti", "n", "u", "m", "K", "M", "G", "T", ""} {
		factor, ok := suffixes[suffix]
		if !ok || !strings.HasSuffix(value, suffix) {
			continue
		}
		number := strings.TrimSuffix(value, suffix)
		parsed, err := strconv.ParseFloat(number, 64)
		return parsed * factor, err == nil && parsed >= 0
	}
	return 0, false
}

func boundedPercent(value float64) float64 {
	return math.Round(math.Max(0, math.Min(100, value))*100) / 100
}
