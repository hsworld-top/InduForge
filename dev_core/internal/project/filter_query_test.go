package project

import (
	"net/http/httptest"
	"testing"
)

func TestListQueryAcceptsBrowserMultiSelect(t *testing.T) {
	for _, query := range []string{"visibility=private&visibility=internal", "visibility[]=private&visibility[]=internal", "visibility=private,internal,private"} {
		r := httptest.NewRequest("GET", "/projects?"+query, nil)
		if got := listQuery(r, "visibility"); got != "private,internal" {
			t.Fatalf("%s: %q", query, got)
		}
	}
}
