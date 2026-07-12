package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qualiguard/qualiguard/internal/model"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Project     model.Project `yaml:"project"`
	Brand       BrandConfig   `yaml:"brand"`
	Sources     []string      `yaml:"sources"`
	Exclusions  []string      `yaml:"exclusions"`
	Inclusions  []string      `yaml:"inclusions"`
	Encoding    string        `yaml:"encoding"`
	Languages   []string      `yaml:"languages"`
	QualityGate string        `yaml:"quality_gate"`
	AI          AIConfig      `yaml:"ai"`
}

type AIConfig struct {
	Enabled  bool            `yaml:"enabled"`
	Provider string          `yaml:"provider"` // openai | gemini | ollama | auto
	OpenAI   OpenAIConfig    `yaml:"openai"`
	Gemini   GeminiConfig    `yaml:"gemini"`
	Ollama   OllamaAIConfig  `yaml:"ollama"`
}

type OpenAIConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type GeminiConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type OllamaAIConfig struct {
	Enabled bool   `yaml:"enabled"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

func Default() Config {
	return Config{
		Project: model.Project{
			Key:  "default",
			Name: "Default Project",
		},
		Brand:      DefaultBrand(),
		Sources:    []string{"."},
		Exclusions: defaultExclusions(),
		Encoding:   "UTF-8",
		Languages:  []string{"python", "javascript", "go", "java", "csharp"},
		AI: AIConfig{
			Enabled:  true,
			Provider: "auto",
			OpenAI: OpenAIConfig{
				Model: "gpt-4o-mini",
			},
			Gemini: GeminiConfig{
				Model: "gemini-2.0-flash",
			},
			Ollama: OllamaAIConfig{
				Enabled: true,
				BaseURL: "http://127.0.0.1:11434/v1",
				Model:   "llama3.2",
			},
		},
	}
}

func defaultExclusions() []string {
	return []string{
		"**/__pycache__/**",
		"**/.venv/**",
		"**/venv/**",
		"**/node_modules/**",
		"**/dist/**",
		"**/build/**",
		"**/.git/**",
		"**/testdata/**",
	}
}

func Load(path string) (Config, error) {
	cfg := Default()

	if path == "" {
		for _, candidate := range []string{"qualiguard.yaml", "qualiguard.yml", ".qualiguard.yaml"} {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}

	if cfg.Project.Key == "" {
		return cfg, fmt.Errorf("project.key is required in config")
	}

	if len(cfg.Sources) == 0 {
		cfg.Sources = []string{"."}
	}

	if len(cfg.Exclusions) == 0 {
		cfg.Exclusions = defaultExclusions()
	}

	cfg.Brand = cfg.Brand.WithDefaults()

	return cfg, nil
}

func (c Config) ResolveSources(baseDir string) ([]string, error) {
	var resolved []string
	for _, src := range c.Sources {
		path := src
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, abs)
	}
	return resolved, nil
}
