package runtimeaccess

import (
	"net/http/httptest"
	"testing"
)

func TestListQueryBounds(t *testing.T) {
	q := parseListQuery(httptest.NewRequest("GET", "/?page=-1&limit=999&keyword=%20hello%20", nil))
	if q.Page != 1 || q.Limit != 100 || q.Keyword != "hello" {
		t.Fatalf("unexpected query %+v", q)
	}
	q = parseListQuery(httptest.NewRequest("GET", "/", nil))
	if q.Page != 1 || q.Limit != 10 {
		t.Fatal(q)
	}
}
