package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Config holds the application configuration.
type Config struct {
	DefaultAgent        string                 `json:"default_agent"`
	APIAddr             string                 `json:"api_addr,omitempty"`
	SaveDir             string                 `json:"save_dir,omitempty"`
	StreamUpdates       bool                   `json:"stream_updates,omitempty"`
	StreamIntervalMS    int                    `json:"stream_interval_ms,omitempty"`
	StreamMaxChunkChars int                    `json:"stream_max_chunk_chars,omitempty"`
	StreamToolEvents    bool                   `json:"stream_tool_events"`
	Agents              map[string]AgentConfig `json:"agents"`
}

// AgentConfig holds configuration for a single agent.
type AgentConfig struct {
	Type          string            `json:"type"`                     // "acp", "cli", or "http"
	Command       string            `json:"command,omitempty"`        // binary path (cli/acp type)
	Args          []string          `json:"args,omitempty"`           // extra args for command (e.g. ["acp"] for cursor)
	Aliases       []string          `json:"aliases,omitempty"`        // custom trigger commands (e.g. ["gpt", "4o"])
	Cwd           string            `json:"cwd,omitempty"`            // working directory (workspace)
	Env           map[string]string `json:"env,omitempty"`            // extra environment variables (cli/acp type)
	Model         string            `json:"model,omitempty"`          // model name
	ModelProvider string            `json:"model_provider,omitempty"` // model provider name
	SystemPrompt  string            `json:"system_prompt,omitempty"`  // system prompt
	Endpoint      string            `json:"endpoint,omitempty"`       // API endpoint (http type)
	APIKey        string            `json:"api_key,omitempty"`        // API key (http type)
	Headers       map[string]string `json:"headers,omitempty"`        // extra HTTP headers (http type)
	MaxHistory    int               `json:"max_history,omitempty"`    // max history (http type)
}

// BuildAliasMap builds a map from custom alias to agent name from all agent configs.
// It logs warnings for conflicts: duplicate aliases and aliases shadowing agent keys.
func BuildAliasMap(agents map[string]AgentConfig) map[string]string {
	// Built-in commands that cannot be overridden
	reserved := map[string]bool{
		"info": true, "help": true, "new": true, "clear": true, "cwd": true,
	}

	m := make(map[string]string)
	for name, cfg := range agents {
		for _, alias := range cfg.Aliases {
			if reserved[alias] {
				log.Printf("[config] WARNING: alias %q for agent %q conflicts with built-in command, ignored", alias, name)
				continue
			}
			if existing, ok := m[alias]; ok {
				log.Printf("[config] WARNING: alias %q is defined by both %q and %q, using %q", alias, existing, name, name)
			}
			m[alias] = name
		}
	}

	// Warn if a custom alias shadows an agent key
	for alias, target := range m {
		if _, isAgent := agents[alias]; isAgent && alias != target {
			log.Printf("[config] WARNING: alias %q (-> %q) shadows agent key %q", alias, target, alias)
		}
	}

	return m
}

// DefaultConfig returns an empty configuration.
func DefaultConfig() *Config {
	return &Config{
		StreamIntervalMS:    1500,
		StreamMaxChunkChars: 1200,
		StreamToolEvents:    true,
		Agents:              make(map[string]AgentConfig),
	}
}

// ConfigPath returns the path to the config file.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".weclaw", "config.json"), nil
}

// Load loads configuration from disk and environment variables.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := ConfigPath()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			loadEnv(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Agents == nil {
		cfg.Agents = make(map[string]AgentConfig)
	}
	applyConfigDefaults(cfg, data)

	loadEnv(cfg)
	return cfg, nil
}

func applyConfigDefaults(cfg *Config, data []byte) {
	if cfg.StreamIntervalMS <= 0 {
		cfg.StreamIntervalMS = 1500
	}
	if cfg.StreamMaxChunkChars <= 0 {
		cfg.StreamMaxChunkChars = 1200
	}
	if cfg.StreamUpdates && !hasJSONKey(data, "stream_tool_events") {
		cfg.StreamToolEvents = true
	}
}

func hasJSONKey(data []byte, key string) bool {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	_, ok := raw[key]
	return ok
}

func loadEnv(cfg *Config) {
	if v := os.Getenv("WECLAW_DEFAULT_AGENT"); v != "" {
		cfg.DefaultAgent = v
	}
	if v := os.Getenv("WECLAW_API_ADDR"); v != "" {
		cfg.APIAddr = v
	}
	if v := os.Getenv("WECLAW_SAVE_DIR"); v != "" {
		cfg.SaveDir = v
	}
}

// Save saves the configuration to disk.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0o600)
}
