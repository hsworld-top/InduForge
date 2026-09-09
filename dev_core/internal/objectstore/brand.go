package objectstore

import "net/url"

// 品牌图片经中心同源读取，不向浏览器返回集群内部对象存储地址。
func BrandURL(tenantCode, key string) string {
	if key == "" {
		return ""
	}
	return "/api/v1/auth/config?" + url.Values{"tenantCode": {tenantCode}, "asset": {key}}.Encode()
}
