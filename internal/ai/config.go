package ai

import (
	"os"
	"strings"

	"github.com/qualiguard/qualiguard/internal/config"
)

type Config struct {
	Enabled  bool
	Provider string
}

var yamlSettings config.AIConfig

func ConfigureFromYAML(cfg config.AIConfig) {
	yamlSettings = cfg
	applyYAMLToEnv(cfg)
}

func applyYAMLToEnv(cfg config.AIConfig) {
	if cfg.Provider != "" && envFirst("QUALIGUARD_AI_PROVIDER") == "" {
		_ = os.Setenv("QUALIGUARD_AI_PROVIDER", cfg.Provider)
	}
	if cfg.OpenAI.APIKey != "" && envFirst("QUALIGUARD_OPENAI_API_KEY", "OPENAI_API_KEY") == "" {
		_ = os.Setenv("QUALIGUARD_OPENAI_API_KEY", cfg.OpenAI.APIKey)
	}
	if cfg.OpenAI.Model != "" && envFirst("QUALIGUARD_OPENAI_MODEL", "OPENAI_MODEL") == "" {
		_ = os.Setenv("QUALIGUARD_OPENAI_MODEL", cfg.OpenAI.Model)
	}
	if cfg.Gemini.APIKey != "" && envFirst("QUALIGUARD_GEMINI_API_KEY", "GEMINI_API_KEY", "GOOGLE_API_KEY") == "" {
		_ = os.Setenv("QUALIGUARD_GEMINI_API_KEY", cfg.Gemini.APIKey)
	}
	if cfg.Gemini.Model != "" && envFirst("QUALIGUARD_GEMINI_MODEL", "GEMINI_MODEL") == "" {
		_ = os.Setenv("QUALIGUARD_GEMINI_MODEL", cfg.Gemini.Model)
	}
	if cfg.Ollama.Enabled && envFirst("QUALIGUARD_OLLAMA_ENABLED") == "" {
		_ = os.Setenv("QUALIGUARD_OLLAMA_ENABLED", "1")
	}
	if cfg.Ollama.BaseURL != "" && envFirst("QUALIGUARD_OLLAMA_BASE_URL", "OLLAMA_BASE_URL") == "" {
		_ = os.Setenv("QUALIGUARD_OLLAMA_BASE_URL", cfg.Ollama.BaseURL)
	}
	if cfg.Ollama.Model != "" && envFirst("QUALIGUARD_OLLAMA_MODEL", "OLLAMA_MODEL") == "" {
		_ = os.Setenv("QUALIGUARD_OLLAMA_MODEL", cfg.Ollama.Model)
	}
}

func LoadConfig() Config {
	enabled := yamlSettings.Enabled
	if yamlSettings == (config.AIConfig{}) {
		enabled = true
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("QUALIGUARD_AI_ENABLED"))) {
	case "0", "false":
		enabled = false
	case "1", "true":
		enabled = true
	}

	provider := envFirst("QUALIGUARD_AI_PROVIDER")
	if provider == "" {
		provider = strings.TrimSpace(yamlSettings.Provider)
	}
	if provider == "" {
		provider = "auto"
	}

	return Config{Enabled: enabled, Provider: provider}
}

func envFirst(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}
