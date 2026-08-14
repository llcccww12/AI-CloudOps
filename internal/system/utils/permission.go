package utils

import (
	"strings"
)

var methodMapping = map[string]int8{
	"GET":    1,
	"POST":   2,
	"PUT":    3,
	"DELETE": 4,
}

// MethodCode 将 HTTP 方法转为系统 API 方法码
func MethodCode(method string) (int8, bool) {
	code, ok := methodMapping[strings.ToUpper(strings.TrimSpace(method))]
	return code, ok
}

// MatchAPIPath 与鉴权中间件一致的路径匹配（精确 + 通配符）
func MatchAPIPath(apiPath, requestPath string, methodCode, apiMethod int8) bool {
	if apiMethod != methodCode {
		return false
	}
	if apiPath == requestPath {
		return true
	}
	if apiPath == "/*" {
		return true
	}
	if !strings.Contains(apiPath, "*") {
		return false
	}
	if strings.HasSuffix(apiPath, "*") {
		prefix := strings.TrimSuffix(apiPath, "*")
		return strings.HasPrefix(requestPath, prefix)
	}
	if strings.HasPrefix(apiPath, "*") {
		suffix := strings.TrimPrefix(apiPath, "*")
		return strings.HasSuffix(requestPath, suffix)
	}
	if strings.Count(apiPath, "*") == 1 {
		parts := strings.Split(apiPath, "*")
		return strings.HasPrefix(requestPath, parts[0]) && strings.HasSuffix(requestPath, parts[1])
	}
	return false
}
