package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig    `yaml:"server"`
	Auth     AuthConfig      `yaml:"auth"`
	Database DatabaseConfig  `yaml:"database"`
	Providers []ProviderConfig `yaml:"providers"`
}

type ServerConfig struct {
	Port      int `yaml:"port"`
	AdminPort int `yaml:"admin_port"`
}

type AuthConfig struct {
	AdminPassword string `yaml:"admin_password"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type ProviderConfig struct {
	Name     string        `yaml:"name"`
	Type     string        `yaml:"type"` // "anthropic" or "openai"
	BaseURL  string        `yaml:"base_url"`
	APIKey   string        `yaml:"api_key"`
	Priority int           `yaml:"priority"`
	Weight   int           `yaml:"weight"`
	Models   []ModelMapping `yaml:"models"`
}

type ModelMapping struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables
	expanded := os.Expand(string(data), func(key string) string {
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		return "${" + key + "}"
	})

	cfg := &Config{
		Server: ServerConfig{
			Port:      8080,
			AdminPort: 8081,
		},
		Database: DatabaseConfig{
			Path: "./data/proxy.db",
		},
	}

	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, err
	}

	// Allow env overrides
	if v := os.Getenv("PROXY_PORT"); v != "" {
		// simple override
		_ = v
	}
	if v := os.Getenv("ADMIN_PASSWORD"); v != "" {
		cfg.Auth.AdminPassword = v
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Auth.AdminPassword == "" {
		c.Auth.AdminPassword = "changeme"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.AdminPort == 0 {
		c.Server.AdminPort = 8081
	}
	return nil
}

// ExpandEnvInString replaces ${VAR} patterns with environment variable values
func ExpandEnvInString(s string) string {
	if !strings.Contains(s, "${") {
		return s
	}
	return os.ExpandEnv(s)
}
