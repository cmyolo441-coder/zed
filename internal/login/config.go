package login

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the values collected during the login flow.
type Config struct {
	Provider  string `json:"provider"`
	APIKey    string `json:"api_key"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
	AccountID string `json:"account_id,omitempty"` // Cloudflare only
}

// ConfigPath returns the path to ~/.config/zed/config.json.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zed", "config.json")
}

// Save writes (or merges) the login values into the config file.
func Save(cfg *Config) error {
	path := ConfigPath()

	// Read existing config if any.
	existing := make(map[string]interface{})
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	// Merge login values.
	if cfg.Provider != "" {
		existing["provider"] = cfg.Provider
	}
	if cfg.APIKey != "" {
		existing["api_key"] = cfg.APIKey
	}
	if cfg.BaseURL != "" {
		existing["base_url"] = cfg.BaseURL
	}
	if cfg.Model != "" {
		existing["model"] = cfg.Model
	}
	if cfg.AccountID != "" {
		existing["account_id"] = cfg.AccountID
	}
	existing["auth_header"] = ""

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load reads the current config file and returns the relevant fields.
func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	cfg := &Config{}
	_ = json.Unmarshal(data, cfg)
	return cfg, nil
}

// ProviderInfo holds metadata for the interactive provider list.
type ProviderInfo struct {
	Name        string
	Description string
	BaseURL     string
	Models      []string
}

// Providers returns the list of supported providers for the /login flow.
var Providers = []ProviderInfo{
	{
		Name:        "nvidia",
		Description: "NVIDIA API — free models, 1M context",
		BaseURL:     "https://integrate.api.nvidia.com/v1/chat/completions",
		Models:      []string{"z-ai/glm-5.2", "minimaxai/minimax-m3", "deepseek-ai/deepseek-v4-pro"},
	},
	{
		Name:        "opencode",
		Description: "OpenCode zen endpoint — free models",
		BaseURL:     "https://opencode.ai/zen/v1/chat/completions",
		Models:      []string{"mimo-v2.5-free", "deepseek-v4-flash-free", "big-pickle"},
	},
	{
		Name:        "cloudflare",
		Description: "Cloudflare Workers AI — GLM-5.2, Kimi K2.7 Code",
		BaseURL:     "https://api.cloudflare.com/client/v4/accounts/ACCOUNT_ID/ai/v1/chat/completions",
		Models:      []string{"@cf/zai-org/glm-5.2", "@cf/moonshotai/kimi-k2.7-code"},
	},
	{
		Name:        "custom",
		Description: "Any OpenAI-compatible endpoint",
		BaseURL:     "",
		Models:      []string{"gpt-4o", "claude-3.5-sonnet", "mimo-v2.5-pro"},
	},
}

// ProviderByName returns provider info by name.
func ProviderByName(name string) (*ProviderInfo, error) {
	for i := range Providers {
		if Providers[i].Name == name {
			return &Providers[i], nil
		}
	}
	return nil, fmt.Errorf("unknown provider: %s", name)
}
