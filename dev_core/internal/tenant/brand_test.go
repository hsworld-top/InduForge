package tenant

import "testing"

func TestBrandKeysRemainWithinTenant(t *testing.T) {
	item := Tenant{ID: "one", Settings: map[string]any{"captchaBackgrounds": []string{"tenants/one/captcha/image.png"}}}
	if err := validateBrandKeys(item); err != nil {
		t.Fatal(err)
	}
	item.Settings["captchaBackgrounds"] = []string{"tenants/two/captcha/image.png"}
	if validateBrandKeys(item) == nil {
		t.Fatal("cross tenant image accepted")
	}
	item.Settings["captchaBackgrounds"] = make([]string, 11)
	if validateBrandKeys(item) == nil {
		t.Fatal("oversized gallery accepted")
	}
	item.Settings = nil
	item.LogoObjectKey = "tenants/two/logo/image.png"
	if validateBrandKeys(item) == nil {
		t.Fatal("cross tenant logo accepted")
	}
}
