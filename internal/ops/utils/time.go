package utils

import (
	"fmt"
	"strings"
	"time"
)

// ParseOptionalTime 解析可选时间；空串/nil 视为未填。
func ParseOptionalTime(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*s)
	if raw == "" || raw == "null" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t, nil
		}
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("时间格式无效: %s", raw)
}
