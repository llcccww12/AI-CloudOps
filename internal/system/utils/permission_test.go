package utils

import "testing"

func TestMethodCode(t *testing.T) {
	code, ok := MethodCode("get")
	if !ok || code != 1 {
		t.Fatalf("GET expected 1, got %d ok=%v", code, ok)
	}
	if _, ok := MethodCode("PATCH"); ok {
		t.Fatal("PATCH should be unsupported")
	}
}

func TestMatchAPIPath(t *testing.T) {
	cases := []struct {
		apiPath, reqPath string
		method, apiMethod int8
		want              bool
	}{
		{"/api/user/list", "/api/user/list", 1, 1, true},
		{"/api/user/list", "/api/user/list", 1, 2, false},
		{"/api/user/*", "/api/user/detail/1", 1, 1, true},
		{"/api/user/*", "/api/role/list", 1, 1, false},
		{"/*", "/anything", 2, 2, true},
		{"*/detail", "/api/user/detail", 1, 1, true},
		{"/api/*/list", "/api/user/list", 1, 1, true},
		{"/api/*/list", "/api/user/create", 1, 1, false},
	}
	for _, c := range cases {
		got := MatchAPIPath(c.apiPath, c.reqPath, c.method, c.apiMethod)
		if got != c.want {
			t.Fatalf("MatchAPIPath(%q,%q,%d,%d)=%v want %v", c.apiPath, c.reqPath, c.method, c.apiMethod, got, c.want)
		}
	}
}
