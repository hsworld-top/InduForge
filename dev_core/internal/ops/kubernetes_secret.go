package ops

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// KubeSecretClient 是仅处理命名 Secret 的最小受控边界；调用方不得记录 data。
type KubeSecretClient interface {
	GetSecret(context.Context, string, string) (map[string]string, bool, error)
	ApplySecret(context.Context, string, string, map[string]string) error
	DeleteSecret(context.Context, string, string) error
}

func (r *KubernetesProjectReconciler) GetSecret(ctx context.Context, ns, name string) (map[string]string, bool, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+"/api/v1/namespaces/"+ns+"/secrets/"+name, nil)
	if e != nil {
		return nil, false, e
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, e := r.client.Do(req)
	if e != nil {
		return nil, false, e
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, false, nil
	}
	if resp.StatusCode/100 != 2 {
		return nil, false, fmt.Errorf("读取 Secret %s 失败: HTTP %d", name, resp.StatusCode)
	}
	var v struct {
		Data map[string]string `json:"data"`
	}
	if e = json.NewDecoder(resp.Body).Decode(&v); e != nil {
		return nil, false, e
	}
	decoded := make(map[string]string, len(v.Data))
	for key, value := range v.Data {
		plain, decodeErr := base64.StdEncoding.DecodeString(value)
		if decodeErr != nil {
			return nil, false, fmt.Errorf("读取 Secret %s 数据失败", name)
		}
		decoded[key] = string(plain)
	}
	return decoded, true, nil
}
func (r *KubernetesProjectReconciler) ApplySecret(ctx context.Context, ns, name string, data map[string]string) error {
	// stringData 不参与 Server-Side Apply 字段归属；对已存在的 Secret 做调和时，
	// 它无法可靠覆盖此前由本控制面写入的 data 键。显式编码 data 后再 SSA，
	// 才能让源凭据轮换真正触发部署专属 Secret 更新。
	encoded := make(map[string]string, len(data))
	for key, value := range data {
		encoded[key] = base64.StdEncoding.EncodeToString([]byte(value))
	}
	b, e := json.Marshal(map[string]any{"apiVersion": "v1", "kind": "Secret", "metadata": map[string]string{"name": name, "namespace": ns}, "type": "Opaque", "data": encoded})
	if e != nil {
		return e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPatch, r.endpoint+"/api/v1/namespaces/"+ns+"/secrets/"+name+"?fieldManager=induforge-center&force=true", strings.NewReader(string(b)))
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/apply-patch+yaml")
	resp, e := r.client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("应用 Secret %s 失败: HTTP %d", name, resp.StatusCode)
	}
	return nil
}
func (r *KubernetesProjectReconciler) DeleteSecret(ctx context.Context, ns, name string) error {
	req, e := http.NewRequestWithContext(ctx, http.MethodDelete, r.endpoint+"/api/v1/namespaces/"+ns+"/secrets/"+name, nil)
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, e := r.client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 && resp.StatusCode/100 != 2 {
		return fmt.Errorf("删除 Secret %s 失败: HTTP %d", name, resp.StatusCode)
	}
	return nil
}
