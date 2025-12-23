package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// QueryConfig defines the configuration for the Query service
type QueryConfig struct {
	Server     QueryServerConfig     `yaml:"server"`
	Database   DatabaseConfig        `yaml:"database"`
	Blockchain QueryBlockchainConfig `yaml:"blockchain"`
	Logging    QueryLoggingConfig    `yaml:"logging"`
}

// QueryServerConfig defines HTTP server configuration for Query service
type QueryServerConfig struct {
	HTTPPort     int    `yaml:"http_port"`
	ReadTimeout  string `yaml:"read_timeout"`
	WriteTimeout string `yaml:"write_timeout"`
	IdleTimeout  string `yaml:"idle_timeout"`
}

// QueryBlockchainConfig defines blockchain client configuration for Query service
type QueryBlockchainConfig struct {
	Enabled          bool   `yaml:"enabled"`
	ChainMakerConfig string `yaml:"chainmaker_config"`
}

// QueryLoggingConfig defines logging configuration for Query service
type QueryLoggingConfig struct {
	Level        string `yaml:"level"`
	Format       string `yaml:"format"`
	AuditEnabled bool   `yaml:"audit_enabled"`
	AuditFile    string `yaml:"audit_file"`
}

// LoadQueryConfig loads query service configuration from the specified YAML file path
func LoadQueryConfig(path string) (*QueryConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", path, err)
	}

	var cfg QueryConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config file: %w", err)
	}

	// Set defaults
	cfg.SetDefaults()

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// SetDefaults sets reasonable default values for the query configuration
func (c *QueryConfig) SetDefaults() {
	// Server defaults
	if c.Server.HTTPPort <= 0 {
		// NOTE: This default must match the value in query.defaults.yml.
		c.Server.HTTPPort = 8083
	}
	if c.Server.ReadTimeout == "" {
		c.Server.ReadTimeout = "30s"
	}
	if c.Server.WriteTimeout == "" {
		c.Server.WriteTimeout = "30s"
	}
	if c.Server.IdleTimeout == "" {
		c.Server.IdleTimeout = "120s"
	}

	// Database defaults
	c.Database.SetDefaults()

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Logging.AuditFile == "" {
		c.Logging.AuditFile = "/var/log/query/audit.log"
	}
}

// Validate validates the query configuration
func (c *QueryConfig) Validate() error {
	// Validate server config
	if c.Server.HTTPPort <= 0 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid http_port: %d (must be between 1-65535)", c.Server.HTTPPort)
	}

	// Validate timeouts
	if _, err := time.ParseDuration(c.Server.ReadTimeout); err != nil {
		return fmt.Errorf("invalid read_timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.Server.WriteTimeout); err != nil {
		return fmt.Errorf("invalid write_timeout: %w", err)
	}
	if _, err := time.ParseDuration(c.Server.IdleTimeout); err != nil {
		return fmt.Errorf("invalid idle_timeout: %w", err)
	}

	// Validate database config
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config error: %w", err)
	}

	// Validate blockchain config
	if c.Blockchain.Enabled && c.Blockchain.ChainMakerConfig == "" {
		return fmt.Errorf("blockchain is enabled but chainmaker_config is not set")
	}

	return nil
}

// LogConfiguration logs the query configuration (excluding sensitive data)
func (c *QueryConfig) LogConfiguration() {
	fmt.Printf("Query Service Configuration:\n")
	fmt.Printf("  HTTP Port: %d\n", c.Server.HTTPPort)
	fmt.Printf("  Read Timeout: %s\n", c.Server.ReadTimeout)
	fmt.Printf("  Write Timeout: %s\n", c.Server.WriteTimeout)
	fmt.Printf("  Idle Timeout: %s\n", c.Server.IdleTimeout)
	fmt.Printf("  Blockchain Enabled: %v\n", c.Blockchain.Enabled)
	fmt.Printf("  Logging Level: %s\n", c.Logging.Level)
	fmt.Printf("  Audit Enabled: %v\n", c.Logging.AuditEnabled)
	c.Database.LogConfiguration()
}
