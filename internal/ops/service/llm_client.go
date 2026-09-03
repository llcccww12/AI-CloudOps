package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func resolveLLMConfig() (apiKey, baseURL, model string, err error) {
	apiKey = strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(viper.GetString("external.llm.api_key"))
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(viper.GetString("llm.api_key"))
	}
	if apiKey == "" {
		return "", "", "", fmt.Errorf("未配置 LLM_API_KEY")
	}

	baseURL = strings.TrimRight(strings.TrimSpace(os.Getenv("LLM_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(viper.GetString("external.llm.base_url")), "/")
	}
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(viper.GetString("llm.base_url")), "/")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model = strings.TrimSpace(os.Getenv("LLM_MODEL"))
	if model == "" {
		model = strings.TrimSpace(viper.GetString("external.llm.model"))
	}
	if model == "" {
		model = strings.TrimSpace(viper.GetString("llm.model"))
	}
	if model == "" {
		if strings.Contains(strings.ToLower(baseURL), "minimaxi") {
			model = "MiniMax-M3"
		} else {
			model = "gpt-4o-mini"
		}
	}
	return apiKey, baseURL, model, nil
}

func truncateLLMErrBody(body []byte) string {
	s := strings.TrimSpace(string(body))
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) > 240 {
		return string(runes[:240]) + "..."
	}
	return s
}
