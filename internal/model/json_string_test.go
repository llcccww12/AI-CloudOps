package model

import (
	"encoding/json"
	"testing"
)

func TestJSONStringUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    JSONString
		wantErr bool
	}{
		{name: "string", raw: `"500"`, want: "500"},
		{name: "number", raw: `500`, want: "500"},
		{name: "null", raw: `null`, want: ""},
		{name: "empty string", raw: `""`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got JSONString
			err := json.Unmarshal([]byte(tt.raw), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestCreateClusterReqAcceptsNumericQuantities(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"name":"k3s-remote",
		"cpu_request":500,
		"cpu_limit":2000,
		"memory_request":512,
		"memory_limit":2048,
		"restrict_namespace":"default",
		"api_server_addr":"https://192.168.239.201:6443",
		"kube_config_content":"apiVersion: v1"
	}`)

	var req CreateClusterReq
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.CpuRequest != "500" || req.MemoryLimit != "2048" {
		t.Fatalf("quantities not coerced: cpu=%q mem=%q", req.CpuRequest, req.MemoryLimit)
	}
	if len(req.RestrictNamespace) != 1 || req.RestrictNamespace[0] != "default" {
		t.Fatalf("restrict_namespace=%v", req.RestrictNamespace)
	}
}
