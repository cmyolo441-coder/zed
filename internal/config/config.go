package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DefaultBaseURL is the OpenAI-compatible endpoint ZED talks to by default.
const DefaultBaseURL = "https://opencode.ai/zen/v1/chat/completions"

// DefaultModel is used when no model is configured.
const DefaultModel = "mimo-v2.5-free"

// DefaultMaxTokens is the default output token budget per turn.
// Individual models may support less; the agent caps to model limit.
const DefaultMaxTokens = 128000

// ModelInfo holds metadata for a known model.
type ModelInfo struct {
	Name       string // model identifier
	MaxTokens  int    // maximum output tokens the model supports
	PricePer1M float64 // cost per 1 million output tokens (USD)
	BaseURL    string // provider endpoint override (empty = use config default)
	APIKey     string // provider API key override (empty = use config default)
	AuthHeader string // auth header name (default "Authorization: Bearer"); e.g. "api-key" for Xiaomi MiMo
}

// KnownModels maps model IDs to their metadata.
var KnownModels = map[string]ModelInfo{
	"mimo-v2.5-free":       {Name: "mimo-v2.5-free", MaxTokens: 128000, PricePer1M: 0.0},
	"deepseek-v4-flash-free": {Name: "deepseek-v4-flash-free", MaxTokens: 128000, PricePer1M: 0.0},
	"big-pickle":           {Name: "big-pickle", MaxTokens: 128000, PricePer1M: 0.0},
	// NVIDIA API models (https://integrate.api.nvidia.com/v1)
	"z-ai/glm-5.2":          {Name: "z-ai/glm-5.2", MaxTokens: 1000000, PricePer1M: 0.0, BaseURL: "https://integrate.api.nvidia.com/v1/chat/completions", APIKey: "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"},
	"minimaxai/minimax-m3":   {Name: "minimaxai/minimax-m3", MaxTokens: 1000000, PricePer1M: 0.0, BaseURL: "https://integrate.api.nvidia.com/v1/chat/completions", APIKey: "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"},
	"deepseek-ai/deepseek-v4-pro": {Name: "deepseek-ai/deepseek-v4-pro", MaxTokens: 1000000, PricePer1M: 0.0, BaseURL: "https://integrate.api.nvidia.com/v1/chat/completions", APIKey: "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"},
	// Xiaomi MiMo (https://api.xiaomimimo.com/v1) — uses api-key header, not Bearer
	"mimo-v2.5-pro": {Name: "mimo-v2.5-pro", MaxTokens: 1000000, PricePer1M: 0.0, BaseURL: "https://api.xiaomimimo.com/v1/chat/completions", APIKey: "sk-s7y2vork2snqeu6qsdh8wbjt8yplu9ykdwem9kky72881zda", AuthHeader: "api-key"},
}

// AvailableModels lists the models supported out of the box.
var AvailableModels = []string{
	// OpenCode endpoint models
	"mimo-v2.5-free",
	"deepseek-v4-flash-free",
	"big-pickle",
	// NVIDIA API models
	"z-ai/glm-5.2",
	"minimaxai/minimax-m3",
	"deepseek-ai/deepseek-v4-pro",
	// Xiaomi MiMo models
	"mimo-v2.5-pro",
}

// LookupModel returns model metadata, falling back to a sensible default.
func LookupModel(name string) ModelInfo {
	if m, ok := KnownModels[name]; ok {
		return m
	}
	return ModelInfo{Name: name, MaxTokens: DefaultMaxTokens, PricePer1M: 0.0}
}

// Config holds all runtime configuration for the ZED agent.
type Config struct {
	Provider   string `json:"provider"`   // "anthropic" | "openai" | "ollama"
	Model      string `json:"model"`      // model id
	APIKey     string `json:"api_key"`    // provider API key
	BaseURL    string `json:"base_url"`   // optional custom endpoint
	AuthHeader string `json:"auth_header"` // auth header name (empty = default "Authorization: Bearer")
	MaxTokens  int    `json:"max_tokens"` // max output tokens per turn
	MaxSteps   int    `json:"max_steps"`  // max ReAct iterations per task
	Theme      string `json:"theme"`      // ui theme
	AutoApply  bool   `json:"auto_apply"` // apply file edits without asking
	Effort     string `json:"effort"`     // effort level: normal|ultraeffort|ultramax|ultracombomax|goal|dream
	WorkDir    string `json:"-"`          // resolved working directory
}

// Default returns a config populated with sensible defaults and env overrides.
func Default() *Config {
	wd, _ := os.Getwd()
	c := &Config{
		Provider:  envOr("ZED_PROVIDER", "openai"),
		Model:     envOr("ZED_MODEL", DefaultModel),
		APIKey:    firstNonEmpty(os.Getenv("ZED_API_KEY"), os.Getenv("ANTHROPIC_API_KEY"), os.Getenv("OPENAI_API_KEY")),
		BaseURL:   envOr("ZED_BASE_URL", DefaultBaseURL),
		MaxTokens: DefaultMaxTokens,
		MaxSteps:  25,
		Theme:     envOr("ZED_THEME", "dracula"),
		AutoApply: false,
		Effort:    envOr("ZED_EFFORT", DefaultEffort),
		WorkDir:   wd,
	}
	return c
}

// Load reads config from ~/.config/zed/config.json if present, then applies
// environment overrides on top of the defaults.
func Load() (*Config, error) {
	c := Default()
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil // no file is fine, use defaults
		}
		return c, err
	}
	fileCfg := &Config{}
	if err := json.Unmarshal(data, fileCfg); err != nil {
		return c, err
	}
	mergeFile(c, fileCfg)
	// If the loaded model has provider-specific overrides (BaseURL/APIKey in
	// KnownModels), apply them. Env vars only win when the model uses the
	// default endpoint — otherwise the saved model/provider pairing is honoured.
	if info, ok := KnownModels[c.Model]; ok && info.BaseURL != "" {
		c.BaseURL = info.BaseURL
		c.APIKey = info.APIKey
		c.AuthHeader = info.AuthHeader
	} else {
		// For default-endpoint models, the key comes from env/default — never a
		// provider-specific key (e.g. nvapi-…) left over in the saved config.
		// Reset unconditionally so a persisted NVIDIA key can't leak into the
		// default endpoint and trigger a 401.
		c.APIKey = firstNonEmpty(
			os.Getenv("ZED_API_KEY"),
			os.Getenv("ANTHROPIC_API_KEY"),
			os.Getenv("OPENAI_API_KEY"),
		)
	}
	return c, nil
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "zed", "config.json")
}

func mergeFile(dst, src *Config) {
	if src.Provider != "" {
		dst.Provider = src.Provider
	}
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.APIKey != "" {
		dst.APIKey = src.APIKey
	}
	if src.BaseURL != "" {
		dst.BaseURL = src.BaseURL
	}
	if src.AuthHeader != "" {
		dst.AuthHeader = src.AuthHeader
	}
	if src.MaxTokens != 0 {
		dst.MaxTokens = src.MaxTokens
	}
	if src.MaxSteps != 0 {
		dst.MaxSteps = src.MaxSteps
	}
	if src.Theme != "" {
		dst.Theme = src.Theme
	}
	if src.Effort != "" {
		dst.Effort = src.Effort
	}
	dst.AutoApply = src.AutoApply
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ApplyModel updates cfg to use the given model, switching BaseURL and APIKey
// if the model has provider-specific overrides in KnownModels.
// If the model has no overrides, BOTH BaseURL and APIKey reset to the
// environment default — otherwise the previous model's key (e.g. an NVIDIA
// nvapi-… key) would leak into the new endpoint and cause a 401.
func ApplyModel(cfg *Config, modelName string) {
	cfg.Model = modelName
	if info, ok := KnownModels[modelName]; ok && info.BaseURL != "" {
		cfg.BaseURL = info.BaseURL
		cfg.AuthHeader = info.AuthHeader
		if info.APIKey != "" {
			cfg.APIKey = info.APIKey
		}
	} else {
		// Reset endpoint AND key to env/default for models without overrides.
		// The key MUST be reset too, or a provider-specific key from a prior
		// /model switch stays attached and the default endpoint rejects it.
		cfg.BaseURL = envOr("ZED_BASE_URL", DefaultBaseURL)
		cfg.AuthHeader = "" // reset to default Bearer auth
		cfg.APIKey = firstNonEmpty(
			os.Getenv("ZED_API_KEY"),
			os.Getenv("ANTHROPIC_API_KEY"),
			os.Getenv("OPENAI_API_KEY"),
		)
	}
}

// EffectiveModel returns the model that should actually be used for the given
// effort level. The user's selected model is always used so /model switches
// take effect in every effort mode (including goal mode).
func EffectiveModel(cfg *Config, effortName string) string {
	return cfg.Model
}

// CappedMaxTokens returns maxTokens capped to the current model's limit.
// This prevents sending a value higher than what the model supports.
func CappedMaxTokens(cfg *Config, maxTokens int) int {
	info := LookupModel(cfg.Model)
	if maxTokens > info.MaxTokens {
		return info.MaxTokens
	}
	return maxTokens
}

// Save writes the current config to ~/.config/zed/config.json so that
// preferences (like effort level, model) persist across restarts.
func (c *Config) Save() error {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	cfgDir := filepath.Join(dir, "zed")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return err
	}
	// Only save persistent fields (skip runtime-only fields).
	snapshot := &Config{
		Provider:   c.Provider,
		Model:      c.Model,
		BaseURL:    c.BaseURL,
		APIKey:     c.APIKey,
		AuthHeader: c.AuthHeader,
		MaxTokens:  c.MaxTokens,
		MaxSteps:  c.MaxSteps,
		Theme:     c.Theme,
		AutoApply: c.AutoApply,
		Effort:    c.Effort,
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cfgDir, "config.json"), data, 0o644)
}

// SetCustomOpenAICompatible configures the agent to talk to an arbitrary
// OpenAI-compatible endpoint with a user-supplied API key, base URL and model.
// This is what /login uses for the "custom" provider option.
func SetCustomOpenAICompatible(cfg *Config, apiKey, baseURL, model string) {
	cfg.Provider = "openai"
	cfg.APIKey = apiKey
	cfg.BaseURL = baseURL
	cfg.AuthHeader = "" // default Authorization: Bearer
	if model != "" {
		cfg.Model = model
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
