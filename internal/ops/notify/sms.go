package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// SMSSender 短信发送接口
type SMSSender interface {
	Enabled() bool
	Send(ctx context.Context, mobile, content string) error
}

type AliyunSMSSender struct {
	logger *zap.Logger
	client *http.Client
}

func NewAliyunSMSSender(logger *zap.Logger) *AliyunSMSSender {
	return &AliyunSMSSender{
		logger: logger,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *AliyunSMSSender) Enabled() bool {
	if !viper.GetBool("ops.sms.enabled") {
		return false
	}
	ak := strings.TrimSpace(viper.GetString("ops.sms.access_key_id"))
	if ak == "" {
		ak = strings.TrimSpace(viper.GetString("external.aliyun.access_key_id"))
	}
	sk := strings.TrimSpace(viper.GetString("ops.sms.access_key_secret"))
	if sk == "" {
		sk = strings.TrimSpace(viper.GetString("external.aliyun.access_key_secret"))
	}
	sign := strings.TrimSpace(viper.GetString("ops.sms.sign_name"))
	tpl := strings.TrimSpace(viper.GetString("ops.sms.template_code"))
	return ak != "" && sk != "" && sign != "" && tpl != ""
}

func (s *AliyunSMSSender) Send(ctx context.Context, mobile, content string) error {
	if !s.Enabled() {
		return fmt.Errorf("短信未配置")
	}
	mobile = strings.TrimSpace(mobile)
	if mobile == "" {
		return fmt.Errorf("手机号为空")
	}
	ak := strings.TrimSpace(viper.GetString("ops.sms.access_key_id"))
	if ak == "" {
		ak = strings.TrimSpace(viper.GetString("external.aliyun.access_key_id"))
	}
	sk := strings.TrimSpace(viper.GetString("ops.sms.access_key_secret"))
	if sk == "" {
		sk = strings.TrimSpace(viper.GetString("external.aliyun.access_key_secret"))
	}
	signName := strings.TrimSpace(viper.GetString("ops.sms.sign_name"))
	templateCode := strings.TrimSpace(viper.GetString("ops.sms.template_code"))
	endpoint := strings.TrimSpace(viper.GetString("ops.sms.endpoint"))
	if endpoint == "" {
		endpoint = "https://dysmsapi.aliyuncs.com/"
	}

	paramJSON, _ := json.Marshal(map[string]string{"content": truncateSMS(content, 200)})
	params := map[string]string{
		"AccessKeyId":      ak,
		"Action":           "SendSms",
		"Format":           "JSON",
		"PhoneNumbers":     mobile,
		"RegionId":         "cn-hangzhou",
		"SignName":         signName,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   fmt.Sprintf("%d", time.Now().UnixNano()),
		"SignatureVersion": "1.0",
		"TemplateCode":     templateCode,
		"TemplateParam":    string(paramJSON),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Version":          "2017-05-25",
	}
	params["Signature"] = signAliyun(params, sk)

	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	_ = json.Unmarshal(body, &result)
	if result.Code != "" && !strings.EqualFold(result.Code, "OK") {
		return fmt.Errorf("短信发送失败: %s %s", result.Code, result.Message)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("短信发送失败: HTTP %d %s", resp.StatusCode, string(body))
	}
	return nil
}

func truncateSMS(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func signAliyun(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(specialURLEncode(k))
		buf.WriteByte('=')
		buf.WriteString(specialURLEncode(params[k]))
	}
	stringToSign := "POST&" + specialURLEncode("/") + "&" + specialURLEncode(buf.String())
	mac := hmac.New(sha1.New, []byte(secret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func specialURLEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
