package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type DatabaseConfig struct {
	Driver   string `mapstructure:"driver" yaml:"driver"`
	Host     string `mapstructure:"host" yaml:"host"`
	Port     int    `mapstructure:"port" yaml:"port"`
	Database string `mapstructure:"database" yaml:"database"`
	Username string `mapstructure:"username" yaml:"username"`
	Password string `mapstructure:"password" yaml:"password"`
	SSLMode  string `mapstructure:"sslmode" yaml:"sslmode"`
}

type MigrationConfig struct {
	Directory string `mapstructure:"directory" yaml:"directory"`
	Table     string `mapstructure:"table" yaml:"table"`
}

type BackupConfig struct {
	Directory     string `mapstructure:"directory" yaml:"directory"`
	Compression   bool   `mapstructure:"compression" yaml:"compression"`
	RetentionDays int    `mapstructure:"retention_days" yaml:"retention_days"`
}

type SeedConfig struct {
	Directory string `mapstructure:"directory" yaml:"directory"`
}

type Config struct {
	Database  DatabaseConfig  `mapstructure:"database" yaml:"database"`
	Migration MigrationConfig `mapstructure:"migration" yaml:"migration"`
	Backup    BackupConfig    `mapstructure:"backup" yaml:"backup"`
	Seed      SeedConfig      `mapstructure:"seed" yaml:"seed"`
	Verbose   bool            `mapstructure:"verbose" yaml:"verbose"`
}

func Load() (*Config, error) {
	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	if err := setDefaults(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults(cfg *Config) error {
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "postgres"
	}
	
	if cfg.Database.Host == "" {
		cfg.Database.Host = "localhost"
	}
	
	if cfg.Database.Port == 0 {
		switch cfg.Database.Driver {
		case "postgres":
			cfg.Database.Port = 5432
		case "mysql":
			cfg.Database.Port = 3306
		case "sqlite3":
			cfg.Database.Port = 0
		}
	}
	
	if cfg.Migration.Directory == "" {
		cfg.Migration.Directory = "./migrations"
	}
	
	if cfg.Migration.Table == "" {
		cfg.Migration.Table = "schema_migrations"
	}
	
	if cfg.Backup.Directory == "" {
		cfg.Backup.Directory = "./backups"
	}
	
	if cfg.Backup.RetentionDays == 0 {
		cfg.Backup.RetentionDays = 30
	}
	
	if cfg.Seed.Directory == "" {
		cfg.Seed.Directory = "./seeds"
	}

	return nil
}

func (c *Config) GetDSN() string {
	switch c.Database.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Database.Host, c.Database.Port, c.Database.Username, 
			c.Database.Password, c.Database.Database, c.Database.SSLMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			c.Database.Username, c.Database.Password, 
			c.Database.Host, c.Database.Port, c.Database.Database)
	case "sqlite3":
		return c.Database.Database
	default:
		return ""
	}
}